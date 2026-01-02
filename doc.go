// Package headlessterm provides a headless VT220-compatible terminal emulator.
//
// This package emulates a terminal without any display, making it ideal for:
//   - Testing terminal applications without a GUI
//   - Building terminal multiplexers and recorders
//   - Creating terminal-based web applications
//   - Automated testing of CLI tools
//   - Screen scraping and automation
//
// # Quick Start
//
// Create a terminal and write ANSI sequences to it:
//
//	term := headlessterm.New()
//	term.WriteString("\x1b[31mHello \x1b[32mWorld\x1b[0m!")
//	fmt.Println(term.String()) // "Hello World!"
//
// # Architecture
//
// The package is organized around these core types:
//
//   - [Terminal]: The main emulator that processes ANSI sequences
//   - [Buffer]: A 2D grid of cells with scrollback support
//   - [Cell]: A single character with colors and attributes
//   - [Cursor]: Tracks position and rendering style
//
// # Terminal
//
// Terminal is the main entry point. It implements [io.Writer] so you can write
// raw bytes containing ANSI escape sequences:
//
//	term := headlessterm.New(
//	    headlessterm.WithSize(24, 80),           // 24 rows, 80 columns
//	    headlessterm.WithScrollback(storage),    // Enable scrollback
//	    headlessterm.WithResponse(ptyWriter),    // Handle terminal responses
//	)
//
//	// Process output from a command
//	cmd := exec.Command("ls", "-la", "--color")
//	cmd.Stdout = term
//	cmd.Run()
//
//	// Read the result
//	for row := 0; row < term.Rows(); row++ {
//	    fmt.Println(term.LineContent(row))
//	}
//
// # Dual Buffers
//
// Terminal maintains two buffers:
//
//   - Primary buffer: Normal mode with optional scrollback storage
//   - Alternate buffer: Used by full-screen apps (vim, less, htop), no scrollback
//
// Applications switch buffers via ANSI sequences (CSI ?1049h/l). Check which
// buffer is active:
//
//	if term.IsAlternateScreen() {
//	    // Full-screen app is running
//	}
//
// # Cells and Attributes
//
// Each cell stores a character with styling information:
//
//	cell := term.Cell(row, col)
//	if cell != nil {
//	    fmt.Printf("Char: %c\n", cell.Char)
//	    fmt.Printf("Bold: %v\n", cell.HasFlag(headlessterm.CellFlagBold))
//	    fmt.Printf("FG: %v\n", cell.Fg)
//	    fmt.Printf("BG: %v\n", cell.Bg)
//	}
//
// Cell flags include: Bold, Dim, Italic, Underline, Blink, Reverse, Hidden, Strike.
//
// # Colors
//
// Colors are stored using Go's [image/color] interface. The package supports:
//
//   - Named colors (indices 0-15 for standard ANSI colors)
//   - 256-color palette (indices 0-255)
//   - True color (24-bit RGB via [color.RGBA])
//
// Use [ResolveDefaultColor] to convert any color to RGBA:
//
//	rgba := headlessterm.ResolveDefaultColor(cell.Fg, true)
//
// # Scrollback
//
// Lines scrolled off the top of the primary buffer can be stored for later access.
// Implement [ScrollbackProvider] or use the built-in memory storage:
//
//	// In-memory scrollback with 10000 line limit
//	storage := headlessterm.NewMemoryScrollback(10000)
//	term := headlessterm.New(headlessterm.WithScrollback(storage))
//
//	// Access scrollback
//	for i := 0; i < term.ScrollbackLen(); i++ {
//	    line := term.ScrollbackLine(i) // []Cell
//	}
//
// # PTY Writer
//
// [PTYWriter] writes terminal responses back to the PTY (cursor position reports, etc.):
//
//	term := headlessterm.New(headlessterm.WithPTYWriter(os.Stdout))
//
// # Providers
//
// Providers handle terminal events and queries. All are optional with no-op defaults:
//
//   - [BellProvider]: Handles bell/beep events
//   - [TitleProvider]: Handles window title changes (OSC 0/1/2)
//   - [ClipboardProvider]: Handles clipboard operations (OSC 52)
//   - [ScrollbackProvider]: Stores lines scrolled off screen
//   - [RecordingProvider]: Captures raw input for replay
//   - [SizeProvider]: Provides pixel dimensions for queries
//   - [SemanticPromptHandler]: Handles semantic prompt marks (OSC 133)
//
// Example with providers:
//
//	term := headlessterm.New(
//	    headlessterm.WithPTYWriter(os.Stdout),
//	    headlessterm.WithBell(&MyBellHandler{}),
//	    headlessterm.WithTitle(&MyTitleHandler{}),
//	)
//
// # Middleware
//
// Middleware intercepts ANSI handler calls for custom behavior:
//
//	mw := &headlessterm.Middleware{
//	    Input: func(r rune, next func(rune)) {
//	        log.Printf("Input: %c", r)
//	        next(r) // Call default handler
//	    },
//	    Bell: func(next func()) {
//	        log.Println("Bell!")
//	        // Don't call next() to suppress the bell
//	    },
//	}
//	term := headlessterm.New(headlessterm.WithMiddleware(mw))
//
// # Terminal Modes
//
// Various terminal behaviors are controlled by mode flags:
//
//	term.HasMode(headlessterm.ModeLineWrap)       // Auto line wrap enabled?
//	term.HasMode(headlessterm.ModeShowCursor)     // Cursor visible?
//	term.HasMode(headlessterm.ModeBracketedPaste) // Bracketed paste enabled?
//
// See [TerminalMode] for all available modes.
//
// # Dirty Tracking
//
// Track which cells changed for efficient rendering:
//
//	if term.HasDirty() {
//	    for _, pos := range term.DirtyCells() {
//	        // Redraw cell at pos.Row, pos.Col
//	    }
//	    term.ClearDirty()
//	}
//
// # Selection
//
// Manage text selections for copy/paste:
//
//	term.SetSelection(
//	    headlessterm.Position{Row: 0, Col: 0},
//	    headlessterm.Position{Row: 2, Col: 10},
//	)
//	text := term.GetSelectedText()
//	term.ClearSelection()
//
// # Search
//
// Find text in the visible screen or scrollback:
//
//	matches := term.Search("error")
//	for _, pos := range matches {
//	    fmt.Printf("Found at row %d, col %d\n", pos.Row, pos.Col)
//	}
//
//	// Search scrollback (returns negative row numbers)
//	scrollbackMatches := term.SearchScrollback("error")
//
// # Snapshots
//
// Capture the terminal state for serialization or rendering:
//
//	// Text only (smallest)
//	snap := term.Snapshot(headlessterm.SnapshotDetailText)
//
//	// With style segments (good for HTML rendering)
//	snap := term.Snapshot(headlessterm.SnapshotDetailStyled)
//
//	// Full cell data (complete state, includes image references)
//	snap := term.Snapshot(headlessterm.SnapshotDetailFull)
//
//	// Convert to JSON
//	data, _ := json.Marshal(snap)
//
// Snapshots include detailed attribute information:
//   - Underline styles: "single", "double", "curly", "dotted", "dashed"
//   - Blink types: "slow", "fast"
//   - Underline color (separate from foreground)
//   - Cell image references with UV coordinates for texture mapping
//
// # Image Support
//
// The terminal supports inline images via Sixel and Kitty graphics protocols:
//
//	// Check if images are enabled
//	if term.SixelEnabled() || term.KittyEnabled() {
//	    // Process image sequences
//	}
//
//	// Access stored images
//	for _, placement := range term.ImagePlacements() {
//	    img := term.Image(placement.ImageID)
//	    // img.Data contains RGBA pixels
//	}
//
//	// Configure image memory budget
//	term.SetImageMaxMemory(100 * 1024 * 1024) // 100MB
//
// # Shell Integration
//
// Track shell prompts and command output (OSC 133):
//
//	term := headlessterm.New(
//	    headlessterm.WithSemanticPromptHandler(&MyHandler{}),
//	)
//
//	// Navigate between prompts (uses absolute rows, including scrollback)
//	currentAbsRow := term.ViewportRowToAbsolute(0) // Convert viewport row to absolute
//	nextAbsRow := term.NextPromptRow(currentAbsRow, -1)
//	prevAbsRow := term.PrevPromptRow(currentAbsRow, -1)
//
//	// Convert absolute row back to viewport for display
//	viewportRow := term.AbsoluteRowToViewport(nextAbsRow) // -1 if in scrollback
//
//	// Get last command output
//	output := term.GetLastCommandOutput()
//
// # Auto-Resize Mode
//
// In auto-resize mode, the buffer grows instead of scrolling:
//
//	term := headlessterm.New(headlessterm.WithAutoResize())
//
//	// Capture complete output without truncation
//	cmd.Stdout = term
//	cmd.Run()
//
//	// Buffer has grown to fit all output
//	fmt.Printf("Total rows: %d\n", term.Rows())
//
// # Thread Safety
//
// All Terminal methods are safe for concurrent use. The terminal uses internal
// locking to protect state. However, if you need to perform multiple operations
// atomically, you should use your own synchronization.
//
// # Supported ANSI Sequences
//
// The terminal supports a comprehensive set of ANSI escape sequences including:
//
//   - Cursor movement (CUU, CUD, CUF, CUB, CUP, HVP, etc.)
//   - Cursor save/restore (DECSC, DECRC)
//   - Erase commands (ED, EL, ECH)
//   - Insert/delete (ICH, DCH, IL, DL)
//   - Scrolling (SU, SD, DECSTBM)
//   - Character attributes (SGR) with full color support
//   - Terminal modes (DECSET, DECRST)
//   - Device status reports (DSR)
//   - Alternate screen buffer
//   - Bracketed paste mode
//   - Mouse reporting
//   - Window title (OSC 0/1/2)
//   - Clipboard (OSC 52)
//   - Hyperlinks (OSC 8)
//   - Shell integration (OSC 133)
//   - Sixel and Kitty graphics
//
// For the complete list of supported sequences, see the [go-ansicode] package
// documentation.
//
// [go-ansicode]: https://github.com/danielgatis/go-ansicode
package headlessterm
