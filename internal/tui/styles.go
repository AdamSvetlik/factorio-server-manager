package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	// Colors
	colorPrimary  = lipgloss.Color("#7CB9E8")
	colorSuccess  = lipgloss.Color("#4ECCA3")
	colorWarning  = lipgloss.Color("#F5A623")
	colorError    = lipgloss.Color("#FF6B6B")
	colorMuted    = lipgloss.Color("#626262")
	colorSelected = lipgloss.Color("#1C3A5E")

	// Header
	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorPrimary).
			MarginBottom(1)

	// Table header
	TableHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(colorPrimary).
				BorderStyle(lipgloss.NormalBorder()).
				BorderBottom(true).
				BorderForeground(colorMuted)

	// Selected row
	SelectedRowStyle = lipgloss.NewStyle().
				Background(colorSelected).
				Foreground(lipgloss.Color("#FFFFFF"))

	// Status styles
	StatusRunning = lipgloss.NewStyle().Foreground(colorSuccess).Bold(true)
	StatusStopped = lipgloss.NewStyle().Foreground(colorError)
	StatusOther   = lipgloss.NewStyle().Foreground(colorWarning)

	// Help bar
	HelpStyle = lipgloss.NewStyle().
			Foreground(colorMuted).
			MarginTop(1)

	// Key hint
	KeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(colorMuted).
			Padding(0, 1)

	// Detail panel
	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(colorMuted).
				Width(12)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFDF5"))

	// Border box
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorMuted).
			Padding(0, 1)
)

// StatusStyle returns the appropriate style for a given container state.
func StatusStyle(state string) lipgloss.Style {
	switch state {
	case "running":
		return StatusRunning
	case "exited", "dead":
		return StatusStopped
	default:
		return StatusOther
	}
}
