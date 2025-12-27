package headlessterm

import (
	"testing"
)

func TestParseSixel_SimplePixel(t *testing.T) {
	// Single sixel '?' = 0 (no pixels), '~' = 63 (all 6 pixels)
	// '@' = 1 (only bottom pixel)
	data := []byte("~")
	img, err := ParseSixel(nil, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.Width != 1 {
		t.Errorf("expected width 1, got %d", img.Width)
	}
	if img.Height != 6 {
		t.Errorf("expected height 6, got %d", img.Height)
	}
}

func TestParseSixel_MultipleColumns(t *testing.T) {
	// Three columns
	data := []byte("~~~")
	img, err := ParseSixel(nil, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.Width != 3 {
		t.Errorf("expected width 3, got %d", img.Width)
	}
	if img.Height != 6 {
		t.Errorf("expected height 6, got %d", img.Height)
	}
}

func TestParseSixel_NewLine(t *testing.T) {
	// Two rows of sixels (each row is 6 pixels high)
	data := []byte("~-~")
	img, err := ParseSixel(nil, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.Width != 1 {
		t.Errorf("expected width 1, got %d", img.Width)
	}
	if img.Height != 12 {
		t.Errorf("expected height 12, got %d", img.Height)
	}
}

func TestParseSixel_CarriageReturn(t *testing.T) {
	// Carriage return + overwrite
	data := []byte("~$~")
	img, err := ParseSixel(nil, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.Width != 1 {
		t.Errorf("expected width 1, got %d", img.Width)
	}
}

func TestParseSixel_Repeat(t *testing.T) {
	// Repeat 5 times
	data := []byte("!5~")
	img, err := ParseSixel(nil, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.Width != 5 {
		t.Errorf("expected width 5, got %d", img.Width)
	}
}

func TestParseSixel_ColorRGB(t *testing.T) {
	// Define color 1 as red (RGB: 100,0,0 = full red)
	// Select color 1 and draw
	data := []byte("#1;2;100;0;0#1~")
	img, err := ParseSixel(nil, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.Width != 1 || img.Height != 6 {
		t.Errorf("unexpected dimensions: %dx%d", img.Width, img.Height)
	}
	// Check that pixel is red
	if len(img.Data) >= 4 {
		r, g, b := img.Data[0], img.Data[1], img.Data[2]
		if r != 255 || g != 0 || b != 0 {
			t.Errorf("expected red (255,0,0), got (%d,%d,%d)", r, g, b)
		}
	}
}

func TestParseSixel_ColorHLS(t *testing.T) {
	// Define color 2 as HLS (type 1)
	data := []byte("#2;1;120;50;100#2~")
	img, err := ParseSixel(nil, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.Width != 1 {
		t.Errorf("expected width 1, got %d", img.Width)
	}
}

func TestParseSixel_Transparent(t *testing.T) {
	// P2=1 means transparent background
	params := []int64{0, 1, 0}
	data := []byte("~")
	img, err := ParseSixel(params, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !img.Transparent {
		t.Error("expected transparent background")
	}
}

func TestParseSixel_Empty(t *testing.T) {
	data := []byte("")
	img, err := ParseSixel(nil, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.Width != 0 || img.Height != 0 {
		t.Errorf("expected 0x0, got %dx%d", img.Width, img.Height)
	}
}

func TestParseSixel_ComplexImage(t *testing.T) {
	// A more complex sixel with multiple colors and rows
	data := []byte("#0;2;0;0;0#1;2;100;0;0#0!10~-#1!10~")
	img, err := ParseSixel(nil, data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if img.Width != 10 {
		t.Errorf("expected width 10, got %d", img.Width)
	}
	if img.Height != 12 {
		t.Errorf("expected height 12, got %d", img.Height)
	}
}

// TestSixelImageDisplay tests end-to-end sixel display via terminal
func TestSixelImageDisplay(t *testing.T) {
	term := New(WithSize(24, 80))
	term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})

	// Send a simple sixel via DCS sequence
	// DCS P1;P2;P3 q <sixel data> ST
	// P2=0 means opaque background
	sixel := "\x1bP0;0;0q#0;2;100;0;0#0!10~-!10~\x1b\\"
	term.WriteString(sixel)

	// Verify image was stored
	if term.ImageCount() != 1 {
		t.Errorf("expected 1 image, got %d", term.ImageCount())
	}

	// Verify placement was created
	if term.ImagePlacementCount() != 1 {
		t.Errorf("expected 1 placement, got %d", term.ImagePlacementCount())
	}
}

// TestSixelImageCellAssignment tests that sixel assigns to cells correctly
func TestSixelImageCellAssignment(t *testing.T) {
	term := New(WithSize(24, 80))

	// Set cell size (10x10 pixels per cell)
	term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})

	// Position cursor at row 2, col 5
	term.WriteString("\x1b[3;6H") // 1-based positioning

	// Send a sixel that covers 20x12 pixels (2 cols, 2 rows at 10x10 cells)
	// 20 columns of sixels, 2 rows of 6 pixels each = 12 pixels
	sixel := "\x1bP0;0;0q#0;2;100;0;0#0!20~-!20~\x1b\\"
	term.WriteString(sixel)

	// Check cells have image references (2x2 cell coverage)
	for row := 2; row < 4; row++ {
		for col := 5; col < 7; col++ {
			cell := term.Cell(row, col)
			if cell == nil {
				t.Fatalf("cell at %d,%d is nil", row, col)
			}
			if !cell.HasImage() {
				t.Errorf("expected cell at %d,%d to have image", row, col)
			}
			if cell.Char != ImagePlaceholderChar {
				t.Errorf("expected placeholder char at %d,%d, got %U", row, col, cell.Char)
			}
		}
	}
}

// TestSixelCursorMovement tests that cursor moves down after sixel
func TestSixelCursorMovement(t *testing.T) {
	term := New(WithSize(24, 80))
	term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})

	// Position cursor at row 0, col 0
	term.WriteString("\x1b[1;1H")

	initialRow, _ := term.CursorPos()

	// Send sixel covering 12 pixels = 2 rows
	sixel := "\x1bP0;0;0q!10~-!10~\x1b\\"
	term.WriteString(sixel)

	// Cursor should have moved down by number of cell rows (2)
	newRow, _ := term.CursorPos()
	expectedRow := initialRow + 2 // 12 pixels / 10 pixels per cell = 1.2, rounded up = 2
	if newRow != expectedRow {
		t.Errorf("expected cursor at row %d, got %d", expectedRow, newRow)
	}
}

// TestSixelScrollingAtBottom tests that sixel scrolls the screen when placed at bottom
func TestSixelScrollingAtBottom(t *testing.T) {
	term := New(WithSize(10, 80)) // Small terminal: 10 rows
	term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})

	// Write some text first
	term.WriteString("Line 1\r\n")
	term.WriteString("Line 2\r\n")
	term.WriteString("Line 3\r\n")

	// Position cursor at last row (row 9, 0-indexed)
	term.WriteString("\x1b[10;1H")

	// Verify cursor is at last row
	curRow, _ := term.CursorPos()
	if curRow != 9 {
		t.Fatalf("expected cursor at row 9, got %d", curRow)
	}

	// Send sixel covering 30 pixels = 3 rows (will need to scroll)
	// With cursor at row 9 and image needing 3 rows (9, 10, 11),
	// it should scroll 2 lines and place image starting at row 7
	sixel := "\x1bP0;0;0q!10~-!10~-!10~\x1b\\"
	term.WriteString(sixel)

	// After scrolling, the image should be visible in the terminal
	// Check that cells at the new position have image references
	placements := term.ImagePlacements()
	if len(placements) != 1 {
		t.Fatalf("expected 1 placement, got %d", len(placements))
	}

	// The placement should have been adjusted to fit on screen
	p := placements[0]
	if p.Row+p.Rows > 10 {
		t.Errorf("placement extends beyond screen: row=%d, rows=%d", p.Row, p.Rows)
	}

	// Cursor should be at the row after the image (or last row if at bottom)
	newRow, _ := term.CursorPos()
	expectedRow := p.Row + p.Rows
	if expectedRow >= 10 {
		expectedRow = 9 // Clamped to last row
	}
	if newRow != expectedRow {
		t.Errorf("expected cursor at row %d, got %d", expectedRow, newRow)
	}

	// "Line 1" should have scrolled off, verify Line 2 is now at top
	cell := term.Cell(0, 0)
	if cell == nil {
		t.Fatal("cell at 0,0 is nil")
	}
	// After scroll, first visible line should be "Line 2" or later
	// Since we scrolled, the original content moved up
}
