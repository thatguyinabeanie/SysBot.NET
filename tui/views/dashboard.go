// Package views contains Bubble Tea models for each TUI screen.
package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thatguyinabeanie/SysBot.NET/tui/client"
	"github.com/thatguyinabeanie/SysBot.NET/tui/components"
)

// ── Message types ─────────────────────────────────────────────────────

// botsMsg carries the result of fetching the bot list.
type botsMsg struct {
	bots []client.BotDto
}

// metaMsg carries the result of fetching hub metadata.
type metaMsg struct {
	meta *client.MetaInfo
}

// queuesMsg carries the result of fetching queue status.
type queuesMsg struct {
	queues *client.QueueStatus
}

// tickMsg triggers a periodic refresh of bot data.
type tickMsg time.Time

// errMsg carries an error from any async command.
type errMsg struct {
	err error
}

// actionDoneMsg signals that a bot action completed. We re-fetch bots after.
type actionDoneMsg struct{}

// confirmDeleteMsg is sent when the user presses 'd' to enter delete confirmation.
type confirmDeleteMsg struct{}

// ── Dashboard model ───────────────────────────────────────────────────

// DashboardModel is the Bubble Tea model for the bot management dashboard.
type DashboardModel struct {
	client         *client.Client
	bots           []client.BotDto
	meta           *client.MetaInfo
	queues         *client.QueueStatus
	cursor         int
	width          int
	height         int
	err            error
	confirmDelete  bool // true while waiting for delete confirmation
}

// NewDashboard creates a new DashboardModel wired to the given API client.
func NewDashboard(c *client.Client) DashboardModel {
	return DashboardModel{
		client: c,
	}
}

// ── Commands (wrap client calls, return messages) ─────────────────────

func fetchBots(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		bots, err := c.ListBots()
		if err != nil {
			return errMsg{err}
		}
		return botsMsg{bots}
	}
}

func fetchMeta(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		meta, err := c.GetMeta()
		if err != nil {
			return errMsg{err}
		}
		return metaMsg{meta}
	}
}

func fetchQueues(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		queues, err := c.GetQueues()
		if err != nil {
			return errMsg{err}
		}
		return queuesMsg{queues}
	}
}

func tickEvery3s() tea.Cmd {
	return tea.Tick(3*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func doBotAction(c *client.Client, id, action string) tea.Cmd {
	return func() tea.Msg {
		_, err := c.BotAction(id, action)
		if err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func doRemoveBot(c *client.Client, id string) tea.Cmd {
	return func() tea.Msg {
		if err := c.RemoveBot(id); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func doStartAll(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		if err := c.StartAll(); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

func doStopAll(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		if err := c.StopAll(); err != nil {
			return errMsg{err}
		}
		return actionDoneMsg{}
	}
}

// ── Tea interface ─────────────────────────────────────────────────────

// Init fetches bots, metadata, and queue status in parallel, then starts
// the periodic refresh ticker.
func (m DashboardModel) Init() tea.Cmd {
	return tea.Batch(
		fetchBots(m.client),
		fetchMeta(m.client),
		fetchQueues(m.client),
		tickEvery3s(),
	)
}

// Update handles input and async messages, returning the concrete DashboardModel
// so the parent model can work with it directly.
func (m DashboardModel) Update(msg tea.Msg) (DashboardModel, tea.Cmd) {
	switch msg := msg.(type) {

	// ── Window resize ────────────────────────────────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	// ── Async data responses ─────────────────────────────────────
	case botsMsg:
		m.bots = msg.bots
		m.err = nil
		// Clamp cursor if bots were removed externally.
		if m.cursor >= len(m.bots) && len(m.bots) > 0 {
			m.cursor = len(m.bots) - 1
		}
		return m, nil

	case metaMsg:
		m.meta = msg.meta
		return m, nil

	case queuesMsg:
		m.queues = msg.queues
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case actionDoneMsg:
		// Re-fetch bots + queues after any action.
		return m, tea.Batch(fetchBots(m.client), fetchQueues(m.client))

	// ── Periodic refresh ─────────────────────────────────────────
	case tickMsg:
		return m, tea.Batch(fetchBots(m.client), tickEvery3s())

	// ── Keyboard input ───────────────────────────────────────────
	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey processes keyboard input for the dashboard.
func (m DashboardModel) handleKey(msg tea.KeyMsg) (DashboardModel, tea.Cmd) {
	// If we're waiting for delete confirmation, only accept y/n.
	if m.confirmDelete {
		switch {
		case key.Matches(msg, key.NewBinding(key.WithKeys("y", "Y"))):
			m.confirmDelete = false
			if m.cursor < len(m.bots) {
				return m, doRemoveBot(m.client, m.bots[m.cursor].ID)
			}
			return m, nil
		default:
			// Any other key cancels the confirmation.
			m.confirmDelete = false
			return m, nil
		}
	}

	switch {
	// Navigation
	case key.Matches(msg, key.NewBinding(key.WithKeys("j", "down"))):
		if m.cursor < len(m.bots)-1 {
			m.cursor++
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("k", "up"))):
		if m.cursor > 0 {
			m.cursor--
		}

	// Single-bot actions (require at least one bot selected)
	case key.Matches(msg, key.NewBinding(key.WithKeys("s"))):
		if m.cursor < len(m.bots) {
			return m, doBotAction(m.client, m.bots[m.cursor].ID, "start")
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("x"))):
		if m.cursor < len(m.bots) {
			return m, doBotAction(m.client, m.bots[m.cursor].ID, "stop")
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("p"))):
		if m.cursor < len(m.bots) {
			bot := m.bots[m.cursor]
			// Toggle pause/resume based on current state.
			action := "pause"
			if bot.IsPaused {
				action = "resume"
			}
			return m, doBotAction(m.client, bot.ID, action)
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("r"))):
		if m.cursor < len(m.bots) {
			return m, doBotAction(m.client, m.bots[m.cursor].ID, "restart")
		}
	case key.Matches(msg, key.NewBinding(key.WithKeys("d"))):
		if m.cursor < len(m.bots) {
			m.confirmDelete = true
		}

	// Bulk actions
	case key.Matches(msg, key.NewBinding(key.WithKeys("S"))):
		return m, doStartAll(m.client)
	case key.Matches(msg, key.NewBinding(key.WithKeys("X"))):
		return m, doStopAll(m.client)
	}

	return m, nil
}

// ── View rendering ────────────────────────────────────────────────────

// Styles used in the dashboard view.
var (
	dashStatusBarStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("235")).
				Foreground(lipgloss.Color("15")).
				Padding(0, 1).
				Bold(true)

	dashModeBadgeStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("4")).
				Foreground(lipgloss.Color("15")).
				Padding(0, 1).
				Bold(true)

	dashQueueOpenStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("2"))

	dashQueueClosedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("1"))

	dashSelectedRowStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Bold(true)

	dashNormalRowStyle = lipgloss.NewStyle()

	dashHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	dashErrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true)

	dashHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("4")).
			Bold(true)

	dashConfirmStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("3")).
				Bold(true)
)

// View renders the dashboard screen.
func (m DashboardModel) View() string {
	var b strings.Builder

	// ── Status bar ───────────────────────────────────────────────
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n\n")

	// ── Error display ────────────────────────────────────────────
	if m.err != nil {
		b.WriteString(dashErrStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n\n")
	}

	// ── Delete confirmation banner ───────────────────────────────
	if m.confirmDelete && m.cursor < len(m.bots) {
		bot := m.bots[m.cursor]
		b.WriteString(dashConfirmStyle.Render(
			fmt.Sprintf("Delete bot %s? [y]es / any key to cancel", bot.ID)))
		b.WriteString("\n\n")
	}

	// ── Bot table ────────────────────────────────────────────────
	if len(m.bots) == 0 {
		b.WriteString("  No bots registered.\n")
	} else {
		b.WriteString(m.renderBotTable())
	}

	b.WriteString("\n")

	// ── Help bar ─────────────────────────────────────────────────
	b.WriteString(m.renderHelpBar())

	return b.String()
}

// renderStatusBar builds the top status line showing mode and queue info.
func (m DashboardModel) renderStatusBar() string {
	var parts []string

	// Mode badge
	mode := "Unknown"
	if m.meta != nil {
		mode = m.meta.Mode
	}
	parts = append(parts, dashModeBadgeStyle.Render(mode))

	// Queue status
	if m.queues != nil {
		var queueLabel string
		if m.queues.CanQueue {
			queueLabel = dashQueueOpenStyle.Render("Queue: Open")
		} else {
			queueLabel = dashQueueClosedStyle.Render("Queue: Closed")
		}
		parts = append(parts, queueLabel)

		// Individual queue counts.
		var counts []string
		for name, qc := range m.queues.Queues {
			if qc != nil && qc.Count > 0 {
				counts = append(counts, fmt.Sprintf("%s:%d", name, qc.Count))
			}
		}
		if len(counts) > 0 {
			parts = append(parts, strings.Join(counts, " "))
		}

		parts = append(parts, fmt.Sprintf("Total: %d", m.queues.TotalCount))
	}

	return dashStatusBarStyle.Render(strings.Join(parts, "  "))
}

// renderBotTable builds the bot list with status lamps and details.
func (m DashboardModel) renderBotTable() string {
	var b strings.Builder

	// Column header
	header := fmt.Sprintf("  %-3s  %-18s  %-14s  %-10s  %s",
		"", "Connection", "Routine", "Status", "Last Log")
	b.WriteString(dashHeaderStyle.Render(header))
	b.WriteString("\n")

	// Maximum width available for the last log column.
	maxLogWidth := m.width - 55
	if maxLogWidth < 10 {
		maxLogWidth = 10
	}

	for i, bot := range m.bots {
		// Status lamp
		lamp := components.StatusLamp(
			bot.IsRunning, bot.IsPaused, bot.IsConnected,
			bot.CurrentRoutine, bot.NextRoutine, bot.LastActive,
		)

		// Connection identifier (IP:Port for WiFi, "USB" for USB)
		var conn string
		if bot.Protocol == "USB" {
			conn = "USB"
		} else {
			conn = fmt.Sprintf("%s:%d", bot.IP, bot.Port)
		}

		// Routine display
		routine := bot.CurrentRoutine
		if bot.NextRoutine != "" && bot.NextRoutine != bot.CurrentRoutine {
			routine += " → " + bot.NextRoutine
		}

		// Status text
		status := "Stopped"
		if bot.IsRunning {
			if bot.IsPaused {
				status = "Paused"
			} else if bot.IsConnected {
				status = "Running"
			} else {
				status = "Connecting"
			}
		}

		// Last log (truncated to available width)
		lastLog := bot.LastLog
		if len(lastLog) > maxLogWidth {
			lastLog = lastLog[:maxLogWidth-1] + "…"
		}

		row := fmt.Sprintf("  %s  %-18s  %-14s  %-10s  %s",
			lamp, conn, routine, status, lastLog)

		if i == m.cursor {
			b.WriteString(dashSelectedRowStyle.Render(row))
		} else {
			b.WriteString(dashNormalRowStyle.Render(row))
		}
		b.WriteString("\n")
	}

	return b.String()
}

// renderHelpBar builds the bottom help line.
func (m DashboardModel) renderHelpBar() string {
	return dashHelpStyle.Render(
		"[s]tart  [x]stop  [p]ause  [r]estart  [d]elete  [S]tart All  [X]Stop All  [j/k]navigate")
}
