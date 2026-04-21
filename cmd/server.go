package cmd

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/AdamSvetlik/factorio-server-manager/internal/config"
	"github.com/AdamSvetlik/factorio-server-manager/internal/docker"
	"github.com/AdamSvetlik/factorio-server-manager/internal/rcon"
	"github.com/AdamSvetlik/factorio-server-manager/internal/server"
	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Manage Factorio server instances",
}

// ── server create ────────────────────────────────────────────────────────────

var serverCreateFlags struct {
	port     int
	rconPort int
	version  string
	desc     string
}

var serverCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new Factorio server instance",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newServerManager()
		if err != nil {
			return err
		}

		opts := server.CreateOptions{
			Name:        args[0],
			Description: serverCreateFlags.desc,
			ImageTag:    serverCreateFlags.version,
			GamePort:    serverCreateFlags.port,
			RCONPort:    serverCreateFlags.rconPort,
		}

		srv, err := mgr.Create(context.Background(), opts)
		if err != nil {
			return err
		}

		fmt.Printf("Server %q created successfully.\n", srv.Name)
		fmt.Printf("  Game port : %d/udp\n", srv.GamePort)
		fmt.Printf("  RCON port : %d/tcp\n", srv.RCONPort)
		fmt.Printf("  Data dir  : %s\n", cfgManager.ServerDir(srv.Name))
		fmt.Printf("\nStart it with: factorio-server-manager server start %s\n", srv.Name)
		return nil
	},
}

// ── server list ──────────────────────────────────────────────────────────────

var serverListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all managed Factorio servers",
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newServerManager()
		if err != nil {
			return err
		}

		servers, err := mgr.List(context.Background())
		if err != nil {
			return err
		}

		if len(servers) == 0 {
			fmt.Println("No servers found. Create one with: factorio-server-manager server create <name>")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tVERSION\tPORT\tUPTIME\tDESCRIPTION")
		for _, s := range servers {
			uptime := "-"
			if s.Uptime > 0 {
				uptime = formatDuration(s.Uptime)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\n",
				s.Config.Name,
				s.State,
				s.Config.ImageTag,
				s.Config.GamePort,
				uptime,
				s.Config.Description,
			)
		}
		return w.Flush()
	},
}

// ── server start ─────────────────────────────────────────────────────────────

var serverStartCmd = &cobra.Command{
	Use:   "start <name>",
	Short: "Start a Factorio server",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newServerManager()
		if err != nil {
			return err
		}
		if err := mgr.Start(context.Background(), args[0]); err != nil {
			return err
		}
		fmt.Printf("Server %q started.\n", args[0])
		return nil
	},
}

// ── server stop ──────────────────────────────────────────────────────────────

var serverStopCmd = &cobra.Command{
	Use:   "stop <name>",
	Short: "Stop a Factorio server",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newServerManager()
		if err != nil {
			return err
		}
		if err := mgr.Stop(context.Background(), args[0]); err != nil {
			return err
		}
		fmt.Printf("Server %q stopped.\n", args[0])
		return nil
	},
}

// ── server status ─────────────────────────────────────────────────────────────

var serverStatusCmd = &cobra.Command{
	Use:   "status <name>",
	Short: "Show detailed status of a Factorio server",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newServerManager()
		if err != nil {
			return err
		}
		info, err := mgr.Status(context.Background(), args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Name       : %s\n", info.Config.Name)
		fmt.Printf("Description: %s\n", info.Config.Description)
		fmt.Printf("Status     : %s\n", info.State)
		fmt.Printf("Version    : %s\n", info.Config.ImageTag)
		fmt.Printf("Game Port  : %d/udp\n", info.Config.GamePort)
		fmt.Printf("RCON Port  : %d/tcp\n", info.Config.RCONPort)
		if info.Uptime > 0 {
			fmt.Printf("Uptime     : %s\n", formatDuration(info.Uptime))
		}
		fmt.Printf("Data Dir   : %s\n", cfgManager.ServerDir(info.Config.Name))
		fmt.Printf("Created    : %s\n", info.Config.CreatedAt.Format(time.RFC3339))
		return nil
	},
}

// ── server delete ────────────────────────────────────────────────────────────

var serverDeleteFlags struct {
	force      bool
	removeData bool
}

var serverDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a Factorio server (stops and removes container)",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if !serverDeleteFlags.force {
			fmt.Printf("This will stop and remove server %q", name)
			if serverDeleteFlags.removeData {
				fmt.Printf(" and DELETE all its data")
			}
			fmt.Print(". Continue? [y/N]: ")
			var confirm string
			fmt.Scanln(&confirm)
			if confirm != "y" && confirm != "Y" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		mgr, err := newServerManager()
		if err != nil {
			return err
		}
		if err := mgr.Delete(context.Background(), name, serverDeleteFlags.removeData); err != nil {
			return err
		}
		fmt.Printf("Server %q deleted.\n", name)
		return nil
	},
}

// ── server logs ──────────────────────────────────────────────────────────────

var serverLogsFlags struct {
	follow bool
	lines  int
}

var serverLogsCmd = &cobra.Command{
	Use:   "logs <name>",
	Short: "Show logs for a Factorio server",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newServerManager()
		if err != nil {
			return err
		}
		return mgr.Logs(context.Background(), args[0], serverLogsFlags.follow, serverLogsFlags.lines, os.Stdout)
	},
}

// ── server update ────────────────────────────────────────────────────────────

var serverUpdateCmd = &cobra.Command{
	Use:   "update <name>",
	Short: "Pull the latest image and recreate the server container",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mgr, err := newServerManager()
		if err != nil {
			return err
		}
		if err := mgr.Update(context.Background(), args[0]); err != nil {
			return err
		}
		fmt.Printf("Server %q updated. Start it with: factorio-server-manager server start %s\n", args[0], args[0])
		return nil
	},
}

// ── server rcon ──────────────────────────────────────────────────────────────

var serverRconCmd = &cobra.Command{
	Use:   "rcon <server> <command>",
	Short: "Execute an RCON command on a running Factorio server",
	Args:  exactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		srv, err := cfgManager.GetServer(args[0])
		if err != nil {
			return err
		}

		client, err := rcon.Connect("127.0.0.1", srv.RCONPort, srv.RCONPassword)
		if err != nil {
			return fmt.Errorf("connect to RCON (is the server running?): %w", err)
		}
		defer client.Close()

		resp, err := client.Execute(args[1])
		if err != nil {
			return err
		}
		if resp != "" {
			fmt.Println(resp)
		}
		return nil
	},
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newServerManager() (*server.Manager, error) {
	dc, err := docker.NewClient()
	if err != nil {
		return nil, fmt.Errorf("connect to Docker: %w\n\nMake sure Docker is running.", err)
	}
	return server.NewManager(cfgManager, dc), nil
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	secs := int(d.Seconds()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm", hours, mins)
	}
	if mins > 0 {
		return fmt.Sprintf("%dm %ds", mins, secs)
	}
	return fmt.Sprintf("%ds", secs)
}

func init() {
	// create flags
	serverCreateCmd.Flags().IntVar(&serverCreateFlags.port, "port", config.DefaultGamePort, "UDP port for the game server")
	serverCreateCmd.Flags().IntVar(&serverCreateFlags.rconPort, "rcon-port", config.DefaultRCONPort, "TCP port for RCON")
	serverCreateCmd.Flags().StringVar(&serverCreateFlags.version, "version", config.DefaultImageTag, "Factorio Docker image tag")
	serverCreateCmd.Flags().StringVar(&serverCreateFlags.desc, "desc", "", "Description for this server")

	// delete flags
	serverDeleteCmd.Flags().BoolVar(&serverDeleteFlags.force, "force", false, "Skip confirmation prompt")
	serverDeleteCmd.Flags().BoolVar(&serverDeleteFlags.removeData, "remove-data", false, "Also delete all server data (saves, mods, config)")

	// logs flags
	serverLogsCmd.Flags().BoolVarP(&serverLogsFlags.follow, "follow", "f", false, "Follow log output")
	serverLogsCmd.Flags().IntVarP(&serverLogsFlags.lines, "lines", "n", 100, "Number of lines to show from the end of the logs")

	// assemble subcommands
	serverCmd.AddCommand(serverCreateCmd)
	serverCmd.AddCommand(serverListCmd)
	serverCmd.AddCommand(serverStartCmd)
	serverCmd.AddCommand(serverStopCmd)
	serverCmd.AddCommand(serverStatusCmd)
	serverCmd.AddCommand(serverDeleteCmd)
	serverCmd.AddCommand(serverLogsCmd)
	serverCmd.AddCommand(serverUpdateCmd)
	serverCmd.AddCommand(serverRconCmd)
}
