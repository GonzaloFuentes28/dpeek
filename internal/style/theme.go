package style

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Colors — htop-inspired palette.
var (
	ColorHeader    = lipgloss.AdaptiveColor{Light: "#1a1a2e", Dark: "#00d4aa"}
	ColorSelected  = lipgloss.AdaptiveColor{Light: "#0066cc", Dark: "#2d2d44"}
	ColorCursor    = lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#00d4aa"}
	ColorNormal    = lipgloss.AdaptiveColor{Light: "#333333", Dark: "#c8c8d0"}
	ColorDim       = lipgloss.AdaptiveColor{Light: "#999999", Dark: "#5a5a6e"}
	ColorStatusBg  = lipgloss.AdaptiveColor{Light: "#e0e0e0", Dark: "#1a1a2e"}
	ColorStatusFg  = lipgloss.AdaptiveColor{Light: "#333333", Dark: "#c8c8d0"}
	ColorFKeyBg    = lipgloss.AdaptiveColor{Light: "#333333", Dark: "#0a0a1a"}
	ColorFKeyLabel = lipgloss.AdaptiveColor{Light: "#ffffff", Dark: "#00d4aa"}
	ColorFKeyDesc  = lipgloss.AdaptiveColor{Light: "#cccccc", Dark: "#8888a0"}
	ColorModified  = lipgloss.AdaptiveColor{Light: "#cc6600", Dark: "#ffaa00"}
	ColorBorder    = lipgloss.AdaptiveColor{Light: "#cccccc", Dark: "#3a3a4e"}
	ColorError     = lipgloss.AdaptiveColor{Light: "#cc0000", Dark: "#ff4444"}
)

// Styles for the CSV table view.
var (
	HeaderStyle = lipgloss.NewStyle().
			Foreground(ColorHeader).
			Bold(true)

	CellStyle = lipgloss.NewStyle().
			Foreground(ColorNormal)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(ColorSelected)

	CursorCellStyle = lipgloss.NewStyle().
			Background(ColorSelected).
			Foreground(ColorCursor).
			Bold(true)

	RowNumberStyle = lipgloss.NewStyle().
			Foreground(ColorDim).
			Align(lipgloss.Right)

	StatusBarStyle = lipgloss.NewStyle().
			Background(ColorStatusBg).
			Foreground(ColorStatusFg)

	ModifiedStyle = lipgloss.NewStyle().
			Foreground(ColorModified).
			Background(ColorStatusBg).
			Bold(true)

	ErrorStatusStyle = lipgloss.NewStyle().
				Foreground(ColorError).
				Background(ColorStatusBg).
				Bold(true)

	TitleStyle = lipgloss.NewStyle().
			Foreground(ColorHeader).
			Bold(true)

	DimStyle = lipgloss.NewStyle().
			Foreground(ColorDim)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)
)

// FKeyItem represents a single F-key shortcut.
type FKeyItem struct {
	Key  string
	Desc string
}

// RenderFKeyBar renders the bottom F-key bar (like htop).
func RenderFKeyBar(width int, items []FKeyItem) string {
	keyStyle := lipgloss.NewStyle().
		Background(ColorFKeyBg).
		Foreground(ColorFKeyLabel).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Background(ColorFKeyBg).
		Foreground(ColorFKeyDesc)

	var parts []string
	for _, item := range items {
		parts = append(parts, keyStyle.Render(item.Key)+descStyle.Render(item.Desc))
	}

	bar := strings.Join(parts, descStyle.Render(" "))

	// Pad to full width
	barLen := lipgloss.Width(bar)
	if barLen < width {
		bar += descStyle.Render(strings.Repeat(" ", width-barLen))
	}

	return bar
}

// RenderStatusBar renders the status bar with the given items.
func RenderStatusBar(width int, items ...string) string {
	sep := StatusBarStyle.Render("  │  ")
	var rendered []string
	for _, item := range items {
		rendered = append(rendered, StatusBarStyle.Render(item))
	}
	content := strings.Join(rendered, sep)
	contentLen := lipgloss.Width(content)

	if contentLen < width {
		content += StatusBarStyle.Render(strings.Repeat(" ", width-contentLen))
	}

	return content
}
