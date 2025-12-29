package headlessterm

import (
	"io"

	"github.com/danielgatis/go-ansicode"
)

// NotificationPayload is re-exported from go-ansicode for use by notification providers.
// This allows consumers to implement NotificationProvider without importing go-ansicode directly.
type NotificationPayload = ansicode.NotificationPayload

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

// MemoryScrollback stores scrollback lines in memory with a configurable limit.
// When the limit is reached, the oldest lines are removed to make room for new ones.
//
// Example:
//
//	storage := headlessterm.NewMemoryScrollback(10000)
//	term := headlessterm.New(headlessterm.WithScrollback(storage))
type MemoryScrollback struct {
	lines    [][]Cell
	maxLines int
}

// NewMemoryScrollback creates a new in-memory scrollback buffer with the given capacity.
// If maxLines is 0, scrollback is unlimited (be careful with memory usage).
func NewMemoryScrollback(maxLines int) *MemoryScrollback {
	return &MemoryScrollback{
		lines:    make([][]Cell, 0),
		maxLines: maxLines,
	}
}

// Push appends a line to scrollback. If maxLines is exceeded, the oldest line is removed.
func (m *MemoryScrollback) Push(line []Cell) {
	// Make a copy to prevent external modifications
	lineCopy := make([]Cell, len(line))
	copy(lineCopy, line)

	m.lines = append(m.lines, lineCopy)

	// Trim oldest lines if over capacity
	if m.maxLines > 0 && len(m.lines) > m.maxLines {
		excess := len(m.lines) - m.maxLines
		m.lines = m.lines[excess:]
	}
}

// Len returns the current number of stored lines.
func (m *MemoryScrollback) Len() int {
	return len(m.lines)
}

// Line returns the line at index, where 0 is the oldest line.
// Returns nil if index is out of range.
func (m *MemoryScrollback) Line(index int) []Cell {
	if index < 0 || index >= len(m.lines) {
		return nil
	}
	return m.lines[index]
}

// Clear removes all stored lines.
func (m *MemoryScrollback) Clear() {
	m.lines = make([][]Cell, 0)
}

// SetMaxLines sets the maximum capacity. If the current length exceeds the new max,
// the oldest lines are removed.
func (m *MemoryScrollback) SetMaxLines(max int) {
	m.maxLines = max
	if max > 0 && len(m.lines) > max {
		excess := len(m.lines) - max
		m.lines = m.lines[excess:]
	}
}

// MaxLines returns the current maximum capacity.
func (m *MemoryScrollback) MaxLines() int {
	return m.maxLines
}

// MemoryRecording stores raw input bytes in memory for replay or debugging.
//
// Example:
//
//	recorder := headlessterm.NewMemoryRecording()
//	term := headlessterm.New(headlessterm.WithRecording(recorder))
//	// ... process terminal output ...
//	data := recorder.Data() // Get all recorded bytes
type MemoryRecording struct {
	data []byte
}

// NewMemoryRecording creates a new in-memory recording buffer.
func NewMemoryRecording() *MemoryRecording {
	return &MemoryRecording{
		data: make([]byte, 0),
	}
}

// Record appends raw bytes to the recording.
func (r *MemoryRecording) Record(data []byte) {
	r.data = append(r.data, data...)
}

// Data returns all captured bytes since the last Clear call.
func (r *MemoryRecording) Data() []byte {
	result := make([]byte, len(r.data))
	copy(result, r.data)
	return result
}

// Clear discards all recorded data.
func (r *MemoryRecording) Clear() {
	r.data = make([]byte, 0)
}

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

// --- Size Provider ---

// SizeProvider provides pixel dimensions for CSI queries.
type SizeProvider interface {
	// WindowSizePixels returns the terminal window size in pixels.
	WindowSizePixels() (width, height int)
	// CellSizePixels returns the size of a single cell in pixels.
	CellSizePixels() (width, height int)
}

// NoopSizeProvider returns default values for size queries.
type NoopSizeProvider struct{}

func (NoopSizeProvider) WindowSizePixels() (width, height int) { return 800, 600 }
func (NoopSizeProvider) CellSizePixels() (width, height int)   { return 10, 20 }

// --- Notification Provider ---

// NotificationProvider handles OSC 99 desktop notifications (Kitty protocol).
// Implementations can display native desktop notifications, log them, or process them
// in any other way appropriate for the platform.
//
// The Kitty notification protocol supports:
//   - Title and body text
//   - Urgency levels (low, normal, critical)
//   - Icons (by name or PNG data)
//   - Sound control (system, silent, error, etc.)
//   - Chunking for large payloads
//   - Query mechanism for capability discovery
//   - Close tracking for notification lifecycle
//
// Note: Buttons are NOT supported (same as Kitty on macOS). Applications
// that query capabilities will see buttons are not reported, and button
// payloads are silently ignored.
//
// See: https://sw.kovidgoyal.net/kitty/desktop-notifications/
type NotificationProvider interface {
	// Notify processes a notification payload and optionally returns a response.
	// For query requests (PayloadType == "?"), implementations should return
	// a string describing supported capabilities in OSC 99 format.
	// For other requests, return empty string.
	Notify(payload *NotificationPayload) string
}

// NoopNotification ignores all notification events.
type NoopNotification struct{}

// Notify discards the notification and returns no response.
func (NoopNotification) Notify(payload *NotificationPayload) string { return "" }

// Ensure implementations satisfy their interfaces
var _ BellProvider = (*NoopBell)(nil)
var _ TitleProvider = (*NoopTitle)(nil)
var _ APCProvider = (*NoopAPC)(nil)
var _ PMProvider = (*NoopPM)(nil)
var _ SOSProvider = (*NoopSOS)(nil)
var _ ClipboardProvider = (*NoopClipboard)(nil)
var _ ScrollbackProvider = (*NoopScrollback)(nil)
var _ ScrollbackProvider = (*MemoryScrollback)(nil)
var _ RecordingProvider = (*NoopRecording)(nil)
var _ RecordingProvider = (*MemoryRecording)(nil)
var _ SizeProvider = (*NoopSizeProvider)(nil)
var _ NotificationProvider = (*NoopNotification)(nil)
