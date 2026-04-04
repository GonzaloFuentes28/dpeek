package model

import (
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/GonzaloFuentes28/dpeek/internal/style"
)

var helpText = []struct {
	key  string
	desc string
}{
	{"↑/↓/←/→, hjkl", "Navigate cells"},
	{"PgUp/PgDn", "Scroll page up/down"},
	{"Home/End", "Jump to first/last cell"},
	{"Enter", "Edit selected cell"},
	{"Tab", "Confirm edit, move to next cell"},
	{"Esc", "Cancel edit / close overlay"},
	{"", ""},
	{"F1", "Toggle this help"},
	{"F2, Ctrl+S", "Save file"},
	{"F3, /", "Search (prefix / for regex)"},
	{"F4", "Filter rows (col:value for column)"},
	{"F5", "Sort by current column"},
	{"F6", "Column statistics"},
	{"Ctrl+G", "Go to line"},
	{"F10, q", "Quit"},
	{"", ""},
	{"y", "Copy to clipboard"},
	{"p", "Paste from clipboard"},
	{"Ctrl+Z", "Undo"},
	{"Ctrl+Y", "Redo"},
	{"n/N", "Next/previous search match"},
}

// RenderHelp renders a centered help overlay.
func RenderHelp(width, height int) string {
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.ColorHeader).
		Padding(1, 2)

	titleStyle := lipgloss.NewStyle().
		Foreground(style.ColorHeader).
		Bold(true)

	keyStyle := lipgloss.NewStyle().
		Foreground(style.ColorFKeyLabel).
		Bold(true).
		Width(18)

	descStyle := lipgloss.NewStyle().
		Foreground(style.ColorNormal)

	var lines []string
	lines = append(lines, titleStyle.Render("dpeek — Keyboard Shortcuts"))
	lines = append(lines, "")

	for _, h := range helpText {
		if h.key == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, keyStyle.Render(h.key)+descStyle.Render(h.desc))
	}

	lines = append(lines, "")
	lines = append(lines, style.DimStyle.Render("Press F1 or Esc to close"))

	content := boxStyle.Render(strings.Join(lines, "\n"))

	// Center in viewport
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	padLeft := ""
	if width > contentWidth {
		padLeft = strings.Repeat(" ", (width-contentWidth)/2)
	}

	padTop := ""
	if height > contentHeight {
		padTop = strings.Repeat("\n", (height-contentHeight)/2)
	}

	// Indent each line
	var centeredLines []string
	for _, line := range strings.Split(content, "\n") {
		centeredLines = append(centeredLines, padLeft+line)
	}

	return padTop + strings.Join(centeredLines, "\n")
}
