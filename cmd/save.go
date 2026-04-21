package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var saveCmd = &cobra.Command{
	Use:   "save",
	Short: "Manage saves for a Factorio server",
}

// ── save list ─────────────────────────────────────────────────────────────────

var saveListCmd = &cobra.Command{
	Use:   "list <server>",
	Short: "List save files for a server",
	Args:  exactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		savesDir := cfgManager.SavesDir(args[0])
		entries, err := os.ReadDir(savesDir)
		if err != nil {
			return fmt.Errorf("read saves directory: %w", err)
		}

		saves := filterZips(entries)
		if len(saves) == 0 {
			fmt.Printf("No saves found for server %q.\n", args[0])
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "SAVE FILE\tSIZE\tMODIFIED")
		for _, e := range saves {
			info, _ := e.Info()
			fmt.Fprintf(w, "%s\t%s\t%s\n",
				e.Name(),
				formatBytes(info.Size()),
				info.ModTime().Format("2006-01-02 15:04:05"),
			)
		}
		return w.Flush()
	},
}

// ── save copy ─────────────────────────────────────────────────────────────────

var saveCopyCmd = &cobra.Command{
	Use:   "copy <file> <server>",
	Short: "Copy a save file into a server's saves directory",
	Args:  exactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		src := args[0]
		serverName := args[1]

		if _, err := cfgManager.GetServer(serverName); err != nil {
			return err
		}

		dest := filepath.Join(cfgManager.SavesDir(serverName), filepath.Base(src))
		if err := copyFile(src, dest); err != nil {
			return fmt.Errorf("copy save: %w", err)
		}
		fmt.Printf("Copied %q to server %q saves.\n", filepath.Base(src), serverName)
		return nil
	},
}

// ── save export ───────────────────────────────────────────────────────────────

var saveExportFlags struct {
	outDir string
}

var saveExportCmd = &cobra.Command{
	Use:   "export <server> <save-file>",
	Short: "Export a save file from a server to a local path",
	Args:  exactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		saveFile := args[1]

		src := filepath.Join(cfgManager.SavesDir(serverName), saveFile)
		destDir := saveExportFlags.outDir
		if destDir == "" {
			var err error
			destDir, err = os.Getwd()
			if err != nil {
				return err
			}
		}
		dest := filepath.Join(destDir, saveFile)

		if err := copyFile(src, dest); err != nil {
			return fmt.Errorf("export save: %w", err)
		}
		fmt.Printf("Exported %q to %s\n", saveFile, dest)
		return nil
	},
}

// ── save delete ───────────────────────────────────────────────────────────────

var saveDeleteCmd = &cobra.Command{
	Use:   "delete <server> <save-file>",
	Short: "Delete a save file from a server",
	Args:  exactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		serverName := args[0]
		saveFile := args[1]

		path := filepath.Join(cfgManager.SavesDir(serverName), saveFile)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("save file %q not found", saveFile)
		}

		fmt.Printf("Delete save %q from server %q? [y/N]: ", saveFile, serverName)
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Aborted.")
			return nil
		}

		if err := os.Remove(path); err != nil {
			return fmt.Errorf("delete save: %w", err)
		}
		fmt.Printf("Deleted %q.\n", saveFile)
		return nil
	},
}

// ── helpers ──────────────────────────────────────────────────────────────────

func filterZips(entries []os.DirEntry) []os.DirEntry {
	var result []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".zip" {
			result = append(result, e)
		}
	}
	return result
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		os.Remove(dest) //nolint:errcheck
		return err
	}
	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

func init() {
	saveExportCmd.Flags().StringVar(&saveExportFlags.outDir, "out", "", "Output directory (default: current directory)")

	saveCmd.AddCommand(saveListCmd)
	saveCmd.AddCommand(saveCopyCmd)
	saveCmd.AddCommand(saveExportCmd)
	saveCmd.AddCommand(saveDeleteCmd)
}
