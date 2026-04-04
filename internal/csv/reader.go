package csv

import (
	"encoding/csv"
	"fmt"
	"os"
)

// DataSet holds a parsed CSV/TSV file in memory.
type DataSet struct {
	Headers    []string
	Rows       [][]string
	OrigIndex  []int // original 1-based line number from file for each row
	HasHeader  bool
	Delimiter  rune
	FilePath   string
	Modified   bool
}

// Load reads a CSV/TSV file into a DataSet.
func Load(path string, delimiter rune, hasHeader bool) (*DataSet, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	r := csv.NewReader(f)
	r.Comma = delimiter
	r.LazyQuotes = true
	r.FieldsPerRecord = -1 // allow variable field counts

	records, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if len(records) == 0 {
		return &DataSet{
			FilePath:  path,
			Delimiter: delimiter,
			HasHeader: hasHeader,
		}, nil
	}

	ds := &DataSet{
		FilePath:  path,
		Delimiter: delimiter,
		HasHeader: hasHeader,
	}

	if hasHeader {
		ds.Headers = records[0]
		ds.Rows = records[1:]
	} else {
		// Generate column names: A, B, C, ...
		colCount := len(records[0])
		ds.Headers = make([]string, colCount)
		for i := range colCount {
			ds.Headers[i] = colName(i)
		}
		ds.Rows = records
	}

	// Build original line indices and normalize column counts
	ds.OrigIndex = make([]int, len(ds.Rows))
	for i, row := range ds.Rows {
		if hasHeader {
			ds.OrigIndex[i] = i + 2 // +1 for 1-based, +1 for header line
		} else {
			ds.OrigIndex[i] = i + 1 // +1 for 1-based
		}
		if len(row) < len(ds.Headers) {
			padded := make([]string, len(ds.Headers))
			copy(padded, row)
			ds.Rows[i] = padded
		}
	}

	return ds, nil
}

// RowCount returns the number of data rows (excluding header).
func (ds *DataSet) RowCount() int {
	return len(ds.Rows)
}

// ColCount returns the number of columns.
func (ds *DataSet) ColCount() int {
	return len(ds.Headers)
}

// SetCell updates a cell value and marks the dataset as modified only if the value changed.
func (ds *DataSet) SetCell(row, col int, value string) {
	if row < 0 || row >= len(ds.Rows) || col < 0 || col >= len(ds.Headers) {
		return
	}
	if ds.Rows[row][col] == value {
		return
	}
	ds.Rows[row][col] = value
	ds.Modified = true
}

// Save writes the dataset back to disk.
func (ds *DataSet) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	w.Comma = ds.Delimiter

	if ds.HasHeader {
		if err := w.Write(ds.Headers); err != nil {
			return fmt.Errorf("write header: %w", err)
		}
	}

	for _, row := range ds.Rows {
		if err := w.Write(row); err != nil {
			return fmt.Errorf("write row: %w", err)
		}
	}

	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}

	ds.Modified = false
	return nil
}

// colName converts a zero-based index to a column name: 0→A, 1→B, ..., 25→Z, 26→AA.
func colName(i int) string {
	name := ""
	for {
		name = string(rune('A'+i%26)) + name
		i = i/26 - 1
		if i < 0 {
			break
		}
	}
	return name
}
