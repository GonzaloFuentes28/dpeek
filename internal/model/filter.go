package model

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	csvpkg "github.com/GonzaloFuentes28/dpeek/internal/csv"
)

// FilterState manages row filtering for CSV mode.
type FilterState struct {
	Active      bool
	Input       textinput.Model
	Query       string
	FilteredIdx []int  // indices of rows that match the filter
	IsFiltered  bool   // true when a filter is actively applied
	Column      int    // specific column to filter (-1 = any)
	ColumnName  string // display name of the filtered column
}

// NewFilterState creates a new filter state.
func NewFilterState() FilterState {
	ti := textinput.New()
	ti.Placeholder = "Filter rows (col:value or any text)..."
	ti.CharLimit = 128
	return FilterState{Input: ti, Column: -1}
}

// Open activates the filter input.
func (f *FilterState) Open() {
	f.Active = true
	f.Input.Focus()
	f.Input.SetValue(f.Query)
	f.Input.CursorEnd()
}

// Close deactivates the filter input without clearing the filter.
func (f *FilterState) Close() {
	f.Active = false
	f.Input.Blur()
}

// Apply runs the filter on the dataset and stores matching row indices.
func (f *FilterState) Apply(data *csvpkg.DataSet) {
	f.Query = f.Input.Value()
	f.FilteredIdx = nil
	f.Column = -1
	f.ColumnName = ""

	if f.Query == "" {
		f.IsFiltered = false
		return
	}

	// Check for column:value syntax
	filterValue := f.Query
	if idx := strings.IndexByte(f.Query, ':'); idx > 0 {
		prefix := f.Query[:idx]
		col := findColumn(data.Headers, prefix)
		if col >= 0 {
			f.Column = col
			f.ColumnName = data.Headers[col]
			filterValue = f.Query[idx+1:]
		}
	}

	query := strings.ToLower(filterValue)
	f.IsFiltered = true

	if f.Column >= 0 {
		for i, row := range data.Rows {
			if f.Column < len(row) && strings.Contains(strings.ToLower(row[f.Column]), query) {
				f.FilteredIdx = append(f.FilteredIdx, i)
			}
		}
	} else {
		for i, row := range data.Rows {
			for _, cell := range row {
				if strings.Contains(strings.ToLower(cell), query) {
					f.FilteredIdx = append(f.FilteredIdx, i)
					break
				}
			}
		}
	}
}

// findColumn returns the index of a header matching name (case-insensitive), or -1.
func findColumn(headers []string, name string) int {
	lower := strings.ToLower(name)
	for i, h := range headers {
		if strings.ToLower(h) == lower {
			return i
		}
	}
	return -1
}

// Clear removes the active filter.
func (f *FilterState) Clear() {
	f.Query = ""
	f.FilteredIdx = nil
	f.IsFiltered = false
	f.Column = -1
	f.ColumnName = ""
	f.Input.SetValue("")
}

// ApplyChunk incrementally filters newly added rows (for progressive loading).
func (f *FilterState) ApplyChunk(data *csvpkg.DataSet, startRow int, count int) {
	if !f.IsFiltered || f.Query == "" {
		return
	}

	// Parse filter value
	filterValue := f.Query
	if f.Column >= 0 {
		if idx := strings.IndexByte(f.Query, ':'); idx > 0 {
			filterValue = f.Query[idx+1:]
		}
	}
	query := strings.ToLower(filterValue)

	endRow := startRow + count
	if endRow > len(data.Rows) {
		endRow = len(data.Rows)
	}

	if f.Column >= 0 {
		for i := startRow; i < endRow; i++ {
			row := data.Rows[i]
			if f.Column < len(row) && strings.Contains(strings.ToLower(row[f.Column]), query) {
				f.FilteredIdx = append(f.FilteredIdx, i)
			}
		}
	} else {
		for i := startRow; i < endRow; i++ {
			for _, cell := range data.Rows[i] {
				if strings.Contains(strings.ToLower(cell), query) {
					f.FilteredIdx = append(f.FilteredIdx, i)
					break
				}
			}
		}
	}
}

// Reapply re-runs the filter on the full dataset (e.g., after sort changes row order).
func (f *FilterState) Reapply(data *csvpkg.DataSet) {
	if !f.IsFiltered || f.Query == "" {
		return
	}
	f.FilteredIdx = nil
	filterValue := f.Query
	if f.Column >= 0 {
		if idx := strings.IndexByte(f.Query, ':'); idx > 0 {
			filterValue = f.Query[idx+1:]
		}
	}
	query := strings.ToLower(filterValue)

	if f.Column >= 0 {
		for i, row := range data.Rows {
			if f.Column < len(row) && strings.Contains(strings.ToLower(row[f.Column]), query) {
				f.FilteredIdx = append(f.FilteredIdx, i)
			}
		}
	} else {
		for i, row := range data.Rows {
			for _, cell := range row {
				if strings.Contains(strings.ToLower(cell), query) {
					f.FilteredIdx = append(f.FilteredIdx, i)
					break
				}
			}
		}
	}
}

// RowCount returns the number of visible rows (filtered or total).
func (f *FilterState) RowCount(totalRows int) int {
	if !f.IsFiltered {
		return totalRows
	}
	return len(f.FilteredIdx)
}

// MapRow converts a visible row index to the actual data row index.
func (f *FilterState) MapRow(visibleRow int) int {
	if !f.IsFiltered {
		return visibleRow
	}
	if visibleRow < 0 || visibleRow >= len(f.FilteredIdx) {
		return 0
	}
	return f.FilteredIdx[visibleRow]
}
