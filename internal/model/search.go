package model

import (
	"regexp"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"

	csvpkg "github.com/GonzaloFuentes28/dpeek/internal/csv"
)

// SearchMatch represents a cell that matches the search query.
type SearchMatch struct {
	Row int
	Col int
}

// SearchState manages incremental search across CSV cells.
type SearchState struct {
	Active       bool
	Input        textinput.Model
	Query        string
	Matches      []SearchMatch
	CurrentMatch int
	IsRegex      bool
	re           *regexp.Regexp
	Err          string
}

// NewSearchState creates a new search state.
func NewSearchState() SearchState {
	ti := textinput.New()
	ti.Placeholder = "Search (prefix / for regex)..."
	ti.CharLimit = 128
	return SearchState{Input: ti}
}

// Open activates the search input.
func (s *SearchState) Open() {
	s.Active = true
	s.Input.Focus()
	s.Input.SetValue(s.Query)
	s.Input.CursorEnd()
}

// Close deactivates the search input.
func (s *SearchState) Close() {
	s.Active = false
	s.Input.Blur()
}

// Compile prepares the search mode from the raw query.
func (s *SearchState) Compile() {
	s.Err = ""
	s.IsRegex = false
	s.re = nil

	raw := s.Input.Value()
	s.Query = raw

	if strings.HasPrefix(raw, "/") {
		pattern := raw[1:]
		if pattern == "" {
			s.Query = ""
			return
		}
		s.IsRegex = true
		var err error
		s.re, err = regexp.Compile("(?i)" + pattern)
		if err != nil {
			s.Err = "Invalid regex: " + err.Error()
			s.re = nil
		}
	}
}

func (s *SearchState) matches(text string) bool {
	if s.IsRegex {
		if s.re == nil {
			return false
		}
		return s.re.MatchString(text)
	}
	return strings.Contains(strings.ToLower(text), strings.ToLower(s.Query))
}

// Execute runs the search across all cells.
func (s *SearchState) Execute(data *csvpkg.DataSet) {
	s.Compile()
	s.Matches = nil
	s.CurrentMatch = 0

	if s.Query == "" || s.Err != "" {
		return
	}

	// Search headers
	for col, h := range data.Headers {
		if s.matches(h) {
			s.Matches = append(s.Matches, SearchMatch{Row: -1, Col: col})
		}
	}

	// Search data cells
	for row, r := range data.Rows {
		for col, cell := range r {
			if s.matches(cell) {
				s.Matches = append(s.Matches, SearchMatch{Row: row, Col: col})
			}
		}
	}
}

// Regex returns the compiled regex, or nil if not in regex mode.
func (s *SearchState) Regex() *regexp.Regexp {
	return s.re
}

// NextMatch moves to the next match and returns its position.
func (s *SearchState) NextMatch() (row, col int, ok bool) {
	if len(s.Matches) == 0 {
		return 0, 0, false
	}
	s.CurrentMatch = (s.CurrentMatch + 1) % len(s.Matches)
	m := s.Matches[s.CurrentMatch]
	return m.Row, m.Col, true
}

// PrevMatch moves to the previous match and returns its position.
func (s *SearchState) PrevMatch() (row, col int, ok bool) {
	if len(s.Matches) == 0 {
		return 0, 0, false
	}
	s.CurrentMatch--
	if s.CurrentMatch < 0 {
		s.CurrentMatch = len(s.Matches) - 1
	}
	m := s.Matches[s.CurrentMatch]
	return m.Row, m.Col, true
}

// IsMatch checks if a cell matches the current search.
func (s *SearchState) IsMatch(row, col int) bool {
	if s.Query == "" {
		return false
	}
	for _, m := range s.Matches {
		if m.Row == row && m.Col == col {
			return true
		}
	}
	return false
}

// IsCurrentMatch checks if a cell is the currently highlighted match.
func (s *SearchState) IsCurrentMatch(row, col int) bool {
	if len(s.Matches) == 0 {
		return false
	}
	m := s.Matches[s.CurrentMatch]
	return m.Row == row && m.Col == col
}

// MatchCount returns the total number of matches.
func (s *SearchState) MatchCount() int {
	return len(s.Matches)
}
