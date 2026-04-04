package detect

import (
	"path/filepath"
	"strings"
)

// Format represents a supported file format.
type Format int

const (
	FormatUnknown Format = iota
	FormatCSV
	FormatTSV
	FormatJSON
	FormatJSONL
)

func (f Format) String() string {
	switch f {
	case FormatCSV:
		return "CSV"
	case FormatTSV:
		return "TSV"
	case FormatJSON:
		return "JSON"
	case FormatJSONL:
		return "JSONL"
	default:
		return "Unknown"
	}
}

// DetectFormat determines the file format from its extension.
// If delimiterOverride is non-zero, CSV/TSV detection uses that delimiter.
// Returns the format and the delimiter rune (only meaningful for CSV/TSV).
func DetectFormat(path string, delimiterOverride rune) (Format, rune) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".csv":
		d := ','
		if delimiterOverride != 0 {
			d = delimiterOverride
		}
		return FormatCSV, d
	case ".tsv":
		d := '\t'
		if delimiterOverride != 0 {
			d = delimiterOverride
		}
		return FormatTSV, d
	case ".json":
		return FormatJSON, 0
	case ".jsonl", ".ndjson":
		return FormatJSONL, 0
	default:
		// If a delimiter is specified, treat as CSV
		if delimiterOverride != 0 {
			return FormatCSV, delimiterOverride
		}
		return FormatUnknown, 0
	}
}
