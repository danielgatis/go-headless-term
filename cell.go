package headlessterm

import "image/color"

// CellFlags is a bitmask of cell rendering attributes.
type CellFlags uint16

const (
	CellFlagBold CellFlags = 1 << iota
	CellFlagDim
	CellFlagItalic
	CellFlagUnderline
	CellFlagDoubleUnderline
	CellFlagCurlyUnderline
	CellFlagDottedUnderline
	CellFlagDashedUnderline
	CellFlagBlinkSlow
	CellFlagBlinkFast
	CellFlagReverse
	CellFlagHidden
	CellFlagStrike
	CellFlagWideChar
	CellFlagWideCharSpacer
	CellFlagDirty
)

// Cell stores the character, colors, and formatting attributes for one grid position.
// Wide characters (2 columns) use a spacer cell in the second position.
type Cell struct {
	Char           rune
	Fg             color.Color
	Bg             color.Color
	UnderlineColor color.Color
	Flags          CellFlags
	Hyperlink      *Hyperlink
	Image          *CellImage // Image reference, nil if no image
}

// Hyperlink associates a cell with a clickable link (OSC 8).
type Hyperlink struct {
	ID  string
	URI string
}

// NewCell creates a cell initialized with space character and default colors.
func NewCell() Cell {
	return Cell{
		Char: ' ',
		Fg:   &NamedColor{Name: NamedColorForeground},
		Bg:   &NamedColor{Name: NamedColorBackground},
	}
}

// Reset clears all attributes and sets the cell to default state (space character, default colors).
func (c *Cell) Reset() {
	c.Char = ' '
	c.Fg = &NamedColor{Name: NamedColorForeground}
	c.Bg = &NamedColor{Name: NamedColorBackground}
	c.UnderlineColor = nil
	c.Flags = 0
	c.Hyperlink = nil
	c.Image = nil
}

// HasFlag returns true if the specified flag is set.
func (c *Cell) HasFlag(flag CellFlags) bool {
	return c.Flags&flag != 0
}

// SetFlag enables the specified flag without affecting others.
func (c *Cell) SetFlag(flag CellFlags) {
	c.Flags |= flag
}

// ClearFlag disables the specified flag without affecting others.
func (c *Cell) ClearFlag(flag CellFlags) {
	c.Flags &^= flag
}

// IsDirty returns true if the cell was modified since the last ClearDirty call.
func (c *Cell) IsDirty() bool {
	return c.HasFlag(CellFlagDirty)
}

// MarkDirty marks the cell as modified for dirty tracking.
func (c *Cell) MarkDirty() {
	c.SetFlag(CellFlagDirty)
}

// ClearDirty resets the dirty tracking flag.
func (c *Cell) ClearDirty() {
	c.ClearFlag(CellFlagDirty)
}

// IsWide returns true if this cell contains a wide character (CJK, emoji, etc.) that occupies 2 columns.
func (c *Cell) IsWide() bool {
	return c.HasFlag(CellFlagWideChar)
}

// IsWideSpacer returns true if this is the second cell of a wide character (should be skipped during rendering).
func (c *Cell) IsWideSpacer() bool {
	return c.HasFlag(CellFlagWideCharSpacer)
}

// Copy returns a deep copy of the cell, including the hyperlink and image pointers.
func (c *Cell) Copy() Cell {
	return Cell{
		Char:           c.Char,
		Fg:             c.Fg,
		Bg:             c.Bg,
		UnderlineColor: c.UnderlineColor,
		Flags:          c.Flags,
		Hyperlink:      c.Hyperlink,
		Image:          c.Image,
	}
}

// HasImage returns true if this cell has an image reference.
func (c *Cell) HasImage() bool {
	return c.Image != nil
}
