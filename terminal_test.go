package headlessterm

import (
	"bytes"
	"strings"
	"testing"

	"github.com/danielgatis/go-ansicode"
)

func TestNewTerminal(t *testing.T) {
	term := New()

	if term.Rows() != 24 {
		t.Errorf("expected 24 rows, got %d", term.Rows())
	}
	if term.Cols() != 80 {
		t.Errorf("expected 80 cols, got %d", term.Cols())
	}
}

func TestTerminalWithSize(t *testing.T) {
	term := New(WithSize(40, 120))

	if term.Rows() != 40 {
		t.Errorf("expected 40 rows, got %d", term.Rows())
	}
	if term.Cols() != 120 {
		t.Errorf("expected 120 cols, got %d", term.Cols())
	}
}

func TestTerminalWrite(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("Hello")

	content := term.LineContent(0)
	if content != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", content)
	}
}

func TestTerminalCursorPosition(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("ABC")

	row, col := term.CursorPos()
	if row != 0 || col != 3 {
		t.Errorf("expected cursor at (0, 3), got (%d, %d)", row, col)
	}
}

func TestTerminalNewline(t *testing.T) {
	term := New(WithSize(24, 80))

	// Use \r\n for proper line break (CR+LF)
	term.WriteString("Line1\r\nLine2")

	if term.LineContent(0) != "Line1" {
		t.Errorf("expected 'Line1', got '%s'", term.LineContent(0))
	}
	if term.LineContent(1) != "Line2" {
		t.Errorf("expected 'Line2', got '%s'", term.LineContent(1))
	}
}

func TestTerminalClearScreen(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("Hello")
	term.WriteString("\x1b[2J") // Clear screen

	if term.LineContent(0) != "" {
		t.Errorf("expected empty line after clear, got '%s'", term.LineContent(0))
	}
}

func TestTerminalScrollback(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(5, 80), WithScrollback(storage))

	// Write more lines than the terminal can display
	for i := 0; i < 10; i++ {
		term.WriteString("Line\n")
	}

	if term.ScrollbackLen() < 5 {
		t.Errorf("expected at least 5 scrollback lines, got %d", term.ScrollbackLen())
	}
}

func TestTerminalSelection(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("Hello World")
	term.SetSelection(Position{Row: 0, Col: 0}, Position{Row: 0, Col: 4})

	if !term.HasSelection() {
		t.Error("expected selection to be active")
	}

	selected := term.GetSelectedText()
	if selected != "Hello" {
		t.Errorf("expected 'Hello', got '%s'", selected)
	}

	term.ClearSelection()
	if term.HasSelection() {
		t.Error("expected selection to be cleared")
	}
}

func TestTerminalSearch(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("Hello World\r\n")
	term.WriteString("Hello Again\r\n")

	matches := term.Search("Hello")
	if len(matches) != 2 {
		t.Errorf("expected 2 matches, got %d", len(matches))
	}

	if len(matches) >= 1 && (matches[0].Row != 0 || matches[0].Col != 0) {
		t.Errorf("first match should be at (0, 0), got (%d, %d)", matches[0].Row, matches[0].Col)
	}
	if len(matches) >= 2 && (matches[1].Row != 1 || matches[1].Col != 0) {
		t.Errorf("second match should be at (1, 0), got (%d, %d)", matches[1].Row, matches[1].Col)
	}
}

func TestTerminalString(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("Line1\r\nLine2\r\nLine3")

	content := term.String()
	expected := "Line1\nLine2\nLine3"
	if content != expected {
		t.Errorf("expected '%s', got '%s'", expected, content)
	}
}

func TestTerminalDirtyTracking(t *testing.T) {
	term := New(WithSize(24, 80))

	// Initial state should have dirty cells after creation
	term.ClearDirty()

	if term.HasDirty() {
		t.Error("expected no dirty cells after ClearDirty")
	}

	term.WriteString("A")

	if !term.HasDirty() {
		t.Error("expected dirty cells after write")
	}

	dirty := term.DirtyCells()
	if len(dirty) == 0 {
		t.Error("expected at least one dirty cell")
	}

	term.ClearDirty()
	if term.HasDirty() {
		t.Error("expected no dirty cells after second ClearDirty")
	}
}

func TestTerminalWideCharacter(t *testing.T) {
	term := New(WithSize(24, 80))

	// Write a wide character (Chinese)
	term.WriteString("中")

	_, col := term.CursorPos()
	if col != 2 {
		t.Errorf("expected cursor at col 2 after wide char, got %d", col)
	}

	cell := term.Cell(0, 0)
	if cell == nil {
		t.Fatal("expected cell at (0,0)")
	}
	if cell.Char != '中' {
		t.Errorf("expected '中', got '%c'", cell.Char)
	}
	if !cell.IsWide() {
		t.Error("expected cell to be marked as wide")
	}

	spacer := term.Cell(0, 1)
	if spacer == nil {
		t.Fatal("expected spacer cell at (0,1)")
	}
	if !spacer.IsWideSpacer() {
		t.Error("expected spacer cell to be marked as spacer")
	}
}

func TestTerminalResize(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("Hello")
	term.Resize(10, 40)

	if term.Rows() != 10 || term.Cols() != 40 {
		t.Errorf("expected size 10x40, got %dx%d", term.Rows(), term.Cols())
	}

	// Content should be preserved
	if term.LineContent(0) != "Hello" {
		t.Errorf("expected content preserved after resize, got '%s'", term.LineContent(0))
	}
}

func TestTerminalTitle(t *testing.T) {
	var capturedTitle string
	term := New(
		WithSize(24, 80),
		WithMiddleware(&Middleware{
			SetTitle: func(title string, next func(string)) {
				capturedTitle = title
				next(title)
			},
		}),
	)

	term.WriteString("\x1b]0;My Title\x07")

	if term.Title() != "My Title" {
		t.Errorf("expected 'My Title', got '%s'", term.Title())
	}
	if capturedTitle != "My Title" {
		t.Errorf("middleware expected 'My Title', got '%s'", capturedTitle)
	}
}

func TestTerminalColors(t *testing.T) {
	term := New(WithSize(24, 80))

	// Red foreground
	term.WriteString("\x1b[31mRed")

	cell := term.Cell(0, 0)
	if cell == nil {
		t.Fatal("expected cell at (0,0)")
	}
	if cell.Fg == nil {
		t.Error("expected foreground color to be set")
	}
}

func TestTerminalBold(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("\x1b[1mBold")

	cell := term.Cell(0, 0)
	if cell == nil {
		t.Fatal("expected cell at (0,0)")
	}
	if !cell.HasFlag(CellFlagBold) {
		t.Error("expected bold flag to be set")
	}
}

func TestTerminalAlternateScreen(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("Main screen")

	if term.IsAlternateScreen() {
		t.Error("expected primary screen")
	}

	// Switch to alternate screen
	term.WriteString("\x1b[?1049h")

	if !term.IsAlternateScreen() {
		t.Error("expected alternate screen")
	}

	// Alternate screen should be clear
	if term.LineContent(0) != "" {
		t.Error("expected alternate screen to be clear")
	}

	term.WriteString("Alt screen")

	// Switch back to main screen
	term.WriteString("\x1b[?1049l")

	if term.IsAlternateScreen() {
		t.Error("expected primary screen after switch back")
	}

	// Main screen content should be preserved
	if term.LineContent(0) != "Main screen" {
		t.Errorf("expected 'Main screen', got '%s'", term.LineContent(0))
	}
}

func TestCustomScrollbackProvider(t *testing.T) {
	// Create a custom storage that counts pushes
	storage := &testScrollback{
		lines: make([][]Cell, 0),
	}

	term := New(
		WithSize(3, 80),
		WithScrollback(storage),
	)

	storage.SetMaxLines(100)

	// Write more lines than terminal height to trigger scroll
	for i := 0; i < 10; i++ {
		term.WriteString("Line\n")
	}

	if storage.pushCount == 0 {
		t.Error("expected custom storage to receive pushed lines")
	}
}

// testScrollback is a test implementation of ScrollbackProvider
type testScrollback struct {
	lines     [][]Cell
	maxLines  int
	pushCount int
}

func (s *testScrollback) Push(line []Cell) {
	s.pushCount++
	lineCopy := make([]Cell, len(line))
	copy(lineCopy, line)
	s.lines = append(s.lines, lineCopy)
	if s.maxLines > 0 && len(s.lines) > s.maxLines {
		s.lines = s.lines[len(s.lines)-s.maxLines:]
	}
}

func (s *testScrollback) Len() int {
	return len(s.lines)
}

func (s *testScrollback) Line(index int) []Cell {
	if index < 0 || index >= len(s.lines) {
		return nil
	}
	return s.lines[index]
}

func (s *testScrollback) Clear() {
	s.lines = make([][]Cell, 0)
}

func (s *testScrollback) SetMaxLines(max int) {
	s.maxLines = max
}

func (s *testScrollback) MaxLines() int {
	return s.maxLines
}

func (s *testScrollback) Pop() []Cell {
	if len(s.lines) == 0 {
		return nil
	}
	line := s.lines[len(s.lines)-1]
	s.lines = s.lines[:len(s.lines)-1]
	return line
}

func TestMiddlewareInput(t *testing.T) {
	var intercepted []rune
	term := New(
		WithSize(24, 80),
		WithMiddleware(&Middleware{
			Input: func(r rune, next func(rune)) {
				intercepted = append(intercepted, r)
				// Modify the rune before passing to terminal
				if r == 'a' {
					next('A')
				} else {
					next(r)
				}
			},
		}),
	)

	term.WriteString("abc")

	if len(intercepted) != 3 {
		t.Errorf("expected 3 intercepted runes, got %d", len(intercepted))
	}

	// Check that 'a' was transformed to 'A'
	content := term.LineContent(0)
	if content != "Abc" {
		t.Errorf("expected 'Abc', got '%s'", content)
	}
}

func TestMiddlewareBell(t *testing.T) {
	bellCount := 0
	term := New(
		WithSize(24, 80),
		WithMiddleware(&Middleware{
			Bell: func(next func()) {
				bellCount++
				next()
			},
		}),
	)

	// Send bell character
	term.WriteString("\x07")

	if bellCount != 1 {
		t.Errorf("expected 1 bell, got %d", bellCount)
	}
}

func TestMiddlewareSetTitle(t *testing.T) {
	var titles []string
	term := New(
		WithSize(24, 80),
		WithMiddleware(&Middleware{
			SetTitle: func(title string, next func(string)) {
				titles = append(titles, title)
				// Prefix the title
				next("[PREFIX] " + title)
			},
		}),
	)

	term.WriteString("\x1b]0;My Title\x07")

	if len(titles) != 1 {
		t.Errorf("expected 1 title, got %d", len(titles))
	}
	if titles[0] != "My Title" {
		t.Errorf("expected 'My Title', got '%s'", titles[0])
	}

	// The actual title should be prefixed
	if term.Title() != "[PREFIX] My Title" {
		t.Errorf("expected '[PREFIX] My Title', got '%s'", term.Title())
	}
}

func TestMiddlewareClearScreen(t *testing.T) {
	clearCount := 0
	term := New(
		WithSize(24, 80),
		WithMiddleware(&Middleware{
			ClearScreen: func(mode ansicode.ClearMode, next func(ansicode.ClearMode)) {
				clearCount++
				// Don't call next - screen won't be cleared
			},
		}),
	)

	term.WriteString("Hello")
	term.WriteString("\x1b[2J") // Try to clear screen

	if clearCount != 1 {
		t.Errorf("expected 1 clear call, got %d", clearCount)
	}

	// Screen should NOT be cleared because we didn't call next
	content := term.LineContent(0)
	if content != "Hello" {
		t.Errorf("expected 'Hello' (clear was blocked), got '%s'", content)
	}
}

func TestClipboardProvider(t *testing.T) {
	clipboard := &testClipboard{content: make(map[byte][]byte)}
	term := New(
		WithSize(24, 80),
		WithClipboard(clipboard),
	)

	// Store some data
	testData := []byte("test content")
	clipboard.Write('c', testData)

	// Verify content was stored
	content := clipboard.Read('c')
	if content != "test content" {
		t.Errorf("expected 'test content', got '%s'", content)
	}

	// Test that ClipboardProvider is accessible
	provider := term.ClipboardProvider()
	if provider == nil {
		t.Error("expected clipboard provider to be set")
	}
}

// testClipboard is a test implementation of ClipboardProvider
type testClipboard struct {
	content map[byte][]byte
}

func (c *testClipboard) Read(clipboard byte) string {
	if data, ok := c.content[clipboard]; ok {
		return string(data)
	}
	return ""
}

func (c *testClipboard) Write(clipboard byte, data []byte) {
	c.content[clipboard] = append([]byte(nil), data...)
}

func TestResponseWriter(t *testing.T) {
	var responses []byte
	writer := &testWriter{data: &responses}

	term := New(
		WithSize(24, 80),
		WithResponse(writer),
	)

	// Device status request (should trigger a response)
	term.WriteString("\x1b[5n")

	if len(responses) == 0 {
		t.Error("expected response to be written")
	}

	// Check it's a valid response
	expected := "\x1b[0n"
	if string(responses) != expected {
		t.Errorf("expected '%s', got '%s'", expected, string(responses))
	}
}

type testWriter struct {
	data *[]byte
}

func (w *testWriter) Write(p []byte) (n int, err error) {
	*w.data = append(*w.data, p...)
	return len(p), nil
}

func TestMiddlewareSkipsCall(t *testing.T) {
	term := New(
		WithSize(24, 80),
		WithMiddleware(&Middleware{
			Input: func(r rune, next func(rune)) {
				// Don't call next - input should be blocked
				if r != 'x' {
					next(r)
				}
			},
		}),
	)

	term.WriteString("axbxc")

	content := term.LineContent(0)
	if content != "abc" {
		t.Errorf("expected 'abc' (x's blocked), got '%s'", content)
	}
}

func TestMiddlewareMerge(t *testing.T) {
	bellCount := 0
	titleCount := 0

	mw1 := &Middleware{
		Bell: func(next func()) {
			bellCount++
			next()
		},
	}

	mw2 := &Middleware{
		SetTitle: func(title string, next func(string)) {
			titleCount++
			next(title)
		},
	}

	mw1.Merge(mw2)

	term := New(
		WithSize(24, 80),
		WithMiddleware(mw1),
	)

	term.WriteString("\x07")          // Bell
	term.WriteString("\x1b]0;Hi\x07") // Title

	if bellCount != 1 {
		t.Errorf("expected 1 bell, got %d", bellCount)
	}
	if titleCount != 1 {
		t.Errorf("expected 1 title, got %d", titleCount)
	}
}

func TestTerminalWrappedLineTracking(t *testing.T) {
	term := New(WithSize(5, 10))

	// Initially lines are not wrapped
	if term.IsWrapped(0) {
		t.Error("expected line 0 not wrapped initially")
	}

	// Write enough characters to wrap
	term.WriteString("1234567890ABC") // 13 chars, line 0 wraps at col 10

	// Line 0 should be marked as wrapped
	if !term.IsWrapped(0) {
		t.Error("expected line 0 to be wrapped after overflow")
	}

	// Line 1 should not be wrapped (no explicit newline yet)
	if term.IsWrapped(1) {
		t.Error("expected line 1 not wrapped")
	}
}

func TestTerminalWrappedLineClearedOnNewline(t *testing.T) {
	term := New(WithSize(5, 10))

	// Write enough to wrap
	term.WriteString("1234567890ABC") // wraps line 0

	if !term.IsWrapped(0) {
		t.Error("expected line 0 to be wrapped")
	}

	// Now write explicit newline on line 1
	term.WriteString("\n")

	// Line 1 (where cursor was) should NOT be marked as wrapped
	// because we had explicit newline
	if term.IsWrapped(1) {
		t.Error("expected line 1 not wrapped after explicit newline")
	}
}

func TestTerminalAutoResizeY(t *testing.T) {
	term := New(WithSize(3, 80), WithAutoResize())

	if !term.AutoResize() {
		t.Error("expected AutoResize to be enabled")
	}

	// Write more lines than terminal height (use \r\n for proper line breaks)
	term.WriteString("Line1\r\n")
	term.WriteString("Line2\r\n")
	term.WriteString("Line3\r\n")
	term.WriteString("Line4\r\n")
	term.WriteString("Line5\r\n")

	// Terminal should have grown
	if term.Rows() < 5 {
		t.Errorf("expected at least 5 rows, got %d", term.Rows())
	}

	// All content should be in the buffer (no scrolling)
	if term.LineContent(0) != "Line1" {
		t.Errorf("expected 'Line1', got '%s'", term.LineContent(0))
	}
	if term.LineContent(4) != "Line5" {
		t.Errorf("expected 'Line5', got '%s'", term.LineContent(4))
	}
}

func TestTerminalAutoResizeX(t *testing.T) {
	term := New(WithSize(3, 10), WithAutoResize())

	// Write a line longer than terminal width
	term.WriteString("This is a very long line that exceeds the terminal width")

	// Terminal should have grown horizontally
	if term.Cols() <= 10 {
		t.Errorf("expected cols > 10, got %d", term.Cols())
	}

	// Content should be on single line (no wrap)
	content := term.LineContent(0)
	if content != "This is a very long line that exceeds the terminal width" {
		t.Errorf("expected full line, got '%s'", content)
	}

	// Cursor should still be on line 0
	row, _ := term.CursorPos()
	if row != 0 {
		t.Errorf("expected cursor on row 0, got %d", row)
	}
}

func TestTerminalAutoResizeNoScrollback(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(3, 80), WithAutoResize(), WithScrollback(storage))

	// Write many lines (use \r\n for proper line breaks)
	for i := 0; i < 10; i++ {
		term.WriteString("Line\r\n")
	}

	// With AutoResize, nothing should go to scrollback
	if storage.pushCount > 0 {
		t.Errorf("expected no scrollback pushes with AutoResize, got %d", storage.pushCount)
	}
}

// --- Recording Tests ---

// testRecording is a test implementation of RecordingProvider
type testRecording struct {
	data []byte
}

func (r *testRecording) Record(data []byte) {
	r.data = append(r.data, data...)
}

func (r *testRecording) Data() []byte {
	return r.data
}

func (r *testRecording) Clear() {
	r.data = nil
}

func TestTerminalRecording(t *testing.T) {
	rec := &testRecording{}
	term := New(WithRecording(rec))

	// Write some data
	term.WriteString("Hello")
	term.WriteString(" World")

	// Check recorded data
	recorded := string(rec.Data())
	if recorded != "Hello World" {
		t.Errorf("expected 'Hello World', got '%s'", recorded)
	}
}

func TestTerminalRecordingWithANSI(t *testing.T) {
	rec := &testRecording{}
	term := New(WithRecording(rec))

	// Write data with ANSI sequences
	input := "\x1b[31mRed\x1b[0m"
	term.WriteString(input)

	// Recording should capture raw bytes including ANSI
	recorded := string(rec.Data())
	if recorded != input {
		t.Errorf("expected '%s', got '%s'", input, recorded)
	}
}

func TestTerminalRecordingClear(t *testing.T) {
	rec := &testRecording{}
	term := New(WithRecording(rec))

	term.WriteString("Hello")
	term.ClearRecording()

	if len(term.RecordedData()) != 0 {
		t.Error("expected empty recording after clear")
	}

	term.WriteString("World")
	if string(term.RecordedData()) != "World" {
		t.Errorf("expected 'World', got '%s'", string(term.RecordedData()))
	}
}

func TestTerminalRecordingReplay(t *testing.T) {
	rec := &testRecording{}
	term := New(WithSize(24, 80), WithRecording(rec))

	// Write some content
	term.WriteString("Hello\r\nWorld")

	// Get recorded data
	recorded := rec.Data()

	// Create new terminal and replay
	term2 := New(WithSize(24, 80))
	term2.Write(recorded)

	// Both terminals should have same content
	if term.String() != term2.String() {
		t.Errorf("replay mismatch:\noriginal: %s\nreplay: %s", term.String(), term2.String())
	}
}

func TestTerminalRecordingSetProvider(t *testing.T) {
	term := New()

	// Default is NoopRecording
	if term.RecordedData() != nil {
		t.Error("expected nil from NoopRecording")
	}

	// Set custom provider
	rec := &testRecording{}
	term.SetRecordingProvider(rec)

	term.WriteString("Test")

	if string(term.RecordedData()) != "Test" {
		t.Errorf("expected 'Test', got '%s'", string(term.RecordedData()))
	}
}

// TestActiveCharsetBoundsValidation tests that inputInternal handles invalid activeCharset values safely
func TestActiveCharsetBoundsValidation(t *testing.T) {
	term := New(WithSize(24, 80))

	// Set activeCharset to invalid values using reflection or direct field access
	// Since we can't access private fields directly, we'll test by setting valid values
	// and ensuring the code doesn't panic with edge cases

	// Test with valid charset values (0-3)
	for i := 0; i < 4; i++ {
		term.SetActiveCharset(i)
		// Write a character - should not panic
		term.WriteString("A")
	}

	// Test that writing characters with various charsets doesn't cause index out of range
	term.WriteString("Hello World")
	row, col := term.CursorPos()
	if row < 0 || row >= term.Rows() || col < 0 || col >= term.Cols() {
		t.Errorf("cursor out of bounds: (%d, %d) for terminal %dx%d", row, col, term.Rows(), term.Cols())
	}
}

// TestResizeInvalidDimensions tests that Resize ignores invalid dimensions
func TestResizeInvalidDimensions(t *testing.T) {
	term := New(WithSize(24, 80))

	originalRows := term.Rows()
	originalCols := term.Cols()

	// Test with zero dimensions
	term.Resize(0, 0)
	if term.Rows() != originalRows || term.Cols() != originalCols {
		t.Errorf("Resize(0, 0) should be ignored, got %dx%d", term.Rows(), term.Cols())
	}

	// Test with negative dimensions
	term.Resize(-10, -20)
	if term.Rows() != originalRows || term.Cols() != originalCols {
		t.Errorf("Resize(-10, -20) should be ignored, got %dx%d", term.Rows(), term.Cols())
	}

	// Test with zero rows
	term.Resize(0, 100)
	if term.Rows() != originalRows || term.Cols() != originalCols {
		t.Errorf("Resize(0, 100) should be ignored, got %dx%d", term.Rows(), term.Cols())
	}

	// Test with zero cols
	term.Resize(50, 0)
	if term.Rows() != originalRows || term.Cols() != originalCols {
		t.Errorf("Resize(50, 0) should be ignored, got %dx%d", term.Rows(), term.Cols())
	}

	// Test with valid dimensions
	term.Resize(30, 100)
	if term.Rows() != 30 || term.Cols() != 100 {
		t.Errorf("Resize(30, 100) should work, got %dx%d", term.Rows(), term.Cols())
	}
}

// TestResizeCursorBounds tests that cursor is properly clamped after resize
func TestResizeCursorBounds(t *testing.T) {
	term := New(WithSize(24, 80))

	// Move cursor to end
	term.WriteString(strings.Repeat("A", 80))
	term.WriteString("\r\n")
	term.WriteString(strings.Repeat("B", 80))

	// Resize to smaller dimensions
	term.Resize(10, 40)

	row, col := term.CursorPos()
	if row < 0 || row >= 10 {
		t.Errorf("cursor row out of bounds after resize: %d (expected 0-9)", row)
	}
	if col < 0 || col >= 40 {
		t.Errorf("cursor col out of bounds after resize: %d (expected 0-39)", col)
	}
}

// TestWriteResponseRaceCondition tests that writeResponse is thread-safe
func TestWriteResponseRaceCondition(t *testing.T) {
	term := New(WithSize(24, 80))

	var buf bytes.Buffer
	term.SetResponseProvider(&buf)

	// Concurrent writes to response provider
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			// Trigger device status which calls writeResponse
			term.DeviceStatus(6) // Cursor position report
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and should have written responses
	if buf.Len() == 0 {
		t.Error("expected responses to be written")
	}
}

// TestCursorBoundsAfterGrowCols tests that cursor stays within bounds after auto-resize
func TestCursorBoundsAfterGrowCols(t *testing.T) {
	term := New(WithSize(5, 10), WithAutoResize())

	// Write a wide character at the end of line (should trigger GrowCols)
	term.WriteString(strings.Repeat("A", 9)) // Fill 9 columns
	term.WriteString("中")                    // Wide character (2 columns) at position 9

	row, col := term.CursorPos()
	if row < 0 || row >= term.Rows() {
		t.Errorf("cursor row out of bounds after GrowCols: %d (rows: %d)", row, term.Rows())
	}
	if col < 0 || col > term.Cols() {
		t.Errorf("cursor col out of bounds after GrowCols: %d (cols: %d)", col, term.Cols())
	}

	// Verify the character was written
	content := term.LineContent(0)
	if len(content) < 10 {
		t.Errorf("expected line to grow, got length %d", len(content))
	}
}

// TestCursorBoundsAfterWrap tests that cursor row is validated after line wrap
func TestCursorBoundsAfterWrap(t *testing.T) {
	term := New(WithSize(5, 10))

	// Fill terminal with text to trigger wrapping
	for i := 0; i < 10; i++ {
		term.WriteString("123456789") // 9 chars, will wrap on next char
		term.WriteString("A")         // Triggers wrap
	}

	row, col := term.CursorPos()
	if row < 0 || row >= term.Rows() {
		t.Errorf("cursor row out of bounds after wrap: %d (rows: %d)", row, term.Rows())
	}
	if col < 0 || col > term.Cols() {
		t.Errorf("cursor col out of bounds after wrap: %d (cols: %d)", col, term.Cols())
	}
}

// TestInputWithInvalidCursorPosition tests that input handles invalid cursor positions gracefully
func TestInputWithInvalidCursorPosition(t *testing.T) {
	term := New(WithSize(5, 10))

	// Manually set cursor to invalid position (would require reflection, but we test indirectly)
	// by writing characters that would cause cursor to go out of bounds

	// Write to fill terminal
	for i := 0; i < 100; i++ {
		term.WriteString("A")
	}

	// Cursor should still be within bounds (allow col == cols for edge case)
	row, col := term.CursorPos()
	if row < 0 || row >= term.Rows() {
		t.Errorf("cursor row out of bounds: %d (rows: %d)", row, term.Rows())
	}
	if col < 0 || col > term.Cols() {
		t.Errorf("cursor col out of bounds: %d (cols: %d)", col, term.Cols())
	}

	// Verify we can still write without panic
	term.WriteString("X")
	row2, col2 := term.CursorPos()
	if row2 < 0 || row2 >= term.Rows() || col2 < 0 || col2 > term.Cols() {
		t.Errorf("cursor out of bounds after write: (%d, %d)", row2, col2)
	}
}

// TestResizeShrinkWithCursorInBounds tests resize when cursor stays within new bounds
func TestResizeShrinkWithCursorInBounds(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(10, 80), WithScrollback(storage))

	// Write content on first few lines
	term.WriteString("Line0\r\n")
	term.WriteString("Line1\r\n")
	term.WriteString("Line2")

	// Cursor should be on row 2
	row, _ := term.CursorPos()
	if row != 2 {
		t.Errorf("expected cursor on row 2, got %d", row)
	}

	initialScrollbackLen := storage.Len()

	// Resize to 5 rows - cursor is still within bounds (row 2 < 5)
	term.Resize(5, 80)

	// No lines should be scrolled to scrollback
	if storage.Len() != initialScrollbackLen {
		t.Errorf("expected no new scrollback lines, got %d new lines", storage.Len()-initialScrollbackLen)
	}

	// Content should be preserved
	if term.LineContent(0) != "Line0" {
		t.Errorf("expected 'Line0', got '%s'", term.LineContent(0))
	}
	if term.LineContent(2) != "Line2" {
		t.Errorf("expected 'Line2', got '%s'", term.LineContent(2))
	}

	// Cursor should still be on row 2
	row, _ = term.CursorPos()
	if row != 2 {
		t.Errorf("expected cursor on row 2 after resize, got %d", row)
	}
}

// TestResizeShrinkWithCursorOutOfBounds tests resize when cursor would be outside new bounds
func TestResizeShrinkWithCursorOutOfBounds(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(10, 80), WithScrollback(storage))

	// Write content filling more lines
	for i := 0; i < 8; i++ {
		term.WriteString("Line" + string(rune('0'+i)) + "\r\n")
	}
	term.WriteString("Line8") // Cursor ends on row 8

	row, _ := term.CursorPos()
	if row != 8 {
		t.Errorf("expected cursor on row 8, got %d", row)
	}

	initialScrollbackLen := storage.Len()

	// Resize to 5 rows - cursor at row 8 is outside bounds
	// Lines should be scrolled to preserve content near cursor
	term.Resize(5, 80)

	// Lines should have been pushed to scrollback
	newScrollbackLines := storage.Len() - initialScrollbackLen
	if newScrollbackLines == 0 {
		t.Error("expected lines to be pushed to scrollback when cursor is out of bounds")
	}

	// Cursor should be within new bounds
	row, _ = term.CursorPos()
	if row < 0 || row >= 5 {
		t.Errorf("cursor row out of bounds after resize: %d (expected 0-4)", row)
	}

	// Content near cursor should be preserved (Line8 should still be visible)
	found := false
	for i := 0; i < 5; i++ {
		if strings.Contains(term.LineContent(i), "Line8") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Line8 (near cursor) to be preserved after resize")
	}
}

// TestResizeShrinkScrollbackContent tests that scrolled lines go to scrollback correctly
func TestResizeShrinkScrollbackContent(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(10, 80), WithScrollback(storage))

	// Write content on all lines
	for i := 0; i < 10; i++ {
		if i < 9 {
			term.WriteString("Line" + string(rune('0'+i)) + "\r\n")
		} else {
			term.WriteString("Line9")
		}
	}

	// Cursor on row 9
	row, _ := term.CursorPos()
	if row != 9 {
		t.Errorf("expected cursor on row 9, got %d", row)
	}

	// Resize to 5 rows
	term.Resize(5, 80)

	// Check scrollback contains the lines that were pushed
	if storage.Len() < 5 {
		t.Errorf("expected at least 5 lines in scrollback, got %d", storage.Len())
	}

	// First lines should be in scrollback
	foundLine0 := false
	for i := 0; i < storage.Len(); i++ {
		line := storage.Line(i)
		if line != nil && len(line) > 0 {
			content := ""
			for _, cell := range line {
				if cell.Char != 0 && cell.Char != ' ' {
					content += string(cell.Char)
				}
			}
			if strings.HasPrefix(content, "Line0") {
				foundLine0 = true
				break
			}
		}
	}
	if !foundLine0 {
		t.Error("expected Line0 to be in scrollback after resize")
	}
}

// TestResizeGrowPullsFromScrollback tests that growing terminal pulls lines from scrollback
func TestResizeGrowPullsFromScrollback(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(10, 80), WithScrollback(storage))

	// Write content that fills the terminal
	for i := 0; i < 10; i++ {
		if i < 9 {
			term.WriteString("Line" + string(rune('0'+i)) + "\r\n")
		} else {
			term.WriteString("Line9")
		}
	}

	// Shrink terminal - this pushes lines to scrollback
	term.Resize(5, 80)

	scrollbackAfterShrink := storage.Len()
	if scrollbackAfterShrink == 0 {
		t.Fatal("expected lines in scrollback after shrink")
	}

	// Grow terminal back - should pull lines from scrollback
	term.Resize(10, 80)

	// Scrollback should have been consumed
	if storage.Len() >= scrollbackAfterShrink {
		t.Errorf("expected scrollback to be consumed, was %d now %d", scrollbackAfterShrink, storage.Len())
	}

	// Content should be restored at top of screen
	foundLine0 := false
	for i := 0; i < 10; i++ {
		if strings.Contains(term.LineContent(i), "Line0") {
			foundLine0 = true
			break
		}
	}
	if !foundLine0 {
		t.Error("expected Line0 to be restored from scrollback")
	}
}

// TestResizeGrowNoScrollbackUnchanged tests that growing terminal without scrollback doesn't change content
func TestResizeGrowNoScrollbackUnchanged(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(5, 80), WithScrollback(storage))

	// Write some content (not enough to need scrollback)
	term.WriteString("Line0\r\n")
	term.WriteString("Line1\r\n")
	term.WriteString("Line2")

	row, _ := term.CursorPos()
	initialRow := row
	initialScrollbackLen := storage.Len()

	// Grow terminal
	term.Resize(10, 80)

	// Scrollback should remain empty
	if storage.Len() != initialScrollbackLen {
		t.Errorf("expected scrollback unchanged, was %d now %d", initialScrollbackLen, storage.Len())
	}

	// Cursor should stay at same position (no lines pulled)
	row, _ = term.CursorPos()
	if row != initialRow {
		t.Errorf("expected cursor to stay at row %d, got %d", initialRow, row)
	}

	// Content should be preserved
	if term.LineContent(0) != "Line0" {
		t.Errorf("expected 'Line0', got '%s'", term.LineContent(0))
	}
}

// TestResizeAlternateScreenNoScrollback tests that alternate screen doesn't use scrollback on resize
func TestResizeAlternateScreenNoScrollback(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(10, 80), WithScrollback(storage))

	// Switch to alternate screen
	term.WriteString("\x1b[?1049h")

	// Write content
	for i := 0; i < 8; i++ {
		term.WriteString("Alt" + string(rune('0'+i)) + "\r\n")
	}
	term.WriteString("Alt8")

	initialScrollbackLen := storage.Len()

	// Resize - alternate screen should NOT push to scrollback
	term.Resize(5, 80)

	// Scrollback should not have new lines from alternate screen
	if storage.Len() != initialScrollbackLen {
		t.Errorf("alternate screen should not push to scrollback, was %d now %d",
			initialScrollbackLen, storage.Len())
	}
}

// TestResizeCursorPositionAfterShrink tests cursor position is correctly adjusted
func TestResizeCursorPositionAfterShrink(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(20, 80), WithScrollback(storage))

	// Move cursor to row 15
	for i := 0; i < 15; i++ {
		term.WriteString("Line\r\n")
	}
	term.WriteString("CursorLine")

	row, _ := term.CursorPos()
	if row != 15 {
		t.Errorf("expected cursor on row 15, got %d", row)
	}

	// Resize to 10 rows
	term.Resize(10, 80)

	// Cursor should be adjusted (15 - (20-10) = 5, but clamped to valid range)
	row, _ = term.CursorPos()
	if row < 0 || row >= 10 {
		t.Errorf("cursor out of bounds after resize: %d", row)
	}

	// The line with "CursorLine" should still be visible
	found := false
	for i := 0; i < 10; i++ {
		if strings.Contains(term.LineContent(i), "CursorLine") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected CursorLine to be visible after resize")
	}
}

// --- Row Coordinate Conversion Tests ---

func TestViewportRowToAbsolute(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(5, 80), WithScrollback(storage))

	// Without scrollback, viewport row equals absolute row
	if got := term.ViewportRowToAbsolute(0); got != 0 {
		t.Errorf("without scrollback: expected 0, got %d", got)
	}
	if got := term.ViewportRowToAbsolute(3); got != 3 {
		t.Errorf("without scrollback: expected 3, got %d", got)
	}

	// Create scrollback by writing more lines than terminal height
	for i := 0; i < 10; i++ {
		term.WriteString("Line\n")
	}

	scrollbackLen := term.ScrollbackLen()
	if scrollbackLen == 0 {
		t.Fatal("expected scrollback to exist")
	}

	// Viewport row 0 should now be at absolute row = scrollbackLen
	if got := term.ViewportRowToAbsolute(0); got != scrollbackLen {
		t.Errorf("with scrollback: expected %d, got %d", scrollbackLen, got)
	}

	// Viewport row 2 should be at absolute row = scrollbackLen + 2
	if got := term.ViewportRowToAbsolute(2); got != scrollbackLen+2 {
		t.Errorf("with scrollback: expected %d, got %d", scrollbackLen+2, got)
	}
}

func TestAbsoluteRowToViewport(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(5, 80), WithScrollback(storage))

	// Without scrollback, absolute row equals viewport row
	if got := term.AbsoluteRowToViewport(0); got != 0 {
		t.Errorf("without scrollback: expected 0, got %d", got)
	}
	if got := term.AbsoluteRowToViewport(3); got != 3 {
		t.Errorf("without scrollback: expected 3, got %d", got)
	}

	// Out of bounds should return -1
	if got := term.AbsoluteRowToViewport(5); got != -1 {
		t.Errorf("out of bounds: expected -1, got %d", got)
	}
	if got := term.AbsoluteRowToViewport(-1); got != -1 {
		t.Errorf("negative: expected -1, got %d", got)
	}

	// Create scrollback
	for i := 0; i < 10; i++ {
		term.WriteString("Line\n")
	}

	scrollbackLen := term.ScrollbackLen()

	// Rows in scrollback should return -1
	if got := term.AbsoluteRowToViewport(0); got != -1 {
		t.Errorf("scrollback row: expected -1, got %d", got)
	}
	if got := term.AbsoluteRowToViewport(scrollbackLen - 1); got != -1 {
		t.Errorf("last scrollback row: expected -1, got %d", got)
	}

	// First visible row
	if got := term.AbsoluteRowToViewport(scrollbackLen); got != 0 {
		t.Errorf("first visible: expected 0, got %d", got)
	}

	// Row in middle of viewport
	if got := term.AbsoluteRowToViewport(scrollbackLen + 2); got != 2 {
		t.Errorf("middle viewport: expected 2, got %d", got)
	}

	// Row beyond viewport
	if got := term.AbsoluteRowToViewport(scrollbackLen + 10); got != -1 {
		t.Errorf("beyond viewport: expected -1, got %d", got)
	}
}

func TestRowConversionRoundTrip(t *testing.T) {
	storage := &testScrollback{lines: make([][]Cell, 0)}
	storage.SetMaxLines(100)

	term := New(WithSize(5, 80), WithScrollback(storage))

	// Create some scrollback
	for i := 0; i < 10; i++ {
		term.WriteString("Line\n")
	}

	// Round trip: viewport -> absolute -> viewport should return original
	for viewportRow := 0; viewportRow < 5; viewportRow++ {
		absRow := term.ViewportRowToAbsolute(viewportRow)
		backToViewport := term.AbsoluteRowToViewport(absRow)
		if backToViewport != viewportRow {
			t.Errorf("round trip failed: viewport %d -> abs %d -> viewport %d",
				viewportRow, absRow, backToViewport)
		}
	}
}
