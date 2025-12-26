package headlessterm

import "io"

// ResponseProvider writes terminal responses (e.g., cursor position reports) back to the PTY.
// Typically an io.Writer connected to the PTY input.
type ResponseProvider = io.Writer

// NoopResponse discards all response data (useful when responses are not needed).
type NoopResponse struct{}

func (NoopResponse) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// --- Bell Provider ---

// BellProvider handles bell/beep events triggered by BEL (0x07) characters.
type BellProvider interface {
	// Ring is called when a bell character is received.
	Ring()
}

// NoopBell ignores all bell events.
type NoopBell struct{}

func (NoopBell) Ring() {}

// --- Title Provider ---

// TitleProvider handles window title changes (OSC 0, 1, 2).
type TitleProvider interface {
	// SetTitle is called when the title changes.
	SetTitle(title string)
	// PushTitle saves the current title to the stack.
	PushTitle()
	// PopTitle restores the title from the stack.
	PopTitle()
}

// NoopTitle ignores all title operations.
type NoopTitle struct{}

func (NoopTitle) SetTitle(title string) {}
func (NoopTitle) PushTitle()            {}
func (NoopTitle) PopTitle()             {}

// --- APC Provider ---

// APCProvider handles Application Program Command sequences (OSC _).
type APCProvider interface {
	// Receive is called with the payload of an APC sequence.
	Receive(data []byte)
}

// NoopAPC ignores all APC sequences.
type NoopAPC struct{}

func (NoopAPC) Receive(data []byte) {}

// --- PM Provider ---

// PMProvider handles Privacy Message sequences (OSC ^).
type PMProvider interface {
	// Receive is called with the payload of a PM sequence.
	Receive(data []byte)
}

// NoopPM ignores all PM sequences.
type NoopPM struct{}

func (NoopPM) Receive(data []byte) {}

// --- SOS Provider ---

// SOSProvider handles Start of String sequences (OSC X).
type SOSProvider interface {
	// Receive is called with the payload of a SOS sequence.
	Receive(data []byte)
}

// NoopSOS ignores all SOS sequences.
type NoopSOS struct{}

func (NoopSOS) Receive(data []byte) {}

// Ensure implementations satisfy their interfaces
var _ ResponseProvider = NoopResponse{}

// ClipboardProvider handles clipboard read/write operations (OSC 52).
type ClipboardProvider interface {
	// Read returns content from the specified clipboard ('c' for clipboard, 'p' for primary selection).
	Read(clipboard byte) string
	// Write stores content to the specified clipboard.
	Write(clipboard byte, data []byte)
}

// ScrollbackProvider stores lines scrolled off the top of the primary buffer.
// Implementations can use in-memory storage, disk, database, etc.
type ScrollbackProvider interface {
	// Push appends a line to scrollback. Oldest lines should be removed if MaxLines is exceeded.
	Push(line []Cell)
	// Len returns the current number of stored lines.
	Len() int
	// Line returns the line at index, where 0 is the oldest line. Returns nil if out of range.
	Line(index int) []Cell
	// Clear removes all stored lines.
	Clear()
	// SetMaxLines sets the maximum capacity. Implementations should trim oldest lines if needed.
	SetMaxLines(max int)
	// MaxLines returns the current maximum capacity.
	MaxLines() int
}

// --- Clipboard Implementations ---

// NoopClipboard ignores all clipboard operations.
type NoopClipboard struct{}

func (NoopClipboard) Read(clipboard byte) string  { return "" }
func (NoopClipboard) Write(clipboard byte, data []byte) {}

// --- Scrollback Implementations ---

// NoopScrollback discards all scrollback lines (useful for alternate buffer which has no scrollback).
type NoopScrollback struct{}

func (NoopScrollback) Push(line []Cell)      {}
func (NoopScrollback) Len() int              { return 0 }
func (NoopScrollback) Line(index int) []Cell { return nil }
func (NoopScrollback) Clear()                {}
func (NoopScrollback) SetMaxLines(max int)   {}
func (NoopScrollback) MaxLines() int         { return 0 }

// --- Recording Provider ---

// RecordingProvider captures raw input bytes before ANSI parsing for replay or debugging.
type RecordingProvider interface {
	// Record appends raw bytes to the recording.
	Record(data []byte)
	// Data returns all captured bytes since the last Clear call.
	Data() []byte
	// Clear discards all recorded data.
	Clear()
}

// NoopRecording discards all input recordings.
type NoopRecording struct{}

func (NoopRecording) Record([]byte) {}
func (NoopRecording) Data() []byte  { return nil }
func (NoopRecording) Clear()        {}

// Ensure implementations satisfy their interfaces
var _ BellProvider = (*NoopBell)(nil)
var _ TitleProvider = (*NoopTitle)(nil)
var _ APCProvider = (*NoopAPC)(nil)
var _ PMProvider = (*NoopPM)(nil)
var _ SOSProvider = (*NoopSOS)(nil)
var _ ClipboardProvider = (*NoopClipboard)(nil)
var _ ScrollbackProvider = (*NoopScrollback)(nil)
var _ RecordingProvider = (*NoopRecording)(nil)
