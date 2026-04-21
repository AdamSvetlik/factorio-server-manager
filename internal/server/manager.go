package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/AdamSvetlik/factorio-server-manager/internal/config"
	"github.com/AdamSvetlik/factorio-server-manager/internal/docker"
)

// ServerInfo combines config and runtime status.
type ServerInfo struct {
	Config  *config.ServerConfig
	State   string
	Uptime  time.Duration
}

// Manager orchestrates server lifecycle operations.
type Manager struct {
	cfg    *config.Manager
	docker *docker.Client
}

// NewManager creates a new server Manager.
func NewManager(cfg *config.Manager, docker *docker.Client) *Manager {
	return &Manager{cfg: cfg, docker: docker}
}

// CreateOptions carries user-supplied options for creating a server.
type CreateOptions struct {
	Name        string
	Description string
	ImageTag    string
	GamePort    int
	RCONPort    int
}

// Create creates a new server: directories, default config, Docker container.
func (m *Manager) Create(ctx context.Context, opts CreateOptions) (*config.ServerConfig, error) {
	// Validate name uniqueness
	if _, err := m.cfg.GetServer(opts.Name); err == nil {
		return nil, fmt.Errorf("server %q already exists", opts.Name)
	}

	imageRef := config.DefaultFactorioImage + ":" + opts.ImageTag

	// Ensure image is available
	exists, err := m.docker.ImageExists(ctx, imageRef)
	if err != nil {
		return nil, err
	}
	if !exists {
		fmt.Printf("Pulling image %s...\n", imageRef)
		if err := m.docker.PullImage(ctx, imageRef); err != nil {
			return nil, err
		}
	}

	// Create directory structure
	if err := m.cfg.InitServerDirs(opts.Name); err != nil {
		return nil, err
	}

	// Initialize default server-settings.json
	if err := m.cfg.InitServerSettings(opts.Name); err != nil {
		return nil, err
	}

	// Generate RCON password
	rconPwd, err := generatePassword(16)
	if err != nil {
		return nil, fmt.Errorf("generate rcon password: %w", err)
	}

	// Create Docker container
	createOpts := docker.CreateOptions{
		Name:         opts.Name,
		DataDir:      m.cfg.ServerDir(opts.Name),
		Image:        imageRef,
		GamePort:     opts.GamePort,
		RCONPort:     opts.RCONPort,
		RCONPassword: rconPwd,
	}
	containerID, err := m.docker.CreateServerContainer(ctx, createOpts)
	if err != nil {
		// Clean up dirs on failure
		m.cfg.RemoveServerDirs(opts.Name) //nolint:errcheck
		return nil, err
	}

	// Write rcon password file
	rconPwPath := m.cfg.ConfigDir(opts.Name) + "/rconpw"
	if err := writeFile(rconPwPath, []byte(rconPwd)); err != nil {
		return nil, fmt.Errorf("write rcon password: %w", err)
	}

	now := time.Now()
	srv := &config.ServerConfig{
		Name:         opts.Name,
		Description:  opts.Description,
		ImageTag:     opts.ImageTag,
		ContainerID:  containerID,
		GamePort:     opts.GamePort,
		RCONPort:     opts.RCONPort,
		RCONPassword: rconPwd,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := m.cfg.AddServer(srv); err != nil {
		return nil, err
	}

	return srv, nil
}

// Start starts a server's Docker container.
func (m *Manager) Start(ctx context.Context, name string) error {
	srv, err := m.cfg.GetServer(name)
	if err != nil {
		return err
	}
	return m.docker.StartContainer(ctx, srv.ContainerID)
}

// Stop stops a server's Docker container.
func (m *Manager) Stop(ctx context.Context, name string) error {
	srv, err := m.cfg.GetServer(name)
	if err != nil {
		return err
	}
	return m.docker.StopContainer(ctx, srv.ContainerID)
}

// Delete stops, removes the container, optionally removes data, and deregisters.
func (m *Manager) Delete(ctx context.Context, name string, removeData bool) error {
	srv, err := m.cfg.GetServer(name)
	if err != nil {
		return err
	}

	// Stop if running (ignore error — may already be stopped)
	m.docker.StopContainer(ctx, srv.ContainerID) //nolint:errcheck

	// Remove container
	if err := m.docker.RemoveContainer(ctx, srv.ContainerID); err != nil {
		return err
	}

	// Optionally remove data directory
	if removeData {
		if err := m.cfg.RemoveServerDirs(name); err != nil {
			return fmt.Errorf("remove server data: %w", err)
		}
	}

	return m.cfg.RemoveServer(name)
}

// Status returns the current status of a server.
func (m *Manager) Status(ctx context.Context, name string) (*ServerInfo, error) {
	srv, err := m.cfg.GetServer(name)
	if err != nil {
		return nil, err
	}

	info, err := m.docker.InspectContainer(ctx, srv.ContainerID)
	if err != nil {
		return &ServerInfo{Config: srv, State: string(config.StatusUnknown)}, nil
	}

	return &ServerInfo{
		Config: srv,
		State:  info.State,
		Uptime: info.Uptime,
	}, nil
}

// List returns info for all registered servers.
func (m *Manager) List(ctx context.Context) ([]*ServerInfo, error) {
	reg, err := m.cfg.LoadRegistry()
	if err != nil {
		return nil, err
	}

	result := make([]*ServerInfo, 0, len(reg.Servers))
	for _, srv := range reg.Servers {
		info := &ServerInfo{Config: srv, State: string(config.StatusUnknown)}
		if srv.ContainerID != "" {
			ci, err := m.docker.InspectContainer(ctx, srv.ContainerID)
			if err == nil {
				info.State = ci.State
				info.Uptime = ci.Uptime
			}
		}
		result = append(result, info)
	}
	return result, nil
}

// Logs streams logs for a server to the given writer.
func (m *Manager) Logs(ctx context.Context, name string, follow bool, out io.Writer) error {
	srv, err := m.cfg.GetServer(name)
	if err != nil {
		return err
	}
	return m.docker.StreamLogs(ctx, srv.ContainerID, follow, out)
}

// Update pulls the latest version of the server's image and recreates the container.
func (m *Manager) Update(ctx context.Context, name string) error {
	srv, err := m.cfg.GetServer(name)
	if err != nil {
		return err
	}

	imageRef := config.DefaultFactorioImage + ":" + srv.ImageTag

	fmt.Printf("Pulling latest %s...\n", imageRef)
	if err := m.docker.PullImage(ctx, imageRef); err != nil {
		return err
	}

	// Stop and remove old container
	m.docker.StopContainer(ctx, srv.ContainerID)    //nolint:errcheck
	m.docker.RemoveContainer(ctx, srv.ContainerID)  //nolint:errcheck

	// Recreate
	createOpts := docker.CreateOptions{
		Name:         srv.Name,
		DataDir:      m.cfg.ServerDir(srv.Name),
		Image:        imageRef,
		GamePort:     srv.GamePort,
		RCONPort:     srv.RCONPort,
		RCONPassword: srv.RCONPassword,
	}
	containerID, err := m.docker.CreateServerContainer(ctx, createOpts)
	if err != nil {
		return err
	}

	srv.ContainerID = containerID
	srv.UpdatedAt = time.Now()
	return m.cfg.UpdateServer(srv)
}

func generatePassword(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func writeFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0600)
}
