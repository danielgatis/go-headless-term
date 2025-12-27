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
