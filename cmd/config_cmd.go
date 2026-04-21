package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage Factorio server configuration",
}

// ── config show ───────────────────────────────────────────────────────────────

var configShowCmd = &cobra.Command{
	Use:   "show <server>",
	Short: "Print the server-settings.json for a server",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		settings, err := cfgManager.LoadServerSettings(args[0])
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(settings)
	},
}

// ── config set ────────────────────────────────────────────────────────────────

var configSetCmd = &cobra.Command{
	Use:   "set <server> <key> <value>",
	Short: "Set a top-level key in server-settings.json",
	Long: `Set a top-level key in server-settings.json.

Examples:
  factorio-server-manager config set myserver name "My Factorio Server"
  factorio-server-manager config set myserver max_players 10
  factorio-server-manager config set myserver require_user_verification true`,
	Args: exactArgs(3),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		key := args[1]
		rawValue := args[2]

		// Parse value: try bool, int, then fall back to string
		var value any
		switch strings.ToLower(rawValue) {
		case "true":
			value = true
		case "false":
			value = false
		default:
			var num float64
			if _, err := fmt.Sscanf(rawValue, "%f", &num); err == nil {
				value = num
			} else {
				value = rawValue
			}
		}

		if err := cfgManager.SetServerSettingValue(serverName, key, value); err != nil {
			return err
		}
		fmt.Printf("Set %s.%s = %v\n", serverName, key, value)
		return nil
	},
}

// ── config edit ───────────────────────────────────────────────────────────────

var configEditCmd = &cobra.Command{
	Use:   "edit <server>",
	Short: "Open server-settings.json in your $EDITOR",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		if _, err := cfgManager.GetServer(serverName); err != nil {
			return err
		}

		settingsPath := filepath.Join(cfgManager.ConfigDir(serverName), "server-settings.json")

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = os.Getenv("VISUAL")
		}
		if editor == "" {
			editor = "vi"
		}

		c := exec.Command(editor, settingsPath)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return c.Run()
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configEditCmd)
}
