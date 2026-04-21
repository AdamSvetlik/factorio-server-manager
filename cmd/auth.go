package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Factorio credentials for mod downloads",
}

// ── auth login ────────────────────────────────────────────────────────────────

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Save Factorio username and token for mod downloads",
	Long: `Save your Factorio credentials for mod portal access.

You can find your token at: https://factorio.com/profile
(Log in, then look for "Service username" and "Token")`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cfgManager.LoadAppConfig()
		if err != nil {
			return err
		}

		reader := bufio.NewReader(os.Stdin)

		fmt.Print("Factorio username: ")
		username, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		username = strings.TrimSpace(username)

		fmt.Print("Factorio token (input hidden): ")
		tokenBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return fmt.Errorf("read token: %w", err)
		}
		token := strings.TrimSpace(string(tokenBytes))

		cfg.FactorioUsername = username
		cfg.FactorioToken = token

		if err := cfgManager.SaveAppConfig(cfg); err != nil {
			return err
		}

		fmt.Printf("Credentials saved for user %q.\n", username)
		return nil
	},
}

// ── auth logout ───────────────────────────────────────────────────────────────

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove saved Factorio credentials",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cfgManager.LoadAppConfig()
		if err != nil {
			return err
		}
		cfg.FactorioUsername = ""
		cfg.FactorioToken = ""
		if err := cfgManager.SaveAppConfig(cfg); err != nil {
			return err
		}
		fmt.Println("Credentials removed.")
		return nil
	},
}

// ── auth status ───────────────────────────────────────────────────────────────

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show whether credentials are configured",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := cfgManager.LoadAppConfig()
		if err != nil {
			return err
		}
		if cfg.FactorioUsername == "" {
			fmt.Println("Not logged in. Run: factorio-server-manager auth login")
		} else {
			fmt.Printf("Logged in as: %s\n", cfg.FactorioUsername)
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
}
