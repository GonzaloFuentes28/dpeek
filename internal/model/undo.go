package model

import (
	jsonpkg "github.com/GonzaloFuentes28/dpeek/internal/json"
)

// UndoStack is a bounded undo/redo stack with save-position tracking.
type UndoStack[T any] struct {
	entries  []T
	pos      int // index of last applied entry (-1 = nothing to undo)
	savedPos int // pos at last save (-1 = never saved / initial state)
	maxSize  int
}

// NewUndoStack creates a new undo stack with the given capacity.
func NewUndoStack[T any](maxSize int) UndoStack[T] {
	return UndoStack[T]{
		pos:      -1,
		savedPos: -1,
		maxSize:  maxSize,
	}
}

// Push adds a new entry, truncating any redo history.
func (s *UndoStack[T]) Push(entry T) {
	// Truncate redo tail — anything after pos is redo history
	newLen := s.pos + 1
	// If the saved position was in the truncated redo tail, it's gone
	if s.savedPos > s.pos {
		s.savedPos = -2 // unreachable — file was saved at a state we can't get back to
	}
	s.entries = s.entries[:newLen]
	s.entries = append(s.entries, entry)
	s.pos++

	// Enforce capacity
	if len(s.entries) > s.maxSize {
		drop := len(s.entries) - s.maxSize
		s.entries = s.entries[drop:]
		s.pos -= drop
		s.savedPos -= drop
	}
}

// Undo returns the most recent entry to reverse, or false if nothing to undo.
func (s *UndoStack[T]) Undo() (T, bool) {
	if s.pos < 0 {
		var zero T
		return zero, false
	}
	entry := s.entries[s.pos]
	s.pos--
	return entry, true
}

// Redo returns the next entry to reapply, or false if nothing to redo.
func (s *UndoStack[T]) Redo() (T, bool) {
	if s.pos+1 >= len(s.entries) {
		var zero T
		return zero, false
	}
	s.pos++
	return s.entries[s.pos], true
}

// CanUndo returns true if there are entries to undo.
func (s *UndoStack[T]) CanUndo() bool {
	return s.pos >= 0
}

// CanRedo returns true if there are entries to redo.
func (s *UndoStack[T]) CanRedo() bool {
	return s.pos+1 < len(s.entries)
}

// MarkSaved records the current position as the saved state.
func (s *UndoStack[T]) MarkSaved() {
	s.savedPos = s.pos
}

// IsModified returns true if the current state differs from the last save.
func (s *UndoStack[T]) IsModified() bool {
	return s.pos != s.savedPos
}

// CellEdit records a single CSV cell edit for undo/redo.
type CellEdit struct {
	OrigLine int // data.OrigIndex[dataRow] — stable across sorts
	Col      int
	OldValue string
	NewValue string
}

// NodeEdit records a single JSON leaf edit for undo/redo.
type NodeEdit struct {
	Node     *jsonpkg.Node
	OldValue string
	OldKind  jsonpkg.NodeKind
	NewValue string
	NewKind  jsonpkg.NodeKind
}
