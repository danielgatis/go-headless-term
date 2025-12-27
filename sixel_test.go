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
