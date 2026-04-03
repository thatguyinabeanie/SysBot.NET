package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/thatguyinabeanie/SysBot.NET/tui/client"
)

// ── Settings message types ────────────────────────────────────────────

// schemaMsg carries the fetched config schema.
type schemaMsg struct {
	schema *client.ConfigSchema
}

// configMsg carries the fetched hub config values.
type configMsg struct {
	config map[string]any
}

// saveResultMsg carries the result of a save operation.
type saveResultMsg struct {
	err error
}

// ── Focus areas in the settings view ──────────────────────────────────
const (
	focusSidebar = 0 // sidebar (categories + sections tree)
	focusFields  = 1 // field list
)

// ── Sidebar item ─────────────────────────────────────────────────────

// sidebarItem is either a category header or a section under a category.
type sidebarItem struct {
	Label    string
	Category string
	Section  string // empty for category headers
	IsHeader bool   // true = category header, false = section
}

// ── Field entry ───────────────────────────────────────────────────────

// fieldEntry represents a single editable field in the current section.
type fieldEntry struct {
	Name   string
	Schema *client.SchemaProperty
	Value  any
}

// ── Settings model ────────────────────────────────────────────────────

// SettingsModel is the Bubble Tea model for the settings editor.
type SettingsModel struct {
	client       *client.Client
	schema       *client.ConfigSchema
	config       map[string]any  // flat hub config from API
	edits        map[string]any  // pending edits to save
	sidebarItems []sidebarItem   // flat list of category headers + sections
	sidebarIdx   int             // cursor position in sidebar
	fields       []fieldEntry    // fields for selected section
	activeField  int             // selected field index
	focus        int             // focusSidebar or focusFields
	editing      bool            // true when inline editing a text/number field
	editInput    textinput.Model // text input for inline editing
	editKey      string          // full config key being edited
	width        int
	height       int
	err          error
}

// NewSettings creates a new SettingsModel wired to the given API client.
func NewSettings(c *client.Client) SettingsModel {
	ti := textinput.New()
	ti.CharLimit = 512
	ti.Width = 40

	return SettingsModel{
		client:     c,
		edits:      make(map[string]any),
		focus:      focusSidebar,
		sidebarIdx: 0,
		editInput:  ti,
	}
}

// ── Commands ──────────────────────────────────────────────────────────

func fetchSchema(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		schema, err := c.GetConfigSchema()
		if err != nil {
			return errMsg{err}
		}
		return schemaMsg{schema}
	}
}

func fetchConfig(c *client.Client) tea.Cmd {
	return func() tea.Msg {
		cfg, err := c.GetHubConfig()
		if err != nil {
			return errMsg{err}
		}
		return configMsg{cfg}
	}
}

func saveConfig(c *client.Client, patch map[string]any) tea.Cmd {
	return func() tea.Msg {
		err := c.PatchHubConfig(patch)
		return saveResultMsg{err}
	}
}

// ── Tea interface ─────────────────────────────────────────────────────

// Init fetches the config schema and current config values in parallel.
func (m SettingsModel) Init() tea.Cmd {
	return tea.Batch(
		fetchSchema(m.client),
		fetchConfig(m.client),
	)
}

// Update handles keyboard input and async responses.
func (m SettingsModel) Update(msg tea.Msg) (SettingsModel, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case schemaMsg:
		m.schema = msg.schema
		m.buildSidebar()
		m.rebuildFields()
		return m, nil

	case configMsg:
		m.config = msg.config
		m.rebuildFields()
		return m, nil

	case saveResultMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			// Clear edits on successful save and re-fetch.
			m.edits = make(map[string]any)
			m.err = nil
			return m, fetchConfig(m.client)
		}
		return m, nil

	case errMsg:
		m.err = msg.err
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// handleKey processes keyboard input for the settings view.
func (m SettingsModel) handleKey(msg tea.KeyMsg) (SettingsModel, tea.Cmd) {
	// If inline editing, route keys to the text input.
	if m.editing {
		switch msg.String() {
		case "esc":
			// Cancel editing.
			m.editing = false
			m.editInput.Blur()
			return m, nil
		case "enter":
			// Confirm the edit.
			m.editing = false
			m.editInput.Blur()
			value := m.editInput.Value()
			if m.editKey != "" {
				m.edits[m.editKey] = value
			}
			// Update the field in memory so the view reflects the change immediately.
			if m.activeField < len(m.fields) {
				m.fields[m.activeField].Value = value
			}
			return m, nil
		default:
			var cmd tea.Cmd
			m.editInput, cmd = m.editInput.Update(msg)
			return m, cmd
		}
	}

	switch msg.String() {

	// ── Enter: drill deeper or activate field ───────────────────
	case "enter":
		switch m.focus {
		case focusSidebar:
			// Only enter fields if the cursor is on a section (not a header).
			if m.sidebarIdx >= 0 && m.sidebarIdx < len(m.sidebarItems) {
				item := m.sidebarItems[m.sidebarIdx]
				if !item.IsHeader && len(m.fields) > 0 {
					m.focus = focusFields
					m.activeField = 0
				}
			}
		case focusFields:
			// Activate the selected field (toggle, cycle, edit).
			if m.activeField < len(m.fields) {
				return m.activateField()
			}
		}
		return m, nil

	// ── Esc: go back one level ──────────────────────────────────
	case "esc":
		if m.focus == focusFields {
			m.focus = focusSidebar
		}
		return m, nil

	// ── j/k or arrows: navigate within current focus ────────────
	case "j", "down":
		switch m.focus {
		case focusSidebar:
			if len(m.sidebarItems) > 0 && m.sidebarIdx < len(m.sidebarItems)-1 {
				m.sidebarIdx++
				m.activeField = 0
				m.rebuildFields()
			}
		case focusFields:
			if m.activeField < len(m.fields)-1 {
				m.activeField++
			}
		}
		return m, nil
	case "k", "up":
		switch m.focus {
		case focusSidebar:
			if m.sidebarIdx > 0 {
				m.sidebarIdx--
				m.activeField = 0
				m.rebuildFields()
			}
		case focusFields:
			if m.activeField > 0 {
				m.activeField--
			}
		}
		return m, nil

	// ── Save ─────────────────────────────────────────────────────
	case "ctrl+s":
		if len(m.edits) > 0 {
			patch := make(map[string]any, len(m.edits))
			for k, v := range m.edits {
				patch[k] = v
			}
			return m, saveConfig(m.client, patch)
		}
		return m, nil
	}

	return m, nil
}

// activateField handles Enter on the currently selected field.
// Booleans toggle immediately, enums cycle, text/number opens the editor.
func (m SettingsModel) activateField() (SettingsModel, tea.Cmd) {
	f := m.fields[m.activeField]
	key := m.fieldKey(f.Name)

	switch f.Schema.Type {
	case "boolean":
		// Toggle the boolean value.
		current, _ := f.Value.(bool)
		newVal := !current
		m.edits[key] = newVal
		m.fields[m.activeField].Value = newVal

	case "enum":
		// Cycle to the next enum value.
		if len(f.Schema.EnumValues) > 0 {
			currentStr := fmt.Sprintf("%v", f.Value)
			nextIdx := 0
			for i, ev := range f.Schema.EnumValues {
				if ev == currentStr {
					nextIdx = (i + 1) % len(f.Schema.EnumValues)
					break
				}
			}
			newVal := f.Schema.EnumValues[nextIdx]
			m.edits[key] = newVal
			m.fields[m.activeField].Value = newVal
		}

	default:
		// Open inline text editor for string, integer, etc.
		m.editing = true
		m.editKey = key
		m.editInput.SetValue(fmt.Sprintf("%v", f.Value))
		m.editInput.Focus()
		return m, textinput.Blink
	}

	return m, nil
}

// ── Data helpers ──────────────────────────────────────────────────────

// buildSidebar constructs a flat list of sidebar items from the schema.
// Categories are sorted alphabetically, and sections within each category
// are also sorted alphabetically.
func (m *SettingsModel) buildSidebar() {
	m.sidebarItems = nil
	if m.schema == nil {
		return
	}

	// Collect and sort category names.
	cats := make([]string, 0, len(m.schema.Categories))
	for cat := range m.schema.Categories {
		cats = append(cats, cat)
	}
	sort.Strings(cats)

	// Build the flat sidebar list with headers and sections.
	for _, cat := range cats {
		// Add the category header.
		m.sidebarItems = append(m.sidebarItems, sidebarItem{
			Label:    cat,
			Category: cat,
			IsHeader: true,
		})

		// Collect and sort sections within this category.
		sectionMap := m.schema.Categories[cat]
		secs := make([]string, 0, len(sectionMap))
		for sec := range sectionMap {
			secs = append(secs, sec)
		}
		sort.Strings(secs)

		// Add each section under the category.
		for _, sec := range secs {
			m.sidebarItems = append(m.sidebarItems, sidebarItem{
				Label:    sec,
				Category: cat,
				Section:  sec,
				IsHeader: false,
			})
		}
	}

	// Reset cursor to the first section (skip leading header).
	m.sidebarIdx = 0
	if len(m.sidebarItems) > 1 && m.sidebarItems[0].IsHeader {
		m.sidebarIdx = 1
	}
}

// rebuildFields builds the field list for the currently selected sidebar item.
// If the cursor is on a category header, no fields are shown.
func (m *SettingsModel) rebuildFields() {
	m.fields = nil
	if m.schema == nil || len(m.sidebarItems) == 0 {
		return
	}

	// Clamp sidebar index.
	if m.sidebarIdx < 0 || m.sidebarIdx >= len(m.sidebarItems) {
		return
	}

	item := m.sidebarItems[m.sidebarIdx]

	// Category headers have no fields to display.
	if item.IsHeader {
		return
	}

	cat := item.Category
	sec := item.Section

	prop, ok := m.schema.Categories[cat][sec]
	if !ok || prop == nil {
		return
	}

	// If the section property is an object with nested properties, list them.
	if prop.Properties != nil {
		for name, sp := range prop.Properties {
			val := m.getFieldValue(cat, sec, name, sp)
			m.fields = append(m.fields, fieldEntry{
				Name:   name,
				Schema: sp,
				Value:  val,
			})
		}
	} else {
		// The section itself is a leaf property.
		val := m.getFieldValue(cat, sec, "", prop)
		m.fields = append(m.fields, fieldEntry{
			Name:   sec,
			Schema: prop,
			Value:  val,
		})
	}

	// Sort fields alphabetically for a stable display.
	sort.Slice(m.fields, func(i, j int) bool {
		return m.fields[i].Name < m.fields[j].Name
	})

	// Clamp activeField.
	if m.activeField >= len(m.fields) {
		if len(m.fields) > 0 {
			m.activeField = len(m.fields) - 1
		} else {
			m.activeField = 0
		}
	}
}

// getFieldValue looks up the current value for a field, checking edits first,
// then the fetched config, then the schema default.
func (m *SettingsModel) getFieldValue(cat, sec, name string, sp *client.SchemaProperty) any {
	key := m.buildFieldKey(cat, sec, name)

	// Check pending edits first.
	if v, ok := m.edits[key]; ok {
		return v
	}

	// Check the live config.
	if m.config != nil {
		// Try to navigate the nested config map.
		if catMap, ok := m.config[cat]; ok {
			if cm, ok := catMap.(map[string]any); ok {
				if secVal, ok := cm[sec]; ok {
					if name == "" {
						return secVal
					}
					if sm, ok := secVal.(map[string]any); ok {
						if v, ok := sm[name]; ok {
							return v
						}
					}
				}
			}
		}
	}

	// Fall back to schema default.
	if sp != nil {
		return sp.Value
	}
	return nil
}

// fieldKey builds the config patch key for the currently selected section/field.
func (m *SettingsModel) fieldKey(fieldName string) string {
	if len(m.sidebarItems) == 0 || m.sidebarIdx < 0 || m.sidebarIdx >= len(m.sidebarItems) {
		return fieldName
	}
	item := m.sidebarItems[m.sidebarIdx]
	if item.IsHeader {
		return fieldName
	}
	return m.buildFieldKey(item.Category, item.Section, fieldName)
}

// buildFieldKey constructs a dotted config key path.
func (m *SettingsModel) buildFieldKey(cat, sec, name string) string {
	if name == "" {
		return cat + "." + sec
	}
	return cat + "." + sec + "." + name
}

// ── View rendering ────────────────────────────────────────────────────

// Settings view styles.
var (
	settsSidebarStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238")).
				Padding(1, 1)

	settsFieldNameStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Bold(true)

	settsFieldValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("6"))

	settsFieldDescStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("243")).
				Italic(true)

	settsSelectedStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("236")).
				Bold(true)

	settsEditedMarker = lipgloss.NewStyle().
				Foreground(lipgloss.Color("3")).
				Bold(true).
				Render("*")

	settsHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	settsErrStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("1")).
			Bold(true)

	settsSavedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("2")).
			Bold(true)

	// Sidebar-specific styles for the tree view.
	settsCatHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("243"))

	settsSectionActiveStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("4"))

	settsSectionInactiveStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("250"))

	settsFocusBorderColor   = lipgloss.Color("4")
	settsUnfocusBorderColor = lipgloss.Color("238")
)

// View renders the settings editor.
func (m SettingsModel) View() string {
	if m.schema == nil {
		return "  Loading settings..."
	}

	var b strings.Builder

	// ── Error display ────────────────────────────────────────────
	if m.err != nil {
		b.WriteString(settsErrStyle.Render("Error: " + m.err.Error()))
		b.WriteString("\n")
	}

	// Layout: sidebar on left, content on right.
	sidebarWidth := 28
	contentWidth := m.width - sidebarWidth - 4
	if contentWidth < 30 {
		contentWidth = 30
	}

	// ── Left sidebar (category headers + sections tree) ──────────
	sidebarContent := m.renderSidebar(sidebarWidth - 4)

	sidebarBorder := settsSidebarStyle
	if m.focus == focusSidebar {
		sidebarBorder = sidebarBorder.BorderForeground(settsFocusBorderColor)
	} else {
		sidebarBorder = sidebarBorder.BorderForeground(settsUnfocusBorderColor)
	}
	sidebar := sidebarBorder.Width(sidebarWidth).Render(sidebarContent)

	// ── Right content (fields for the selected section) ──────────
	var content strings.Builder

	// Show a section title at the top of the fields area.
	if m.sidebarIdx >= 0 && m.sidebarIdx < len(m.sidebarItems) {
		item := m.sidebarItems[m.sidebarIdx]
		if !item.IsHeader {
			title := item.Category + " > " + item.Section
			content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")).Render(title))
			content.WriteString("\n\n")
		}
	}

	// Field list.
	if len(m.fields) == 0 {
		if m.sidebarIdx >= 0 && m.sidebarIdx < len(m.sidebarItems) && m.sidebarItems[m.sidebarIdx].IsHeader {
			content.WriteString("  Select a section to view fields.")
		} else {
			content.WriteString("  No fields in this section.")
		}
	} else {
		for i, f := range m.fields {
			line := m.renderField(i, f)
			if i == m.activeField && m.focus == focusFields {
				// Replace leading spaces with a "› " indicator.
				line = "› " + strings.TrimLeft(line, " ")
				content.WriteString(line)
			} else {
				content.WriteString(line)
			}
			content.WriteString("\n")
		}
	}

	// Join sidebar and content side by side.
	joined := lipgloss.JoinHorizontal(lipgloss.Top, sidebar, "  ", content.String())
	b.WriteString(joined)
	b.WriteString("\n\n")

	// ── Inline editor ────────────────────────────────────────────
	if m.editing {
		b.WriteString("  Edit: ")
		b.WriteString(m.editInput.View())
		b.WriteString("\n")
	}

	// ── Status / help bar ────────────────────────────────────────
	var helpParts []string
	switch m.focus {
	case focusSidebar:
		helpParts = append(helpParts, "[j/k]navigate  [enter]select section  [ctrl+s]save")
	case focusFields:
		helpParts = append(helpParts, "[j/k]navigate  [enter]edit  [esc]back  [ctrl+s]save")
	}
	if len(m.edits) > 0 {
		helpParts = append(helpParts, fmt.Sprintf("  %d unsaved change(s)", len(m.edits)))
	}
	b.WriteString(settsHelpStyle.Render(strings.Join(helpParts, "")))

	return b.String()
}

// renderSidebar builds the sidebar tree with category headers and indented sections.
func (m SettingsModel) renderSidebar(width int) string {
	lines := make([]string, len(m.sidebarItems))
	for i, item := range m.sidebarItems {
		if item.IsHeader {
			// Category headers are rendered in bold/dim, not selectable via Enter.
			label := item.Label
			if i == m.sidebarIdx {
				// Cursor is on a header — highlight it but differently.
				label = "► " + label
			} else {
				label = "  " + label
			}
			label = truncateOrPad(label, width)
			lines[i] = settsCatHeaderStyle.Render(label)
		} else {
			// Section items are indented under their category.
			var label string
			if i == m.sidebarIdx {
				label = "  ► " + item.Label
				label = truncateOrPad(label, width)
				lines[i] = settsSectionActiveStyle.Render(label)
			} else {
				label = "    " + item.Label
				label = truncateOrPad(label, width)
				lines[i] = settsSectionInactiveStyle.Render(label)
			}
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

// renderField formats a single field as a two-line block:
//
//	Line 1: Name ···················· Value [*]
//	Line 2: Description (truncated to fit)
func (m SettingsModel) renderField(idx int, f fieldEntry) string {
	// Available width for the field content (minus sidebar + padding).
	maxWidth := m.width - 34
	if maxWidth < 40 {
		maxWidth = 40
	}

	// Check if this field has pending edits.
	key := m.fieldKey(f.Name)
	edited := ""
	if _, ok := m.edits[key]; ok {
		edited = " *"
	}

	// Format the value.
	valStr := formatValue(f.Schema, f.Value)

	// Line 1: Name + dots + value.
	name := f.Name
	value := valStr + edited
	nameLen := len([]rune(name))
	valueLen := len([]rune(value))
	dotsLen := maxWidth - nameLen - valueLen - 4 // 2 padding + 2 spaces around dots
	if dotsLen < 2 {
		dotsLen = 2
	}
	dots := settsFieldDescStyle.Render(strings.Repeat("·", dotsLen))

	line1 := fmt.Sprintf("  %s %s %s",
		settsFieldNameStyle.Render(name),
		dots,
		settsFieldValueStyle.Render(value),
	)

	// Line 2: Description (truncated).
	if f.Schema != nil && f.Schema.Description != "" {
		desc := f.Schema.Description
		descRunes := []rune(desc)
		descMax := maxWidth - 4
		if descMax > 0 && len(descRunes) > descMax {
			desc = string(descRunes[:descMax-1]) + "…"
		}
		line2 := "    " + settsFieldDescStyle.Render(desc)
		return line1 + "\n" + line2
	}

	return line1
}

// formatValue converts a field value to a display string based on its schema type.
func formatValue(sp *client.SchemaProperty, val any) string {
	if val == nil {
		return "<not set>"
	}

	if sp != nil && sp.Type == "boolean" {
		b, ok := val.(bool)
		if ok {
			if b {
				return "✓ true"
			}
			return "✗ false"
		}
	}

	return fmt.Sprintf("%v", val)
}
