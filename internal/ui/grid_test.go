package ui

import "testing"

func TestGridSetItemsCalculatesDimensions(t *testing.T) {
	m := NewGridModel()
	items := []GridItem{"first", "second line"}
	m.SetItems(items)

	if m.itemWidth == 0 {
		t.Fatalf("expected item width to be calculated")
	}
	if m.itemHeight == 0 {
		t.Fatalf("expected item height to be calculated")
	}
	if got := m.GetIndex(); got != 0 {
		t.Fatalf("expected cursor to start at 0, got %d", got)
	}
}

func TestGridSetCursorClamps(t *testing.T) {
	m := NewGridModel()
	m.SetItems([]GridItem{"one", "two"})

	m.SetCursor(-1)
	if m.cursor != 0 {
		t.Fatalf("expected cursor to clamp to 0, got %d", m.cursor)
	}

	m.SetCursor(10)
	if m.cursor != 1 {
		t.Fatalf("expected cursor to clamp to last index, got %d", m.cursor)
	}
}

// Navigation is handled by bubbletea key messages; direct cursor mutation is covered
// via SetCursor and SetItems tests above.
