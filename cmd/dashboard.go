package cmd

import (
	"fmt"

	"github.com/AdamSvetlik/factorio-server-manager/internal/docker"
	"github.com/AdamSvetlik/factorio-server-manager/internal/server"
	"github.com/AdamSvetlik/factorio-server-manager/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var dashboardCmd = &cobra.Command{
	Use:     "dashboard",
	Aliases: []string{"dash"},
	Short:   "Open the interactive TUI dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		dc, err := docker.NewClient()
		if err != nil {
			return fmt.Errorf("connect to Docker: %w\n\nMake sure Docker is running.", err)
		}
		defer dc.Close()

		mgr := server.NewManager(cfgManager, dc)
		model := tui.NewDashboard(mgr)

		p := tea.NewProgram(model, tea.WithAltScreen())
		_, err = p.Run()
		return err
	},
}
