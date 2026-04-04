package json

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
)

// NodeKind represents the JSON type of a tree node.
type NodeKind int

const (
	KindObject NodeKind = iota
	KindArray
	KindString
	KindNumber
	KindBool
	KindNull
)

func (k NodeKind) String() string {
	switch k {
	case KindObject:
		return "object"
	case KindArray:
		return "array"
	case KindString:
		return "string"
	case KindNumber:
		return "number"
	case KindBool:
		return "bool"
	case KindNull:
		return "null"
	default:
		return "unknown"
	}
}

// Node represents a single node in the JSON tree.
type Node struct {
	Key      string  // key name (empty for root or array elements)
	Kind     NodeKind
	Value    string  // stringified leaf value
	Children []*Node
	Expanded bool
	Depth    int
	Parent   *Node
	Index    int // index within parent array (-1 if not an array element)
}

// IsLeaf returns true if the node has no children.
func (n *Node) IsLeaf() bool {
	return len(n.Children) == 0
}

// SetValue updates a leaf node's value string and kind, validating the type.
// Returns an error if the value can't be parsed for the target type.
func (n *Node) SetValue(raw string) error {
	if !n.IsLeaf() {
		return fmt.Errorf("cannot edit non-leaf node")
	}

	trimmed := strings.TrimSpace(raw)

	// Try to auto-detect type from the raw input
	switch {
	case trimmed == "null":
		n.Kind = KindNull
		n.Value = "null"
	case trimmed == "true" || trimmed == "false":
		n.Kind = KindBool
		n.Value = trimmed
	case isNumeric(trimmed):
		n.Kind = KindNumber
		n.Value = trimmed
	default:
		n.Kind = KindString
		// If user typed quotes, strip them; otherwise add them for display
		if len(trimmed) >= 2 && trimmed[0] == '"' && trimmed[len(trimmed)-1] == '"' {
			n.Value = trimmed
		} else {
			n.Value = fmt.Sprintf("%q", trimmed)
		}
	}
	return nil
}

// EditableValue returns the value in a form suitable for editing
// (strings without quotes, everything else as-is).
func (n *Node) EditableValue() string {
	if n.Kind == KindString && len(n.Value) >= 2 && n.Value[0] == '"' {
		// Unquote
		var s string
		if err := json.Unmarshal([]byte(n.Value), &s); err == nil {
			return s
		}
	}
	return n.Value
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	// Allow leading minus
	start := 0
	if s[0] == '-' {
		start = 1
		if len(s) == 1 {
			return false
		}
	}
	hasDot := false
	hasE := false
	for i := start; i < len(s); i++ {
		c := s[i]
		switch {
		case c >= '0' && c <= '9':
			continue
		case c == '.' && !hasDot && !hasE:
			hasDot = true
		case (c == 'e' || c == 'E') && !hasE && i > start:
			hasE = true
			if i+1 < len(s) && (s[i+1] == '+' || s[i+1] == '-') {
				i++
			}
		default:
			return false
		}
	}
	return true
}

// ToInterface converts the tree back to a Go interface{} for JSON serialization.
func (n *Node) ToInterface() interface{} {
	switch n.Kind {
	case KindObject:
		m := make(map[string]interface{}, len(n.Children))
		for _, c := range n.Children {
			m[c.Key] = c.ToInterface()
		}
		return m
	case KindArray:
		a := make([]interface{}, len(n.Children))
		for i, c := range n.Children {
			a[i] = c.ToInterface()
		}
		return a
	case KindString:
		var s string
		if err := json.Unmarshal([]byte(n.Value), &s); err == nil {
			return s
		}
		return n.Value
	case KindNumber:
		var f float64
		if err := json.Unmarshal([]byte(n.Value), &f); err == nil {
			return f
		}
		return n.Value
	case KindBool:
		return n.Value == "true"
	case KindNull:
		return nil
	}
	return nil
}

// Save serializes the tree and writes it to the given path as formatted JSON.
func Save(root *Node, path string) error {
	data := root.ToInterface()
	out, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize: %w", err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

// SaveJSONL serializes a root array node as JSONL (one JSON object per line).
func SaveJSONL(root *Node, path string) error {
	if root.Kind != KindArray {
		return Save(root, path)
	}

	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create %s: %w", path, err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, child := range root.Children {
		data := child.ToInterface()
		line, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("serialize line: %w", err)
		}
		w.Write(line)
		w.WriteByte('\n')
	}

	if err := w.Flush(); err != nil {
		return fmt.Errorf("flush: %w", err)
	}
	return nil
}

// ChildCount returns the number of direct children.
func (n *Node) ChildCount() int {
	return len(n.Children)
}

// Toggle expands or collapses a non-leaf node.
func (n *Node) Toggle() {
	if !n.IsLeaf() {
		n.Expanded = !n.Expanded
	}
}

// ExpandAll expands this node and all descendants.
func (n *Node) ExpandAll() {
	if !n.IsLeaf() {
		n.Expanded = true
		for _, c := range n.Children {
			c.ExpandAll()
		}
	}
}

// CollapseAll collapses this node and all descendants.
func (n *Node) CollapseAll() {
	if !n.IsLeaf() {
		n.Expanded = false
		for _, c := range n.Children {
			c.CollapseAll()
		}
	}
}

// VisibleNodes returns a flat list of currently visible nodes
// (respecting expanded/collapsed state), starting from this node.
func (n *Node) VisibleNodes() []*Node {
	var result []*Node
	n.collectVisible(&result)
	return result
}

func (n *Node) collectVisible(result *[]*Node) {
	*result = append(*result, n)
	if n.Expanded {
		for _, c := range n.Children {
			c.collectVisible(result)
		}
	}
}

// Summary returns a short description for collapsed containers.
// e.g. "{3 keys}" or "[5 items]"
func (n *Node) Summary() string {
	count := len(n.Children)
	switch n.Kind {
	case KindObject:
		if count == 1 {
			return "{1 key}"
		}
		return fmt.Sprintf("{%d keys}", count)
	case KindArray:
		if count == 1 {
			return "[1 item]"
		}
		return fmt.Sprintf("[%d items]", count)
	default:
		return n.Value
	}
}

// DisplayKey returns the key to show for this node.
func (n *Node) DisplayKey() string {
	if n.Key != "" {
		return n.Key
	}
	if n.Index >= 0 {
		return fmt.Sprintf("[%d]", n.Index)
	}
	return ""
}

// Search finds nodes whose key or value matches the query.
// If re is non-nil, it is used for matching; otherwise case-insensitive substring.
func (n *Node) Search(query string, re *regexp.Regexp) []*Node {
	var matches []*Node
	query = strings.ToLower(query)
	n.searchRecursive(query, re, &matches)
	return matches
}

func (n *Node) searchRecursive(query string, re *regexp.Regexp, matches *[]*Node) {
	searchValue := n.Value
	// Use unquoted value for string nodes so regex anchors work naturally
	if n.Kind == KindString {
		searchValue = n.EditableValue()
	}

	if re != nil {
		if re.MatchString(n.Key) || re.MatchString(searchValue) {
			*matches = append(*matches, n)
		}
	} else {
		if strings.Contains(strings.ToLower(n.Key), query) ||
			strings.Contains(strings.ToLower(searchValue), query) {
			*matches = append(*matches, n)
		}
	}
	for _, c := range n.Children {
		c.searchRecursive(query, re, matches)
	}
}

// EnsureVisible expands all ancestors so this node is visible.
func (n *Node) EnsureVisible() {
	for p := n.Parent; p != nil; p = p.Parent {
		p.Expanded = true
	}
}

// Parse reads a JSON file and builds a tree.
func Parse(path string) (*Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}

	var raw interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	root := buildNode("", raw, 0, nil, -1)
	root.Expanded = true // root is always expanded
	return root, nil
}

// ParseJSONL reads a JSONL file (one JSON object per line) and builds a tree
// with a root array containing each line as a child.
func ParseJSONL(path string) (*Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	root := &Node{
		Kind:     KindArray,
		Expanded: true,
		Depth:    0,
		Index:    -1,
	}

	scanner := bufio.NewScanner(f)
	// Increase buffer for large lines
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	idx := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var raw interface{}
		if err := json.Unmarshal([]byte(line), &raw); err != nil {
			return nil, fmt.Errorf("parse line %d: %w", idx+1, err)
		}

		child := buildNode("", raw, 1, root, idx)
		root.Children = append(root.Children, child)
		idx++
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}

	return root, nil
}

func buildNode(key string, val interface{}, depth int, parent *Node, index int) *Node {
	n := &Node{
		Key:    key,
		Depth:  depth,
		Parent: parent,
		Index:  index,
	}

	switch v := val.(type) {
	case map[string]interface{}:
		n.Kind = KindObject
		n.Expanded = depth < 2 // auto-expand first 2 levels

		// Sort keys for consistent ordering
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			child := buildNode(k, v[k], depth+1, n, -1)
			n.Children = append(n.Children, child)
		}

	case []interface{}:
		n.Kind = KindArray
		n.Expanded = depth < 2

		for i, item := range v {
			child := buildNode("", item, depth+1, n, i)
			n.Children = append(n.Children, child)
		}

	case string:
		n.Kind = KindString
		n.Value = fmt.Sprintf("%q", v)

	case float64:
		n.Kind = KindNumber
		// Display integers without decimal point
		if v == float64(int64(v)) {
			n.Value = fmt.Sprintf("%d", int64(v))
		} else {
			n.Value = fmt.Sprintf("%g", v)
		}

	case bool:
		n.Kind = KindBool
		n.Value = fmt.Sprintf("%t", v)

	case nil:
		n.Kind = KindNull
		n.Value = "null"
	}

	return n
}
