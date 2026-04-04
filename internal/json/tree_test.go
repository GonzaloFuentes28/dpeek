package json

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if root.Kind != KindObject {
		t.Errorf("root kind: expected Object, got %v", root.Kind)
	}
	if !root.Expanded {
		t.Error("root should be expanded")
	}
	if root.ChildCount() == 0 {
		t.Error("root should have children")
	}
}

func TestParseTypes(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find specific nodes and check types
	typeMap := make(map[string]NodeKind)
	for _, child := range root.Children {
		typeMap[child.Key] = child.Kind
	}

	if typeMap["company"] != KindString {
		t.Errorf("company should be string, got %v", typeMap["company"])
	}
	if typeMap["founded"] != KindNumber {
		t.Errorf("founded should be number, got %v", typeMap["founded"])
	}
	if typeMap["active"] != KindBool {
		t.Errorf("active should be bool, got %v", typeMap["active"])
	}
	if typeMap["address"] != KindObject {
		t.Errorf("address should be object, got %v", typeMap["address"])
	}
	if typeMap["employees"] != KindArray {
		t.Errorf("employees should be array, got %v", typeMap["employees"])
	}
	if typeMap["metadata"] != KindNull {
		t.Errorf("metadata should be null, got %v", typeMap["metadata"])
	}
}

func TestVisibleNodes(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// With default expansion (depth < 2), should have visible nodes
	visible := root.VisibleNodes()
	if len(visible) == 0 {
		t.Error("should have visible nodes")
	}

	// Root should be first
	if visible[0] != root {
		t.Error("first visible should be root")
	}
}

func TestToggle(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	before := len(root.VisibleNodes())

	// Collapse root
	root.Toggle()
	if root.Expanded {
		t.Error("root should be collapsed after toggle")
	}

	after := len(root.VisibleNodes())
	if after != 1 {
		t.Errorf("collapsed root should show 1 node, got %d", after)
	}

	// Expand again
	root.Toggle()
	restored := len(root.VisibleNodes())
	if restored != before {
		t.Errorf("expanded should restore %d nodes, got %d", before, restored)
	}
}

func TestExpandCollapseAll(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	root.ExpandAll()
	expandedCount := len(root.VisibleNodes())

	root.CollapseAll()
	collapsedCount := len(root.VisibleNodes())

	if collapsedCount >= expandedCount {
		t.Errorf("collapsed (%d) should be less than expanded (%d)", collapsedCount, expandedCount)
	}
}

func TestSearch(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	matches := root.Search("Madrid", nil)
	if len(matches) == 0 {
		t.Error("should find matches for 'Madrid'")
	}

	matches = root.Search("nonexistent", nil)
	if len(matches) != 0 {
		t.Errorf("should find 0 matches for 'nonexistent', got %d", len(matches))
	}
}

func TestSearchRegex(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	re := regexp.MustCompile("(?i)^mad")
	matches := root.Search("", re)
	if len(matches) == 0 {
		t.Error("should find regex matches for '^mad'")
	}

	reNo := regexp.MustCompile("^zzz$")
	matches = root.Search("", reNo)
	if len(matches) != 0 {
		t.Errorf("should find 0 regex matches for '^zzz$', got %d", len(matches))
	}
}

func TestParseJSONL(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.jsonl")
	root, err := ParseJSONL(path)
	if err != nil {
		t.Fatalf("ParseJSONL failed: %v", err)
	}

	if root.Kind != KindArray {
		t.Errorf("root kind: expected Array, got %v", root.Kind)
	}
	if root.ChildCount() != 5 {
		t.Errorf("expected 5 children (lines), got %d", root.ChildCount())
	}

	// Each child should be an object
	for i, child := range root.Children {
		if child.Kind != KindObject {
			t.Errorf("child %d: expected Object, got %v", i, child.Kind)
		}
		if child.Index != i {
			t.Errorf("child %d: expected index %d, got %d", i, i, child.Index)
		}
	}
}

func TestSetValue(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Find the "company" node (string leaf)
	var companyNode *Node
	for _, c := range root.Children {
		if c.Key == "company" {
			companyNode = c
			break
		}
	}
	if companyNode == nil {
		t.Fatal("company node not found")
	}

	// Edit string
	if err := companyNode.SetValue("New Corp"); err != nil {
		t.Fatalf("SetValue string failed: %v", err)
	}
	if companyNode.Kind != KindString {
		t.Errorf("expected string kind, got %v", companyNode.Kind)
	}
	if companyNode.Value != `"New Corp"` {
		t.Errorf("expected '\"New Corp\"', got %q", companyNode.Value)
	}

	// Edit to number
	if err := companyNode.SetValue("42"); err != nil {
		t.Fatalf("SetValue number failed: %v", err)
	}
	if companyNode.Kind != KindNumber {
		t.Errorf("expected number kind, got %v", companyNode.Kind)
	}

	// Edit to bool
	if err := companyNode.SetValue("true"); err != nil {
		t.Fatalf("SetValue bool failed: %v", err)
	}
	if companyNode.Kind != KindBool {
		t.Errorf("expected bool kind, got %v", companyNode.Kind)
	}

	// Edit to null
	if err := companyNode.SetValue("null"); err != nil {
		t.Fatalf("SetValue null failed: %v", err)
	}
	if companyNode.Kind != KindNull {
		t.Errorf("expected null kind, got %v", companyNode.Kind)
	}

	// Cannot edit non-leaf
	if err := root.SetValue("nope"); err == nil {
		t.Error("expected error editing non-leaf node")
	}
}

func TestEditableValue(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	for _, c := range root.Children {
		if c.Key == "company" {
			ev := c.EditableValue()
			if ev != "Acme Corp" {
				t.Errorf("expected 'Acme Corp', got %q", ev)
			}
		}
		if c.Key == "founded" {
			ev := c.EditableValue()
			if ev != "2015" {
				t.Errorf("expected '2015', got %q", ev)
			}
		}
	}
}

func TestToInterfaceRoundtrip(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	iface := root.ToInterface()
	m, ok := iface.(map[string]interface{})
	if !ok {
		t.Fatal("expected root to serialize as map")
	}

	if m["company"] != "Acme Corp" {
		t.Errorf("company: expected 'Acme Corp', got %v", m["company"])
	}
	if m["founded"] != float64(2015) {
		t.Errorf("founded: expected 2015, got %v", m["founded"])
	}
	if m["active"] != true {
		t.Errorf("active: expected true, got %v", m["active"])
	}
	if m["metadata"] != nil {
		t.Errorf("metadata: expected nil, got %v", m["metadata"])
	}
}

func TestSaveRoundtrip(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Modify a value
	for _, c := range root.Children {
		if c.Key == "company" {
			c.SetValue("Changed Corp")
		}
	}

	// Save to temp
	tmp := filepath.Join(t.TempDir(), "out.json")
	if err := Save(root, tmp); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	// Reload and verify
	root2, err := Parse(tmp)
	if err != nil {
		t.Fatalf("Parse saved file failed: %v", err)
	}

	for _, c := range root2.Children {
		if c.Key == "company" {
			if c.EditableValue() != "Changed Corp" {
				t.Errorf("expected 'Changed Corp', got %q", c.EditableValue())
			}
		}
	}
}

func TestSaveJSONLRoundtrip(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.jsonl")
	root, err := ParseJSONL(path)
	if err != nil {
		t.Fatalf("ParseJSONL failed: %v", err)
	}

	// Save as JSONL
	tmp := filepath.Join(t.TempDir(), "out.jsonl")
	if err := SaveJSONL(root, tmp); err != nil {
		t.Fatalf("SaveJSONL failed: %v", err)
	}

	// Reload as JSONL (not regular JSON)
	root2, err := ParseJSONL(tmp)
	if err != nil {
		t.Fatalf("ParseJSONL reload failed: %v", err)
	}

	if root2.ChildCount() != 5 {
		t.Errorf("expected 5 children after roundtrip, got %d", root2.ChildCount())
	}

	// Verify it's actually JSONL format (each line is valid JSON)
	data, _ := os.ReadFile(tmp)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines in JSONL file, got %d", len(lines))
	}
}

func TestSummary(t *testing.T) {
	path := filepath.Join("..", "..", "testdata", "sample.json")
	root, err := Parse(path)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	summary := root.Summary()
	if summary == "" {
		t.Error("root summary should not be empty")
	}

	// Find the employees array
	for _, child := range root.Children {
		if child.Key == "employees" {
			s := child.Summary()
			if s != "[3 items]" {
				t.Errorf("employees summary: expected '[3 items]', got %q", s)
			}
		}
	}
}
