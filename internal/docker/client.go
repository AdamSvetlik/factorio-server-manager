package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const labelManaged = "factorio-server-manager"

// Client wraps the Docker SDK client with Factorio-specific helpers.
type Client struct {
	cli *client.Client
}

// NewClient creates a new Docker client using the environment (DOCKER_HOST, etc.).
func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("connect to Docker: %w", err)
	}
	return &Client{cli: cli}, nil
}

// Close releases the underlying Docker client connection.
func (c *Client) Close() error {
	return c.cli.Close()
}

// ContainerInfo holds runtime info about a container.
type ContainerInfo struct {
	ID      string
	Status  string
	State   string
	Uptime  time.Duration
	Image   string
}

// CreateServerContainer creates a new Factorio server container.
func (c *Client) CreateServerContainer(ctx context.Context, opts CreateOptions) (string, error) {
	gamePortBinding := nat.Port(fmt.Sprintf("%d/udp", opts.GamePort))
	rconPortBinding := nat.Port(fmt.Sprintf("%d/tcp", opts.RCONPort))

	hostConfig := &container.HostConfig{
		Binds: []string{
			fmt.Sprintf("%s:/factorio", opts.DataDir),
		},
		PortBindings: nat.PortMap{
			gamePortBinding: []nat.PortBinding{{HostPort: fmt.Sprintf("%d", opts.GamePort)}},
			rconPortBinding: []nat.PortBinding{{HostPort: fmt.Sprintf("%d", opts.RCONPort)}},
		},
		RestartPolicy: container.RestartPolicy{Name: "unless-stopped"},
	}

	containerConfig := &container.Config{
		Image: opts.Image,
		ExposedPorts: nat.PortSet{
			gamePortBinding: struct{}{},
			rconPortBinding: struct{}{},
		},
		Env: buildEnv(opts),
		Labels: map[string]string{
			labelManaged:     "true",
			"fsm.server":     opts.Name,
		},
	}

	resp, err := c.cli.ContainerCreate(ctx, containerConfig, hostConfig, nil, nil, containerName(opts.Name))
	if err != nil {
		return "", fmt.Errorf("create container: %w", err)
	}
	return resp.ID, nil
}

// StartContainer starts an existing container by ID.
func (c *Client) StartContainer(ctx context.Context, containerID string) error {
	if err := c.cli.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		return fmt.Errorf("start container: %w", err)
	}
	return nil
}

// StopContainer gracefully stops a container.
func (c *Client) StopContainer(ctx context.Context, containerID string) error {
	timeout := 30
	if err := c.cli.ContainerStop(ctx, containerID, container.StopOptions{Timeout: &timeout}); err != nil {
		return fmt.Errorf("stop container: %w", err)
	}
	return nil
}

// RemoveContainer forcefully removes a container.
func (c *Client) RemoveContainer(ctx context.Context, containerID string) error {
	if err := c.cli.ContainerRemove(ctx, containerID, container.RemoveOptions{Force: true}); err != nil {
		return fmt.Errorf("remove container: %w", err)
	}
	return nil
}

// InspectContainer returns runtime info for a container by ID.
func (c *Client) InspectContainer(ctx context.Context, containerID string) (*ContainerInfo, error) {
	info, err := c.cli.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("inspect container: %w", err)
	}

	ci := &ContainerInfo{
		ID:    info.ID[:12],
		State: info.State.Status,
		Image: info.Config.Image,
	}

	if info.State.Running {
		started, err := time.Parse(time.RFC3339Nano, info.State.StartedAt)
		if err == nil {
			ci.Uptime = time.Since(started)
		}
		ci.Status = "running"
	} else {
		ci.Status = info.State.Status
	}

	return ci, nil
}

// ContainerExists returns true if a container with the given name exists.
func (c *Client) ContainerExists(ctx context.Context, name string) (bool, string, error) {
	f := filters.NewArgs()
	f.Add("name", containerName(name))
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return false, "", fmt.Errorf("list containers: %w", err)
	}
	for _, cont := range containers {
		for _, n := range cont.Names {
			if strings.TrimPrefix(n, "/") == containerName(name) {
				return true, cont.ID, nil
			}
		}
	}
	return false, "", nil
}

// StreamLogs streams container logs to the given writer.
func (c *Client) StreamLogs(ctx context.Context, containerID string, follow bool, out io.Writer) error {
	logs, err := c.cli.ContainerLogs(ctx, containerID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     follow,
		Timestamps: false,
		Tail:       "100",
	})
	if err != nil {
		return fmt.Errorf("get logs: %w", err)
	}
	defer logs.Close()

	// Docker multiplexes stdout/stderr; strip the 8-byte header from each frame.
	buf := make([]byte, 4096)
	header := make([]byte, 8)
	for {
		_, err := io.ReadFull(logs, header)
		if err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return err
		}
		// bytes 4-7 are the frame size (big-endian)
		size := int(header[4])<<24 | int(header[5])<<16 | int(header[6])<<8 | int(header[7])
		for size > 0 {
			n := size
			if n > len(buf) {
				n = len(buf)
			}
			nr, err := io.ReadFull(logs, buf[:n])
			if nr > 0 {
				out.Write(buf[:nr]) //nolint:errcheck
			}
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					return nil
				}
				return err
			}
			size -= nr
		}
	}
	return nil
}

// PullImage pulls a Docker image, printing progress to stdout.
func (c *Client) PullImage(ctx context.Context, imageRef string) error {
	out, err := c.cli.ImagePull(ctx, imageRef, image.PullOptions{})
	if err != nil {
		return fmt.Errorf("pull image %s: %w", imageRef, err)
	}
	defer out.Close()
	io.Copy(os.Stdout, out) //nolint:errcheck
	return nil
}

// ImageExists checks whether an image is already present locally.
func (c *Client) ImageExists(ctx context.Context, imageRef string) (bool, error) {
	f := filters.NewArgs()
	f.Add("reference", imageRef)
	images, err := c.cli.ImageList(ctx, image.ListOptions{Filters: f})
	if err != nil {
		return false, fmt.Errorf("list images: %w", err)
	}
	return len(images) > 0, nil
}

// ListManagedContainers returns all containers managed by this tool.
func (c *Client) ListManagedContainers(ctx context.Context) ([]types.Container, error) {
	f := filters.NewArgs()
	f.Add("label", labelManaged+"=true")
	containers, err := c.cli.ContainerList(ctx, container.ListOptions{
		All:     true,
		Filters: f,
	})
	if err != nil {
		return nil, fmt.Errorf("list containers: %w", err)
	}
	return containers, nil
}

// CreateOptions carries parameters for creating a new server container.
type CreateOptions struct {
	Name         string
	DataDir      string
	Image        string
	GamePort     int
	RCONPort     int
	RCONPassword string
}

func containerName(serverName string) string {
	return "factorio-" + serverName
}

func buildEnv(opts CreateOptions) []string {
	env := []string{
		"LOAD_LATEST_SAVE=true",
		fmt.Sprintf("PORT=%d", opts.GamePort),
		fmt.Sprintf("RCON_PORT=%d", opts.RCONPort),
	}
	if opts.RCONPassword != "" {
		env = append(env, "RCON_PASSWORD="+opts.RCONPassword)
	}
	return env
}
