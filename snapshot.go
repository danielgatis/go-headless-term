package headlessterm

import (
	"encoding/base64"
	"fmt"
	"image/color"
)

// SnapshotDetail specifies the level of detail in a snapshot.
type SnapshotDetail string

const (
	// SnapshotDetailText returns plain text only.
	SnapshotDetailText SnapshotDetail = "text"
	// SnapshotDetailStyled returns text with style segments per line.
	SnapshotDetailStyled SnapshotDetail = "styled"
	// SnapshotDetailFull returns full cell-by-cell data.
	SnapshotDetailFull SnapshotDetail = "full"
)

// Snapshot represents a complete terminal screen capture.
type Snapshot struct {
	Size   SnapshotSize    `json:"size"`
	Cursor SnapshotCursor  `json:"cursor"`
	Lines  []SnapshotLine  `json:"lines"`
	Images []SnapshotImage `json:"images,omitempty"`
}

// SnapshotSize holds terminal dimensions.
type SnapshotSize struct {
	Rows int `json:"rows"`
	Cols int `json:"cols"`
}

// SnapshotCursor holds cursor state.
type SnapshotCursor struct {
	Row     int    `json:"row"`
	Col     int    `json:"col"`
	Visible bool   `json:"visible"`
	Style   string `json:"style"`
}

// SnapshotLine represents a single line in the snapshot.
type SnapshotLine struct {
	Text     string            `json:"text"`
	Segments []SnapshotSegment `json:"segments,omitempty"`
	Cells    []SnapshotCell    `json:"cells,omitempty"`
}

// SnapshotSegment represents a styled text segment within a line.
type SnapshotSegment struct {
	Text       string         `json:"text"`
	Fg         string         `json:"fg,omitempty"`
	Bg         string         `json:"bg,omitempty"`
	Attributes SnapshotAttrs  `json:"attrs,omitempty"`
	Hyperlink  *SnapshotLink  `json:"hyperlink,omitempty"`
}

// SnapshotCell represents a single cell with full attributes.
type SnapshotCell struct {
	Char       string         `json:"char"`
	Fg         string         `json:"fg"`
	Bg         string         `json:"bg"`
	Attributes SnapshotAttrs  `json:"attrs,omitempty"`
	Hyperlink  *SnapshotLink  `json:"hyperlink,omitempty"`
	Wide       bool           `json:"wide,omitempty"`
	WideSpacer bool           `json:"wide_spacer,omitempty"`
}

// SnapshotAttrs holds text formatting attributes.
type SnapshotAttrs struct {
	Bold          bool `json:"bold,omitempty"`
	Dim           bool `json:"dim,omitempty"`
	Italic        bool `json:"italic,omitempty"`
	Underline     bool `json:"underline,omitempty"`
	Blink         bool `json:"blink,omitempty"`
	Reverse       bool `json:"reverse,omitempty"`
	Hidden        bool `json:"hidden,omitempty"`
	Strikethrough bool `json:"strikethrough,omitempty"`
}

// SnapshotLink holds hyperlink information.
type SnapshotLink struct {
	ID  string `json:"id,omitempty"`
	URI string `json:"uri"`
}

// SnapshotImage holds image placement metadata (without pixel data).
type SnapshotImage struct {
	ID          uint32 `json:"id"`           // Unique image ID
	PlacementID uint32 `json:"placement_id"` // Unique placement ID
	Row         int    `json:"row"`          // Position row (cells)
	Col         int    `json:"col"`          // Position column (cells)
	Rows        int    `json:"rows"`         // Size in rows (cells)
	Cols        int    `json:"cols"`         // Size in columns (cells)
	PixelWidth  uint32 `json:"pixel_width"`  // Original image width (pixels)
	PixelHeight uint32 `json:"pixel_height"` // Original image height (pixels)
	ZIndex      int32  `json:"z_index"`      // Z-index for layering
}

// ImageSnapshot holds complete image data for retrieval.
type ImageSnapshot struct {
	ID     uint32 `json:"id"`
	Width  uint32 `json:"width"`
	Height uint32 `json:"height"`
	Format string `json:"format"` // "rgba" (raw RGBA pixels, base64 encoded)
	Data   string `json:"data"`   // Base64 encoded image data
}

// GetImageData returns the image data for the given ID, or nil if not found.
func (t *Terminal) GetImageData(id uint32) *ImageSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	img := t.images.Image(id)
	if img == nil {
		return nil
	}

	return &ImageSnapshot{
		ID:     img.ID,
		Width:  img.Width,
		Height: img.Height,
		Format: "rgba",
		Data:   base64.StdEncoding.EncodeToString(img.Data),
	}
}

// Snapshot creates a snapshot of the current terminal state.
// The detail parameter controls how much information is included.
func (t *Terminal) Snapshot(detail SnapshotDetail) *Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	snap := &Snapshot{
		Size: SnapshotSize{
			Rows: t.rows,
			Cols: t.cols,
		},
		Cursor: SnapshotCursor{
			Row:     t.cursor.Row,
			Col:     t.cursor.Col,
			Visible: t.cursor.Visible,
			Style:   cursorStyleToString(t.cursor.Style),
		},
		Lines: make([]SnapshotLine, t.rows),
	}

	for row := 0; row < t.rows; row++ {
		snap.Lines[row] = t.snapshotLine(row, detail)
	}

	// Include image placements
	snap.Images = t.snapshotImages()

	return snap
}

// snapshotImages returns all image placements with metadata.
func (t *Terminal) snapshotImages() []SnapshotImage {
	placements := t.images.Placements()
	if len(placements) == 0 {
		return nil
	}

	images := make([]SnapshotImage, 0, len(placements))
	for _, p := range placements {
		img := t.images.Image(p.ImageID)
		if img == nil {
			continue
		}

		images = append(images, SnapshotImage{
			ID:          p.ImageID,
			PlacementID: p.ID,
			Row:         p.Row,
			Col:         p.Col,
			Rows:        p.Rows,
			Cols:        p.Cols,
			PixelWidth:  img.Width,
			PixelHeight: img.Height,
			ZIndex:      p.ZIndex,
		})
	}

	return images
}

// snapshotLine creates a snapshot of a single line.
func (t *Terminal) snapshotLine(row int, detail SnapshotDetail) SnapshotLine {
	line := SnapshotLine{
		Text: t.activeBuffer.LineContent(row),
	}

	switch detail {
	case SnapshotDetailText:
		// Just text, already set

	case SnapshotDetailStyled:
		line.Segments = t.lineToSegments(row)

	case SnapshotDetailFull:
		line.Cells = t.lineToCells(row)
	}

	return line
}

// lineToSegments converts a line to styled segments (runs of same style).
func (t *Terminal) lineToSegments(row int) []SnapshotSegment {
	var segments []SnapshotSegment
	var current *SnapshotSegment
	var currentChars []rune

	for col := 0; col < t.cols; col++ {
		cell := t.activeBuffer.Cell(row, col)
		if cell == nil {
			continue
		}
		if cell.IsWideSpacer() {
			continue
		}

		fg := colorToHex(cell.Fg)
		bg := colorToHex(cell.Bg)
		attrs := cellAttrsToSnapshot(cell)
		link := cellHyperlinkToSnapshot(cell)

		// Check if we need to start a new segment
		if current == nil || !segmentMatches(current, fg, bg, attrs, link) {
			// Save current segment if exists
			if current != nil && len(currentChars) > 0 {
				current.Text = string(currentChars)
				segments = append(segments, *current)
			}

			// Start new segment
			current = &SnapshotSegment{
				Fg:         fg,
				Bg:         bg,
				Attributes: attrs,
				Hyperlink:  link,
			}
			currentChars = nil
		}

		ch := cell.Char
		if ch == 0 {
			ch = ' '
		}
		currentChars = append(currentChars, ch)
	}

	// Don't forget the last segment
	if current != nil && len(currentChars) > 0 {
		current.Text = string(currentChars)
		segments = append(segments, *current)
	}

	return segments
}

// lineToCells converts a line to full cell data.
func (t *Terminal) lineToCells(row int) []SnapshotCell {
	cells := make([]SnapshotCell, 0, t.cols)

	for col := 0; col < t.cols; col++ {
		cell := t.activeBuffer.Cell(row, col)
		if cell == nil {
			cells = append(cells, SnapshotCell{
				Char: " ",
				Fg:   colorToHex(nil),
				Bg:   colorToHex(nil),
			})
			continue
		}

		ch := cell.Char
		if ch == 0 {
			ch = ' '
		}

		sc := SnapshotCell{
			Char:       string(ch),
			Fg:         colorToHex(cell.Fg),
			Bg:         colorToHex(cell.Bg),
			Attributes: cellAttrsToSnapshot(cell),
			Hyperlink:  cellHyperlinkToSnapshot(cell),
			Wide:       cell.IsWide(),
			WideSpacer: cell.IsWideSpacer(),
		}

		cells = append(cells, sc)
	}

	return cells
}

// segmentMatches checks if segment matches the given style.
func segmentMatches(seg *SnapshotSegment, fg, bg string, attrs SnapshotAttrs, link *SnapshotLink) bool {
	if seg.Fg != fg || seg.Bg != bg {
		return false
	}
	if seg.Attributes != attrs {
		return false
	}
	// Compare hyperlinks
	if seg.Hyperlink == nil && link == nil {
		return true
	}
	if seg.Hyperlink == nil || link == nil {
		return false
	}
	return seg.Hyperlink.URI == link.URI && seg.Hyperlink.ID == link.ID
}

// colorToHex converts a color to hex string.
func colorToHex(c color.Color) string {
	if c == nil {
		return ""
	}

	rgba := resolveDefaultColor(c, true)
	return fmt.Sprintf("#%02x%02x%02x", rgba.R, rgba.G, rgba.B)
}

// cellAttrsToSnapshot extracts cell attributes.
func cellAttrsToSnapshot(cell *Cell) SnapshotAttrs {
	return SnapshotAttrs{
		Bold:          cell.HasFlag(CellFlagBold),
		Dim:           cell.HasFlag(CellFlagDim),
		Italic:        cell.HasFlag(CellFlagItalic),
		Underline:     cell.HasFlag(CellFlagUnderline) || cell.HasFlag(CellFlagDoubleUnderline) || cell.HasFlag(CellFlagCurlyUnderline) || cell.HasFlag(CellFlagDottedUnderline) || cell.HasFlag(CellFlagDashedUnderline),
		Blink:         cell.HasFlag(CellFlagBlinkSlow) || cell.HasFlag(CellFlagBlinkFast),
		Reverse:       cell.HasFlag(CellFlagReverse),
		Hidden:        cell.HasFlag(CellFlagHidden),
		Strikethrough: cell.HasFlag(CellFlagStrike),
	}
}

// cellHyperlinkToSnapshot extracts hyperlink info.
func cellHyperlinkToSnapshot(cell *Cell) *SnapshotLink {
	if cell.Hyperlink == nil {
		return nil
	}
	return &SnapshotLink{
		ID:  cell.Hyperlink.ID,
		URI: cell.Hyperlink.URI,
	}
}

// cursorStyleToString converts cursor style to string.
func cursorStyleToString(style CursorStyle) string {
	switch style {
	case CursorStyleBlinkingBlock, CursorStyleSteadyBlock:
		return "block"
	case CursorStyleBlinkingUnderline, CursorStyleSteadyUnderline:
		return "underline"
	case CursorStyleBlinkingBar, CursorStyleSteadyBar:
		return "bar"
	default:
		return "block"
	}
}
