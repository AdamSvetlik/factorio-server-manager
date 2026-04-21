package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/AdamSvetlik/factorio-server-manager/internal/mods"
	"github.com/spf13/cobra"
)

var modCmd = &cobra.Command{
	Use:   "mod",
	Short: "Manage mods for a Factorio server",
}

// ── mod list ──────────────────────────────────────────────────────────────────

var modListCmd = &cobra.Command{
	Use:   "list <server>",
	Short: "List installed mods for a server",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		if _, err := cfgManager.GetServer(serverName); err != nil {
			return err
		}

		modsDir := cfgManager.ModsDir(serverName)
		installed, err := mods.ListInstalled(modsDir)
		if err != nil {
			return err
		}

		if len(installed) == 0 {
			fmt.Printf("No mods installed for server %q.\n", serverName)
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "MOD FILE")
		for _, name := range installed {
			fmt.Fprintln(w, name)
		}
		return w.Flush()
	},
}

// ── mod search ────────────────────────────────────────────────────────────────

var modSearchFlags struct {
	page     int
	pageSize int
}

var modSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search the Factorio mod portal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newModClient()
		result, err := client.Search(args[0], modSearchFlags.page, modSearchFlags.pageSize)
		if err != nil {
			return err
		}

		if len(result.Results) == 0 {
			fmt.Printf("No mods found for %q.\n", args[0])
			return nil
		}

		fmt.Printf("Found %d mods (page %d/%d):\n\n",
			result.Pagination.Count, result.Pagination.Page, result.Pagination.PageCount)

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTITLE\tOWNER\tDOWNLOADS\tLATEST")
		for _, m := range result.Results {
			latest := "-"
			if m.LatestRelease != nil {
				latest = m.LatestRelease.Version
			}
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n",
				m.Name, truncate(m.Title, 40), m.Owner, m.DownloadsCount, latest)
		}
		return w.Flush()
	},
}

// ── mod info ──────────────────────────────────────────────────────────────────

var modInfoCmd = &cobra.Command{
	Use:   "info <mod-name>",
	Short: "Show details about a mod from the mod portal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		client := newModClient()
		info, err := client.GetMod(args[0])
		if err != nil {
			return err
		}

		fmt.Printf("Name       : %s\n", info.Name)
		fmt.Printf("Title      : %s\n", info.Title)
		fmt.Printf("Owner      : %s\n", info.Owner)
		fmt.Printf("Downloads  : %d\n", info.DownloadsCount)
		fmt.Printf("Summary    : %s\n", info.Summary)
		if len(info.Releases) > 0 {
			fmt.Printf("\nReleases (%d):\n", len(info.Releases))
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "  VERSION\tFACTORIO\tRELEASED")
			for _, r := range info.Releases {
				fmt.Fprintf(w, "  %s\t%s\t%s\n",
					r.Version,
					r.InfoJSON.FactorioVersion,
					r.ReleasedAt.Format("2006-01-02"),
				)
			}
			w.Flush() //nolint:errcheck
		}
		return nil
	},
}

// ── mod install ───────────────────────────────────────────────────────────────

var modInstallFlags struct {
	version string
}

var modInstallCmd = &cobra.Command{
	Use:   "install <server> <mod-name>",
	Short: "Download and install a mod from the mod portal",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		modName := args[1]

		if _, err := cfgManager.GetServer(serverName); err != nil {
			return err
		}

		client := newModClient()
		info, err := client.GetMod(modName)
		if err != nil {
			return err
		}

		var release *mods.ModRelease
		if modInstallFlags.version != "" {
			for i := range info.Releases {
				if info.Releases[i].Version == modInstallFlags.version {
					release = &info.Releases[i]
					break
				}
			}
			if release == nil {
				return fmt.Errorf("version %q not found for mod %q", modInstallFlags.version, modName)
			}
		} else {
			release = info.LatestRelease
		}

		if release == nil {
			return fmt.Errorf("no releases found for mod %q", modName)
		}

		modsDir := cfgManager.ModsDir(serverName)
		fmt.Printf("Installing %s v%s...\n", modName, release.Version)
		if err := client.Download(release.DownloadURL, modsDir, release.FileName); err != nil {
			return err
		}
		fmt.Printf("Installed %s to server %q.\n", release.FileName, serverName)
		return nil
	},
}

// ── mod remove ────────────────────────────────────────────────────────────────

var modRemoveCmd = &cobra.Command{
	Use:   "remove <server> <mod-file>",
	Short: "Remove a mod file from a server",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		modFile := args[1]

		if _, err := cfgManager.GetServer(serverName); err != nil {
			return err
		}

		modsDir := cfgManager.ModsDir(serverName)
		if err := mods.RemoveMod(modsDir, modFile); err != nil {
			return err
		}
		fmt.Printf("Removed %q from server %q.\n", modFile, serverName)
		return nil
	},
}

// ── mod update ────────────────────────────────────────────────────────────────

var modUpdateCmd = &cobra.Command{
	Use:   "update <server>",
	Short: "Update all installed mods for a server to their latest versions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		if _, err := cfgManager.GetServer(serverName); err != nil {
			return err
		}

		modsDir := cfgManager.ModsDir(serverName)
		installed, err := mods.ListInstalled(modsDir)
		if err != nil {
			return err
		}

		if len(installed) == 0 {
			fmt.Printf("No mods installed for server %q.\n", serverName)
			return nil
		}

		client := newModClient()
		updated := 0

		for _, fileName := range installed {
			// Parse mod name from filename (modname_version.zip)
			modName := modNameFromFile(fileName)
			if modName == "" {
				continue
			}

			info, err := client.GetMod(modName)
			if err != nil {
				fmt.Printf("  Warning: could not fetch info for %s: %v\n", modName, err)
				continue
			}
			if info.LatestRelease == nil {
				continue
			}

			// Skip if already at latest
			if fileName == info.LatestRelease.FileName {
				fmt.Printf("  %s: already up to date (%s)\n", modName, info.LatestRelease.Version)
				continue
			}

			fmt.Printf("  Updating %s to v%s...\n", modName, info.LatestRelease.Version)
			if err := client.Download(info.LatestRelease.DownloadURL, modsDir, info.LatestRelease.FileName); err != nil {
				fmt.Printf("  Warning: failed to update %s: %v\n", modName, err)
				continue
			}
			// Remove old version
			mods.RemoveMod(modsDir, fileName) //nolint:errcheck
			updated++
		}

		fmt.Printf("\nUpdated %d mod(s).\n", updated)
		return nil
	},
}

// ── helpers ──────────────────────────────────────────────────────────────────

func newModClient() *mods.Client {
	cfg, _ := cfgManager.LoadAppConfig()
	username, token := "", ""
	if cfg != nil {
		username = cfg.FactorioUsername
		token = cfg.FactorioToken
	}
	return mods.NewClient(username, token)
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// modNameFromFile extracts the mod name from a filename like "modname_1.2.3.zip".
func modNameFromFile(fileName string) string {
	name := filepath.Base(fileName)
	// Strip .zip
	if len(name) > 4 {
		name = name[:len(name)-4]
	}
	// Find last underscore followed by a digit (version separator)
	for i := len(name) - 1; i > 0; i-- {
		if name[i] == '_' && i+1 < len(name) && name[i+1] >= '0' && name[i+1] <= '9' {
			return name[:i]
		}
	}
	return name
}

func init() {
	modSearchCmd.Flags().IntVar(&modSearchFlags.page, "page", 1, "Page number")
	modSearchCmd.Flags().IntVar(&modSearchFlags.pageSize, "page-size", 20, "Results per page")

	modInstallCmd.Flags().StringVar(&modInstallFlags.version, "version", "", "Specific mod version to install (default: latest)")

	modCmd.AddCommand(modListCmd)
	modCmd.AddCommand(modSearchCmd)
	modCmd.AddCommand(modInfoCmd)
	modCmd.AddCommand(modInstallCmd)
	modCmd.AddCommand(modRemoveCmd)
	modCmd.AddCommand(modUpdateCmd)
}
