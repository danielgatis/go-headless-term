package headlessterm

import (
	"crypto/sha256"
	"sync"
	"time"
)

// ImageFormat represents the format of image data.
type ImageFormat uint8

const (
	ImageFormatRGBA ImageFormat = iota // 32-bit RGBA (4 bytes per pixel)
	ImageFormatRGB                     // 24-bit RGB (3 bytes per pixel)
	ImageFormatPNG                     // PNG encoded
)

// ImageData stores decoded image pixels and metadata.
type ImageData struct {
	ID        uint32      // Unique image ID
	Width     uint32      // Image width in pixels
	Height    uint32      // Image height in pixels
	Data      []byte      // RGBA pixel data (always converted to RGBA internally)
	Hash      [32]byte    // SHA-256 hash for deduplication
	CreatedAt time.Time   // For LRU eviction
	AccessedAt time.Time  // Last access time
}

// ImagePlacement represents a displayed instance of an image.
type ImagePlacement struct {
	ID          uint32 // Unique placement ID
	ImageID     uint32 // Reference to ImageData

	// Position in terminal (cell coordinates)
	Row, Col    int

	// Size in cells
	Cols, Rows  int

	// Source region (crop from original image)
	SrcX, SrcY  uint32
	SrcW, SrcH  uint32

	// Z-index for layering (-1 = behind text, 0+ = in front)
	ZIndex      int32

	// Sub-cell offset in pixels
	OffsetX, OffsetY uint32
}

// CellImage is a lightweight reference stored in each Cell.
// It contains UV coordinates for rendering the correct slice of the image.
type CellImage struct {
	PlacementID uint32  // Reference to ImagePlacement
	ImageID     uint32  // Direct reference to ImageData for quick lookup

	// Normalized texture coordinates (0.0 - 1.0)
	U0, V0      float32 // Top-left corner
	U1, V1      float32 // Bottom-right corner

	// Z-index for render ordering
	ZIndex      int32
}

// ImageManager handles storage, placement, and lifecycle of terminal images.
type ImageManager struct {
	mu sync.RWMutex

	images      map[uint32]*ImageData      // ID -> image data
	placements  map[uint32]*ImagePlacement // PlacementID -> placement
	hashToID    map[[32]byte]uint32        // Hash -> ID for deduplication

	nextImageID     uint32
	nextPlacementID uint32

	// Memory management
	maxMemory   int64 // Budget in bytes (default 320MB)
	usedMemory  int64

	// Kitty protocol state
	accumulator            []byte      // For chunked transfers
	accumulatorID          uint32      // Image ID for current accumulation
	accumulatorMore        bool        // More chunks expected
	accumulatorFormat      KittyFormat // Format from first chunk
	accumulatorWidth       uint32      // Width from first chunk
	accumulatorHeight      uint32      // Height from first chunk
	accumulatorCompression byte        // Compression from first chunk
}

// NewImageManager creates a new ImageManager with default settings.
func NewImageManager() *ImageManager {
	return &ImageManager{
		images:     make(map[uint32]*ImageData),
		placements: make(map[uint32]*ImagePlacement),
		hashToID:   make(map[[32]byte]uint32),
		maxMemory:  320 * 1024 * 1024, // 320MB default
	}
}

// SetMaxMemory sets the maximum memory budget for images.
func (m *ImageManager) SetMaxMemory(bytes int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxMemory = bytes
}

// Store adds image data and returns its ID.
// If an identical image exists (same hash), returns the existing ID.
func (m *ImageManager) Store(width, height uint32, data []byte) uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Calculate hash for deduplication
	hash := sha256.Sum256(data)

	// Check for duplicate
	if existingID, ok := m.hashToID[hash]; ok {
		if img, ok := m.images[existingID]; ok {
			img.AccessedAt = time.Now()
			return existingID
		}
	}

	// Allocate new ID
	m.nextImageID++
	id := m.nextImageID

	now := time.Now()
	img := &ImageData{
		ID:         id,
		Width:      width,
		Height:     height,
		Data:       data,
		Hash:       hash,
		CreatedAt:  now,
		AccessedAt: now,
	}

	m.images[id] = img
	m.hashToID[hash] = id
	m.usedMemory += int64(len(data))

	// Prune if over budget
	if m.usedMemory > m.maxMemory {
		m.pruneLocked()
	}

	return id
}

// StoreWithID adds image data with a specific ID (used by Kitty protocol).
func (m *ImageManager) StoreWithID(id, width, height uint32, data []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	hash := sha256.Sum256(data)

	// Remove old image with same ID if exists
	if old, ok := m.images[id]; ok {
		m.usedMemory -= int64(len(old.Data))
		delete(m.hashToID, old.Hash)
	}

	now := time.Now()
	img := &ImageData{
		ID:         id,
		Width:      width,
		Height:     height,
		Data:       data,
		Hash:       hash,
		CreatedAt:  now,
		AccessedAt: now,
	}

	m.images[id] = img
	m.hashToID[hash] = id
	m.usedMemory += int64(len(data))

	if id >= m.nextImageID {
		m.nextImageID = id + 1
	}

	if m.usedMemory > m.maxMemory {
		m.pruneLocked()
	}
}

// Image returns the image data for the given ID, or nil if not found.
func (m *ImageManager) Image(id uint32) *ImageData {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if img, ok := m.images[id]; ok {
		img.AccessedAt = time.Now()
		return img
	}
	return nil
}

// Place creates a new placement and returns its ID.
func (m *ImageManager) Place(p *ImagePlacement) uint32 {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.nextPlacementID++
	p.ID = m.nextPlacementID
	m.placements[p.ID] = p

	return p.ID
}

// Placement returns the placement for the given ID, or nil if not found.
func (m *ImageManager) Placement(id uint32) *ImagePlacement {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.placements[id]
}

// Placements returns all current placements.
func (m *ImageManager) Placements() []*ImagePlacement {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*ImagePlacement, 0, len(m.placements))
	for _, p := range m.placements {
		result = append(result, p)
	}
	return result
}

// RemovePlacement removes a placement by ID.
func (m *ImageManager) RemovePlacement(id uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.placements, id)
}

// RemovePlacementsForImage removes all placements for a given image ID.
func (m *ImageManager) RemovePlacementsForImage(imageID uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, p := range m.placements {
		if p.ImageID == imageID {
			delete(m.placements, id)
		}
	}
}

// DeleteImage removes an image and all its placements.
func (m *ImageManager) DeleteImage(id uint32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if img, ok := m.images[id]; ok {
		m.usedMemory -= int64(len(img.Data))
		delete(m.hashToID, img.Hash)
		delete(m.images, id)
	}

	// Remove associated placements
	for pid, p := range m.placements {
		if p.ImageID == id {
			delete(m.placements, pid)
		}
	}
}

// Clear removes all images and placements.
func (m *ImageManager) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.images = make(map[uint32]*ImageData)
	m.placements = make(map[uint32]*ImagePlacement)
	m.hashToID = make(map[[32]byte]uint32)
	m.usedMemory = 0
	m.accumulator = nil
}

// UsedMemory returns the current memory usage in bytes.
func (m *ImageManager) UsedMemory() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.usedMemory
}

// ImageCount returns the number of stored images.
func (m *ImageManager) ImageCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.images)
}

// PlacementCount returns the number of active placements.
func (m *ImageManager) PlacementCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.placements)
}

// pruneLocked removes least recently used images until under budget.
// Must be called with lock held.
func (m *ImageManager) pruneLocked() {
	// Find images not referenced by any placement
	referenced := make(map[uint32]bool)
	for _, p := range m.placements {
		referenced[p.ImageID] = true
	}

	// Collect unreferenced images sorted by access time
	type candidate struct {
		id   uint32
		time time.Time
		size int64
	}
	var candidates []candidate

	for id, img := range m.images {
		if !referenced[id] {
			candidates = append(candidates, candidate{id, img.AccessedAt, int64(len(img.Data))})
		}
	}

	// Sort by access time (oldest first)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if candidates[j].time.Before(candidates[i].time) {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	// Remove until under budget
	for _, c := range candidates {
		if m.usedMemory <= m.maxMemory {
			break
		}
		if img, ok := m.images[c.id]; ok {
			delete(m.hashToID, img.Hash)
			delete(m.images, c.id)
			m.usedMemory -= c.size
		}
	}
}

// DeletePlacementsByPosition removes placements that overlap a given cell position.
func (m *ImageManager) DeletePlacementsByPosition(row, col int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, p := range m.placements {
		if row >= p.Row && row < p.Row+p.Rows &&
			col >= p.Col && col < p.Col+p.Cols {
			delete(m.placements, id)
		}
	}
}

// DeletePlacementsByZIndex removes placements with a specific z-index.
func (m *ImageManager) DeletePlacementsByZIndex(z int32) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, p := range m.placements {
		if p.ZIndex == z {
			delete(m.placements, id)
		}
	}
}

// DeletePlacementsInRow removes all placements that intersect a given row.
func (m *ImageManager) DeletePlacementsInRow(row int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, p := range m.placements {
		if row >= p.Row && row < p.Row+p.Rows {
			delete(m.placements, id)
		}
	}
}

// DeletePlacementsInColumn removes all placements that intersect a given column.
func (m *ImageManager) DeletePlacementsInColumn(col int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, p := range m.placements {
		if col >= p.Col && col < p.Col+p.Cols {
			delete(m.placements, id)
		}
	}
}
