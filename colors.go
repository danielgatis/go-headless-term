package headlessterm

import "image/color"

// DefaultPalette is the standard 256-color palette: 16 named colors (0-15), 216 color cube (16-231), 24 grayscale (232-255).
var DefaultPalette = [256]color.RGBA{
	// Standard colors (0-7)
	{0, 0, 0, 255},       // Black
	{205, 49, 49, 255},   // Red
	{13, 188, 121, 255},  // Green
	{229, 229, 16, 255},  // Yellow
	{36, 114, 200, 255},  // Blue
	{188, 63, 188, 255},  // Magenta
	{17, 168, 205, 255},  // Cyan
	{229, 229, 229, 255}, // White

	// Bright colors (8-15)
	{102, 102, 102, 255}, // Bright Black
	{241, 76, 76, 255},   // Bright Red
	{35, 209, 139, 255},  // Bright Green
	{245, 245, 67, 255},  // Bright Yellow
	{59, 142, 234, 255},  // Bright Blue
	{214, 112, 214, 255}, // Bright Magenta
	{41, 184, 219, 255},  // Bright Cyan
	{255, 255, 255, 255}, // Bright White

	// 216 colors (16-231)
	// Generated programmatically below

	// Grayscale (232-255)
	// Generated programmatically below
}

func init() {
	// Generate 216 color cube (16-231)
	i := 16
	for r := 0; r < 6; r++ {
		for g := 0; g < 6; g++ {
			for b := 0; b < 6; b++ {
				DefaultPalette[i] = color.RGBA{
					R: uint8(r * 51),
					G: uint8(g * 51),
					B: uint8(b * 51),
					A: 255,
				}
				i++
			}
		}
	}

	// Generate grayscale (232-255)
	for j := 0; j < 24; j++ {
		gray := uint8(8 + j*10)
		DefaultPalette[232+j] = color.RGBA{gray, gray, gray, 255}
	}
}

// DefaultForeground is the default text color (light gray).
var DefaultForeground = color.RGBA{229, 229, 229, 255}

// DefaultBackground is the default background color (black).
var DefaultBackground = color.RGBA{0, 0, 0, 255}

// DefaultCursorColor is the default cursor rendering color (light gray).
var DefaultCursorColor = color.RGBA{229, 229, 229, 255}

// Named color indices for semantic colors (used with NamedColor).
const (
	NamedColorForeground       = 256 // Default foreground text color
	NamedColorBackground       = 257 // Default background color
	NamedColorCursor           = 258 // Cursor color
	NamedColorDimBlack         = 259 // Dim black
	NamedColorDimRed           = 260 // Dim red
	NamedColorDimGreen         = 261 // Dim green
	NamedColorDimYellow        = 262 // Dim yellow
	NamedColorDimBlue          = 263 // Dim blue
	NamedColorDimMagenta       = 264 // Dim magenta
	NamedColorDimCyan          = 265 // Dim cyan
	NamedColorDimWhite         = 266 // Dim white
	NamedColorBrightForeground = 267 // Bright foreground (white)
	NamedColorDimForeground    = 268 // Dim foreground
)

// resolveDefaultColor converts a color.Color to RGBA using the default palette.
// If c is nil, returns the default foreground or background based on fg.
// IndexedColor and NamedColor are resolved using DefaultPalette.
func resolveDefaultColor(c color.Color, fg bool) color.RGBA {
	if c == nil {
		if fg {
			return DefaultForeground
		}
		return DefaultBackground
	}

	switch v := c.(type) {
	case color.RGBA:
		return v
	case *IndexedColor:
		if v.Index >= 0 && v.Index < 256 {
			return DefaultPalette[v.Index]
		}
		if fg {
			return DefaultForeground
		}
		return DefaultBackground
	case *NamedColor:
		return resolveNamedColor(v.Name, fg)
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

// resolveNamedColor resolves a named color index to RGBA.
func resolveNamedColor(name int, fg bool) color.RGBA {
	switch {
	case name >= 0 && name < 16:
		return DefaultPalette[name]
	case name == 256: // NamedColorForeground
		return DefaultForeground
	case name == 257: // NamedColorBackground
		return DefaultBackground
	case name == 258: // NamedColorCursor
		return DefaultCursorColor
	case name >= 259 && name <= 266: // Dim colors
		baseIndex := name - 259
		base := DefaultPalette[baseIndex]
		return color.RGBA{
			R: uint8(float64(base.R) * 0.66),
			G: uint8(float64(base.G) * 0.66),
			B: uint8(float64(base.B) * 0.66),
			A: 255,
		}
	case name == 267: // NamedColorBrightForeground
		return DefaultPalette[15] // Bright White
	case name == 268: // NamedColorDimForeground
		return color.RGBA{
			R: uint8(float64(DefaultForeground.R) * 0.66),
			G: uint8(float64(DefaultForeground.G) * 0.66),
			B: uint8(float64(DefaultForeground.B) * 0.66),
			A: 255,
		}
	default:
		if fg {
			return DefaultForeground
		}
		return DefaultBackground
	}
}
