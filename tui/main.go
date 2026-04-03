// Package main is the entry point for the SysBot.NET TUI.
// It connects to the SysBot.Pokemon.Web backend via REST and optionally
// SignalR, then launches a Bubble Tea terminal interface with three views:
// Dashboard, Logs, and Settings.
package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/thatguyinabeanie/SysBot.NET/tui/client"
	"github.com/thatguyinabeanie/SysBot.NET/tui/components"
	"github.com/thatguyinabeanie/SysBot.NET/tui/views"
)

// ── CLI flags ─────────────────────────────────────────────────────────

var (
	flagURL     = flag.String("url", "http://localhost:5050", "Base URL of the SysBot.Pokemon.Web server")
	flagManaged = flag.Bool("managed", false, "Spawn the .NET server as a subprocess")
	flagProject = flag.String("project", "SysBot.Pokemon.Web", ".NET project path for managed mode")
)

// ── App model ─────────────────────────────────────────────────────────

// appModel is the root Bubble Tea model that manages tab navigation
// and delegates to the three child views.
type appModel struct {
	activeTab int
	tabs      []string
	dashboard views.DashboardModel
	settings  views.SettingsModel
	logs      views.LogsModel
	width     int
	height    int
}

// Init returns a batch of all three child Init commands so every view
// can start its initial data fetches in parallel.
func (m appModel) Init() tea.Cmd {
	return tea.Batch(
		m.dashboard.Init(),
		m.settings.Init(),
		m.logs.Init(),
	)
}

// Update handles global key bindings (tab switching, quit) and delegates
// everything else to the currently active view.
func (m appModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {

	// ── Global key handling ──────────────────────────────────────
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		// Number keys switch tabs directly.
		case "1":
			m.activeTab = 0
			return m, nil
		case "2":
			m.activeTab = 1
			return m, nil
		case "3":
			m.activeTab = 2
			return m, nil

		// Tab key cycles through tabs.
		case "tab":
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
			return m, nil

		// All other keys go to the active view.
		default:
			return m.delegateToActiveView(msg)
		}

	// ── Window resize — propagate to all views ──────────────────
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Subtract 2 lines for the tab bar + separator.
		childSize := tea.WindowSizeMsg{
			Width:  msg.Width,
			Height: msg.Height - 2,
		}

		m.dashboard, cmd = m.dashboard.Update(childSize)
		var cmd2, cmd3 tea.Cmd
		m.settings, cmd2 = m.settings.Update(childSize)
		m.logs, cmd3 = m.logs.Update(childSize)
		return m, tea.Batch(cmd, cmd2, cmd3)

	// ── SignalR log entries → forward to logs view ───────────────
	case views.LogEntryMsg:
		m.logs, cmd = m.logs.Update(msg)
		return m, cmd

	// ── Everything else → broadcast to all views ────────────────
	// Async responses (schema loaded, bots fetched, etc.) use unexported
	// message types. Each view ignores messages it doesn't recognize.
	// Broadcasting ensures messages arrive even if that tab isn't active.
	default:
		return m.broadcastToAllViews(msg)
	}
}

// delegateToActiveView sends a message to whichever view is currently
// visible and stores the updated model back.
func (m appModel) delegateToActiveView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch m.activeTab {
	case 0:
		m.dashboard, cmd = m.dashboard.Update(msg)
	case 1:
		m.settings, cmd = m.settings.Update(msg)
	case 2:
		m.logs, cmd = m.logs.Update(msg)
	}

	return m, cmd
}

// broadcastToAllViews sends a message to every view. Used for async
// responses that may arrive while a different tab is active.
func (m appModel) broadcastToAllViews(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd1, cmd2, cmd3 tea.Cmd
	m.dashboard, cmd1 = m.dashboard.Update(msg)
	m.settings, cmd2 = m.settings.Update(msg)
	m.logs, cmd3 = m.logs.Update(msg)
	return m, tea.Batch(cmd1, cmd2, cmd3)
}

// View renders the tab bar at the top followed by the active view's content.
func (m appModel) View() string {
	// Tab bar.
	tabBar := components.RenderTabs(m.tabs, m.activeTab)

	// Active view content.
	var content string
	switch m.activeTab {
	case 0:
		content = m.dashboard.View()
	case 1:
		content = m.settings.View()
	case 2:
		content = m.logs.View()
	}

	return tabBar + "\n" + content
}

// ── Managed mode helpers ──────────────────────────────────────────────

// startManagedServer spawns `dotnet run --project <path>` as a subprocess
// and returns the exec.Cmd. The caller is responsible for cleanup.
func startManagedServer(project string) *exec.Cmd {
	cmd := exec.Command("dotnet", "run", "--project", project)
	// Pipe stdout/stderr to /dev/null — the TUI will read logs via SignalR.
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start managed server: %v\n", err)
		os.Exit(1)
	}

	return cmd
}

// waitForServer polls GET {url}/api/meta every 500ms until it returns 200
// or the timeout (30s) is reached. Returns an error on timeout.
func waitForServer(url string) error {
	deadline := time.Now().Add(30 * time.Second)
	endpoint := url + "/api/meta"

	for time.Now().Before(deadline) {
		resp, err := http.Get(endpoint)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	return fmt.Errorf("server did not become ready within 30 seconds")
}

// stopManagedServer sends SIGINT to the subprocess. If it doesn't exit
// within 5 seconds, it forcefully kills it.
func stopManagedServer(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}

	// Try graceful shutdown first.
	_ = cmd.Process.Signal(os.Interrupt)

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-done:
		// Process exited gracefully.
	case <-time.After(5 * time.Second):
		// Force kill after timeout.
		_ = cmd.Process.Kill()
	}
}

// ── Main ──────────────────────────────────────────────────────────────

func main() {
	flag.Parse()

	serverURL := *flagURL

	// ── Managed mode: spawn and wait for the .NET server ─────────
	var managedCmd *exec.Cmd
	if *flagManaged {
		fmt.Printf("Starting managed server (%s)...\n", *flagProject)
		managedCmd = startManagedServer(*flagProject)

		// Make sure we clean up even on unexpected signals.
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		go func() {
			<-sigCh
			stopManagedServer(managedCmd)
			os.Exit(0)
		}()

		fmt.Printf("Waiting for server at %s...\n", serverURL)
		if err := waitForServer(serverURL); err != nil {
			fmt.Fprintf(os.Stderr, "Could not connect to server at %s. Is it running?\n", serverURL)
			stopManagedServer(managedCmd)
			os.Exit(1)
		}
	}

	// ── Create clients ──────────────────────────────────────────
	fmt.Printf("Connecting to %s...\n", serverURL)

	restClient := client.NewClient(serverURL)

	// SignalR is optional — if it fails, logs just won't stream in real time.
	signalrClient, signalrErr := client.NewSignalRClient(serverURL)
	if signalrErr != nil {
		fmt.Fprintf(os.Stderr, "SignalR connection failed (logs won't stream): %v\n", signalrErr)
	}

	// ── Verify the server is actually reachable ─────────────────
	if !*flagManaged {
		_, err := restClient.GetMeta()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not connect to server at %s. Is it running?\n", serverURL)
			os.Exit(1)
		}
	}

	// ── Build the root model ────────────────────────────────────
	model := appModel{
		activeTab: 0,
		tabs:      []string{"1: Dashboard", "2: Settings", "3: Logs"},
		dashboard: views.NewDashboard(restClient),
		settings:  views.NewSettings(restClient),
		logs:      views.NewLogs(),
	}

	// ── Create and run the Bubble Tea program ───────────────────
	p := tea.NewProgram(model, tea.WithAltScreen())

	// Wire SignalR callbacks to push messages into the Bubble Tea event loop.
	if signalrClient != nil {
		signalrClient.OnLog = func(entry client.LogEntry) {
			p.Send(views.LogEntryMsg(entry))
		}
		signalrClient.OnEcho = func(ts, msg string) {
			p.Send(views.LogEntryMsg(client.LogEntry{
				Timestamp: ts,
				Identity:  "Echo",
				Message:   msg,
			}))
		}
		go signalrClient.Listen()
	}

	// Run blocks until the user quits.
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
	}

	// ── Cleanup ─────────────────────────────────────────────────
	if signalrClient != nil {
		signalrClient.Close()
	}
	if managedCmd != nil {
		stopManagedServer(managedCmd)
	}
}
