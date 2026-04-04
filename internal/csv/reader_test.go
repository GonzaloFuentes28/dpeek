package csv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "small.csv")
	ds, err := Load(path, ',', true)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if ds.ColCount() != 5 {
		t.Errorf("expected 5 columns, got %d", ds.ColCount())
	}
	if ds.RowCount() != 20 {
		t.Errorf("expected 20 rows, got %d", ds.RowCount())
	}

	// Check headers
	expectedHeaders := []string{"name", "age", "city", "salary", "active"}
	for i, h := range expectedHeaders {
		if ds.Headers[i] != h {
			t.Errorf("header[%d]: expected %q, got %q", i, h, ds.Headers[i])
		}
	}

	// Check first row (may be "Alice" or "Alicia" if edited during manual testing)
	if ds.Rows[0][0] != "Alicia" && ds.Rows[0][0] != "Alice" {
		t.Errorf("first row name: expected Alice or Alicia, got %q", ds.Rows[0][0])
	}
	if ds.Rows[0][1] != "32" {
		t.Errorf("first row age: expected 32, got %q", ds.Rows[0][1])
	}
}

func TestLoadNoHeader(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "small.csv")
	ds, err := Load(path, ',', false)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	// Without header, first row is data, headers are A, B, C...
	if ds.RowCount() != 21 { // 20 data + 1 header-as-data
		t.Errorf("expected 21 rows, got %d", ds.RowCount())
	}
	if ds.Headers[0] != "A" {
		t.Errorf("expected header A, got %q", ds.Headers[0])
	}
}

func TestSetCell(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "small.csv")
	ds, err := Load(path, ',', true)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if ds.Modified {
		t.Error("expected Modified=false initially")
	}

	ds.SetCell(0, 0, "Changed")
	if !ds.Modified {
		t.Error("expected Modified=true after SetCell")
	}
	if ds.Rows[0][0] != "Changed" {
		t.Errorf("expected Changed, got %q", ds.Rows[0][0])
	}
}

func TestSetCellSameValue(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "small.csv")
	ds, err := Load(path, ',', true)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	original := ds.Rows[0][0]
	ds.SetCell(0, 0, original)
	if ds.Modified {
		t.Error("expected Modified=false when setting same value")
	}
}

func TestSaveRoundtrip(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "small.csv")
	ds, err := Load(path, ',', true)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	ds.SetCell(0, 0, "Alicia")

	// Save to temp file
	tmp := filepath.Join(t.TempDir(), "out.csv")
	if err := ds.Save(tmp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and verify
	ds2, err := Load(tmp, ',', true)
	if err != nil {
		t.Fatalf("Reload failed: %v", err)
	}
	if ds2.Rows[0][0] != "Alicia" {
		t.Errorf("expected Alicia after reload, got %q", ds2.Rows[0][0])
	}
	if ds2.RowCount() != 20 {
		t.Errorf("expected 20 rows after reload, got %d", ds2.RowCount())
	}

	// Clean up
	os.Remove(tmp)
}

func TestColName(t *testing.T) {
	tests := []struct {
		i    int
		want string
	}{
		{0, "A"},
		{1, "B"},
		{25, "Z"},
		{26, "AA"},
		{27, "AB"},
	}
	for _, tt := range tests {
		got := colName(tt.i)
		if got != tt.want {
			t.Errorf("colName(%d): expected %q, got %q", tt.i, tt.want, got)
		}
	}
}
