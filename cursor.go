package headlessterm

// CursorStyle determines how the cursor is rendered.
type CursorStyle int

const (
	CursorStyleBlinkingBlock CursorStyle = iota
	CursorStyleSteadyBlock
	CursorStyleBlinkingUnderline
	CursorStyleSteadyUnderline
	CursorStyleBlinkingBar
	CursorStyleSteadyBar
)

// Cursor tracks the current position and rendering style (0-based coordinates).
type Cursor struct {
	Row     int
	Col     int
	Style   CursorStyle
	Visible bool
}

// NewCursor creates a cursor at (0, 0) with blinking block style, visible.
func NewCursor() *Cursor {
	return &Cursor{
		Row:     0,
		Col:     0,
		Style:   CursorStyleBlinkingBlock,
		Visible: true,
	}
}

// SavedCursor stores cursor position, cell attributes, and charset state for restoration.
// Used when switching between primary and alternate screens.
type SavedCursor struct {
	Row            int
	Col            int
	Attrs          CellTemplate
	OriginMode     bool
	CharsetIndex   int
	Charsets       [4]Charset
}

// CellTemplate defines default attributes applied to newly written characters.
// Modified by SGR (Select Graphic Rendition) escape sequences.
type CellTemplate struct {
	Cell
}

// NewCellTemplate creates a template with default attributes (no colors, no flags).
func NewCellTemplate() CellTemplate {
	return CellTemplate{
		Cell: NewCell(),
	}
}

// Charset selects the character encoding variant.
type Charset int

const (
	CharsetASCII Charset = iota
	CharsetLineDrawing
)

// CharsetIndex selects one of four character set slots (G0-G3).
type CharsetIndex int

const (
	CharsetIndexG0 CharsetIndex = iota
	CharsetIndexG1
	CharsetIndexG2
	CharsetIndexG3
)
