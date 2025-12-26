package headlessterm

import (
	"image"
	"image/color"
	"io"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

// FontFinder locates font files by name (useful for avoiding font library dependencies).
type FontFinder interface {
	// Find returns the filesystem path to a font file matching the given name.
	Find(name string) (string, error)
}

// ScreenshotConfig controls how the terminal is rendered to an image.
type ScreenshotConfig struct {
	// Font face to use for rendering. If nil and FontName is empty, uses basicfont.Face7x13.
	Font font.Face

	// FontFinder is used to find fonts by name. Optional.
	FontFinder FontFinder

	// FontName is the font name to find using FontFinder.
	FontName string

	// FontSize is the font size when using FontFinder. Default 14.
	FontSize float64

	// CellWidth and CellHeight override the cell dimensions.
	// If zero, derived from font metrics.
	CellWidth  int
	CellHeight int

	// Palette is the 256-color palette. If nil, uses DefaultPalette.
	Palette *[256]color.RGBA

	// DefaultFG is the default foreground color. If nil, uses DefaultForeground.
	DefaultFG *color.RGBA

	// DefaultBG is the default background color. If nil, uses DefaultBackground.
	DefaultBG *color.RGBA

	// CursorColor is the cursor color. If nil, uses inverted colors.
	CursorColor *color.RGBA

	// ShowCursor controls whether to render the cursor. Default true.
	ShowCursor *bool
}

// LoadFont loads a TrueType or OpenType font from a file path.
func LoadFont(path string, size float64) (font.Face, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return LoadFontFromReader(f, size)
}

// LoadFontFromReader loads a TrueType or OpenType font from an io.Reader.
func LoadFontFromReader(r io.Reader, size float64) (font.Face, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return LoadFontFromBytes(data, size)
}

// LoadFontFromBytes loads a TrueType or OpenType font from raw bytes.
func LoadFontFromBytes(data []byte, size float64) (font.Face, error) {
	ft, err := opentype.Parse(data)
	if err != nil {
		return nil, err
	}

	face, err := opentype.NewFace(ft, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}

	return face, nil
}

// Screenshot renders the terminal to an RGBA image using default settings (basicfont, default palette).
func (t *Terminal) Screenshot() *image.RGBA {
	return t.ScreenshotWithConfig(&ScreenshotConfig{})
}

// ScreenshotWithConfig renders the terminal to an RGBA image with custom font, colors, and cursor settings.
func (t *Terminal) ScreenshotWithConfig(cfg *ScreenshotConfig) *image.RGBA {
	t.mu.RLock()
	defer t.mu.RUnlock()

	face := cfg.Font
	if face == nil && cfg.FontFinder != nil && cfg.FontName != "" {
		// Use FontFinder to load font by name
		size := cfg.FontSize
		if size == 0 {
			size = 14
		}
		if path, err := cfg.FontFinder.Find(cfg.FontName); err == nil {
			if loadedFace, err := LoadFont(path, size); err == nil {
				face = loadedFace
			}
		}
	}
	if face == nil {
		face = basicfont.Face7x13
	}

	cellWidth := cfg.CellWidth
	cellHeight := cfg.CellHeight
	if cellWidth == 0 || cellHeight == 0 {
		metrics := face.Metrics()
		if cellWidth == 0 {
			// Measure a character to get width
			adv, _ := face.GlyphAdvance('M')
			cellWidth = adv.Ceil()
			if cellWidth == 0 {
				cellWidth = 7 // fallback for basicfont
			}
		}
		if cellHeight == 0 {
			cellHeight = metrics.Height.Ceil()
		}
	}

	palette := cfg.Palette
	if palette == nil {
		palette = &DefaultPalette
	}

	defaultFG := cfg.DefaultFG
	if defaultFG == nil {
		defaultFG = &DefaultForeground
	}

	defaultBG := cfg.DefaultBG
	if defaultBG == nil {
		defaultBG = &DefaultBackground
	}

	showCursor := true
	if cfg.ShowCursor != nil {
		showCursor = *cfg.ShowCursor
	}

	// Create image
	imgWidth := t.cols * cellWidth
	imgHeight := t.rows * cellHeight
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// Fill background
	for y := 0; y < imgHeight; y++ {
		for x := 0; x < imgWidth; x++ {
			img.Set(x, y, defaultBG)
		}
	}

	// Render each cell
	for row := 0; row < t.rows; row++ {
		for col := 0; col < t.cols; col++ {
			cell := t.activeBuffer.Cell(row, col)
			if cell == nil || cell.IsWideSpacer() {
				continue
			}

			x := col * cellWidth
			y := row * cellHeight

			// Get colors using custom palette
			fg := resolveColorWithPalette(cell.Fg, true, palette, defaultFG, defaultBG)
			bg := resolveColorWithPalette(cell.Bg, false, palette, defaultFG, defaultBG)

			// Handle reverse video
			if cell.HasFlag(CellFlagReverse) {
				fg, bg = bg, fg
			}

			// Handle dim
			if cell.HasFlag(CellFlagDim) {
				fg = color.RGBA{
					R: uint8(float64(fg.R) * 0.66),
					G: uint8(float64(fg.G) * 0.66),
					B: uint8(float64(fg.B) * 0.66),
					A: fg.A,
				}
			}

			// Fill cell background
			for py := 0; py < cellHeight; py++ {
				for px := 0; px < cellWidth; px++ {
					img.Set(x+px, y+py, bg)
				}
			}

			// Draw character
			ch := cell.Char
			if ch == 0 || ch == ' ' {
				continue
			}

			// Calculate baseline
			metrics := face.Metrics()
			baseline := y + metrics.Ascent.Ceil()

			// Draw the character
			d := &font.Drawer{
				Dst:  img,
				Src:  image.NewUniform(fg),
				Face: face,
				Dot:  fixed.P(x, baseline),
			}
			d.DrawString(string(ch))

			// Handle underline
			if cell.HasFlag(CellFlagUnderline) || cell.HasFlag(CellFlagDoubleUnderline) ||
				cell.HasFlag(CellFlagCurlyUnderline) || cell.HasFlag(CellFlagDottedUnderline) ||
				cell.HasFlag(CellFlagDashedUnderline) {
				underlineColor := fg
				if cell.UnderlineColor != nil {
					underlineColor = resolveColorWithPalette(cell.UnderlineColor, true, palette, defaultFG, defaultBG)
				}
				underlineY := baseline + 2
				for px := 0; px < cellWidth; px++ {
					if underlineY < imgHeight {
						img.Set(x+px, underlineY, underlineColor)
					}
				}
			}

			// Handle strikethrough
			if cell.HasFlag(CellFlagStrike) {
				strikeY := y + cellHeight/2
				for px := 0; px < cellWidth; px++ {
					img.Set(x+px, strikeY, fg)
				}
			}
		}
	}

	// Draw cursor if visible
	if showCursor && t.cursor.Visible {
		cursorX := t.cursor.Col * cellWidth
		cursorY := t.cursor.Row * cellHeight

		if cfg.CursorColor != nil {
			// Use specified cursor color
			for py := 0; py < cellHeight; py++ {
				for px := 0; px < cellWidth; px++ {
					cx, cy := cursorX+px, cursorY+py
					if cx < imgWidth && cy < imgHeight {
						img.Set(cx, cy, cfg.CursorColor)
					}
				}
			}
		} else {
			// Invert colors
			for py := 0; py < cellHeight; py++ {
				for px := 0; px < cellWidth; px++ {
					cx, cy := cursorX+px, cursorY+py
					if cx < imgWidth && cy < imgHeight {
						existing := img.RGBAAt(cx, cy)
						inverted := color.RGBA{
							R: 255 - existing.R,
							G: 255 - existing.G,
							B: 255 - existing.B,
							A: 255,
						}
						img.Set(cx, cy, inverted)
					}
				}
			}
		}
	}

	return img
}

// resolveColorWithPalette resolves a color using a custom palette.
func resolveColorWithPalette(c color.Color, fg bool, palette *[256]color.RGBA, defaultFG, defaultBG *color.RGBA) color.RGBA {
	if c == nil {
		if fg {
			return *defaultFG
		}
		return *defaultBG
	}

	switch v := c.(type) {
	case color.RGBA:
		return v
	case *IndexedColor:
		if v.Index >= 0 && v.Index < 256 {
			return palette[v.Index]
		}
		if fg {
			return *defaultFG
		}
		return *defaultBG
	case *NamedColor:
		return resolveNamedColorWithPalette(v.Name, fg, palette, defaultFG, defaultBG)
	default:
		r, g, b, a := c.RGBA()
		return color.RGBA{
			R: uint8(r >> 8),
			G: uint8(g >> 8),
			B: uint8(b >> 8),
			A: uint8(a >> 8),
		}
	}
}

// resolveNamedColorWithPalette resolves a named color using a custom palette.
func resolveNamedColorWithPalette(name int, fg bool, palette *[256]color.RGBA, defaultFG, defaultBG *color.RGBA) color.RGBA {
	switch {
	case name >= 0 && name < 16:
		return palette[name]
	case name == 256: // NamedColorForeground
		return *defaultFG
	case name == 257: // NamedColorBackground
		return *defaultBG
	case name == 258: // NamedColorCursor
		return *defaultFG // Use foreground as cursor default
	case name >= 259 && name <= 266: // Dim colors
		baseIndex := name - 259
		base := palette[baseIndex]
		return color.RGBA{
			R: uint8(float64(base.R) * 0.66),
			G: uint8(float64(base.G) * 0.66),
			B: uint8(float64(base.B) * 0.66),
			A: 255,
		}
	case name == 267: // NamedColorBrightForeground
		return palette[15] // Bright White
	case name == 268: // NamedColorDimForeground
		return color.RGBA{
			R: uint8(float64(defaultFG.R) * 0.66),
			G: uint8(float64(defaultFG.G) * 0.66),
			B: uint8(float64(defaultFG.B) * 0.66),
			A: 255,
		}
	default:
		if fg {
			return *defaultFG
		}
		return *defaultBG
	}
}
