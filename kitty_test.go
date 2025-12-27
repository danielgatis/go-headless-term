package headlessterm

import (
	"encoding/base64"
	"testing"
)

func TestParseKittyGraphics_Basic(t *testing.T) {
	// Simple transmit and display command
	data := []byte("Ga=T,f=32,s=2,v=2;AAAAAAAAAAAAAAAAAAAAAAA=")
	cmd, err := ParseKittyGraphics(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Action != KittyActionTransmitDisplay {
		t.Errorf("expected action T, got %c", cmd.Action)
	}
	if cmd.Format != KittyFormatRGBA {
		t.Errorf("expected format 32, got %d", cmd.Format)
	}
	if cmd.Width != 2 {
		t.Errorf("expected width 2, got %d", cmd.Width)
	}
	if cmd.Height != 2 {
		t.Errorf("expected height 2, got %d", cmd.Height)
	}
}

func TestParseKittyGraphics_Query(t *testing.T) {
	data := []byte("Ga=q,i=1;")
	cmd, err := ParseKittyGraphics(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Action != KittyActionQuery {
		t.Errorf("expected action q, got %c", cmd.Action)
	}
	if cmd.ImageID != 1 {
		t.Errorf("expected image ID 1, got %d", cmd.ImageID)
	}
}

func TestParseKittyGraphics_Delete(t *testing.T) {
	data := []byte("Ga=d,d=a;")
	cmd, err := ParseKittyGraphics(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Action != KittyActionDelete {
		t.Errorf("expected action d, got %c", cmd.Action)
	}
	if cmd.Delete != KittyDeleteAll {
		t.Errorf("expected delete all, got %c", cmd.Delete)
	}
}

func TestParseKittyGraphics_Chunked(t *testing.T) {
	data := []byte("Ga=T,m=1;AAAA")
	cmd, err := ParseKittyGraphics(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmd.More {
		t.Error("expected more=true")
	}
}

func TestParseKittyGraphics_WithZIndex(t *testing.T) {
	data := []byte("Ga=p,i=1,z=-1;")
	cmd, err := ParseKittyGraphics(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.ZIndex != -1 {
		t.Errorf("expected z-index -1, got %d", cmd.ZIndex)
	}
}

func TestParseKittyGraphics_Placement(t *testing.T) {
	data := []byte("Ga=p,i=1,c=10,r=5,X=2,Y=3;")
	cmd, err := ParseKittyGraphics(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cmd.Cols != 10 {
		t.Errorf("expected cols 10, got %d", cmd.Cols)
	}
	if cmd.Rows != 5 {
		t.Errorf("expected rows 5, got %d", cmd.Rows)
	}
	if cmd.CellOffsetX != 2 {
		t.Errorf("expected offsetX 2, got %d", cmd.CellOffsetX)
	}
	if cmd.CellOffsetY != 3 {
		t.Errorf("expected offsetY 3, got %d", cmd.CellOffsetY)
	}
}

func TestParseKittyGraphics_DoNotMoveCursor(t *testing.T) {
	data := []byte("Ga=T,C=1;")
	cmd, err := ParseKittyGraphics(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cmd.DoNotMoveCursor {
		t.Error("expected DoNotMoveCursor=true")
	}
}

func TestKittyCommand_DecodeRGBA(t *testing.T) {
	// 2x2 RGBA image (16 bytes)
	rgba := make([]byte, 16)
	for i := range rgba {
		rgba[i] = 255
	}
	payload := base64.StdEncoding.EncodeToString(rgba)

	cmd := &KittyCommand{
		Format:  KittyFormatRGBA,
		Width:   2,
		Height:  2,
		Payload: rgba,
	}

	data, w, h, err := cmd.DecodeImageData()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 2 || h != 2 {
		t.Errorf("expected 2x2, got %dx%d", w, h)
	}
	if len(data) != 16 {
		t.Errorf("expected 16 bytes, got %d", len(data))
	}
	_ = payload // just to use the variable
}

func TestKittyCommand_DecodeRGB(t *testing.T) {
	// 2x2 RGB image (12 bytes) -> converted to RGBA (16 bytes)
	rgb := make([]byte, 12)
	for i := range rgb {
		rgb[i] = 128
	}

	cmd := &KittyCommand{
		Format:  KittyFormatRGB,
		Width:   2,
		Height:  2,
		Payload: rgb,
	}

	data, w, h, err := cmd.DecodeImageData()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if w != 2 || h != 2 {
		t.Errorf("expected 2x2, got %dx%d", w, h)
	}
	if len(data) != 16 {
		t.Errorf("expected 16 bytes RGBA, got %d", len(data))
	}
	// Check alpha is 255
	if data[3] != 255 {
		t.Errorf("expected alpha 255, got %d", data[3])
	}
}

func TestFormatKittyResponse(t *testing.T) {
	resp := FormatKittyResponse(42, "", false)
	expected := "\x1b_Gi=42;OK\x1b\\"
	if resp != expected {
		t.Errorf("expected %q, got %q", expected, resp)
	}

	respErr := FormatKittyResponse(0, "ENOENT", true)
	expectedErr := "\x1b_G;ENOENT\x1b\\"
	if respErr != expectedErr {
		t.Errorf("expected %q, got %q", expectedErr, respErr)
	}
}

// TestKittyImageDisplay tests end-to-end image display via terminal
func TestKittyImageDisplay(t *testing.T) {
	// Create terminal with known cell size
	term := New(WithSize(24, 80))

	// Create a 2x2 RGBA image (16 bytes) with all white pixels
	rgba := make([]byte, 16)
	for i := range rgba {
		rgba[i] = 255
	}
	payload := base64.StdEncoding.EncodeToString(rgba)

	// Send Kitty graphics command via APC sequence
	// a=T (transmit and display), f=32 (RGBA), s=2 (width), v=2 (height)
	apc := "\x1b_Ga=T,f=32,s=2,v=2;" + payload + "\x1b\\"
	term.WriteString(apc)

	// Verify image was stored
	if term.ImageCount() != 1 {
		t.Errorf("expected 1 image, got %d", term.ImageCount())
	}

	// Verify placement was created
	if term.ImagePlacementCount() != 1 {
		t.Errorf("expected 1 placement, got %d", term.ImagePlacementCount())
	}
}

// TestKittyImageCellAssignment tests that cells get image references
func TestKittyImageCellAssignment(t *testing.T) {
	term := New(WithSize(24, 80))

	// Set cell size for calculations (20x20 pixels per cell)
	term.SetSizeProvider(&testSizeProvider{cellW: 20, cellH: 20})

	// Create a 40x40 RGBA image (should cover 2x2 cells)
	width, height := 40, 40
	rgba := make([]byte, width*height*4)
	for i := range rgba {
		rgba[i] = 128
	}
	payload := base64.StdEncoding.EncodeToString(rgba)

	// Position cursor at row 5, col 10
	term.WriteString("\x1b[6;11H") // 1-based positioning

	// Transmit and display
	apc := "\x1b_Ga=T,f=32,s=40,v=40;" + payload + "\x1b\\"
	term.WriteString(apc)

	// Check cells have image references
	for row := 5; row < 7; row++ {
		for col := 10; col < 12; col++ {
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

// TestKittyImageUVCoordinates tests that UV coordinates are calculated correctly
func TestKittyImageUVCoordinates(t *testing.T) {
	term := New(WithSize(24, 80))

	// Set cell size (10x10 pixels per cell)
	term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})

	// Create a 20x20 RGBA image (should cover 2x2 cells exactly)
	width, height := 20, 20
	rgba := make([]byte, width*height*4)
	payload := base64.StdEncoding.EncodeToString(rgba)

	// Position cursor at origin
	term.WriteString("\x1b[1;1H")

	// Transmit and display
	apc := "\x1b_Ga=T,f=32,s=20,v=20;" + payload + "\x1b\\"
	term.WriteString(apc)

	// Check UV coordinates for each cell
	testCases := []struct {
		row, col        int
		u0, v0, u1, v1  float32
	}{
		{0, 0, 0.0, 0.0, 0.5, 0.5}, // Top-left
		{0, 1, 0.5, 0.0, 1.0, 0.5}, // Top-right
		{1, 0, 0.0, 0.5, 0.5, 1.0}, // Bottom-left
		{1, 1, 0.5, 0.5, 1.0, 1.0}, // Bottom-right
	}

	for _, tc := range testCases {
		cell := term.Cell(tc.row, tc.col)
		if cell == nil || cell.Image == nil {
			t.Fatalf("cell at %d,%d has no image", tc.row, tc.col)
		}

		img := cell.Image
		if !floatClose(img.U0, tc.u0) || !floatClose(img.V0, tc.v0) ||
			!floatClose(img.U1, tc.u1) || !floatClose(img.V1, tc.v1) {
			t.Errorf("cell %d,%d: expected UV (%v,%v)-(%v,%v), got (%v,%v)-(%v,%v)",
				tc.row, tc.col, tc.u0, tc.v0, tc.u1, tc.v1,
				img.U0, img.V0, img.U1, img.V1)
		}
	}
}

// TestKittyChunkedTransfer tests multi-chunk image transmission
func TestKittyChunkedTransfer(t *testing.T) {
	term := New(WithSize(24, 80))
	term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})

	// Create a 20x20 RGBA image
	width, height := 20, 20
	rgba := make([]byte, width*height*4)
	for i := range rgba {
		rgba[i] = uint8(i % 256)
	}

	// Split into chunks
	chunk1 := rgba[:800]
	chunk2 := rgba[800:]
	payload1 := base64.StdEncoding.EncodeToString(chunk1)
	payload2 := base64.StdEncoding.EncodeToString(chunk2)

	// First chunk with m=1 (more coming)
	apc1 := "\x1b_Ga=T,f=32,s=20,v=20,m=1;" + payload1 + "\x1b\\"
	term.WriteString(apc1)

	// No image yet (incomplete)
	if term.ImageCount() != 0 {
		t.Errorf("expected 0 images during chunked transfer, got %d", term.ImageCount())
	}

	// Final chunk with m=0 (or no m)
	apc2 := "\x1b_Gm=0;" + payload2 + "\x1b\\"
	term.WriteString(apc2)

	// Now image should be complete
	if term.ImageCount() != 1 {
		t.Errorf("expected 1 image after chunked transfer, got %d", term.ImageCount())
	}
}

// TestKittyImageDelete tests image deletion
func TestKittyImageDelete(t *testing.T) {
	term := New(WithSize(24, 80))
	term.SetSizeProvider(&testSizeProvider{cellW: 10, cellH: 10})

	// Create and display an image
	rgba := make([]byte, 100*4)
	payload := base64.StdEncoding.EncodeToString(rgba)
	apc := "\x1b_Ga=T,f=32,s=10,v=10,i=42;" + payload + "\x1b\\"
	term.WriteString(apc)

	if term.ImageCount() != 1 {
		t.Fatalf("expected 1 image, got %d", term.ImageCount())
	}

	// Delete all visible placements
	term.WriteString("\x1b_Ga=d,d=a;\x1b\\")

	// Placements should be removed
	if term.ImagePlacementCount() != 0 {
		t.Errorf("expected 0 placements after delete, got %d", term.ImagePlacementCount())
	}

	// Image data should still exist (d=a only deletes placements)
	if term.ImageCount() != 1 {
		t.Errorf("expected 1 image after placement delete, got %d", term.ImageCount())
	}

	// Delete image data too
	term.WriteString("\x1b_Ga=d,d=I,i=42;\x1b\\")

	if term.ImageCount() != 0 {
		t.Errorf("expected 0 images after data delete, got %d", term.ImageCount())
	}
}

// testSizeProvider is a test implementation of SizeProvider
type testSizeProvider struct {
	cellW, cellH int
}

func (p *testSizeProvider) CellSizePixels() (width, height int) {
	return p.cellW, p.cellH
}

func (p *testSizeProvider) WindowSizePixels() (width, height int) {
	return 800, 600
}

// floatClose checks if two floats are approximately equal
func floatClose(a, b float32) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < 0.01
}
