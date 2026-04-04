package detect

import "testing"

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		path      string
		delim     rune
		wantFmt   Format
		wantDelim rune
	}{
		{"data.csv", 0, FormatCSV, ','},
		{"data.CSV", 0, FormatCSV, ','},
		{"data.tsv", 0, FormatTSV, '\t'},
		{"data.json", 0, FormatJSON, 0},
		{"data.jsonl", 0, FormatJSONL, 0},
		{"data.ndjson", 0, FormatJSONL, 0},
		{"data.txt", 0, FormatUnknown, 0},
		// Delimiter override
		{"data.csv", ';', FormatCSV, ';'},
		{"data.tsv", '|', FormatTSV, '|'},
		// Unknown extension with delimiter → treat as CSV
		{"data.txt", '|', FormatCSV, '|'},
	}

	for _, tt := range tests {
		fmt, delim := DetectFormat(tt.path, tt.delim)
		if fmt != tt.wantFmt {
			t.Errorf("DetectFormat(%q, %q): format = %v, want %v", tt.path, string(tt.delim), fmt, tt.wantFmt)
		}
		if delim != tt.wantDelim {
			t.Errorf("DetectFormat(%q, %q): delim = %q, want %q", tt.path, string(tt.delim), string(delim), string(tt.wantDelim))
		}
	}
}
