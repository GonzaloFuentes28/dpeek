package model

import (
	"sort"
	"strconv"
	"strings"

	csvpkg "github.com/GonzaloFuentes28/dpeek/internal/csv"
)

// SortState manages column sorting for CSV mode.
type SortState struct {
	Column    int
	Ascending bool
	Active    bool // true when data has been sorted
}

// Toggle cycles sort on a column: ascending → descending → reset (original order).
func (s *SortState) Toggle(col int, data *csvpkg.DataSet) {
	if s.Active && s.Column == col {
		if s.Ascending {
			// asc → desc
			s.Ascending = false
			s.apply(data)
		} else {
			// desc → reset to original order
			s.Reset(data)
		}
	} else {
		s.Column = col
		s.Ascending = true
		s.Active = true
		s.apply(data)
	}
}

// Reset restores original file order by sorting on OrigIndex.
func (s *SortState) Reset(data *csvpkg.DataSet) {
	s.Active = false

	n := len(data.Rows)
	indices := make([]int, n)
	for i := range n {
		indices[i] = i
	}
	sort.SliceStable(indices, func(i, j int) bool {
		return data.OrigIndex[indices[i]] < data.OrigIndex[indices[j]]
	})

	newRows := make([][]string, n)
	newOrig := make([]int, n)
	for i, idx := range indices {
		newRows[i] = data.Rows[idx]
		newOrig[i] = data.OrigIndex[idx]
	}
	data.Rows = newRows
	data.OrigIndex = newOrig
}

// Apply sorts the data using the current sort settings.
func (s *SortState) Apply(data *csvpkg.DataSet) {
	if !s.Active {
		return
	}
	s.apply(data)
}

func (s *SortState) apply(data *csvpkg.DataSet) {
	col := s.Column
	asc := s.Ascending

	// Detect if column is numeric
	numeric := isNumericColumn(data, col)

	// Build index pairs so we can sort Rows and OrigIndex together
	n := len(data.Rows)
	indices := make([]int, n)
	for i := range n {
		indices[i] = i
	}

	sort.SliceStable(indices, func(i, j int) bool {
		a := cellValue(data.Rows[indices[i]], col)
		b := cellValue(data.Rows[indices[j]], col)

		var less bool
		if numeric {
			fa, _ := strconv.ParseFloat(a, 64)
			fb, _ := strconv.ParseFloat(b, 64)
			less = fa < fb
		} else {
			less = strings.ToLower(a) < strings.ToLower(b)
		}

		if asc {
			return less
		}
		return !less
	})

	// Reorder both slices according to sorted indices
	newRows := make([][]string, n)
	newOrig := make([]int, n)
	for i, idx := range indices {
		newRows[i] = data.Rows[idx]
		newOrig[i] = data.OrigIndex[idx]
	}
	data.Rows = newRows
	data.OrigIndex = newOrig
}

// Direction returns "▲" for ascending, "▼" for descending.
func (s *SortState) Direction() string {
	if s.Ascending {
		return "▲"
	}
	return "▼"
}

func isNumericColumn(data *csvpkg.DataSet, col int) bool {
	sampleSize := min(100, len(data.Rows))
	numericCount := 0
	total := 0

	for i := range sampleSize {
		val := cellValue(data.Rows[i], col)
		if val == "" {
			continue
		}
		total++
		if _, err := strconv.ParseFloat(val, 64); err == nil {
			numericCount++
		}
	}

	// Consider numeric if >80% of non-empty values parse as numbers
	return total > 0 && float64(numericCount)/float64(total) > 0.8
}

func cellValue(row []string, col int) string {
	if col < len(row) {
		return row[col]
	}
	return ""
}
