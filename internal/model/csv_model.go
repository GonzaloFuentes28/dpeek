package model

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	csvpkg "github.com/GonzaloFuentes28/dpeek/internal/csv"
	"github.com/GonzaloFuentes28/dpeek/internal/style"
)

const (
	minColWidth     = 4
	rowNumWidth     = 6
	colPadding      = 1
	headerSeparator = '─'
)

// overlay represents what's currently displayed on top of the table.
type overlay int

const (
	overlayNone overlay = iota
	overlayHelp
	overlayStats
)

// CSVModel is the bubbletea model for the CSV/TSV table view.
type CSVModel struct {
	data *csvpkg.DataSet

	// Cursor position (in visible/filtered space)
	cursorRow int
	cursorCol int

	// Viewport scroll offset
	scrollRow int
	scrollCol int

	// Terminal dimensions
	width  int
	height int

	// Computed layout
	colWidths []int
	viewRows  int // rows visible in the viewport

	// Editing
	editing   bool
	editInput textinput.Model

	// Go-to-line
	gotoActive bool
	gotoInput  textinput.Model

	// Overlays
	activeOverlay overlay

	// Features
	search SearchState
	filter FilterState
	sort   SortState
	undo   UndoStack[CellEdit]

	// Status message (temporary)
	statusMsg string
}

// NewCSVModel creates a new CSV table model.
func NewCSVModel(data *csvpkg.DataSet) CSVModel {
	ti := textinput.New()
	ti.CharLimit = 256

	gi := textinput.New()
	gi.Placeholder = "Line number..."
	gi.CharLimit = 10

	m := CSVModel{
		data:      data,
		editInput: ti,
		gotoInput: gi,
		search:    NewSearchState(),
		filter:    NewFilterState(),
		undo:      NewUndoStack[CellEdit](1000),
	}
	m.computeColWidths()
	return m
}

func (m *CSVModel) computeColWidths() {
	if m.data.ColCount() == 0 {
		return
	}

	m.colWidths = make([]int, m.data.ColCount())

	for col := range m.data.ColCount() {
		w := len(m.data.Headers[col])

		for row := range m.data.RowCount() {
			if col < len(m.data.Rows[row]) {
				if cellLen := len(m.data.Rows[row][col]); cellLen > w {
					w = cellLen
				}
			}
		}

		w += colPadding * 2
		if w < minColWidth {
			w = minColWidth
		}
		m.colWidths[col] = w
	}
}

// visibleColRange returns the range of columns that fit in the viewport.
func (m *CSVModel) visibleColRange() (start, end int) {
	available := m.width - rowNumWidth - 1
	start = m.scrollCol
	used := 0
	for i := start; i < m.data.ColCount(); i++ {
		needed := m.colWidths[i] + 1
		if used+needed > available && i > start {
			return start, i
		}
		used += needed
	}
	return start, m.data.ColCount()
}

// rowCount returns the number of visible rows (respecting filter).
func (m *CSVModel) rowCount() int {
	return m.filter.RowCount(m.data.RowCount())
}

// dataRow maps a visible row index to the actual data row index.
func (m *CSVModel) dataRow(visibleRow int) int {
	return m.filter.MapRow(visibleRow)
}

// Init implements tea.Model.
func (m CSVModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m CSVModel) Update(msg tea.Msg) (CSVModel, tea.Cmd) {
	// Route to active input handler
	if m.editing {
		return m.updateEditing(msg)
	}
	if m.search.Active {
		return m.updateSearch(msg)
	}
	if m.filter.Active {
		return m.updateFilter(msg)
	}
	if m.gotoActive {
		return m.updateGoto(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		// 4 lines reserved: title, header, separator, status bar, fkey bar
		m.viewRows = m.height - 5
		if m.viewRows < 1 {
			m.viewRows = 1
		}

	case tea.MouseMsg:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			if m.cursorRow > 0 {
				m.cursorRow -= 3
				if m.cursorRow < 0 {
					m.cursorRow = 0
				}
				m.ensureRowVisible()
			}
		case tea.MouseButtonWheelDown:
			if m.cursorRow < m.rowCount()-1 {
				m.cursorRow += 3
				if m.cursorRow >= m.rowCount() {
					m.cursorRow = m.rowCount() - 1
				}
				m.ensureRowVisible()
			}
		}

	case tea.KeyMsg:
		// Close overlays first
		if m.activeOverlay != overlayNone {
			switch msg.String() {
			case "esc", "f1", "f6":
				m.activeOverlay = overlayNone
			}
			return m, nil
		}

		m.statusMsg = ""
		switch msg.String() {
		// Navigation
		case "up", "k":
			if m.cursorRow > 0 {
				m.cursorRow--
				m.ensureRowVisible()
			}
		case "down", "j":
			if m.cursorRow < m.rowCount()-1 {
				m.cursorRow++
				m.ensureRowVisible()
			}
		case "left", "h":
			if m.cursorCol > 0 {
				m.cursorCol--
				m.ensureColVisible()
			}
		case "right", "l":
			if m.cursorCol < m.data.ColCount()-1 {
				m.cursorCol++
				m.ensureColVisible()
			}
		case "pgup":
			m.cursorRow -= m.viewRows
			if m.cursorRow < 0 {
				m.cursorRow = 0
			}
			m.ensureRowVisible()
		case "pgdown":
			m.cursorRow += m.viewRows
			if m.cursorRow >= m.rowCount() {
				m.cursorRow = m.rowCount() - 1
			}
			if m.cursorRow < 0 {
				m.cursorRow = 0
			}
			m.ensureRowVisible()
		case "home":
			m.cursorRow = 0
			m.cursorCol = 0
			m.scrollRow = 0
			m.scrollCol = 0
		case "end":
			m.cursorRow = m.rowCount() - 1
			if m.cursorRow < 0 {
				m.cursorRow = 0
			}
			m.cursorCol = m.data.ColCount() - 1
			if m.cursorCol < 0 {
				m.cursorCol = 0
			}
			m.ensureRowVisible()
			m.ensureColVisible()

		// Editing
		case "enter":
			if m.rowCount() > 0 && m.data.ColCount() > 0 {
				m.startEditing()
			}

		// F-keys
		case "f1":
			m.activeOverlay = overlayHelp

		case "f2", "ctrl+s":
			if m.undo.IsModified() {
				if err := m.data.Save(m.data.FilePath); err != nil {
					m.statusMsg = fmt.Sprintf("Error: %v", err)
				} else {
					m.undo.MarkSaved()
					m.statusMsg = "Saved!"
				}
			} else {
				m.statusMsg = "No changes to save"
			}

		case "f3", "/":
			m.search.Open()

		case "ctrl+g":
			m.gotoActive = true
			m.gotoInput.SetValue("")
			m.gotoInput.Focus()

		case "f4":
			if m.filter.IsFiltered {
				// Toggle: if already filtered, clear it
				m.filter.Clear()
				m.cursorRow = 0
				m.scrollRow = 0
			} else {
				m.filter.Open()
			}

		case "f5":
			dataCol := m.cursorCol
			m.sort.Toggle(dataCol, m.data)
			if m.filter.IsFiltered {
				m.filter.Apply(m.data) // re-apply filter after sort
			}
			if m.sort.Active {
				m.statusMsg = fmt.Sprintf("Sorted by %s %s", m.data.Headers[dataCol], m.sort.Direction())
			} else {
				m.statusMsg = "Sort reset — original order"
			}

		case "f6":
			m.activeOverlay = overlayStats

		// Copy/Paste
		case "y":
			if m.rowCount() > 0 {
				dataRow := m.dataRow(m.cursorRow)
				value := m.data.Rows[dataRow][m.cursorCol]
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
			if m.rowCount() > 0 {
				pasted, err := clipboard.ReadAll()
				if err != nil {
					m.statusMsg = fmt.Sprintf("Clipboard: %v", err)
				} else {
					dataRow := m.dataRow(m.cursorRow)
					m.applyEdit(dataRow, m.cursorCol, pasted)
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
				row := m.resolveOrigLine(edit.OrigLine)
				if row >= 0 {
					m.data.Rows[row][edit.Col] = edit.OldValue
					m.jumpToCell(row, edit.Col)
					m.statusMsg = "Undone"
				}
			}
		case "ctrl+y":
			if edit, ok := m.undo.Redo(); ok {
				row := m.resolveOrigLine(edit.OrigLine)
				if row >= 0 {
					m.data.Rows[row][edit.Col] = edit.NewValue
					m.jumpToCell(row, edit.Col)
					m.statusMsg = "Redone"
				}
			}

		// Search navigation
		case "n":
			if m.search.Query != "" {
				if row, col, ok := m.search.NextMatch(); ok {
					m.jumpToCell(row, col)
				}
			}
		case "N":
			if m.search.Query != "" {
				if row, col, ok := m.search.PrevMatch(); ok {
					m.jumpToCell(row, col)
				}
			}

		case "esc":
			if m.search.Query != "" {
				m.search.Query = ""
				m.search.Matches = nil
			} else if m.filter.IsFiltered {
				m.filter.Clear()
				m.cursorRow = 0
				m.scrollRow = 0
			}
		}
	}

	return m, nil
}

func (m *CSVModel) jumpToCell(row, col int) {
	if row >= 0 {
		m.cursorRow = row
		// If filtered, find the visible index for this data row
		if m.filter.IsFiltered {
			for i, idx := range m.filter.FilteredIdx {
				if idx == row {
					m.cursorRow = i
					break
				}
			}
		}
	}
	if col >= 0 {
		m.cursorCol = col
	}
	m.ensureRowVisible()
	m.ensureColVisible()
}

func (m *CSVModel) startEditing() {
	m.editing = true
	dataRow := m.dataRow(m.cursorRow)
	currentValue := ""
	if dataRow < len(m.data.Rows) && m.cursorCol < len(m.data.Rows[dataRow]) {
		currentValue = m.data.Rows[dataRow][m.cursorCol]
	}
	m.editInput.SetValue(currentValue)
	m.editInput.Focus()
	m.editInput.CursorEnd()
}

// applyEdit sets a cell value and pushes an undo entry if the value changed.
func (m *CSVModel) applyEdit(dataRow, col int, newValue string) {
	if dataRow < 0 || dataRow >= len(m.data.Rows) || col < 0 || col >= len(m.data.Headers) {
		return
	}
	oldValue := m.data.Rows[dataRow][col]
	if oldValue == newValue {
		return
	}
	m.undo.Push(CellEdit{
		OrigLine: m.data.OrigIndex[dataRow],
		Col:      col,
		OldValue: oldValue,
		NewValue: newValue,
	})
	m.data.Rows[dataRow][col] = newValue
}

// resolveOrigLine finds the current data row index for an original line number.
func (m *CSVModel) resolveOrigLine(origLine int) int {
	for i, ol := range m.data.OrigIndex {
		if ol == origLine {
			return i
		}
	}
	return -1
}

func (m CSVModel) updateEditing(msg tea.Msg) (CSVModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.editing = false
			m.editInput.Blur()
			return m, nil
		case "enter":
			dataRow := m.dataRow(m.cursorRow)
			m.applyEdit(dataRow, m.cursorCol, m.editInput.Value())
			m.editing = false
			m.editInput.Blur()
			if m.cursorRow < m.rowCount()-1 {
				m.cursorRow++
				m.ensureRowVisible()
			}
			return m, nil
		case "tab":
			dataRow := m.dataRow(m.cursorRow)
			m.applyEdit(dataRow, m.cursorCol, m.editInput.Value())
			m.editing = false
			m.editInput.Blur()
			if m.cursorCol < m.data.ColCount()-1 {
				m.cursorCol++
				m.ensureColVisible()
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return m, cmd
}

func (m CSVModel) updateSearch(msg tea.Msg) (CSVModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.search.Close()
			return m, nil
		case "enter":
			m.search.Execute(m.data)
			m.search.Close()
			if m.search.Err != "" {
				m.statusMsg = m.search.Err
			} else if row, col, ok := m.search.NextMatch(); ok {
				m.jumpToCell(row, col)
				m.statusMsg = fmt.Sprintf("%d matches", m.search.MatchCount())
			} else if m.search.Query != "" {
				m.statusMsg = "No matches"
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.search.Input, cmd = m.search.Input.Update(msg)
	return m, cmd
}

func (m CSVModel) updateFilter(msg tea.Msg) (CSVModel, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.filter.Close()
			return m, nil
		case "enter":
			m.filter.Apply(m.data)
			m.filter.Close()
			m.cursorRow = 0
			m.scrollRow = 0
			if m.filter.IsFiltered {
				m.statusMsg = fmt.Sprintf("Showing %d/%d rows", len(m.filter.FilteredIdx), m.data.RowCount())
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.filter.Input, cmd = m.filter.Input.Update(msg)
	return m, cmd
}

func (m CSVModel) updateGoto(msg tea.Msg) (CSVModel, tea.Cmd) {
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
				m.statusMsg = "Invalid line number"
				return m, nil
			}
			// Convert 1-based to 0-based
			target := line - 1
			if target < 0 {
				target = 0
			}
			if target >= m.rowCount() {
				target = m.rowCount() - 1
			}
			m.cursorRow = target
			m.cursorCol = 0
			m.ensureRowVisible()
			m.statusMsg = fmt.Sprintf("Jumped to line %d", target+1)
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.gotoInput, cmd = m.gotoInput.Update(msg)
	return m, cmd
}

func (m *CSVModel) ensureRowVisible() {
	if m.cursorRow < m.scrollRow {
		m.scrollRow = m.cursorRow
	}
	if m.cursorRow >= m.scrollRow+m.viewRows {
		m.scrollRow = m.cursorRow - m.viewRows + 1
	}
}

func (m *CSVModel) ensureColVisible() {
	if m.cursorCol < m.scrollCol {
		m.scrollCol = m.cursorCol
	}
	_, end := m.visibleColRange()
	for m.cursorCol >= end && m.scrollCol < m.data.ColCount()-1 {
		m.scrollCol++
		_, end = m.visibleColRange()
	}
}

// View implements tea.Model.
func (m CSVModel) View() string {
	if m.data.ColCount() == 0 {
		return "Empty file"
	}

	// Render overlays on top
	if m.activeOverlay == overlayHelp {
		return RenderHelp(m.width, m.height)
	}
	if m.activeOverlay == overlayStats {
		stats := ComputeStats(m.data, m.cursorCol)
		return RenderStats(stats, m.width, m.height)
	}

	var b strings.Builder

	// Title line
	title := style.TitleStyle.Render(m.data.FilePath)
	pos := style.DimStyle.Render(fmt.Sprintf("  Row %d/%d  Col %d/%d",
		m.cursorRow+1, m.rowCount(),
		m.cursorCol+1, m.data.ColCount()))
	b.WriteString(title + pos + "\n")

	colStart, colEnd := m.visibleColRange()

	// Header
	b.WriteString(m.renderRow(-1, colStart, colEnd))
	b.WriteString("\n")

	// Separator
	b.WriteString(m.renderSeparator(colStart, colEnd))
	b.WriteString("\n")

	// Data rows
	totalVisible := m.rowCount()
	endRow := m.scrollRow + m.viewRows
	if endRow > totalVisible {
		endRow = totalVisible
	}
	for visibleRow := m.scrollRow; visibleRow < endRow; visibleRow++ {
		dataRow := m.dataRow(visibleRow)
		b.WriteString(m.renderDataRow(dataRow, visibleRow, colStart, colEnd))
		b.WriteString("\n")
	}

	// Pad remaining lines
	rendered := endRow - m.scrollRow
	for i := rendered; i < m.viewRows; i++ {
		b.WriteString(style.DimStyle.Render("~") + "\n")
	}

	// Status bar
	items := m.buildStatusItems()
	b.WriteString(style.RenderStatusBar(m.width, items...))
	b.WriteString("\n")

	// F-key bar
	b.WriteString(style.RenderFKeyBar(m.width, m.fkeys()))

	return b.String()
}

func (m CSVModel) buildStatusItems() []string {
	// Scroll percentage
	pct := 0
	if m.rowCount() > 0 {
		pct = (m.cursorRow + 1) * 100 / m.rowCount()
	}

	items := []string{
		fmt.Sprintf("%d rows", m.data.RowCount()),
		fmt.Sprintf("%d cols", m.data.ColCount()),
		fmt.Sprintf("%d%%", pct),
	}

	if m.filter.IsFiltered {
		if m.filter.ColumnName != "" {
			items = append(items, fmt.Sprintf("FILTER [%s]: %d/%d", m.filter.ColumnName, len(m.filter.FilteredIdx), m.data.RowCount()))
		} else {
			items = append(items, fmt.Sprintf("FILTER: %d/%d", len(m.filter.FilteredIdx), m.data.RowCount()))
		}
	}
	if m.sort.Active {
		items = append(items, fmt.Sprintf("SORT: %s %s", m.data.Headers[m.sort.Column], m.sort.Direction()))
	}
	if m.undo.IsModified() {
		items = append(items, style.ModifiedStyle.Render("MODIFIED"))
	}
	if m.statusMsg != "" {
		items = append(items, m.statusMsg)
	}
	if m.editing {
		items = append(items, "EDIT: "+m.editInput.View())
	}
	if m.search.Active {
		items = append(items, "SEARCH: "+m.search.Input.View())
	}
	if m.filter.Active {
		items = append(items, "FILTER: "+m.filter.Input.View())
	}
	if m.gotoActive {
		items = append(items, "GO TO: "+m.gotoInput.View())
	}
	if m.search.Query != "" && !m.search.Active {
		if m.search.IsRegex {
			items = append(items, fmt.Sprintf("regex: %s (%d matches, n/N)", m.search.Query[1:], m.search.MatchCount()))
		} else {
			items = append(items, fmt.Sprintf("/%s (%d matches, n/N)", m.search.Query, m.search.MatchCount()))
		}
	}

	return items
}

func (m CSVModel) fkeys() []style.FKeyItem {
	fkeys := []style.FKeyItem{
		{Key: "F1", Desc: "Help"},
		{Key: "F2", Desc: "Save"},
		{Key: "F3", Desc: "Search"},
	}
	if m.filter.IsFiltered {
		fkeys = append(fkeys, style.FKeyItem{Key: "F4", Desc: "Unfilter"})
	} else {
		fkeys = append(fkeys, style.FKeyItem{Key: "F4", Desc: "Filter"})
	}
	fkeys = append(fkeys,
		style.FKeyItem{Key: "F5", Desc: "Sort"},
		style.FKeyItem{Key: "F6", Desc: "Stats"},
		style.FKeyItem{Key: "F10", Desc: "Quit"},
	)
	return fkeys
}

// renderRow renders the header row (row=-1).
func (m CSVModel) renderRow(row int, colStart, colEnd int) string {
	var parts []string
	parts = append(parts, style.RowNumberStyle.Render(padLeft("", rowNumWidth)))

	for col := colStart; col < colEnd; col++ {
		header := m.data.Headers[col]
		// Add sort indicator
		if m.sort.Active && m.sort.Column == col {
			header = header + " " + m.sort.Direction()
		}

		w := m.colWidths[col]
		cell := padRight(truncate(header, w-colPadding*2), w)
		cell = style.HeaderStyle.Render(cell)
		parts = append(parts, cell)
	}

	sep := style.DimStyle.Render("│")
	return strings.Join(parts, sep)
}

// renderDataRow renders a data row with proper styling (cursor, search highlight).
func (m CSVModel) renderDataRow(dataRow, visibleRow, colStart, colEnd int) string {
	var parts []string
	// Show original file line number
	lineNum := dataRow + 1
	if dataRow < len(m.data.OrigIndex) {
		lineNum = m.data.OrigIndex[dataRow]
	}
	parts = append(parts, style.RowNumberStyle.Render(padLeft(fmt.Sprintf("%d", lineNum), rowNumWidth)))

	for col := colStart; col < colEnd; col++ {
		var value string
		if col < len(m.data.Rows[dataRow]) {
			value = m.data.Rows[dataRow][col]
		}

		w := m.colWidths[col]
		cell := padRight(truncate(value, w-colPadding*2), w)

		// Style priority: cursor > current search match > search match > selected row > normal
		isCursor := visibleRow == m.cursorRow && col == m.cursorCol
		isSelectedRow := visibleRow == m.cursorRow
		isCurrentMatch := m.search.IsCurrentMatch(dataRow, col)
		isMatch := m.search.IsMatch(dataRow, col)

		switch {
		case isCursor:
			cell = style.CursorCellStyle.Render(cell)
		case isCurrentMatch:
			cell = lipgloss.NewStyle().
				Background(style.ColorHeader).
				Foreground(lipgloss.Color("#000000")).
				Bold(true).
				Render(cell)
		case isMatch:
			cell = lipgloss.NewStyle().
				Background(lipgloss.AdaptiveColor{Light: "#fff3cd", Dark: "#3a3a00"}).
				Foreground(style.ColorNormal).
				Render(cell)
		case isSelectedRow:
			cell = style.SelectedRowStyle.Render(cell)
		default:
			cell = style.CellStyle.Render(cell)
		}

		parts = append(parts, cell)
	}

	sep := style.DimStyle.Render("│")
	return strings.Join(parts, sep)
}

func (m CSVModel) renderSeparator(colStart, colEnd int) string {
	var parts []string
	parts = append(parts, strings.Repeat(string(headerSeparator), rowNumWidth))
	for col := colStart; col < colEnd; col++ {
		parts = append(parts, strings.Repeat(string(headerSeparator), m.colWidths[col]))
	}
	return style.DimStyle.Render(strings.Join(parts, "┼"))
}

// HasActiveInput returns true if an input field or overlay is active.
func (m CSVModel) HasActiveInput() bool {
	return m.editing || m.search.Active || m.filter.Active || m.gotoActive || m.activeOverlay != overlayNone
}

// Helper functions

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

func padRight(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func padLeft(s string, width int) string {
	runes := []rune(s)
	if len(runes) >= width {
		return s
	}
	return strings.Repeat(" ", width-len(runes)) + s
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
