package model

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	jsonpkg "github.com/GonzaloFuentes28/dpeek/internal/json"
	"github.com/GonzaloFuentes28/dpeek/internal/style"
)

// Styles for JSON tree rendering.
var (
	jsonKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#0550ae", Dark: "#79c0ff"}).
			Bold(true)

	jsonStringStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#0a3069", Dark: "#a5d6ff"})

	jsonNumberStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#0550ae", Dark: "#79c0ff"})

	jsonBoolStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#953800", Dark: "#ffa657"})

	jsonNullStyle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#999999", Dark: "#5a5a6e"})

	jsonBracketStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#8b949e"})

	jsonCursorStyle = lipgloss.NewStyle().
			Background(style.ColorSelected).
			Bold(true)

	jsonMatchStyle = lipgloss.NewStyle().
			Background(lipgloss.AdaptiveColor{Light: "#fff3cd", Dark: "#3a3a00"})

	jsonCurrentMatchStyle = lipgloss.NewStyle().
				Background(style.ColorHeader).
				Foreground(lipgloss.Color("#000000")).
				Bold(true)
)

// JSONModel is the bubbletea model for the JSON tree view.
type JSONModel struct {
	root     *jsonpkg.Node
	visible  []*jsonpkg.Node // cached flat list of visible nodes
	cursor   int
	scrollOff int
	filePath string
	isJSONL  bool // true if the source was a JSONL file

	width  int
	height int

	// Editing
	editing   bool
	editInput textinput.Model
	undo      UndoStack[NodeEdit]

	// Go-to-line
	gotoActive bool
	gotoInput  textinput.Model

	// Features
	search        SearchState
	activeOverlay overlay

	// Progressive loading
	loading     bool
	chunkReader *jsonpkg.JSONLChunkReader

	statusMsg string
}

// jsonlChunkMsg delivers a batch of nodes from background JSONL loading.
type jsonlChunkMsg struct {
	Nodes []*jsonpkg.Node
}

// jsonlLoadDoneMsg signals that all JSONL lines have been loaded.
type jsonlLoadDoneMsg struct{}

// jsonlLoadErrMsg signals a JSONL loading error.
type jsonlLoadErrMsg struct{ Err error }

// NewJSONModel creates a new JSON tree model.
func NewJSONModel(root *jsonpkg.Node, filePath string, isJSONL bool) JSONModel {
	ti := textinput.New()
	ti.CharLimit = 512

	gi := textinput.New()
	gi.Placeholder = "Node number..."
	gi.CharLimit = 10

	m := JSONModel{
		root:      root,
		filePath:  filePath,
		isJSONL:   isJSONL,
		search:    NewSearchState(),
		editInput: ti,
		gotoInput: gi,
		undo:      NewUndoStack[NodeEdit](1000),
	}
	m.refreshVisible()
	return m
}

// NewJSONModelChunked creates a JSON model with progressive JSONL loading.
func NewJSONModelChunked(root *jsonpkg.Node, filePath string, cr *jsonpkg.JSONLChunkReader) JSONModel {
	ti := textinput.New()
	ti.CharLimit = 512

	gi := textinput.New()
	gi.Placeholder = "Node number..."
	gi.CharLimit = 10

	m := JSONModel{
		root:        root,
		filePath:    filePath,
		isJSONL:     true,
		search:      NewSearchState(),
		editInput:   ti,
		gotoInput:   gi,
		undo:        NewUndoStack[NodeEdit](1000),
		loading:     true,
		chunkReader: cr,
	}
	m.refreshVisible()
	return m
}

func (m JSONModel) readNextJSONLChunkCmd() tea.Cmd {
	cr := m.chunkReader
	return func() tea.Msg {
		nodes, err := cr.ReadNextChunk(jsonpkg.DefaultJSONLChunkSize)
		if err != nil {
			return jsonlLoadErrMsg{Err: err}
		}
		if len(nodes) == 0 {
			return jsonlLoadDoneMsg{}
		}
		return jsonlChunkMsg{Nodes: nodes}
	}
}

func (m *JSONModel) refreshVisible() {
	m.visible = m.root.VisibleNodes()
}

func (m *JSONModel) viewRows() int {
	// Reserve: title, status bar, fkey bar
	rows := m.height - 3
	if rows < 1 {
		return 1
	}
	return rows
}

// HasActiveInput returns true if an input or overlay is active.
func (m JSONModel) HasActiveInput() bool {
	return m.editing || m.search.Active || m.gotoActive || m.activeOverlay != overlayNone
}

// HasDismissableState returns true if Esc has something to close/clear.
func (m JSONModel) HasDismissableState() bool {
	return m.HasActiveInput() || m.search.Query != ""
}

// Init implements tea.Model.
func (m JSONModel) Init() tea.Cmd {
	if m.loading && m.chunkReader != nil {
		return m.readNextJSONLChunkCmd()
	}
	return nil
}

// Update implements tea.Model.
func (m JSONModel) Update(msg tea.Msg) (JSONModel, tea.Cmd) {
	if m.editing {
		return m.updateEditing(msg)
	}
	if m.search.Active {
		return m.updateSearch(msg)
	}
	if m.gotoActive {
		return m.updateGoto(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case jsonlChunkMsg:
		for _, node := range msg.Nodes {
			node.Parent = m.root
			m.root.Children = append(m.root.Children, node)
		}
		m.refreshVisible()
		return m, m.readNextJSONLChunkCmd()

	case jsonlLoadDoneMsg:
		m.loading = false
		if m.chunkReader != nil {
			m.chunkReader.Close()
			m.chunkReader = nil
		}
		m.refreshVisible()
		m.statusMsg = fmt.Sprintf("Loaded %d items", len(m.root.Children))
		return m, nil

	case jsonlLoadErrMsg:
		m.loading = false
		if m.chunkReader != nil {
			m.chunkReader.Close()
			m.chunkReader = nil
		}
		m.statusMsg = fmt.Sprintf("Load error: %v", msg.Err)
		return m, nil

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.cursor > 0 {
				m.cursor -= 3
				if m.cursor < 0 {
					m.cursor = 0
				}
				m.ensureVisible()
			}
		case tea.MouseButtonWheelDown:
			if m.cursor < len(m.visible)-1 {
				m.cursor += 3
				if m.cursor >= len(m.visible) {
					m.cursor = len(m.visible) - 1
				}
				m.ensureVisible()
			}
		}

	case tea.KeyMsg:
		if m.activeOverlay != overlayNone {
			switch msg.String() {
			case "esc", "f1":
				m.activeOverlay = overlayNone
			}
			return m, nil
		}

		m.statusMsg = ""
		switch msg.String() {
		// Navigation
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				m.ensureVisible()
			}
		case "down", "j":
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				m.ensureVisible()
			}
		case "pgup":
			m.cursor -= m.viewRows()
			if m.cursor < 0 {
				m.cursor = 0
			}
			m.ensureVisible()
		case "pgdown":
			m.cursor += m.viewRows()
			if m.cursor >= len(m.visible) {
				m.cursor = len(m.visible) - 1
			}
			m.ensureVisible()
		case "home":
			m.cursor = 0
			m.scrollOff = 0
		case "end":
			m.cursor = len(m.visible) - 1
			m.ensureVisible()

		// Edit / Expand/collapse
		case "enter", " ":
			if m.cursor < len(m.visible) {
				node := m.visible[m.cursor]
				if node.IsLeaf() {
					// Edit leaf value
					m.startEditing(node)
				} else {
					node.Toggle()
					m.refreshVisible()
					if m.cursor >= len(m.visible) {
						m.cursor = len(m.visible) - 1
					}
				}
			}

		// Save
		case "f2", "ctrl+s":
			if m.undo.IsModified() {
				var err error
				if m.isJSONL {
					err = jsonpkg.SaveJSONL(m.root, m.filePath)
				} else {
					err = jsonpkg.Save(m.root, m.filePath)
				}
				if err != nil {
					m.statusMsg = fmt.Sprintf("Error: %v", err)
				} else {
					m.undo.MarkSaved()
					m.statusMsg = "Saved!"
				}
			} else {
				m.statusMsg = "No changes to save"
			}
		case "right", "l":
			if m.cursor < len(m.visible) {
				node := m.visible[m.cursor]
				if !node.IsLeaf() && !node.Expanded {
					node.Expanded = true
					m.refreshVisible()
				} else if m.cursor < len(m.visible)-1 {
					m.cursor++
					m.ensureVisible()
				}
			}
		case "left", "h":
			if m.cursor < len(m.visible) {
				node := m.visible[m.cursor]
				if !node.IsLeaf() && node.Expanded {
					node.Expanded = false
					m.refreshVisible()
				} else if node.Parent != nil {
					// Jump to parent
					for i, v := range m.visible {
						if v == node.Parent {
							m.cursor = i
							m.ensureVisible()
							break
						}
					}
				}
			}

		// Expand/collapse all
		case "e":
			m.root.ExpandAll()
			m.refreshVisible()
		case "c":
			// Collapse all except root
			for _, child := range m.root.Children {
				child.CollapseAll()
			}
			m.refreshVisible()
			if m.cursor >= len(m.visible) {
				m.cursor = len(m.visible) - 1
			}
			m.ensureVisible()

		// F-keys
		case "f1":
			m.activeOverlay = overlayHelp
		case "f3", "/":
			m.search.Open()
		case "ctrl+g":
			m.gotoActive = true
			m.gotoInput.SetValue("")
			m.gotoInput.Focus()

		// Copy/Paste
		case "y":
			if m.cursor < len(m.visible) {
				node := m.visible[m.cursor]
				var value string
				if node.IsLeaf() {
					value = node.EditableValue()
				} else {
					data, err := json.MarshalIndent(node.ToInterface(), "", "  ")
					if err != nil {
						m.statusMsg = fmt.Sprintf("Error: %v", err)
						break
					}
					value = string(data)
				}
				if err := clipboard.WriteAll(value); err != nil {
					m.statusMsg = fmt.Sprintf("Clipboard: %v", err)
				} else {
					display := value
					if len(display) > 30 {
						display = display[:30] + "…"
					}
					m.statusMsg = fmt.Sprintf("Copied \"%s\"", display)
				}
			}
		case "p":
			if m.cursor < len(m.visible) {
				node := m.visible[m.cursor]
				if !node.IsLeaf() {
					m.statusMsg = "Cannot paste into container"
					break
				}
				pasted, err := clipboard.ReadAll()
				if err != nil {
					m.statusMsg = fmt.Sprintf("Clipboard: %v", err)
				} else {
					m.applyNodeEdit(node, pasted)
					display := pasted
					if len(display) > 30 {
						display = display[:30] + "…"
					}
					m.statusMsg = fmt.Sprintf("Pasted \"%s\"", display)
				}
			}

		// Undo/Redo
		case "ctrl+z":
			if edit, ok := m.undo.Undo(); ok {
				edit.Node.Value = edit.OldValue
				edit.Node.Kind = edit.OldKind
				// Navigate cursor to the node
				for i, v := range m.visible {
					if v == edit.Node {
						m.cursor = i
						m.ensureVisible()
						break
					}
				}
				m.statusMsg = "Undone"
			}
		case "ctrl+y":
			if edit, ok := m.undo.Redo(); ok {
				edit.Node.Value = edit.NewValue
				edit.Node.Kind = edit.NewKind
				for i, v := range m.visible {
					if v == edit.Node {
						m.cursor = i
						m.ensureVisible()
						break
					}
				}
				m.statusMsg = "Redone"
			}

		// Search navigation
		case "n":
			if m.search.Query != "" {
				m.nextSearchMatch()
			}
		case "N":
			if m.search.Query != "" {
				m.prevSearchMatch()
			}

		case "esc":
			if m.search.Query != "" {
				m.search.Query = ""
				m.search.Matches = nil
			}
		}
	}

	return m, nil
}

func (m *JSONModel) startEditing(node *jsonpkg.Node) {
	m.editing = true
	m.editInput.SetValue(node.EditableValue())
	m.editInput.Focus()
	m.editInput.CursorEnd()
}

// applyNodeEdit sets a leaf node's value and pushes an undo entry if changed.
func (m *JSONModel) applyNodeEdit(node *jsonpkg.Node, newValue string) {
	oldValue := node.Value
	oldKind := node.Kind
	if err := node.SetValue(newValue); err != nil {
		m.statusMsg = fmt.Sprintf("Error: %v", err)
		return
	}
	if node.Value != oldValue || node.Kind != oldKind {
		m.undo.Push(NodeEdit{
			Node:     node,
			OldValue: oldValue,
			OldKind:  oldKind,
			NewValue: node.Value,
			NewKind:  node.Kind,
		})
	}
}

func (m JSONModel) updateEditing(msg tea.Msg) (JSONModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.editing = false
			m.editInput.Blur()
			return m, nil
		case "enter":
			if m.cursor < len(m.visible) {
				node := m.visible[m.cursor]
				m.applyNodeEdit(node, m.editInput.Value())
			}
			m.editing = false
			m.editInput.Blur()
			// Move to next node
			if m.cursor < len(m.visible)-1 {
				m.cursor++
				m.ensureVisible()
			}
			return m, nil
		case "tab":
			if m.cursor < len(m.visible) {
				node := m.visible[m.cursor]
				m.applyNodeEdit(node, m.editInput.Value())
			}
			m.editing = false
			m.editInput.Blur()
			// Move to next leaf
			for i := m.cursor + 1; i < len(m.visible); i++ {
				if m.visible[i].IsLeaf() {
					m.cursor = i
					m.ensureVisible()
					break
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return m, cmd
}

func (m JSONModel) updateSearch(msg tea.Msg) (JSONModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.search.Close()
			return m, nil
		case "enter":
			m.search.Compile()
			m.search.Close()

			if m.search.Err != "" {
				m.statusMsg = m.search.Err
				return m, nil
			}
			if m.search.Query == "" {
				m.search.Matches = nil
				return m, nil
			}

			// Search the tree
			matches := m.root.Search(m.search.Query, m.search.Regex())
			m.search.Matches = nil
			m.search.CurrentMatch = 0

			for _, node := range matches {
				// Find the node's index in visible or make it visible
				node.EnsureVisible()
			}
			m.refreshVisible()

			// Build match list from visible nodes
			for i, v := range m.visible {
				for _, match := range matches {
					if v == match {
						m.search.Matches = append(m.search.Matches, SearchMatch{Row: i, Col: 0})
						break
					}
				}
			}

			if len(m.search.Matches) > 0 {
				m.cursor = m.search.Matches[0].Row
				m.ensureVisible()
				m.statusMsg = fmt.Sprintf("%d matches", len(m.search.Matches))
			} else {
				m.statusMsg = "No matches"
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.search.Input, cmd = m.search.Input.Update(msg)
	return m, cmd
}

func (m *JSONModel) nextSearchMatch() {
	if len(m.search.Matches) == 0 {
		return
	}
	m.search.CurrentMatch = (m.search.CurrentMatch + 1) % len(m.search.Matches)
	m.cursor = m.search.Matches[m.search.CurrentMatch].Row
	m.ensureVisible()
}

func (m *JSONModel) prevSearchMatch() {
	if len(m.search.Matches) == 0 {
		return
	}
	m.search.CurrentMatch--
	if m.search.CurrentMatch < 0 {
		m.search.CurrentMatch = len(m.search.Matches) - 1
	}
	m.cursor = m.search.Matches[m.search.CurrentMatch].Row
	m.ensureVisible()
}

func (m JSONModel) updateGoto(msg tea.Msg) (JSONModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.gotoActive = false
			m.gotoInput.Blur()
			return m, nil
		case "enter":
			m.gotoActive = false
			m.gotoInput.Blur()
			lineStr := strings.TrimSpace(m.gotoInput.Value())
			if lineStr == "" {
				return m, nil
			}
			line, err := strconv.Atoi(lineStr)
			if err != nil {
				m.statusMsg = "Invalid node number"
				return m, nil
			}
			target := line - 1
			if target < 0 {
				target = 0
			}
			if target >= len(m.visible) {
				target = len(m.visible) - 1
			}
			m.cursor = target
			m.ensureVisible()
			m.statusMsg = fmt.Sprintf("Jumped to node %d", target+1)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.gotoInput, cmd = m.gotoInput.Update(msg)
	return m, cmd
}

func (m *JSONModel) ensureVisible() {
	if m.cursor < m.scrollOff {
		m.scrollOff = m.cursor
	}
	if m.cursor >= m.scrollOff+m.viewRows() {
		m.scrollOff = m.cursor - m.viewRows() + 1
	}
}

func (m *JSONModel) isSearchMatch(visibleIdx int) bool {
	for _, match := range m.search.Matches {
		if match.Row == visibleIdx {
			return true
		}
	}
	return false
}

func (m *JSONModel) isCurrentSearchMatch(visibleIdx int) bool {
	if len(m.search.Matches) == 0 {
		return false
	}
	return m.search.Matches[m.search.CurrentMatch].Row == visibleIdx
}

// View implements tea.Model.
func (m JSONModel) View() string {
	if m.activeOverlay == overlayHelp {
		return RenderJSONHelp(m.width, m.height)
	}

	var b strings.Builder

	// Title
	title := style.TitleStyle.Render(m.filePath)
	totalNodes := m.countNodes(m.root)
	pos := style.DimStyle.Render(fmt.Sprintf("  %d/%d nodes  %d visible",
		m.cursor+1, totalNodes, len(m.visible)))
	b.WriteString(title + pos + "\n")

	// Tree rows
	endIdx := m.scrollOff + m.viewRows()
	if endIdx > len(m.visible) {
		endIdx = len(m.visible)
	}

	for i := m.scrollOff; i < endIdx; i++ {
		node := m.visible[i]
		line := m.renderNode(node, i)
		b.WriteString(line + "\n")
	}

	// Pad
	rendered := endIdx - m.scrollOff
	for i := rendered; i < m.viewRows(); i++ {
		b.WriteString(style.DimStyle.Render("~") + "\n")
	}

	// Status bar
	items := m.buildStatusItems()
	b.WriteString(style.RenderStatusBar(m.width, items...))
	b.WriteString("\n")

	// F-key bar
	fkeys := []style.FKeyItem{
		{Key: "F1", Desc: "Help"},
		{Key: "F2", Desc: "Save"},
		{Key: "F3", Desc: "Search"},
		{Key: "e", Desc: "ExpandAll"},
		{Key: "c", Desc: "CollapseAll"},
		{Key: "F10", Desc: "Quit"},
	}
	b.WriteString(style.RenderFKeyBar(m.width, fkeys))

	return b.String()
}

func (m JSONModel) buildStatusItems() []string {
	pct := 0
	if len(m.visible) > 0 {
		pct = (m.cursor + 1) * 100 / len(m.visible)
	}

	nodeLabel := fmt.Sprintf("%d nodes", m.countNodes(m.root))
	if m.loading {
		nodeLabel = fmt.Sprintf("Loading... %d items", len(m.root.Children))
	}
	items := []string{
		nodeLabel,
		fmt.Sprintf("%d%%", pct),
	}
	if m.undo.IsModified() {
		items = append(items, style.ModifiedStyle.Render("MODIFIED"))
	}
	if m.editing {
		items = append(items, "EDIT: "+m.editInput.View())
	}
	if m.search.Active {
		items = append(items, "SEARCH: "+m.search.Input.View())
	}
	if m.gotoActive {
		items = append(items, "GO TO: "+m.gotoInput.View())
	}
	if m.search.Query != "" && !m.search.Active {
		if m.search.IsRegex {
			items = append(items, fmt.Sprintf("regex: %s (%d matches, n/N)", m.search.Query[1:], len(m.search.Matches)))
		} else {
			items = append(items, fmt.Sprintf("/%s (%d matches, n/N)", m.search.Query, len(m.search.Matches)))
		}
	}
	if m.statusMsg != "" {
		items = append(items, m.statusMsg)
	}
	return items
}

func (m JSONModel) countNodes(n *jsonpkg.Node) int {
	count := 1
	for _, c := range n.Children {
		count += m.countNodes(c)
	}
	return count
}

func (m JSONModel) renderNode(node *jsonpkg.Node, visibleIdx int) string {
	isCursor := visibleIdx == m.cursor
	isCurrentMatch := m.isCurrentSearchMatch(visibleIdx)
	isMatch := m.isSearchMatch(visibleIdx)

	// Indentation
	indent := strings.Repeat("  ", node.Depth)

	var line strings.Builder

	// Expand/collapse indicator
	if !node.IsLeaf() {
		if node.Expanded {
			line.WriteString(jsonBracketStyle.Render("▼ "))
		} else {
			line.WriteString(jsonBracketStyle.Render("▶ "))
		}
	} else {
		line.WriteString("  ")
	}

	// Key
	displayKey := node.DisplayKey()
	if displayKey != "" {
		line.WriteString(jsonKeyStyle.Render(displayKey))
		line.WriteString(style.DimStyle.Render(": "))
	}

	// Value or summary
	if node.IsLeaf() {
		line.WriteString(m.renderValue(node))
	} else if !node.Expanded {
		line.WriteString(jsonBracketStyle.Render(node.Summary()))
	} else {
		// Opening bracket
		switch node.Kind {
		case jsonpkg.KindObject:
			line.WriteString(jsonBracketStyle.Render("{"))
		case jsonpkg.KindArray:
			line.WriteString(jsonBracketStyle.Render("["))
		}
	}

	content := indent + line.String()

	// Apply highlight style
	switch {
	case isCursor:
		// Pad to full width for cursor highlight
		contentWidth := lipgloss.Width(content)
		if contentWidth < m.width {
			content += strings.Repeat(" ", m.width-contentWidth)
		}
		content = jsonCursorStyle.Render(content)
	case isCurrentMatch:
		contentWidth := lipgloss.Width(content)
		if contentWidth < m.width {
			content += strings.Repeat(" ", m.width-contentWidth)
		}
		content = jsonCurrentMatchStyle.Render(content)
	case isMatch:
		contentWidth := lipgloss.Width(content)
		if contentWidth < m.width {
			content += strings.Repeat(" ", m.width-contentWidth)
		}
		content = jsonMatchStyle.Render(content)
	}

	return content
}

func (m JSONModel) renderValue(node *jsonpkg.Node) string {
	switch node.Kind {
	case jsonpkg.KindString:
		return jsonStringStyle.Render(node.Value)
	case jsonpkg.KindNumber:
		return jsonNumberStyle.Render(node.Value)
	case jsonpkg.KindBool:
		return jsonBoolStyle.Render(node.Value)
	case jsonpkg.KindNull:
		return jsonNullStyle.Render(node.Value)
	default:
		return node.Value
	}
}

// RenderJSONHelp renders the help overlay for JSON mode.
func RenderJSONHelp(width, height int) string {
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

	helpItems := []struct{ key, desc string }{
		{"↑/↓, j/k", "Navigate nodes"},
		{"←/h", "Collapse or go to parent"},
		{"→/l", "Expand or go to child"},
		{"Enter/Space", "Toggle or edit leaf value"},
		{"Tab", "Confirm edit, jump to next leaf"},
		{"Esc", "Cancel edit / close overlay"},
		{"PgUp/PgDn", "Scroll page"},
		{"Home/End", "Jump to first/last node"},
		{"", ""},
		{"e", "Expand all nodes"},
		{"c", "Collapse all nodes"},
		{"F1", "Toggle this help"},
		{"F2, Ctrl+S", "Save file"},
		{"F3, /", "Search (prefix / for regex)"},
		{"Ctrl+G", "Go to node number"},
		{"y", "Copy to clipboard"},
		{"p", "Paste from clipboard"},
		{"Ctrl+Z", "Undo"},
		{"Ctrl+Y", "Redo"},
		{"n/N", "Next/previous match"},
		{"F10, q", "Quit"},
	}

	var lines []string
	lines = append(lines, titleStyle.Render("dpeek — JSON Shortcuts"))
	lines = append(lines, "")
	for _, h := range helpItems {
		if h.key == "" {
			lines = append(lines, "")
			continue
		}
		lines = append(lines, keyStyle.Render(h.key)+descStyle.Render(h.desc))
	}
	lines = append(lines, "")
	lines = append(lines, style.DimStyle.Render("Press F1 or Esc to close"))

	content := boxStyle.Render(strings.Join(lines, "\n"))

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

	var centeredLines []string
	for _, line := range strings.Split(content, "\n") {
		centeredLines = append(centeredLines, padLeft+line)
	}

	return padTop + strings.Join(centeredLines, "\n")
}
