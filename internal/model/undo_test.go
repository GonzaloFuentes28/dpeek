package model

import "testing"

func TestUndoStackPushAndUndo(t *testing.T) {
	s := NewUndoStack[string](10)

	if s.CanUndo() {
		t.Error("expected CanUndo=false on empty stack")
	}

	s.Push("a")
	s.Push("b")
	s.Push("c")

	if !s.CanUndo() {
		t.Error("expected CanUndo=true")
	}

	val, ok := s.Undo()
	if !ok || val != "c" {
		t.Errorf("expected c, got %q", val)
	}
	val, ok = s.Undo()
	if !ok || val != "b" {
		t.Errorf("expected b, got %q", val)
	}
	val, ok = s.Undo()
	if !ok || val != "a" {
		t.Errorf("expected a, got %q", val)
	}
	_, ok = s.Undo()
	if ok {
		t.Error("expected Undo to fail on empty stack")
	}
}

func TestUndoStackRedo(t *testing.T) {
	s := NewUndoStack[string](10)
	s.Push("a")
	s.Push("b")

	s.Undo()
	s.Undo()

	if !s.CanRedo() {
		t.Error("expected CanRedo=true")
	}

	val, ok := s.Redo()
	if !ok || val != "a" {
		t.Errorf("expected a, got %q", val)
	}
	val, ok = s.Redo()
	if !ok || val != "b" {
		t.Errorf("expected b, got %q", val)
	}
	_, ok = s.Redo()
	if ok {
		t.Error("expected Redo to fail at end")
	}
}

func TestUndoStackPushTruncatesRedo(t *testing.T) {
	s := NewUndoStack[string](10)
	s.Push("a")
	s.Push("b")
	s.Push("c")

	s.Undo() // undo c
	s.Undo() // undo b
	s.Push("d")

	// Redo should fail — b and c were truncated
	if s.CanRedo() {
		t.Error("expected CanRedo=false after push truncates redo tail")
	}

	val, ok := s.Undo()
	if !ok || val != "d" {
		t.Errorf("expected d, got %q", val)
	}
	val, ok = s.Undo()
	if !ok || val != "a" {
		t.Errorf("expected a, got %q", val)
	}
}

func TestUndoStackOverflow(t *testing.T) {
	s := NewUndoStack[int](3)
	s.Push(1)
	s.Push(2)
	s.Push(3)
	s.Push(4) // drops 1

	var values []int
	for {
		val, ok := s.Undo()
		if !ok {
			break
		}
		values = append(values, val)
	}

	if len(values) != 3 {
		t.Errorf("expected 3 entries, got %d", len(values))
	}
	if values[0] != 4 || values[1] != 3 || values[2] != 2 {
		t.Errorf("expected [4,3,2], got %v", values)
	}
}

func TestUndoStackIsModified(t *testing.T) {
	s := NewUndoStack[string](10)

	if s.IsModified() {
		t.Error("expected not modified initially")
	}

	s.Push("a")
	if !s.IsModified() {
		t.Error("expected modified after push")
	}

	s.MarkSaved()
	if s.IsModified() {
		t.Error("expected not modified after save")
	}

	s.Push("b")
	if !s.IsModified() {
		t.Error("expected modified after new push")
	}

	s.Undo()
	if s.IsModified() {
		t.Error("expected not modified after undo back to saved state")
	}

	s.Undo()
	if !s.IsModified() {
		t.Error("expected modified when before saved state")
	}
}

func TestUndoStackSavedPosAfterTruncate(t *testing.T) {
	s := NewUndoStack[string](10)
	s.Push("a")
	s.Push("b")
	s.MarkSaved() // saved at pos=1 (after b)

	s.Undo()      // undo b, pos=0
	s.Push("c")   // truncates b, saved state is gone

	// Can never get back to saved state
	if !s.IsModified() {
		t.Error("expected modified — saved state was truncated")
	}

	s.Undo() // undo c
	if !s.IsModified() {
		t.Error("expected modified — saved state still unreachable")
	}
}
