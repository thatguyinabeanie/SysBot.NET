package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thatguyinabeanie/SysBot.NET/tui/client"
)

// maxLogEntries is the maximum number of log entries kept in memory.
// When the buffer exceeds this size, the oldest entries are trimmed.
const maxLogEntries = 5000

// LogEntryMsg is sent by the parent model when SignalR delivers a log entry.
type LogEntryMsg client.LogEntry

// ── Log view styles ───────────────────────────────────────────────────

var (
	logTimestampStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	logIdentityStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	logMessageStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
	logFilterBarStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	logFollowOnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	logFollowOffStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
)

// ── Logs model ────────────────────────────────────────────────────────

// LogsModel is the Bubble Tea model for the streaming log viewer.
type LogsModel struct {
	entries   []client.LogEntry // all log entries in memory
	viewport  viewport.Model    // scrollable log area
	filter    textinput.Model   // filter input field
	follow    bool              // auto-scroll to bottom on new entries
	filtering bool              // true when filter input is focused
	width     int
	height    int
}

// NewLogs creates a new LogsModel with sensible defaults.
func NewLogs() LogsModel {
	// Initialize the filter text input.
	ti := textinput.New()
	ti.Placeholder = "filter logs..."
	ti.CharLimit = 256
	ti.Width = 40

	// Initialize the viewport (will be resized on first WindowSizeMsg).
	vp := viewport.New(80, 20)

	return LogsModel{
		entries:  make([]client.LogEntry, 0, 256),
		viewport: vp,
		filter:   ti,
		follow:   true,
	}
}

// Init has nothing to initialize — log entries arrive via LogEntryMsg from the parent.
func (m LogsModel) Init() tea.Cmd {
	return nil
}

// Update handles keyboard input and incoming log entries.
func (m LogsModel) Update(msg tea.Msg) (LogsModel, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// Reserve 2 lines: 1 for the filter bar, 1 for spacing.
		vpHeight := m.height - 2
		if vpHeight < 1 {
			vpHeight = 1
		}
		m.viewport.Width = m.width
		m.viewport.Height = vpHeight
		m.filter.Width = m.width - 30
		if m.filter.Width < 20 {
			m.filter.Width = 20
		}
		// Re-render content at the new size.
		m.viewport.SetContent(m.renderLogContent())
		if m.follow {
			m.viewport.GotoBottom()
		}
		return m, nil

	case LogEntryMsg:
		// Append the new entry.
		m.entries = append(m.entries, client.LogEntry(msg))

		// Trim from front if we exceed the max buffer size.
		if len(m.entries) > maxLogEntries {
			excess := len(m.entries) - maxLogEntries
			m.entries = m.entries[excess:]
		}

		// Re-render and optionally follow.
		m.viewport.SetContent(m.renderLogContent())
		if m.follow {
			m.viewport.GotoBottom()
		}
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Pass unhandled messages to the viewport for scrolling.
	if !m.filtering {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return m, tea.Batch(cmds...)
}

// handleKey processes keyboard input for the log viewer.
func (m LogsModel) handleKey(msg tea.KeyMsg) (LogsModel, tea.Cmd) {
	// When the filter input is focused, route keys to it.
	if m.filtering {
		switch msg.String() {
		case "esc":
			// Unfocus the filter input.
			m.filtering = false
			m.filter.Blur()
			// Re-render with current filter text (keeps the filter active).
			m.viewport.SetContent(m.renderLogContent())
			if m.follow {
				m.viewport.GotoBottom()
			}
			return m, nil
		case "enter":
			// Confirm filter and unfocus.
			m.filtering = false
			m.filter.Blur()
			m.viewport.SetContent(m.renderLogContent())
			if m.follow {
				m.viewport.GotoBottom()
			}
			return m, nil
		default:
			// Forward to the text input.
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			// Re-render as the filter text changes.
			m.viewport.SetContent(m.renderLogContent())
			if m.follow {
				m.viewport.GotoBottom()
			}
			return m, cmd
		}
	}

	// Normal mode key bindings.
	switch msg.String() {
	case "/":
		// Focus the filter input.
		m.filtering = true
		m.filter.Focus()
		return m, textinput.Blink
	case "f":
		// Toggle follow mode.
		m.follow = !m.follow
		if m.follow {
			m.viewport.GotoBottom()
		}
		return m, nil
	case "c":
		// Clear the log buffer.
		m.entries = m.entries[:0]
		m.viewport.SetContent("")
		return m, nil
	}

	// Pass navigation keys to the viewport (pgup, pgdn, etc.).
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	// If the user scrolled manually, disable follow.
	if m.follow && !m.viewport.AtBottom() {
		m.follow = false
	}

	return m, cmd
}

// renderLogContent builds the full text content for the viewport,
// applying the current filter.
func (m LogsModel) renderLogContent() string {
	filterText := strings.ToLower(strings.TrimSpace(m.filter.Value()))

	var b strings.Builder
	for _, entry := range m.entries {
		// Apply filter — match against identity or message (case-insensitive).
		if filterText != "" {
			lower := strings.ToLower(entry.Identity + " " + entry.Message)
			if !strings.Contains(lower, filterText) {
				continue
			}
		}

		// Format: [timestamp] identity: message
		line := fmt.Sprintf("%s %s %s",
			logTimestampStyle.Render(entry.Timestamp),
			logIdentityStyle.Render(entry.Identity),
			logMessageStyle.Render(entry.Message),
		)
		b.WriteString(line)
		b.WriteString("\n")
	}

	return b.String()
}

// View renders the log viewer screen.
func (m LogsModel) View() string {
	var b strings.Builder

	// ── Filter bar ───────────────────────────────────────────────
	b.WriteString(m.renderFilterBar())
	b.WriteString("\n")

	// ── Log viewport ─────────────────────────────────────────────
	b.WriteString(m.viewport.View())

	return b.String()
}

// renderFilterBar builds the filter bar showing the input and follow state.
func (m LogsModel) renderFilterBar() string {
	var parts []string

	// Follow indicator
	if m.follow {
		parts = append(parts, logFollowOnStyle.Render("● Live"))
	} else {
		parts = append(parts, logFollowOffStyle.Render("○ Paused"))
	}

	// Filter input or hint
	if m.filtering {
		parts = append(parts, logFilterBarStyle.Render("Filter: ")+m.filter.View())
	} else {
		filterVal := m.filter.Value()
		if filterVal != "" {
			parts = append(parts, logFilterBarStyle.Render("Filter: "+filterVal))
		} else {
			parts = append(parts, logFilterBarStyle.Render("[/]filter  [f]ollow  [c]lear"))
		}
	}

	// Entry count
	parts = append(parts, lipgloss.NewStyle().
		Foreground(lipgloss.Color("243")).
		Render(fmt.Sprintf("%d entries", len(m.entries))))

	return strings.Join(parts, "  ")
}
