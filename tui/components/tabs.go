package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Tab bar styles.
var (
	// activeTabStyle is applied to the currently selected tab.
	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Background(lipgloss.Color("4")).
			Foreground(lipgloss.Color("15")).
			Padding(0, 2)

	// inactiveTabStyle is applied to unselected tabs.
	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250")).
				Padding(0, 2)

	// tabSeparator sits between tabs.
	tabSeparator = lipgloss.NewStyle().
			Foreground(lipgloss.Color("238")).
			Render("│")
)

// RenderTabs builds a horizontal tab bar from the given labels.
// The tab at index `active` is highlighted with a bold style and background color.
func RenderTabs(tabs []string, active int) string {
	return RenderTabsWithFocus(tabs, active, true)
}

// RenderTabsWithFocus builds a horizontal tab bar. When focused is true,
// the active tab gets an underline to indicate it can be navigated.
func RenderTabsWithFocus(tabs []string, active int, focused bool) string {
	rendered := make([]string, len(tabs))
	for i, t := range tabs {
		if i == active {
			style := activeTabStyle
			if focused {
				style = style.Underline(true)
			}
			rendered[i] = style.Render(t)
		} else {
			rendered[i] = inactiveTabStyle.Render(t)
		}
	}
	return strings.Join(rendered, tabSeparator)
}

// Sidebar styles.
var (
	// activeSidebarItem is the style for the currently selected sidebar entry.
	activeSidebarItem = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("4"))

	// inactiveSidebarItem is the style for unselected sidebar entries.
	inactiveSidebarItem = lipgloss.NewStyle().
				Foreground(lipgloss.Color("250"))
)

// RenderSidebar builds a vertical list of items. The item at index `active`
// is prefixed with "► " and highlighted. Each line is padded/truncated to
// fit within `width` columns.
func RenderSidebar(items []string, active int, width int) string {
	lines := make([]string, len(items))
	for i, item := range items {
		var line string
		if i == active {
			// Active item gets an arrow prefix.
			line = "► " + item
			line = truncateOrPad(line, width)
			lines[i] = activeSidebarItem.Render(line)
		} else {
			// Inactive items are indented to align with the active item text.
			line = "  " + item
			line = truncateOrPad(line, width)
			lines[i] = inactiveSidebarItem.Render(line)
		}
	}
	return strings.Join(lines, "\n")
}

// truncateOrPad ensures a string is exactly `width` runes long,
// padding with spaces or trimming with an ellipsis as needed.
func truncateOrPad(s string, width int) string {
	runes := []rune(s)
	if len(runes) > width {
		if width > 1 {
			return string(runes[:width-1]) + "…"
		}
		return string(runes[:width])
	}
	if len(runes) < width {
		return s + strings.Repeat(" ", width-len(runes))
	}
	return s
}
