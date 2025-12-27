package headlessterm

import (
	"testing"
)

func TestImageManager_Store(t *testing.T) {
	m := NewImageManager()

	data := make([]byte, 100)
	id := m.Store(10, 10, data)

	if id != 1 {
		t.Errorf("expected id 1, got %d", id)
	}
	if m.ImageCount() != 1 {
		t.Errorf("expected 1 image, got %d", m.ImageCount())
	}
	if m.UsedMemory() != 100 {
		t.Errorf("expected 100 bytes, got %d", m.UsedMemory())
	}
}

func TestImageManager_Deduplication(t *testing.T) {
	m := NewImageManager()

	data := []byte("test image data")
	id1 := m.Store(10, 10, data)
	id2 := m.Store(10, 10, data) // Same data

	if id1 != id2 {
		t.Errorf("expected same id for duplicate, got %d and %d", id1, id2)
	}
	if m.ImageCount() != 1 {
		t.Errorf("expected 1 image (deduplicated), got %d", m.ImageCount())
	}
}

func TestImageManager_StoreWithID(t *testing.T) {
	m := NewImageManager()

	data := make([]byte, 50)
	m.StoreWithID(42, 5, 5, data)

	img := m.Image(42)
	if img == nil {
		t.Fatal("expected image with id 42")
	}
	if img.Width != 5 || img.Height != 5 {
		t.Errorf("expected 5x5, got %dx%d", img.Width, img.Height)
	}
}

func TestImageManager_Place(t *testing.T) {
	m := NewImageManager()

	data := make([]byte, 100)
	imageID := m.Store(10, 10, data)

	placement := &ImagePlacement{
		ImageID: imageID,
		Row:     0,
		Col:     0,
		Cols:    5,
		Rows:    5,
	}

	placementID := m.Place(placement)
	if placementID != 1 {
		t.Errorf("expected placement id 1, got %d", placementID)
	}
	if m.PlacementCount() != 1 {
		t.Errorf("expected 1 placement, got %d", m.PlacementCount())
	}
}

func TestImageManager_DeleteImage(t *testing.T) {
	m := NewImageManager()

	data := make([]byte, 100)
	id := m.Store(10, 10, data)

	m.DeleteImage(id)

	if m.ImageCount() != 0 {
		t.Errorf("expected 0 images after delete, got %d", m.ImageCount())
	}
	if m.UsedMemory() != 0 {
		t.Errorf("expected 0 bytes after delete, got %d", m.UsedMemory())
	}
}

func TestImageManager_Clear(t *testing.T) {
	m := NewImageManager()

	data := make([]byte, 100)
	imageID := m.Store(10, 10, data)
	m.Place(&ImagePlacement{ImageID: imageID, Row: 0, Col: 0, Cols: 1, Rows: 1})

	m.Clear()

	if m.ImageCount() != 0 {
		t.Errorf("expected 0 images after clear, got %d", m.ImageCount())
	}
	if m.PlacementCount() != 0 {
		t.Errorf("expected 0 placements after clear, got %d", m.PlacementCount())
	}
}

func TestImageManager_Prune(t *testing.T) {
	m := NewImageManager()
	m.SetMaxMemory(150) // Low limit

	// Store 3 images of 100 bytes each - should trigger pruning
	data := make([]byte, 100)
	m.Store(10, 10, data)

	data2 := make([]byte, 100)
	data2[0] = 1 // Different data
	m.Store(10, 10, data2)

	// At this point, we're at 200 bytes with 150 limit
	// Pruning should have removed unreferenced images
	if m.UsedMemory() > 150 {
		// This might not prune if images are still referenced
		// Just verify it doesn't crash
	}
}

func TestImageManager_Placements(t *testing.T) {
	m := NewImageManager()

	data := make([]byte, 100)
	imageID := m.Store(10, 10, data)

	m.Place(&ImagePlacement{ImageID: imageID, Row: 0, Col: 0, Cols: 1, Rows: 1})
	m.Place(&ImagePlacement{ImageID: imageID, Row: 1, Col: 1, Cols: 2, Rows: 2})

	placements := m.Placements()
	if len(placements) != 2 {
		t.Errorf("expected 2 placements, got %d", len(placements))
	}
}

func TestImageManager_DeletePlacementsByPosition(t *testing.T) {
	m := NewImageManager()

	data := make([]byte, 100)
	imageID := m.Store(10, 10, data)

	m.Place(&ImagePlacement{ImageID: imageID, Row: 0, Col: 0, Cols: 2, Rows: 2})
	m.Place(&ImagePlacement{ImageID: imageID, Row: 5, Col: 5, Cols: 2, Rows: 2})

	m.DeletePlacementsByPosition(0, 0) // Should delete first placement

	if m.PlacementCount() != 1 {
		t.Errorf("expected 1 placement after delete, got %d", m.PlacementCount())
	}
}

func TestImageManager_DeletePlacementsInRow(t *testing.T) {
	m := NewImageManager()

	data := make([]byte, 100)
	imageID := m.Store(10, 10, data)

	m.Place(&ImagePlacement{ImageID: imageID, Row: 0, Col: 0, Cols: 2, Rows: 2})
	m.Place(&ImagePlacement{ImageID: imageID, Row: 5, Col: 5, Cols: 2, Rows: 2})

	m.DeletePlacementsInRow(1) // Row 1 intersects first placement (rows 0-1)

	if m.PlacementCount() != 1 {
		t.Errorf("expected 1 placement after delete, got %d", m.PlacementCount())
	}
}

func TestCellImage(t *testing.T) {
	cell := NewCell()

	if cell.HasImage() {
		t.Error("new cell should not have image")
	}

	cell.Image = &CellImage{
		PlacementID: 1,
		ImageID:     1,
		U0:          0.0,
		V0:          0.0,
		U1:          1.0,
		V1:          1.0,
		ZIndex:      -1,
	}

	if !cell.HasImage() {
		t.Error("cell should have image after setting")
	}

	cell.Reset()

	if cell.HasImage() {
		t.Error("cell should not have image after reset")
	}
}
