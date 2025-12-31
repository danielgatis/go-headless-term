package headlessterm

import (
	"image/color"
)

// SixelImage represents a decoded Sixel image.
type SixelImage struct {
	Width       uint32
	Height      uint32
	Data        []byte // RGBA pixel data
	Transparent bool   // Whether background is transparent
}

// sixelParser handles parsing of Sixel data.
type sixelParser struct {
	palette     [256]color.RGBA
	colorIndex  int
	x, y        int
	maxX, maxY  int
	pixels      map[int]map[int]color.RGBA
	transparent bool
}

// ParseSixel parses Sixel data and returns an RGBA image.
// params contains the DCS parameters (P1;P2;P3).
// data contains the raw Sixel bytes after 'q'.
func ParseSixel(params []int64, data []byte) (*SixelImage, error) {
	p := &sixelParser{
		pixels:     make(map[int]map[int]color.RGBA),
		colorIndex: 0,
	}

	// Initialize default VGA palette
	p.initDefaultPalette()

	// Parse DCS parameters
	// P1: pixel aspect ratio numerator (ignored)
	// P2: background select (0=device default, 1=no change, 2=set to color 0)
	// P3: horizontal grid size (ignored)
	if len(params) >= 2 && params[1] == 1 {
		p.transparent = true
	}

	// Parse sixel data
	p.parse(data)

	// Convert to RGBA image
	return p.toImage(), nil
}

// initDefaultPalette sets up the default VGA 16-color palette.
func (p *sixelParser) initDefaultPalette() {
	// Standard VGA colors
	vgaColors := []color.RGBA{
		{0, 0, 0, 255},       // 0: Black
		{0, 0, 205, 255},     // 1: Blue
		{205, 0, 0, 255},     // 2: Red
		{205, 0, 205, 255},   // 3: Magenta
		{0, 205, 0, 255},     // 4: Green
		{0, 205, 205, 255},   // 5: Cyan
		{205, 205, 0, 255},   // 6: Yellow
		{205, 205, 205, 255}, // 7: White
		{0, 0, 0, 255},       // 8: Black (repeat for HLS)
		{0, 0, 255, 255},     // 9: Bright Blue
		{255, 0, 0, 255},     // 10: Bright Red
		{255, 0, 255, 255},   // 11: Bright Magenta
		{0, 255, 0, 255},     // 12: Bright Green
		{0, 255, 255, 255},   // 13: Bright Cyan
		{255, 255, 0, 255},   // 14: Bright Yellow
		{255, 255, 255, 255}, // 15: Bright White
	}

	copy(p.palette[:], vgaColors)

	// Fill remaining with grayscale
	for i := 16; i < 256; i++ {
		gray := uint8((i - 16) * 255 / 239)
		p.palette[i] = color.RGBA{gray, gray, gray, 255}
	}
}

// parse processes the sixel byte stream.
func (p *sixelParser) parse(data []byte) {
	i := 0
	for i < len(data) {
		b := data[i]
		i++

		switch {
		case b == '$':
			// Carriage return - go to beginning of current sixel line
			p.x = 0

		case b == '-':
			// New line - move down 6 pixels and go to beginning
			p.x = 0
			p.y += 6

		case b == '!':
			// Repeat introducer: !<count><sixel>
			count, newI := p.parseNumber(data, i)
			i = newI
			if i < len(data) {
				sixel := data[i]
				i++
				if sixel >= '?' && sixel <= '~' {
					p.drawSixel(sixel, int(count))
				}
			}

		case b == '#':
			// Color introducer: #<index> or #<index>;<type>;<v1>;<v2>;<v3>
			colorNum, newI := p.parseNumber(data, i)
			i = newI

			if i < len(data) && data[i] == ';' {
				// Color definition
				i++ // skip ';'
				colorType, newI := p.parseNumber(data, i)
				i = newI

				if i < len(data) && data[i] == ';' {
					i++ // skip ';'
					v1, newI := p.parseNumber(data, i)
					i = newI

					if i < len(data) && data[i] == ';' {
						i++ // skip ';'
						v2, newI := p.parseNumber(data, i)
						i = newI

						if i < len(data) && data[i] == ';' {
							i++ // skip ';'
							v3, newI := p.parseNumber(data, i)
							i = newI

							if colorNum >= 0 && colorNum < 256 {
								if colorType == 1 {
									// HLS color
									p.palette[colorNum] = hlsToRGB(int(v1), int(v2), int(v3))
								} else {
									// RGB color (type 2 or default)
									// Values are 0-100 percentage
									r := uint8(v1 * 255 / 100)
									g := uint8(v2 * 255 / 100)
									b := uint8(v3 * 255 / 100)
									p.palette[colorNum] = color.RGBA{r, g, b, 255}
								}
							}
						}
					}
				}
			}

			// Select color
			if colorNum >= 0 && colorNum < 256 {
				p.colorIndex = int(colorNum)
			}

		case b >= '?' && b <= '~':
			// Sixel data character
			p.drawSixel(b, 1)

		case b == '"':
			// Raster attributes: "<Pan>;<Pad>;<Ph>;<Pv>
			// Pan/Pad = pixel aspect ratio, Ph/Pv = width/height
			// We parse but mostly ignore these
			for i < len(data) && data[i] != '$' && data[i] != '-' &&
				data[i] != '#' && data[i] != '!' &&
				!(data[i] >= '?' && data[i] <= '~') {
				i++
			}
		}
	}
}

// parseNumber parses a decimal number from data starting at index i.
func (p *sixelParser) parseNumber(data []byte, i int) (int64, int) {
	var n int64
	for i < len(data) && data[i] >= '0' && data[i] <= '9' {
		n = n*10 + int64(data[i]-'0')
		i++
	}
	return n, i
}

// drawSixel draws a sixel character at the current position.
// A sixel represents 6 vertical pixels encoded in 6 bits.
func (p *sixelParser) drawSixel(b byte, count int) {
	if count <= 0 {
		count = 1
	}

	// Convert from sixel encoding (?-~ maps to 0-63)
	bits := b - '?'

	c := p.palette[p.colorIndex]

	for r := 0; r < count; r++ {
		// Each bit represents a vertical pixel (bit 0 = top)
		for bit := 0; bit < 6; bit++ {
			if bits&(1<<bit) != 0 {
				py := p.y + bit
				px := p.x

				if p.pixels[py] == nil {
					p.pixels[py] = make(map[int]color.RGBA)
				}
				p.pixels[py][px] = c

				if px > p.maxX {
					p.maxX = px
				}
				if py > p.maxY {
					p.maxY = py
				}
			}
		}
		p.x++
	}
}

// toImage converts the parsed pixels to an RGBA image.
func (p *sixelParser) toImage() *SixelImage {
	// No pixels drawn
	if len(p.pixels) == 0 {
		return &SixelImage{
			Width:  0,
			Height: 0,
			Data:   nil,
		}
	}

	width := uint32(p.maxX + 1)
	height := uint32(p.maxY + 1)

	// Allocate RGBA buffer
	data := make([]byte, width*height*4)

	// Fill with transparent or background color
	if p.transparent {
		// Leave as zero (transparent)
	} else {
		// Fill with color 0 (background)
		bg := p.palette[0]
		for i := uint32(0); i < width*height; i++ {
			data[i*4+0] = bg.R
			data[i*4+1] = bg.G
			data[i*4+2] = bg.B
			data[i*4+3] = bg.A
		}
	}

	// Copy pixels
	for y, row := range p.pixels {
		for x, c := range row {
			if x >= 0 && x < int(width) && y >= 0 && y < int(height) {
				offset := (uint32(y)*width + uint32(x)) * 4
				data[offset+0] = c.R
				data[offset+1] = c.G
				data[offset+2] = c.B
				data[offset+3] = c.A
			}
		}
	}

	return &SixelImage{
		Width:       width,
		Height:      height,
		Data:        data,
		Transparent: p.transparent,
	}
}

// hlsToRGB converts HLS color to RGB.
// Sixel uses non-standard HLS where:
// - Hue: 0-360 degrees (blue=0, red=120, green=240)
// - Lightness: 0-100
// - Saturation: 0-100
func hlsToRGB(h, l, s int) color.RGBA {
	if s == 0 {
		// Achromatic (gray)
		v := uint8(l * 255 / 100)
		return color.RGBA{v, v, v, 255}
	}

	// Normalize values
	hNorm := float64(h) / 360.0
	lNorm := float64(l) / 100.0
	sNorm := float64(s) / 100.0

	// Rotate hue for Sixel's non-standard color wheel
	// Sixel: blue=0, red=120, green=240
	// Standard: red=0, green=120, blue=240
	hNorm = hNorm + 1.0/3.0 // Shift by 120 degrees
	if hNorm >= 1.0 {
		hNorm -= 1.0
	}

	var q float64
	if lNorm < 0.5 {
		q = lNorm * (1 + sNorm)
	} else {
		q = lNorm + sNorm - lNorm*sNorm
	}
	p := 2*lNorm - q

	r := hueToRGB(p, q, hNorm+1.0/3.0)
	g := hueToRGB(p, q, hNorm)
	b := hueToRGB(p, q, hNorm-1.0/3.0)

	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 255,
	}
}

// hueToRGB is a helper for HLS to RGB conversion.
func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	if t < 1.0/6.0 {
		return p + (q-p)*6*t
	}
	if t < 1.0/2.0 {
		return q
	}
	if t < 2.0/3.0 {
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
}
