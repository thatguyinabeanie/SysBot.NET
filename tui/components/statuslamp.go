// Package components provides shared UI building blocks for the TUI views.
package components

import (
	"time"

	"github.com/charmbracelet/lipgloss"
)

// Color constants for status lamps.
var (
	colorGray   = lipgloss.Color("240") // not running
	colorCyan   = lipgloss.Color("6")   // running but not connected
	colorYellow = lipgloss.Color("3")   // paused or idle, or stale 30-60s
	colorGreen  = lipgloss.Color("2")   // active, recent (<30s)
	colorRed    = lipgloss.Color("1")   // active, stale (>60s)
)

// StatusLamp returns a colored Unicode dot representing the bot's current state.
//
// The logic follows this priority order:
//  1. Not running at all                        -> gray hollow circle
//  2. Running but not connected to the Switch    -> cyan dotted circle
//  3. Paused, or both routines are "Idle"        -> yellow filled circle
//  4. Active with lastActive < 30s ago           -> green filled circle
//  5. Active with lastActive < 60s ago           -> yellow filled circle
//  6. Active with lastActive >= 60s ago          -> red filled circle
func StatusLamp(isRunning, isPaused, isConnected bool, currentRoutine, nextRoutine string, lastActive time.Time) string {
	// 1. Not running
	if !isRunning {
		return lipgloss.NewStyle().Foreground(colorGray).Render("○")
	}

	// 2. Running but not connected
	if !isConnected {
		return lipgloss.NewStyle().Foreground(colorCyan).Render("◌")
	}

	// 3. Paused or both routines idle
	if isPaused || (currentRoutine == "Idle" && nextRoutine == "Idle") {
		return lipgloss.NewStyle().Foreground(colorYellow).Render("●")
	}

	// 4-6. Active — color based on how recently the bot was active.
	elapsed := time.Since(lastActive)

	switch {
	case elapsed < 30*time.Second:
		return lipgloss.NewStyle().Foreground(colorGreen).Render("●")
	case elapsed < 60*time.Second:
		return lipgloss.NewStyle().Foreground(colorYellow).Render("●")
	default:
		return lipgloss.NewStyle().Foreground(colorRed).Render("●")
	}
}
