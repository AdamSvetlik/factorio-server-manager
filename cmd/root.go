package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AdamSvetlik/factorio-server-manager/internal/config"
	"github.com/spf13/cobra"
)

const (
	ansiRed   = "\033[31m"
	ansiReset = "\033[0m"
)

// exactArgs is like cobra.ExactArgs but also prints the command help when the
// argument count is wrong, so the user sees usage alongside the error.
func exactArgs(n int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if err := cobra.ExactArgs(n)(cmd, args); err != nil {
			cmd.Help() //nolint:errcheck
			fmt.Fprintln(os.Stderr)
			return err
		}
		return nil
	}
}

var (
	dataDirFlag string
	cfgManager  *config.Manager
)

var rootCmd = &cobra.Command{
	Use:   "factorio-server-manager",
	Short: "Manage multiple Factorio server instances via Docker",
	Long: `factorio-server-manager is a CLI tool for managing multiple Factorio dedicated
server instances running in Docker containers (factoriotools/factorio image).

It handles server lifecycle, saves, mods, and configuration from a single tool.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		mgr := config.NewManager(dataDirFlag)
		if err := mgr.Init(); err != nil {
			return fmt.Errorf("initializing data dir: %w", err)
		}
		cfgManager = mgr
		return nil
	},
}

// Execute is the entrypoint called from main.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: %s%s\n", ansiRed, err.Error(), ansiReset)
		os.Exit(1)
	}
}

func init() {
	defaultDataDir := filepath.Join(homeDir(), ".factorio-manager")
	rootCmd.PersistentFlags().StringVar(&dataDirFlag, "data-dir", defaultDataDir, "Data directory for all server state")

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(saveCmd)
	rootCmd.AddCommand(modCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(dashboardCmd)
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}
