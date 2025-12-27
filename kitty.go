package headlessterm

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"io"
	"strconv"
	"strings"
)

// KittyAction represents the action to perform.
type KittyAction byte

const (
	KittyActionTransmit        KittyAction = 't' // Transmit image data
	KittyActionTransmitDisplay KittyAction = 'T' // Transmit and display
	KittyActionQuery           KittyAction = 'q' // Query terminal support
	KittyActionDisplay         KittyAction = 'p' // Display (put) image
	KittyActionDelete          KittyAction = 'd' // Delete image(s)
	KittyActionFrame           KittyAction = 'f' // Transmit animation frame
	KittyActionAnimate         KittyAction = 'a' // Control animation
	KittyActionCompose         KittyAction = 'c' // Compose animation frames
)

// KittyTransmission represents how image data is transmitted.
type KittyTransmission byte

const (
	KittyTransmitDirect    KittyTransmission = 'd' // Direct (inline base64)
	KittyTransmitFile      KittyTransmission = 'f' // File path
	KittyTransmitTempFile  KittyTransmission = 't' // Temporary file
	KittyTransmitSharedMem KittyTransmission = 's' // Shared memory
)

// KittyFormat represents the image format.
type KittyFormat uint32

const (
	KittyFormatRGB  KittyFormat = 24  // 24-bit RGB
	KittyFormatRGBA KittyFormat = 32  // 32-bit RGBA (default)
	KittyFormatPNG  KittyFormat = 100 // PNG encoded
)

// KittyDelete represents what to delete.
type KittyDelete byte

const (
	KittyDeleteAll          KittyDelete = 'a' // All visible placements
	KittyDeleteAllWithData  KittyDelete = 'A' // All visible + image data
	KittyDeleteByID         KittyDelete = 'i' // By image ID
	KittyDeleteByIDWithData KittyDelete = 'I' // By image ID + image data
	KittyDeleteByNumber     KittyDelete = 'n' // By image number
	KittyDeleteByNumData    KittyDelete = 'N' // By image number + data
	KittyDeleteAtCursor     KittyDelete = 'c' // At cursor position
	KittyDeleteAtCursorData KittyDelete = 'C' // At cursor + data
	KittyDeleteAtPos        KittyDelete = 'p' // At specific position
	KittyDeleteAtPosData    KittyDelete = 'P' // At position + data
	KittyDeleteByCol        KittyDelete = 'x' // By column
	KittyDeleteByColData    KittyDelete = 'X' // By column + data
	KittyDeleteByRow        KittyDelete = 'y' // By row
	KittyDeleteByRowData    KittyDelete = 'Y' // By row + data
	KittyDeleteByZIndex     KittyDelete = 'z' // By z-index
	KittyDeleteByZIndexData KittyDelete = 'Z' // By z-index + data
)

// KittyCommand represents a parsed Kitty graphics command.
type KittyCommand struct {
	Action       KittyAction
	Transmission KittyTransmission
	Format       KittyFormat
	Compression  byte // 'z' for zlib

	// Image identification
	ImageID     uint32 // i=
	ImageNumber uint32 // I=
	PlacementID uint32 // p=

	// Transmission parameters
	Width  uint32 // s= (source width in pixels)
	Height uint32 // v= (source height in pixels)
	Size   uint32 // S= (data size for file/shm)
	Offset uint32 // O= (data offset for file/shm)
	More   bool   // m= (more data chunks coming)

	// Display parameters
	SrcX, SrcY     uint32 // x=, y= (source region origin)
	SrcW, SrcH     uint32 // w=, h= (source region size)
	Cols, Rows     uint32 // c=, r= (target cell size)
	CellOffsetX    uint32 // X= (x offset within cell)
	CellOffsetY    uint32 // Y= (y offset within cell)
	ZIndex         int32  // z= (z-index for layering)
	DoNotMoveCursor bool  // C= (1 = don't move cursor)

	// Delete parameters
	Delete KittyDelete // d=

	// Query/response
	Quiet uint32 // q= (0=normal, 1=suppress OK, 2=suppress all)

	// Payload data (base64 decoded)
	Payload []byte
}

// ParseKittyGraphics parses a Kitty graphics APC sequence.
// The data should be the content after ESC_G (without the ESC_G prefix and ST terminator).
func ParseKittyGraphics(data []byte) (*KittyCommand, error) {
	cmd := &KittyCommand{
		Action:       KittyActionTransmitDisplay, // Default action
		Transmission: KittyTransmitDirect,
		Format:       KittyFormatRGBA, // Default format
	}

	// Skip 'G' prefix if present
	if len(data) > 0 && data[0] == 'G' {
		data = data[1:]
	}

	// Find separator between control data and payload
	sepIdx := bytes.IndexByte(data, ';')
	var controlData, payload []byte

	if sepIdx >= 0 {
		controlData = data[:sepIdx]
		payload = data[sepIdx+1:]
	} else {
		controlData = data
	}

	// Parse control data (key=value pairs separated by commas)
	if len(controlData) > 0 {
		pairs := bytes.Split(controlData, []byte(","))
		for _, pair := range pairs {
			eqIdx := bytes.IndexByte(pair, '=')
			if eqIdx < 0 || eqIdx == 0 {
				continue
			}

			key := pair[0]
			value := pair[eqIdx+1:]

			switch key {
			case 'a':
				if len(value) > 0 {
					cmd.Action = KittyAction(value[0])
				}
			case 't':
				if len(value) > 0 {
					cmd.Transmission = KittyTransmission(value[0])
				}
			case 'f':
				cmd.Format = KittyFormat(parseUint32(value))
			case 'o':
				if len(value) > 0 {
					cmd.Compression = value[0]
				}
			case 'i':
				cmd.ImageID = parseUint32(value)
			case 'I':
				cmd.ImageNumber = parseUint32(value)
			case 'p':
				cmd.PlacementID = parseUint32(value)
			case 's':
				cmd.Width = parseUint32(value)
			case 'v':
				cmd.Height = parseUint32(value)
			case 'S':
				cmd.Size = parseUint32(value)
			case 'O':
				cmd.Offset = parseUint32(value)
			case 'm':
				cmd.More = parseUint32(value) == 1
			case 'x':
				cmd.SrcX = parseUint32(value)
			case 'y':
				cmd.SrcY = parseUint32(value)
			case 'w':
				cmd.SrcW = parseUint32(value)
			case 'h':
				cmd.SrcH = parseUint32(value)
			case 'c':
				cmd.Cols = parseUint32(value)
			case 'r':
				cmd.Rows = parseUint32(value)
			case 'X':
				cmd.CellOffsetX = parseUint32(value)
			case 'Y':
				cmd.CellOffsetY = parseUint32(value)
			case 'z':
				cmd.ZIndex = parseInt32(value)
			case 'C':
				cmd.DoNotMoveCursor = parseUint32(value) == 1
			case 'd':
				if len(value) > 0 {
					cmd.Delete = KittyDelete(value[0])
				}
			case 'q':
				cmd.Quiet = parseUint32(value)
			}
		}
	}

	// Decode payload if present
	if len(payload) > 0 {
		decoded, err := base64.StdEncoding.DecodeString(string(payload))
		if err != nil {
			// Try without padding
			decoded, err = base64.RawStdEncoding.DecodeString(string(payload))
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 payload: %w", err)
			}
		}
		cmd.Payload = decoded
	}

	return cmd, nil
}

// DecodeImageData decodes the image payload based on format and compression.
// Returns RGBA pixel data, width, and height.
func (cmd *KittyCommand) DecodeImageData() ([]byte, uint32, uint32, error) {
	data := cmd.Payload

	// Decompress if needed
	if cmd.Compression == 'z' && len(data) > 0 {
		r, err := zlib.NewReader(bytes.NewReader(data))
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to create zlib reader: %w", err)
		}
		defer r.Close()

		decompressed, err := io.ReadAll(r)
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to decompress data: %w", err)
		}
		data = decompressed
	}

	switch cmd.Format {
	case KittyFormatPNG:
		return decodePNG(data)

	case KittyFormatRGB:
		if cmd.Width == 0 || cmd.Height == 0 {
			return nil, 0, 0, fmt.Errorf("RGB format requires width and height")
		}
		expected := int(cmd.Width * cmd.Height * 3)
		if len(data) < expected {
			return nil, 0, 0, fmt.Errorf("insufficient RGB data: got %d, expected %d", len(data), expected)
		}
		// Convert RGB to RGBA
		rgba := make([]byte, cmd.Width*cmd.Height*4)
		for i := uint32(0); i < cmd.Width*cmd.Height; i++ {
			rgba[i*4+0] = data[i*3+0]
			rgba[i*4+1] = data[i*3+1]
			rgba[i*4+2] = data[i*3+2]
			rgba[i*4+3] = 255
		}
		return rgba, cmd.Width, cmd.Height, nil

	case KittyFormatRGBA:
		if cmd.Width == 0 || cmd.Height == 0 {
			return nil, 0, 0, fmt.Errorf("RGBA format requires width and height")
		}
		expected := int(cmd.Width * cmd.Height * 4)
		if len(data) < expected {
			return nil, 0, 0, fmt.Errorf("insufficient RGBA data: got %d, expected %d", len(data), expected)
		}
		return data[:expected], cmd.Width, cmd.Height, nil

	default:
		return nil, 0, 0, fmt.Errorf("unsupported format: %d", cmd.Format)
	}
}

// decodePNG decodes PNG data to RGBA pixels.
func decodePNG(data []byte) ([]byte, uint32, uint32, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		// Try as generic image
		img, _, err = image.Decode(bytes.NewReader(data))
		if err != nil {
			return nil, 0, 0, fmt.Errorf("failed to decode PNG: %w", err)
		}
	}

	bounds := img.Bounds()
	width := uint32(bounds.Dx())
	height := uint32(bounds.Dy())

	rgba := make([]byte, width*height*4)

	for y := 0; y < int(height); y++ {
		for x := 0; x < int(width); x++ {
			r, g, b, a := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			offset := (uint32(y)*width + uint32(x)) * 4
			rgba[offset+0] = uint8(r >> 8)
			rgba[offset+1] = uint8(g >> 8)
			rgba[offset+2] = uint8(b >> 8)
			rgba[offset+3] = uint8(a >> 8)
		}
	}

	return rgba, width, height, nil
}

// parseUint32 parses a byte slice as uint32.
func parseUint32(b []byte) uint32 {
	n, _ := strconv.ParseUint(string(b), 10, 32)
	return uint32(n)
}

// parseInt32 parses a byte slice as int32.
func parseInt32(b []byte) int32 {
	n, _ := strconv.ParseInt(string(b), 10, 32)
	return int32(n)
}

// FormatKittyResponse formats a Kitty graphics response.
func FormatKittyResponse(imageID uint32, message string, isError bool) string {
	var sb strings.Builder
	sb.WriteString("\x1b_G")
	if imageID > 0 {
		sb.WriteString(fmt.Sprintf("i=%d", imageID))
	}
	sb.WriteString(";")
	if isError {
		sb.WriteString(message)
	} else {
		sb.WriteString("OK")
	}
	sb.WriteString("\x1b\\")
	return sb.String()
}
