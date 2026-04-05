package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

const (
	// DefaultChunkSize is the number of rows to load per chunk.
	DefaultChunkSize = 50000
	// ColWidthSampleSize limits how many rows are scanned for column width estimation.
	ColWidthSampleSize = 1000
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

// ChunkReader holds the state needed to continue reading chunks from a CSV file.
type ChunkReader struct {
	Reader    *csv.Reader
	File      *os.File
	ColCount  int
	HasHeader bool
	NextIndex int // next OrigIndex to assign
}

// LoadChunk reads the header and first chunk of rows from a CSV file.
// Returns a partial DataSet and a ChunkReader for continued reading.
func LoadChunk(path string, delimiter rune, hasHeader bool, chunkSize int) (*DataSet, *ChunkReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s: %w", path, err)
	}

	r := csv.NewReader(f)
	r.Comma = delimiter
	r.LazyQuotes = true
	r.FieldsPerRecord = -1

	ds := &DataSet{
		FilePath:  path,
		Delimiter: delimiter,
		HasHeader: hasHeader,
	}

	// Read first record to determine headers
	firstRecord, err := r.Read()
	if err != nil {
		f.Close()
		if err == io.EOF {
			return ds, nil, nil
		}
		return nil, nil, fmt.Errorf("parse %s: %w", path, err)
	}

	if hasHeader {
		ds.Headers = firstRecord
	} else {
		colCount := len(firstRecord)
		ds.Headers = make([]string, colCount)
		for i := range colCount {
			ds.Headers[i] = colName(i)
		}
		// First record is data
		padded := normalizeRow(firstRecord, len(ds.Headers))
		ds.Rows = append(ds.Rows, padded)
		ds.OrigIndex = append(ds.OrigIndex, 1)
	}

	// Read first chunk
	for range chunkSize {
		record, err := r.Read()
		if err != nil {
			f.Close()
			if err == io.EOF {
				return ds, nil, nil // file fully read
			}
			return nil, nil, fmt.Errorf("parse %s: %w", path, err)
		}
		padded := normalizeRow(record, len(ds.Headers))
		ds.Rows = append(ds.Rows, padded)
		ds.OrigIndex = append(ds.OrigIndex, 0) // placeholder, fixed below
	}

	// Fix OrigIndex values
	for i := range ds.Rows {
		if hasHeader {
			ds.OrigIndex[i] = i + 2
		} else {
			ds.OrigIndex[i] = i + 1
		}
	}

	cr := &ChunkReader{
		Reader:    r,
		File:      f,
		ColCount:  len(ds.Headers),
		HasHeader: hasHeader,
		NextIndex: len(ds.Rows),
	}
	return ds, cr, nil
}

// ReadNextChunk reads the next batch of rows from an open ChunkReader.
// Returns empty slices when EOF is reached.
func (cr *ChunkReader) ReadNextChunk(chunkSize int) ([][]string, []int, error) {
	var rows [][]string
	var origIndex []int

	for range chunkSize {
		record, err := cr.Reader.Read()
		if err != nil {
			if err == io.EOF {
				return rows, origIndex, nil
			}
			return nil, nil, fmt.Errorf("read chunk: %w", err)
		}
		padded := normalizeRow(record, cr.ColCount)
		rows = append(rows, padded)
		idx := cr.NextIndex
		if cr.HasHeader {
			origIndex = append(origIndex, idx+2)
		} else {
			origIndex = append(origIndex, idx+1)
		}
		cr.NextIndex++
	}

	return rows, origIndex, nil
}

// Close closes the underlying file.
func (cr *ChunkReader) Close() {
	if cr.File != nil {
		cr.File.Close()
	}
}

// normalizeRow pads a row to the expected column count.
func normalizeRow(row []string, colCount int) []string {
	if len(row) >= colCount {
		return row
	}
	padded := make([]string, colCount)
	copy(padded, row)
	return padded
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
