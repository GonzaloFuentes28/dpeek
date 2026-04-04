package model

import (
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	csvpkg "github.com/GonzaloFuentes28/dpeek/internal/csv"
)

func loadTestCSV(t *testing.T) *csvpkg.DataSet {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "small.csv")
	ds, err := csvpkg.Load(path, ',', true)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	return ds
}

func sizedModel(t *testing.T) CSVModel {
	t.Helper()
	m := NewCSVModel(loadTestCSV(t))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	return m
}

func TestCSVModelNavigation(t *testing.T) {
	m := sizedModel(t)

	if m.cursorRow != 0 || m.cursorCol != 0 {
		t.Errorf("initial cursor: expected (0,0), got (%d,%d)", m.cursorRow, m.cursorCol)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyDown})
	if m.cursorRow != 1 {
		t.Errorf("after down: expected row=1, got %d", m.cursorRow)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})
	if m.cursorCol != 1 {
		t.Errorf("after right: expected col=1, got %d", m.cursorCol)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursorRow != 0 {
		t.Errorf("after up: expected row=0, got %d", m.cursorRow)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.cursorCol != 0 {
		t.Errorf("after left: expected col=0, got %d", m.cursorCol)
	}

	// Bounds checking
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if m.cursorRow != 0 {
		t.Errorf("should stay at row=0, got %d", m.cursorRow)
	}
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if m.cursorCol != 0 {
		t.Errorf("should stay at col=0, got %d", m.cursorCol)
	}
}

func TestCSVModelPageNavigation(t *testing.T) {
	ds := loadTestCSV(t)
	m := NewCSVModel(ds)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 15})

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	if m.cursorRow != 10 {
		t.Errorf("after pgdown: expected row=10, got %d", m.cursorRow)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgDown})
	lastRow := m.rowCount() - 1
	if m.cursorRow != lastRow {
		t.Errorf("after pgdown again: expected row=%d, got %d", lastRow, m.cursorRow)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyPgUp})
	if m.cursorRow != lastRow-10 {
		t.Errorf("after pgup: expected row=%d, got %d", lastRow-10, m.cursorRow)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyHome})
	if m.cursorRow != 0 || m.cursorCol != 0 {
		t.Errorf("after home: expected (0,0), got (%d,%d)", m.cursorRow, m.cursorCol)
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnd})
	if m.cursorRow != lastRow || m.cursorCol != 4 {
		t.Errorf("after end: expected (%d,4), got (%d,%d)", lastRow, m.cursorRow, m.cursorCol)
	}
}

func TestCSVModelView(t *testing.T) {
	m := sizedModel(t)
	view := m.View()

	for _, want := range []string{"name", "age", "20 rows", "5 cols", "F1", "F10"} {
		if !strings.Contains(view, want) {
			t.Errorf("view should contain %q", want)
		}
	}
}

func TestCSVModelEditing(t *testing.T) {
	m := sizedModel(t)

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.editing {
		t.Error("should be in editing mode after Enter")
	}
	if !m.HasActiveInput() {
		t.Error("HasActiveInput should be true while editing")
	}

	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.editing {
		t.Error("should exit editing mode after Esc")
	}
}

func TestCSVModelHelp(t *testing.T) {
	m := sizedModel(t)

	// Open help
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyF1})
	if m.activeOverlay != overlayHelp {
		t.Error("F1 should open help overlay")
	}
	if !m.HasActiveInput() {
		t.Error("HasActiveInput should be true with overlay")
	}

	view := m.View()
	if !strings.Contains(view, "Keyboard Shortcuts") {
		t.Error("help view should contain 'Keyboard Shortcuts'")
	}

	// Close help
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.activeOverlay != overlayNone {
		t.Error("Esc should close help overlay")
	}
}

func TestCSVModelSearch(t *testing.T) {
	m := sizedModel(t)

	// Open search
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyF3})
	if !m.search.Active {
		t.Error("F3 should activate search")
	}

	// Type a query
	for _, r := range "Madrid" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Execute search
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if m.search.Active {
		t.Error("search should close after Enter")
	}
	if m.search.MatchCount() == 0 {
		t.Error("should find matches for 'Madrid'")
	}
}

func TestCSVModelFilter(t *testing.T) {
	m := sizedModel(t)
	totalRows := m.rowCount()

	// Open filter
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyF4})
	if !m.filter.Active {
		t.Error("F4 should activate filter")
	}

	// Type filter
	for _, r := range "true" {
		m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Apply filter
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if !m.filter.IsFiltered {
		t.Error("filter should be active after Enter")
	}
	if m.rowCount() >= totalRows {
		t.Errorf("filtered rows (%d) should be less than total (%d)", m.rowCount(), totalRows)
	}

	// Clear filter with F4
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyF4})
	if m.filter.IsFiltered {
		t.Error("F4 should toggle filter off")
	}
	if m.rowCount() != totalRows {
		t.Errorf("after clearing filter: expected %d rows, got %d", totalRows, m.rowCount())
	}
}

func TestCSVModelSort(t *testing.T) {
	m := sizedModel(t)

	// Sort by first column (name)
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyF5})
	if !m.sort.Active {
		t.Error("F5 should activate sort")
	}
	if !m.sort.Ascending {
		t.Error("first sort should be ascending")
	}

	// Sort again to reverse
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyF5})
	if m.sort.Ascending {
		t.Error("second F5 should toggle to descending")
	}
}

func TestCSVModelStats(t *testing.T) {
	m := sizedModel(t)

	// Move to age column
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyRight})

	// Open stats
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyF6})
	if m.activeOverlay != overlayStats {
		t.Error("F6 should open stats overlay")
	}

	view := m.View()
	if !strings.Contains(view, "Column:") {
		t.Error("stats view should contain 'Column:'")
	}
	if !strings.Contains(view, "integer") {
		t.Error("age column should be detected as integer")
	}

	// Close
	m, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	if m.activeOverlay != overlayNone {
		t.Error("Esc should close stats")
	}
}
