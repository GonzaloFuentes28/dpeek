package model

import (
	"path/filepath"
	"strings"
	"testing"

	csvpkg "github.com/GonzaloFuentes28/dpeek/internal/csv"
)

func loadTestData(t *testing.T) *csvpkg.DataSet {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "small.csv")
	ds, err := csvpkg.Load(path, ',', true)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	return ds
}

func TestFilterGlobal(t *testing.T) {
	ds := loadTestData(t)
	f := NewFilterState()
	f.Input.SetValue("Madrid")
	f.Apply(ds)

	if !f.IsFiltered {
		t.Error("expected IsFiltered=true")
	}
	if f.Column != -1 {
		t.Errorf("expected Column=-1 (global), got %d", f.Column)
	}
	if len(f.FilteredIdx) == 0 {
		t.Error("expected matches for 'Madrid'")
	}
}

func TestFilterByColumn(t *testing.T) {
	ds := loadTestData(t)
	f := NewFilterState()
	f.Input.SetValue("city:Madrid")
	f.Apply(ds)

	if !f.IsFiltered {
		t.Error("expected IsFiltered=true")
	}
	if f.ColumnName != "city" {
		t.Errorf("expected ColumnName='city', got %q", f.ColumnName)
	}
	if len(f.FilteredIdx) == 0 {
		t.Error("expected matches for city:Madrid")
	}

	// Verify all matches are in the city column
	cityCol := -1
	for i, h := range ds.Headers {
		if h == "city" {
			cityCol = i
			break
		}
	}
	for _, idx := range f.FilteredIdx {
		if cityCol < len(ds.Rows[idx]) {
			cell := ds.Rows[idx][cityCol]
			if !strings.Contains(strings.ToLower(cell), "madrid") {
				t.Errorf("row %d city=%q does not contain 'Madrid'", idx, cell)
			}
		}
	}
}

func TestFilterByColumnCaseInsensitive(t *testing.T) {
	ds := loadTestData(t)
	f := NewFilterState()
	f.Input.SetValue("CITY:Madrid")
	f.Apply(ds)

	if f.ColumnName != "city" {
		t.Errorf("expected case-insensitive match to 'city', got %q", f.ColumnName)
	}
	if len(f.FilteredIdx) == 0 {
		t.Error("expected matches")
	}
}

func TestFilterNonExistentColumnFallback(t *testing.T) {
	ds := loadTestData(t)
	f := NewFilterState()
	f.Input.SetValue("nonexistent:value")
	f.Apply(ds)

	if f.Column != -1 {
		t.Errorf("expected global fallback (Column=-1), got %d", f.Column)
	}
}

func TestFilterClear(t *testing.T) {
	ds := loadTestData(t)
	f := NewFilterState()
	f.Input.SetValue("city:Madrid")
	f.Apply(ds)
	f.Clear()

	if f.IsFiltered {
		t.Error("expected IsFiltered=false after clear")
	}
	if f.Column != -1 {
		t.Errorf("expected Column=-1 after clear, got %d", f.Column)
	}
	if f.ColumnName != "" {
		t.Errorf("expected ColumnName='' after clear, got %q", f.ColumnName)
	}
}
