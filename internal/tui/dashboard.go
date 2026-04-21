package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/AdamSvetlik/factorio-server-manager/internal/server"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const refreshInterval = 3 * time.Second

// tickMsg is sent on each refresh interval.
type tickMsg struct{}

// serversLoadedMsg carries the refreshed server list.
type serversLoadedMsg struct {
	servers []*server.ServerInfo
	err     error
}

// actionResultMsg carries the result of a start/stop action.
type actionResultMsg struct {
	err error
}

// Dashboard is the main bubbletea model for the TUI.
type Dashboard struct {
	mgr      *server.Manager
	servers  []*server.ServerInfo
	cursor   int
	width    int
	height   int
	loading  bool
	err      error
	spinner  spinner.Model
	logView  viewport.Model
	showLog  bool
	logLines []string
	status   string
}

// NewDashboard creates a new Dashboard model.
func NewDashboard(mgr *server.Manager) *Dashboard {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	vp := viewport.New(80, 20)

	return &Dashboard{
		mgr:     mgr,
		loading: true,
		spinner: s,
		logView: vp,
	}
}

func (d *Dashboard) Init() tea.Cmd {
	return tea.Batch(
		d.spinner.Tick,
		d.loadServers(),
		d.scheduleRefresh(),
	)
}

func (d *Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return d.handleKey(msg)

	case tea.WindowSizeMsg:
		d.width = msg.Width
		d.height = msg.Height
		d.logView.Width = msg.Width - 4
		d.logView.Height = d.height - 12

	case tickMsg:
		return d, tea.Batch(d.loadServers(), d.scheduleRefresh())

	case serversLoadedMsg:
		d.loading = false
		if msg.err != nil {
			d.err = msg.err
		} else {
			d.servers = msg.servers
			if d.cursor >= len(d.servers) && len(d.servers) > 0 {
				d.cursor = len(d.servers) - 1
			}
		}

	case actionResultMsg:
		if msg.err != nil {
			d.status = "Error: " + msg.err.Error()
		}
		return d, d.loadServers()

	case spinner.TickMsg:
		var cmd tea.Cmd
		d.spinner, cmd = d.spinner.Update(msg)
		return d, cmd
	}

	if d.showLog {
		var cmd tea.Cmd
		d.logView, cmd = d.logView.Update(msg)
		return d, cmd
	}

	return d, nil
}

func (d *Dashboard) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if d.showLog {
		switch msg.String() {
		case "q", "esc":
			d.showLog = false
			d.logLines = nil
		}
		return d, nil
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return d, tea.Quit

	case "j", "down":
		if d.cursor < len(d.servers)-1 {
			d.cursor++
		}

	case "k", "up":
		if d.cursor > 0 {
			d.cursor--
		}

	case "s":
		if len(d.servers) > 0 {
			name := d.servers[d.cursor].Config.Name
			d.status = fmt.Sprintf("Starting %s...", name)
			return d, d.startServer(name)
		}

	case "S":
		if len(d.servers) > 0 {
			name := d.servers[d.cursor].Config.Name
			d.status = fmt.Sprintf("Stopping %s...", name)
			return d, d.stopServer(name)
		}

	case "r":
		d.loading = true
		d.status = ""
		return d, d.loadServers()
	}

	return d, nil
}

func (d *Dashboard) View() string {
	if d.showLog {
		return d.renderLogView()
	}

	var sb strings.Builder

	// Header
	title := HeaderStyle.Render("  Factorio Server Manager")
	sb.WriteString(title + "\n")

	if d.err != nil {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorError).Render("Error: "+d.err.Error()) + "\n")
	}

	if d.loading && len(d.servers) == 0 {
		sb.WriteString(d.spinner.View() + " Loading servers...\n")
	} else {
		sb.WriteString(d.renderTable())
	}

	if len(d.servers) > 0 {
		sb.WriteString("\n")
		sb.WriteString(d.renderDetail())
	}

	// Status bar
	if d.status != "" {
		sb.WriteString("\n" + StatusOther.Render(d.status) + "\n")
	}

	// Help
	sb.WriteString("\n" + d.renderHelp())

	return sb.String()
}

func (d *Dashboard) renderTable() string {
	cols := []int{20, 10, 10, 8, 12}
	headers := []string{"NAME", "STATUS", "VERSION", "PORT", "UPTIME"}

	var sb strings.Builder

	// Header row
	headerCells := make([]string, len(headers))
	for i, h := range headers {
		headerCells[i] = lipgloss.NewStyle().Width(cols[i]).Bold(true).Foreground(colorPrimary).Render(h)
	}
	sb.WriteString(strings.Join(headerCells, " ") + "\n")
	sb.WriteString(strings.Repeat("─", sumCols(cols)+len(cols)-1) + "\n")

	if len(d.servers) == 0 {
		sb.WriteString(HelpStyle.Render("No servers. Create one with: factorio-server-manager server create <name>") + "\n")
		return sb.String()
	}

	for i, s := range d.servers {
		uptime := "-"
		if s.Uptime > 0 {
			uptime = formatDur(s.Uptime)
		}

		cells := []string{
			lipgloss.NewStyle().Width(cols[0]).Render(truncStr(s.Config.Name, cols[0])),
			StatusStyle(s.State).Width(cols[1]).Render(s.State),
			lipgloss.NewStyle().Width(cols[2]).Render(s.Config.ImageTag),
			lipgloss.NewStyle().Width(cols[3]).Render(fmt.Sprintf("%d", s.Config.GamePort)),
			lipgloss.NewStyle().Width(cols[4]).Render(uptime),
		}
		row := strings.Join(cells, " ")

		if i == d.cursor {
			row = SelectedRowStyle.Render(row)
		}
		sb.WriteString(row + "\n")
	}

	return sb.String()
}

func (d *Dashboard) renderDetail() string {
	if len(d.servers) == 0 || d.cursor >= len(d.servers) {
		return ""
	}
	s := d.servers[d.cursor]

	lines := []struct{ label, value string }{
		{"Name", s.Config.Name},
		{"Description", s.Config.Description},
		{"Status", StatusStyle(s.State).Render(s.State)},
		{"Image", s.Config.ImageTag},
		{"Game Port", fmt.Sprintf("%d/udp", s.Config.GamePort)},
		{"RCON Port", fmt.Sprintf("%d/tcp", s.Config.RCONPort)},
	}
	if s.Uptime > 0 {
		lines = append(lines, struct{ label, value string }{"Uptime", formatDur(s.Uptime)})
	}

	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString(DetailLabelStyle.Render(l.label+":") + " " + DetailValueStyle.Render(l.value) + "\n")
	}
	return BoxStyle.Width(d.width - 4).Render(sb.String())
}

func (d *Dashboard) renderLogView() string {
	title := HeaderStyle.Render(fmt.Sprintf("  Logs — press q or esc to go back"))
	return title + "\n" + d.logView.View()
}

func (d *Dashboard) renderHelp() string {
	keys := []struct{ key, desc string }{
		{"j/k", "navigate"},
		{"s", "start"},
		{"S", "stop"},
		{"r", "refresh"},
		{"q", "quit"},
	}
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = KeyStyle.Render(k.key) + " " + HelpStyle.Render(k.desc)
	}
	return strings.Join(parts, "  ")
}

// ── commands (return tea.Cmd) ─────────────────────────────────────────────────

func (d *Dashboard) loadServers() tea.Cmd {
	return func() tea.Msg {
		servers, err := d.mgr.List(context.Background())
		return serversLoadedMsg{servers: servers, err: err}
	}
}

func (d *Dashboard) scheduleRefresh() tea.Cmd {
	return tea.Tick(refreshInterval, func(_ time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (d *Dashboard) startServer(name string) tea.Cmd {
	return func() tea.Msg {
		err := d.mgr.Start(context.Background(), name)
		return actionResultMsg{err: err}
	}
}

func (d *Dashboard) stopServer(name string) tea.Cmd {
	return func() tea.Msg {
		err := d.mgr.Stop(context.Background(), name)
		return actionResultMsg{err: err}
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func formatDur(d time.Duration) string {
	d = d.Round(time.Second)
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	mins := int(d.Minutes()) % 60
	if days > 0 {
		return fmt.Sprintf("%dd%dh%dm", days, hours, mins)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
	return fmt.Sprintf("%dm%ds", mins, int(d.Seconds())%60)
}

func truncStr(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func sumCols(cols []int) int {
	n := 0
	for _, c := range cols {
		n += c
	}
	return n
}
