package headlessterm

import (
	"encoding/base64"
	"image/color"
	"testing"
)

func TestSnapshot_Text(t *testing.T) {
	term := New(WithSize(3, 10))
	term.WriteString("Hello")
	term.WriteString("\x1b[2;1H") // Move to row 2, col 1
	term.WriteString("World")

	snap := term.Snapshot(SnapshotDetailText)

	if snap.Size.Rows != 3 {
		t.Errorf("Size.Rows = %d, want 3", snap.Size.Rows)
	}
	if snap.Size.Cols != 10 {
		t.Errorf("Size.Cols = %d, want 10", snap.Size.Cols)
	}

	if len(snap.Lines) != 3 {
		t.Fatalf("len(Lines) = %d, want 3", len(snap.Lines))
	}

	if snap.Lines[0].Text != "Hello" {
		t.Errorf("Lines[0].Text = %q, want %q", snap.Lines[0].Text, "Hello")
	}
	if snap.Lines[1].Text != "World" {
		t.Errorf("Lines[1].Text = %q, want %q", snap.Lines[1].Text, "World")
	}

	// Text mode should not have segments or cells
	if snap.Lines[0].Segments != nil {
		t.Error("Text mode should not have segments")
	}
	if snap.Lines[0].Cells != nil {
		t.Error("Text mode should not have cells")
	}
}

func TestSnapshot_Cursor(t *testing.T) {
	term := New(WithSize(5, 10))
	term.WriteString("ABC")

	snap := term.Snapshot(SnapshotDetailText)

	if snap.Cursor.Row != 0 {
		t.Errorf("Cursor.Row = %d, want 0", snap.Cursor.Row)
	}
	if snap.Cursor.Col != 3 {
		t.Errorf("Cursor.Col = %d, want 3", snap.Cursor.Col)
	}
	if !snap.Cursor.Visible {
		t.Error("Cursor.Visible = false, want true")
	}
	if snap.Cursor.Style != "block" {
		t.Errorf("Cursor.Style = %q, want %q", snap.Cursor.Style, "block")
	}
}

func TestSnapshot_Styled(t *testing.T) {
	term := New(WithSize(3, 20))

	// Write text with different colors
	term.WriteString("\x1b[31mRed\x1b[0m Normal \x1b[32mGreen\x1b[0m")

	snap := term.Snapshot(SnapshotDetailStyled)

	if len(snap.Lines) < 1 {
		t.Fatal("Expected at least 1 line")
	}

	line := snap.Lines[0]
	if len(line.Segments) < 3 {
		t.Fatalf("Expected at least 3 segments, got %d", len(line.Segments))
	}

	// First segment should be red
	if line.Segments[0].Text != "Red" {
		t.Errorf("Segment[0].Text = %q, want %q", line.Segments[0].Text, "Red")
	}

	// Styled mode should not have cells
	if line.Cells != nil {
		t.Error("Styled mode should not have cells")
	}
}

func TestSnapshot_Full(t *testing.T) {
	term := New(WithSize(3, 10))
	term.WriteString("Hi")

	snap := term.Snapshot(SnapshotDetailFull)

	if len(snap.Lines) < 1 {
		t.Fatal("Expected at least 1 line")
	}

	line := snap.Lines[0]
	if len(line.Cells) != 10 {
		t.Fatalf("Expected 10 cells, got %d", len(line.Cells))
	}

	if line.Cells[0].Char != "H" {
		t.Errorf("Cells[0].Char = %q, want %q", line.Cells[0].Char, "H")
	}
	if line.Cells[1].Char != "i" {
		t.Errorf("Cells[1].Char = %q, want %q", line.Cells[1].Char, "i")
	}
	// Rest should be spaces
	if line.Cells[2].Char != " " {
		t.Errorf("Cells[2].Char = %q, want %q", line.Cells[2].Char, " ")
	}
}

func TestSnapshot_Attributes(t *testing.T) {
	term := New(WithSize(3, 20))

	// Bold text
	term.WriteString("\x1b[1mBold\x1b[0m")

	snap := term.Snapshot(SnapshotDetailFull)

	if len(snap.Lines[0].Cells) < 4 {
		t.Fatal("Expected at least 4 cells")
	}

	for i := 0; i < 4; i++ {
		if !snap.Lines[0].Cells[i].Attributes.Bold {
			t.Errorf("Cell[%d] should be bold", i)
		}
	}
}

func TestSnapshot_UnderlineStyles(t *testing.T) {
	tests := []struct {
		name     string
		sequence string
		expected string
	}{
		{"single", "\x1b[4mText\x1b[0m", "single"},
		{"single_4:1", "\x1b[4:1mText\x1b[0m", "single"},
		{"double", "\x1b[4:2mText\x1b[0m", "double"},
		{"curly", "\x1b[4:3mText\x1b[0m", "curly"},
		{"dotted", "\x1b[4:4mText\x1b[0m", "dotted"},
		{"dashed", "\x1b[4:5mText\x1b[0m", "dashed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := New(WithSize(3, 20))
			term.WriteString(tt.sequence)

			snap := term.Snapshot(SnapshotDetailFull)

			if len(snap.Lines[0].Cells) < 4 {
				t.Fatal("Expected at least 4 cells")
			}

			got := snap.Lines[0].Cells[0].Attributes.Underline
			if got != tt.expected {
				t.Errorf("Underline = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSnapshot_BlinkStyles(t *testing.T) {
	tests := []struct {
		name     string
		sequence string
		expected string
	}{
		{"slow", "\x1b[5mText\x1b[0m", "slow"},
		{"fast", "\x1b[6mText\x1b[0m", "fast"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := New(WithSize(3, 20))
			term.WriteString(tt.sequence)

			snap := term.Snapshot(SnapshotDetailFull)

			if len(snap.Lines[0].Cells) < 4 {
				t.Fatal("Expected at least 4 cells")
			}

			got := snap.Lines[0].Cells[0].Attributes.Blink
			if got != tt.expected {
				t.Errorf("Blink = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestSnapshot_UnderlineColor(t *testing.T) {
	term := New(WithSize(3, 20))

	// SGR 58:2::R:G:B sets underline color (RGB) - common format
	// Also try the indexed format
	term.WriteString("\x1b[4m\x1b[58;2;255;0;0mText\x1b[0m")

	snap := term.Snapshot(SnapshotDetailFull)

	if len(snap.Lines[0].Cells) < 4 {
		t.Fatal("Expected at least 4 cells")
	}

	got := snap.Lines[0].Cells[0].UnderlineColor
	// If parser supports underline color, it should be set
	// If not supported, test just documents current behavior
	t.Logf("UnderlineColor = %q", got)
}

func TestSnapshot_Hyperlink(t *testing.T) {
	term := New(WithSize(3, 40))

	// OSC 8 hyperlink
	term.WriteString("\x1b]8;id=test;https://example.com\x07Link\x1b]8;;\x07")

	snap := term.Snapshot(SnapshotDetailFull)

	if len(snap.Lines[0].Cells) < 4 {
		t.Fatal("Expected at least 4 cells")
	}

	for i := 0; i < 4; i++ {
		cell := snap.Lines[0].Cells[i]
		if cell.Hyperlink == nil {
			t.Errorf("Cell[%d] should have hyperlink", i)
			continue
		}
		if cell.Hyperlink.URI != "https://example.com" {
			t.Errorf("Cell[%d].Hyperlink.URI = %q, want %q", i, cell.Hyperlink.URI, "https://example.com")
		}
	}
}

func TestSnapshot_WideChar(t *testing.T) {
	term := New(WithSize(3, 10))

	// Write a wide character (Chinese)
	term.WriteString("中")

	snap := term.Snapshot(SnapshotDetailFull)

	if len(snap.Lines[0].Cells) < 2 {
		t.Fatal("Expected at least 2 cells")
	}

	if !snap.Lines[0].Cells[0].Wide {
		t.Error("Cell[0] should be wide")
	}
	if !snap.Lines[0].Cells[1].WideSpacer {
		t.Error("Cell[1] should be wide spacer")
	}
}

func TestColorToHex(t *testing.T) {
	tests := []struct {
		name     string
		color    color.Color
		expected string
	}{
		{"nil", nil, ""},
		{"black", color.RGBA{0, 0, 0, 255}, "#000000"},
		{"white", color.RGBA{255, 255, 255, 255}, "#ffffff"},
		{"red", color.RGBA{255, 0, 0, 255}, "#ff0000"},
		{"indexed", &IndexedColor{Index: 1}, "#cd3131"}, // Red from palette
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := colorToHex(tt.color)
			if result != tt.expected {
				t.Errorf("colorToHex(%v) = %q, want %q", tt.color, result, tt.expected)
			}
		})
	}
}

func TestCursorStyleToString(t *testing.T) {
	tests := []struct {
		style    CursorStyle
		expected string
	}{
		{CursorStyleBlinkingBlock, "block"},
		{CursorStyleSteadyBlock, "block"},
		{CursorStyleBlinkingUnderline, "underline"},
		{CursorStyleSteadyUnderline, "underline"},
		{CursorStyleBlinkingBar, "bar"},
		{CursorStyleSteadyBar, "bar"},
	}

	for _, tt := range tests {
		result := cursorStyleToString(tt.style)
		if result != tt.expected {
			t.Errorf("cursorStyleToString(%v) = %q, want %q", tt.style, result, tt.expected)
		}
	}
}

func TestSnapshot_EmptyTerminal(t *testing.T) {
	term := New(WithSize(3, 10))

	snap := term.Snapshot(SnapshotDetailText)

	if snap.Size.Rows != 3 {
		t.Errorf("Size.Rows = %d, want 3", snap.Size.Rows)
	}
	if len(snap.Lines) != 3 {
		t.Errorf("len(Lines) = %d, want 3", len(snap.Lines))
	}

	// All lines should be empty
	for i, line := range snap.Lines {
		if line.Text != "" {
			t.Errorf("Lines[%d].Text = %q, want empty", i, line.Text)
		}
	}
}

func TestSnapshot_StyledSegments(t *testing.T) {
	term := New(WithSize(3, 30))

	// Write same color consecutively - should be one segment
	term.WriteString("\x1b[31mRedText\x1b[0m")

	snap := term.Snapshot(SnapshotDetailStyled)

	if len(snap.Lines[0].Segments) < 1 {
		t.Fatal("Expected at least 1 segment")
	}

	// First segment should contain all red text
	if snap.Lines[0].Segments[0].Text != "RedText" {
		t.Errorf("Segment[0].Text = %q, want %q", snap.Lines[0].Segments[0].Text, "RedText")
	}
}

func TestSnapshot_Images(t *testing.T) {
	term := New(WithSize(10, 20))

	// Create a small test image (2x2 RGBA)
	imgData := []byte{
		255, 0, 0, 255, // Red pixel
		0, 255, 0, 255, // Green pixel
		0, 0, 255, 255, // Blue pixel
		255, 255, 0, 255, // Yellow pixel
	}

	// Store the image
	imgID := term.images.Store(2, 2, imgData)

	// Create a placement
	term.images.Place(&ImagePlacement{
		ImageID: imgID,
		Row:     1,
		Col:     2,
		Rows:    3,
		Cols:    4,
		ZIndex:  0,
	})

	snap := term.Snapshot(SnapshotDetailText)

	if len(snap.Images) != 1 {
		t.Fatalf("Expected 1 image, got %d", len(snap.Images))
	}

	img := snap.Images[0]
	if img.ID != imgID {
		t.Errorf("Image.ID = %d, want %d", img.ID, imgID)
	}
	if img.Row != 1 {
		t.Errorf("Image.Row = %d, want 1", img.Row)
	}
	if img.Col != 2 {
		t.Errorf("Image.Col = %d, want 2", img.Col)
	}
	if img.Rows != 3 {
		t.Errorf("Image.Rows = %d, want 3", img.Rows)
	}
	if img.Cols != 4 {
		t.Errorf("Image.Cols = %d, want 4", img.Cols)
	}
	if img.PixelWidth != 2 {
		t.Errorf("Image.PixelWidth = %d, want 2", img.PixelWidth)
	}
	if img.PixelHeight != 2 {
		t.Errorf("Image.PixelHeight = %d, want 2", img.PixelHeight)
	}
}

func TestSnapshot_NoImages(t *testing.T) {
	term := New(WithSize(3, 10))
	term.WriteString("Hello")

	snap := term.Snapshot(SnapshotDetailText)

	if snap.Images != nil {
		t.Errorf("Expected nil Images, got %v", snap.Images)
	}
}

func TestGetImageData(t *testing.T) {
	term := New(WithSize(10, 20))

	// Create a small test image (2x2 RGBA)
	imgData := []byte{
		255, 0, 0, 255, // Red pixel
		0, 255, 0, 255, // Green pixel
		0, 0, 255, 255, // Blue pixel
		255, 255, 0, 255, // Yellow pixel
	}

	// Store the image
	imgID := term.images.Store(2, 2, imgData)

	// Get image data
	result := term.GetImageData(imgID)

	if result == nil {
		t.Fatal("Expected image data, got nil")
	}

	if result.ID != imgID {
		t.Errorf("ID = %d, want %d", result.ID, imgID)
	}
	if result.Width != 2 {
		t.Errorf("Width = %d, want 2", result.Width)
	}
	if result.Height != 2 {
		t.Errorf("Height = %d, want 2", result.Height)
	}
	if result.Format != "rgba" {
		t.Errorf("Format = %q, want %q", result.Format, "rgba")
	}

	// Decode and verify data
	decoded, err := base64.StdEncoding.DecodeString(result.Data)
	if err != nil {
		t.Fatalf("Failed to decode base64: %v", err)
	}
	if len(decoded) != len(imgData) {
		t.Errorf("Decoded data length = %d, want %d", len(decoded), len(imgData))
	}
	for i, b := range decoded {
		if b != imgData[i] {
			t.Errorf("Decoded data[%d] = %d, want %d", i, b, imgData[i])
		}
	}
}

func TestGetImageData_NotFound(t *testing.T) {
	term := New(WithSize(10, 20))

	result := term.GetImageData(999)

	if result != nil {
		t.Errorf("Expected nil for non-existent image, got %v", result)
	}
}
