package headlessterm

import (
	"testing"
)

func TestNewBuffer(t *testing.T) {
	b := NewBuffer(24, 80)

	if b.Rows() != 24 {
		t.Errorf("expected 24 rows, got %d", b.Rows())
	}
	if b.Cols() != 80 {
		t.Errorf("expected 80 cols, got %d", b.Cols())
	}
}

func TestBufferCell(t *testing.T) {
	b := NewBuffer(24, 80)

	cell := b.Cell(0, 0)
	if cell == nil {
		t.Fatal("expected cell at (0,0)")
	}

	cell.Char = 'A'

	retrieved := b.Cell(0, 0)
	if retrieved.Char != 'A' {
		t.Errorf("expected 'A', got '%c'", retrieved.Char)
	}
}

func TestBufferCellOutOfBounds(t *testing.T) {
	b := NewBuffer(24, 80)

	if b.Cell(-1, 0) != nil {
		t.Error("expected nil for negative row")
	}
	if b.Cell(0, -1) != nil {
		t.Error("expected nil for negative col")
	}
	if b.Cell(24, 0) != nil {
		t.Error("expected nil for row >= rows")
	}
	if b.Cell(0, 80) != nil {
		t.Error("expected nil for col >= cols")
	}
}

func TestBufferClearRow(t *testing.T) {
	b := NewBuffer(24, 80)

	b.Cell(0, 0).Char = 'A'
	b.Cell(0, 1).Char = 'B'

	b.ClearRow(0)

	if b.Cell(0, 0).Char != ' ' {
		t.Error("expected cell to be cleared")
	}
	if b.Cell(0, 1).Char != ' ' {
		t.Error("expected cell to be cleared")
	}
}

func TestBufferScrollUp(t *testing.T) {
	b := NewBuffer(5, 10)

	for row := 0; row < 5; row++ {
		b.Cell(row, 0).Char = rune('0' + row)
	}

	b.ScrollUp(0, 5, 1)

	// Row 0 should now have what was in row 1
	if b.Cell(0, 0).Char != '1' {
		t.Errorf("expected '1', got '%c'", b.Cell(0, 0).Char)
	}
	// Last row should be cleared
	if b.Cell(4, 0).Char != ' ' {
		t.Errorf("expected space, got '%c'", b.Cell(4, 0).Char)
	}
}

func TestBufferScrollDown(t *testing.T) {
	b := NewBuffer(5, 10)

	for row := 0; row < 5; row++ {
		b.Cell(row, 0).Char = rune('0' + row)
	}

	b.ScrollDown(0, 5, 1)

	// Row 1 should now have what was in row 0
	if b.Cell(1, 0).Char != '0' {
		t.Errorf("expected '0', got '%c'", b.Cell(1, 0).Char)
	}
	// First row should be cleared
	if b.Cell(0, 0).Char != ' ' {
		t.Errorf("expected space, got '%c'", b.Cell(0, 0).Char)
	}
}

func TestBufferScrollback(t *testing.T) {
	storage := &testScrollbackBuffer{lines: make([][]Cell, 0), maxLines: 100}
	b := NewBufferWithStorage(5, 10, storage)

	for row := 0; row < 5; row++ {
		b.Cell(row, 0).Char = rune('A' + row)
	}

	// Scroll up, line 0 should go to scrollback
	b.ScrollUp(0, 5, 1)

	if b.ScrollbackLen() != 1 {
		t.Errorf("expected 1 scrollback line, got %d", b.ScrollbackLen())
	}

	line := b.ScrollbackLine(0)
	if line == nil {
		t.Fatal("expected scrollback line")
	}
	if line[0].Char != 'A' {
		t.Errorf("expected 'A' in scrollback, got '%c'", line[0].Char)
	}
}

// testScrollbackBuffer is a test implementation of ScrollbackProvider
type testScrollbackBuffer struct {
	lines    [][]Cell
	maxLines int
}

func (s *testScrollbackBuffer) Push(line []Cell) {
	lineCopy := make([]Cell, len(line))
	copy(lineCopy, line)
	s.lines = append(s.lines, lineCopy)
	if s.maxLines > 0 && len(s.lines) > s.maxLines {
		s.lines = s.lines[len(s.lines)-s.maxLines:]
	}
}

func (s *testScrollbackBuffer) Len() int              { return len(s.lines) }
func (s *testScrollbackBuffer) Line(index int) []Cell { return s.lines[index] }
func (s *testScrollbackBuffer) Clear()                { s.lines = make([][]Cell, 0) }
func (s *testScrollbackBuffer) SetMaxLines(max int)   { s.maxLines = max }
func (s *testScrollbackBuffer) MaxLines() int         { return s.maxLines }

func (s *testScrollbackBuffer) Pop() []Cell {
	if len(s.lines) == 0 {
		return nil
	}
	line := s.lines[len(s.lines)-1]
	s.lines = s.lines[:len(s.lines)-1]
	return line
}

func TestBufferLineContent(t *testing.T) {
	b := NewBuffer(24, 80)

	b.Cell(0, 0).Char = 'H'
	b.Cell(0, 1).Char = 'e'
	b.Cell(0, 2).Char = 'l'
	b.Cell(0, 3).Char = 'l'
	b.Cell(0, 4).Char = 'o'

	content := b.LineContent(0)
	if content != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", content)
	}
}

func TestBufferTabStops(t *testing.T) {
	b := NewBuffer(24, 80)

	// Default tab stops at 0, 8, 16, etc.
	next := b.NextTabStop(0)
	if next != 8 {
		t.Errorf("expected next tab at 8, got %d", next)
	}

	next = b.NextTabStop(8)
	if next != 16 {
		t.Errorf("expected next tab at 16, got %d", next)
	}

	prev := b.PrevTabStop(16)
	if prev != 8 {
		t.Errorf("expected prev tab at 8, got %d", prev)
	}
}

func TestBufferResize(t *testing.T) {
	b := NewBuffer(10, 20)

	b.Cell(0, 0).Char = 'A'
	b.Cell(5, 10).Char = 'B'

	b.Resize(20, 40)

	if b.Rows() != 20 || b.Cols() != 40 {
		t.Errorf("expected 20x40, got %dx%d", b.Rows(), b.Cols())
	}

	// Content should be preserved
	if b.Cell(0, 0).Char != 'A' {
		t.Error("expected content to be preserved")
	}
	if b.Cell(5, 10).Char != 'B' {
		t.Error("expected content to be preserved")
	}
}

func TestBufferDirtyTracking(t *testing.T) {
	b := NewBuffer(24, 80)

	b.ClearAllDirty()

	if b.HasDirty() {
		t.Error("expected no dirty cells")
	}

	b.MarkDirty(0, 0)

	if !b.HasDirty() {
		t.Error("expected dirty cells")
	}

	dirty := b.DirtyCells()
	if len(dirty) != 1 {
		t.Errorf("expected 1 dirty cell, got %d", len(dirty))
	}
	if dirty[0].Row != 0 || dirty[0].Col != 0 {
		t.Error("expected dirty cell at (0,0)")
	}
}

func TestBufferInsertBlanks(t *testing.T) {
	b := NewBuffer(24, 80)

	b.Cell(0, 0).Char = 'A'
	b.Cell(0, 1).Char = 'B'
	b.Cell(0, 2).Char = 'C'

	b.InsertBlanks(0, 1, 2)

	if b.Cell(0, 0).Char != 'A' {
		t.Errorf("expected 'A', got '%c'", b.Cell(0, 0).Char)
	}
	if b.Cell(0, 1).Char != ' ' {
		t.Errorf("expected space, got '%c'", b.Cell(0, 1).Char)
	}
	if b.Cell(0, 2).Char != ' ' {
		t.Errorf("expected space, got '%c'", b.Cell(0, 2).Char)
	}
	if b.Cell(0, 3).Char != 'B' {
		t.Errorf("expected 'B', got '%c'", b.Cell(0, 3).Char)
	}
}

func TestBufferDeleteChars(t *testing.T) {
	b := NewBuffer(24, 80)

	b.Cell(0, 0).Char = 'A'
	b.Cell(0, 1).Char = 'B'
	b.Cell(0, 2).Char = 'C'
	b.Cell(0, 3).Char = 'D'

	b.DeleteChars(0, 1, 2)

	if b.Cell(0, 0).Char != 'A' {
		t.Errorf("expected 'A', got '%c'", b.Cell(0, 0).Char)
	}
	if b.Cell(0, 1).Char != 'D' {
		t.Errorf("expected 'D', got '%c'", b.Cell(0, 1).Char)
	}
}

func TestBufferWrappedLineTracking(t *testing.T) {
	b := NewBuffer(5, 10)

	// Initially no lines are wrapped
	if b.IsWrapped(0) {
		t.Error("expected line 0 not wrapped initially")
	}

	// Set wrapped
	b.SetWrapped(0, true)
	if !b.IsWrapped(0) {
		t.Error("expected line 0 to be wrapped")
	}

	// Clear wrapped
	b.SetWrapped(0, false)
	if b.IsWrapped(0) {
		t.Error("expected line 0 not wrapped after clear")
	}

	// Out of bounds should not panic
	b.SetWrapped(-1, true)
	b.SetWrapped(100, true)
	if b.IsWrapped(-1) {
		t.Error("expected false for out of bounds")
	}
	if b.IsWrapped(100) {
		t.Error("expected false for out of bounds")
	}
}

func TestBufferWrappedLineTrackingWithScroll(t *testing.T) {
	b := NewBuffer(5, 10)

	// Set some wrapped flags
	b.SetWrapped(0, true)
	b.SetWrapped(1, false)
	b.SetWrapped(2, true)

	// Scroll up
	b.ScrollUp(0, 5, 1)

	// Wrapped flags should move with lines
	if b.IsWrapped(0) != false { // was line 1
		t.Error("expected line 0 not wrapped after scroll")
	}
	if b.IsWrapped(1) != true { // was line 2
		t.Error("expected line 1 wrapped after scroll")
	}
	if b.IsWrapped(4) { // new line should not be wrapped
		t.Error("expected new line not wrapped")
	}
}

func TestBufferGrowRows(t *testing.T) {
	b := NewBuffer(5, 10)

	b.Cell(0, 0).Char = 'A'
	b.Cell(4, 0).Char = 'E'

	b.GrowRows(3)

	if b.Rows() != 8 {
		t.Errorf("expected 8 rows, got %d", b.Rows())
	}

	// Content should be preserved
	if b.Cell(0, 0).Char != 'A' {
		t.Error("expected content preserved")
	}
	if b.Cell(4, 0).Char != 'E' {
		t.Error("expected content preserved")
	}

	// New rows should be empty
	if b.Cell(7, 0).Char != ' ' {
		t.Error("expected new row to be empty")
	}
}

func TestBufferGrowCols(t *testing.T) {
	b := NewBuffer(5, 10)

	b.Cell(0, 0).Char = 'A'
	b.Cell(0, 9).Char = 'B'

	b.GrowCols(0, 20)

	if b.Cols() != 20 {
		t.Errorf("expected 20 cols, got %d", b.Cols())
	}

	// Content should be preserved
	if b.Cell(0, 0).Char != 'A' {
		t.Error("expected content preserved")
	}
	if b.Cell(0, 9).Char != 'B' {
		t.Error("expected content preserved")
	}

	// New cells should be empty
	if b.Cell(0, 15).Char != ' ' {
		t.Error("expected new cell to be empty")
	}
}
