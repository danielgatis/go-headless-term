package headlessterm

import (
	"encoding/base64"
	"fmt"
	"image/color"

	"github.com/danielgatis/go-ansicode"
)

// ApplicationCommandReceived processes an APC sequence and delegates to the configured provider.
func (t *Terminal) ApplicationCommandReceived(data []byte) {
	if t.middleware != nil && t.middleware.ApplicationCommandReceived != nil {
		t.middleware.ApplicationCommandReceived(data, t.applicationCommandReceivedInternal)
		return
	}
	t.applicationCommandReceivedInternal(data)
}

func (t *Terminal) applicationCommandReceivedInternal(data []byte) {
	// Check for Kitty graphics protocol (starts with 'G')
	if len(data) > 0 && data[0] == 'G' {
		t.handleKittyGraphics(data)
		return
	}

	// Forward to APC provider for other APC sequences
	if t.apcProvider != nil {
		t.apcProvider.Receive(data)
	}
}

// handleKittyGraphics processes a Kitty graphics protocol command.
func (t *Terminal) handleKittyGraphics(data []byte) {
	cmd, err := ParseKittyGraphics(data)
	if err != nil {
		return
	}

	switch cmd.Action {
	case KittyActionQuery:
		// Respond that we support the protocol
		if cmd.Quiet < 2 {
			response := FormatKittyResponse(cmd.ImageID, "", false)
			t.writeResponseString(response)
		}

	case KittyActionTransmit:
		// Transmit image data (store but don't display yet)
		t.kittyTransmit(cmd)

	case KittyActionTransmitDisplay:
		// Transmit and display immediately
		t.kittyTransmit(cmd)
		if !cmd.More {
			t.kittyDisplay(cmd)
		}

	case KittyActionDisplay:
		// Display an already transmitted image
		t.kittyDisplay(cmd)

	case KittyActionDelete:
		// Delete image(s)
		t.kittyDelete(cmd)
	}
}

// kittyTransmit handles image data transmission.
func (t *Terminal) kittyTransmit(cmd *KittyCommand) {
	// Handle chunked transfer
	if cmd.More {
		t.images.mu.Lock()
		t.images.accumulator = append(t.images.accumulator, cmd.Payload...)
		t.images.accumulatorID = cmd.ImageID
		t.images.accumulatorMore = true
		t.images.mu.Unlock()
		return
	}

	// Get complete payload
	var payload []byte
	t.images.mu.Lock()
	if t.images.accumulatorMore {
		payload = append(t.images.accumulator, cmd.Payload...)
		t.images.accumulator = nil
		t.images.accumulatorMore = false
	} else {
		payload = cmd.Payload
	}
	t.images.mu.Unlock()

	// Update command with complete payload
	cmd.Payload = payload

	// Decode image data
	rgba, width, height, err := cmd.DecodeImageData()
	if err != nil || width == 0 || height == 0 {
		if cmd.Quiet < 2 {
			response := FormatKittyResponse(cmd.ImageID, "ENODATA", true)
			t.writeResponseString(response)
		}
		return
	}

	// Store the image
	if cmd.ImageID > 0 {
		t.images.StoreWithID(cmd.ImageID, width, height, rgba)
	} else {
		cmd.ImageID = t.images.Store(width, height, rgba)
	}

	// Send OK response
	if cmd.Quiet < 1 {
		response := FormatKittyResponse(cmd.ImageID, "", false)
		t.writeResponseString(response)
	}
}

// kittyDisplay displays an image at the current cursor position.
func (t *Terminal) kittyDisplay(cmd *KittyCommand) {
	img := t.images.Image(cmd.ImageID)
	if img == nil {
		if cmd.Quiet < 2 {
			response := FormatKittyResponse(cmd.ImageID, "ENOENT", true)
			t.writeResponseString(response)
		}
		return
	}

	// Calculate cell coverage
	cellW, cellH := t.getCellSizePixels()

	// Determine source region
	srcW := cmd.SrcW
	srcH := cmd.SrcH
	if srcW == 0 {
		srcW = img.Width - cmd.SrcX
	}
	if srcH == 0 {
		srcH = img.Height - cmd.SrcY
	}

	// Determine target size in cells
	cols := int(cmd.Cols)
	rows := int(cmd.Rows)
	if cols == 0 {
		cols = int((srcW + uint32(cellW) - 1) / uint32(cellW))
	}
	if rows == 0 {
		rows = int((srcH + uint32(cellH) - 1) / uint32(cellH))
	}

	// Get cursor position
	t.mu.Lock()
	curRow := t.cursor.Row
	curCol := t.cursor.Col
	t.mu.Unlock()

	// Create placement
	placement := &ImagePlacement{
		ImageID: cmd.ImageID,
		Row:     curRow,
		Col:     curCol,
		Cols:    cols,
		Rows:    rows,
		SrcX:    cmd.SrcX,
		SrcY:    cmd.SrcY,
		SrcW:    srcW,
		SrcH:    srcH,
		ZIndex:  cmd.ZIndex,
		OffsetX: cmd.CellOffsetX,
		OffsetY: cmd.CellOffsetY,
	}

	placementID := t.images.Place(placement)

	// Assign image references to cells
	t.assignImageToCells(cmd.ImageID, placementID, placement, img.Width, img.Height, cellW, cellH)

	// Move cursor if not suppressed
	if !cmd.DoNotMoveCursor {
		t.mu.Lock()
		t.cursor.Col += cols
		if t.cursor.Col >= t.cols {
			t.cursor.Col = 0
			t.cursor.Row++
			if t.cursor.Row >= t.rows {
				t.cursor.Row = t.rows - 1
			}
		}
		t.mu.Unlock()
	}

	// Send OK response
	if cmd.Quiet < 1 {
		response := FormatKittyResponse(cmd.ImageID, "", false)
		t.writeResponseString(response)
	}
}

// kittyDelete handles image deletion commands.
func (t *Terminal) kittyDelete(cmd *KittyCommand) {
	t.mu.Lock()
	curRow := t.cursor.Row
	curCol := t.cursor.Col
	t.mu.Unlock()

	switch cmd.Delete {
	case KittyDeleteAll, KittyDeleteAllWithData:
		// Delete all visible placements
		t.images.Clear()

	case KittyDeleteByID, KittyDeleteByIDWithData:
		t.images.RemovePlacementsForImage(cmd.ImageID)
		if cmd.Delete == KittyDeleteByIDWithData {
			t.images.DeleteImage(cmd.ImageID)
		}

	case KittyDeleteAtCursor, KittyDeleteAtCursorData:
		t.images.DeletePlacementsByPosition(curRow, curCol)

	case KittyDeleteByCol, KittyDeleteByColData:
		t.images.DeletePlacementsInColumn(curCol)

	case KittyDeleteByRow, KittyDeleteByRowData:
		t.images.DeletePlacementsInRow(curRow)

	case KittyDeleteByZIndex, KittyDeleteByZIndexData:
		t.images.DeletePlacementsByZIndex(cmd.ZIndex)
	}
}

// Backspace moves the cursor one column left, stopping at column 0.
func (t *Terminal) Backspace() {
	if t.middleware != nil && t.middleware.Backspace != nil {
		t.middleware.Backspace(t.backspaceInternal)
		return
	}
	t.backspaceInternal()
}

func (t *Terminal) backspaceInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cursor.Col > 0 {
		t.cursor.Col--
	}
}

// Bell triggers the bell provider if configured.
func (t *Terminal) Bell() {
	if t.middleware != nil && t.middleware.Bell != nil {
		t.middleware.Bell(t.bellInternal)
		return
	}
	t.bellInternal()
}

func (t *Terminal) bellInternal() {
	if t.bellProvider != nil {
		t.bellProvider.Ring()
	}
}

// CarriageReturn moves the cursor to column 0 of the current row.
func (t *Terminal) CarriageReturn() {
	if t.middleware != nil && t.middleware.CarriageReturn != nil {
		t.middleware.CarriageReturn(t.carriageReturnInternal)
		return
	}
	t.carriageReturnInternal()
}

func (t *Terminal) carriageReturnInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursor.Col = 0
}

// ClearLine clears portions of the current line based on mode (right of cursor, left of cursor, or entire line).
func (t *Terminal) ClearLine(mode ansicode.LineClearMode) {
	if t.middleware != nil && t.middleware.ClearLine != nil {
		t.middleware.ClearLine(mode, t.clearLineInternal)
		return
	}
	t.clearLineInternal(mode)
}

func (t *Terminal) clearLineInternal(mode ansicode.LineClearMode) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch mode {
	case ansicode.LineClearModeRight:
		t.activeBuffer.ClearRowRange(t.cursor.Row, t.cursor.Col, t.cols)
	case ansicode.LineClearModeLeft:
		t.activeBuffer.ClearRowRange(t.cursor.Row, 0, t.cursor.Col+1)
	case ansicode.LineClearModeAll:
		t.activeBuffer.ClearRow(t.cursor.Row)
	}
}

// ClearScreen clears screen regions based on mode (below cursor, above cursor, entire screen, or saved lines).
func (t *Terminal) ClearScreen(mode ansicode.ClearMode) {
	if t.middleware != nil && t.middleware.ClearScreen != nil {
		t.middleware.ClearScreen(mode, t.clearScreenInternal)
		return
	}
	t.clearScreenInternal(mode)
}

func (t *Terminal) clearScreenInternal(mode ansicode.ClearMode) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch mode {
	case ansicode.ClearModeBelow:
		// Clear from cursor to end of screen
		t.activeBuffer.ClearRowRange(t.cursor.Row, t.cursor.Col, t.cols)
		for row := t.cursor.Row + 1; row < t.rows; row++ {
			t.activeBuffer.ClearRow(row)
		}
	case ansicode.ClearModeAbove:
		// Clear from beginning to cursor
		for row := 0; row < t.cursor.Row; row++ {
			t.activeBuffer.ClearRow(row)
		}
		t.activeBuffer.ClearRowRange(t.cursor.Row, 0, t.cursor.Col+1)
	case ansicode.ClearModeAll:
		t.activeBuffer.ClearAll()
	case ansicode.ClearModeSaved:
		// Clear saved lines (scrollback) - not implemented for now
		t.activeBuffer.ClearAll()
	}
}

// ClearTabs removes tab stops at the current column or all columns based on mode.
func (t *Terminal) ClearTabs(mode ansicode.TabulationClearMode) {
	if t.middleware != nil && t.middleware.ClearTabs != nil {
		t.middleware.ClearTabs(mode, t.clearTabsInternal)
		return
	}
	t.clearTabsInternal(mode)
}

func (t *Terminal) clearTabsInternal(mode ansicode.TabulationClearMode) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch mode {
	case ansicode.TabulationClearModeCurrent:
		t.activeBuffer.ClearTabStop(t.cursor.Col)
	case ansicode.TabulationClearModeAll:
		t.activeBuffer.ClearAllTabStops()
	}
}

// ClipboardLoad reads from the clipboard provider and sends the response via OSC 52.
func (t *Terminal) ClipboardLoad(clipboard byte, terminator string) {
	if t.middleware != nil && t.middleware.ClipboardLoad != nil {
		t.middleware.ClipboardLoad(clipboard, terminator, t.clipboardLoadInternal)
		return
	}
	t.clipboardLoadInternal(clipboard, terminator)
}

func (t *Terminal) clipboardLoadInternal(clipboard byte, terminator string) {
	if t.clipboardProvider == nil {
		return
	}
	content := t.clipboardProvider.Read(clipboard)
	if content != "" {
		// OSC 52 response: OSC 52 ; clipboard ; base64-data ST
		encoded := base64.StdEncoding.EncodeToString([]byte(content))
		response := "\x1b]52;" + string(clipboard) + ";" + encoded + terminator
		t.writeResponseString(response)
	}
}

// ClipboardStore writes data to the clipboard provider via OSC 52.
func (t *Terminal) ClipboardStore(clipboard byte, data []byte) {
	if t.middleware != nil && t.middleware.ClipboardStore != nil {
		t.middleware.ClipboardStore(clipboard, data, t.clipboardStoreInternal)
		return
	}
	t.clipboardStoreInternal(clipboard, data)
}

func (t *Terminal) clipboardStoreInternal(clipboard byte, data []byte) {
	if t.clipboardProvider != nil {
		t.clipboardProvider.Write(clipboard, data)
	}
}

// ConfigureCharset sets the character set for one of the four slots (G0-G3).
func (t *Terminal) ConfigureCharset(index ansicode.CharsetIndex, charset ansicode.Charset) {
	if t.middleware != nil && t.middleware.ConfigureCharset != nil {
		t.middleware.ConfigureCharset(index, charset, t.configureCharsetInternal)
		return
	}
	t.configureCharsetInternal(index, charset)
}

func (t *Terminal) configureCharsetInternal(index ansicode.CharsetIndex, charset ansicode.Charset) {
	t.mu.Lock()
	defer t.mu.Unlock()

	idx := CharsetIndex(index)
	cs := Charset(charset)

	if idx >= 0 && idx <= CharsetIndexG3 {
		t.charsets[idx] = cs
	}
}

// Decaln fills the entire screen with 'E' characters (DEC screen alignment test).
func (t *Terminal) Decaln() {
	if t.middleware != nil && t.middleware.Decaln != nil {
		t.middleware.Decaln(t.decalnInternal)
		return
	}
	t.decalnInternal()
}

func (t *Terminal) decalnInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.activeBuffer.FillWithE()
}

// DeleteChars removes n characters at the cursor, shifting remaining characters left.
func (t *Terminal) DeleteChars(n int) {
	if t.middleware != nil && t.middleware.DeleteChars != nil {
		t.middleware.DeleteChars(n, t.deleteCharsInternal)
		return
	}
	t.deleteCharsInternal(n)
}

func (t *Terminal) deleteCharsInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.activeBuffer.DeleteChars(t.cursor.Row, t.cursor.Col, n)
}

// DeleteLines removes n lines at the cursor within the scroll region, shifting remaining lines up.
func (t *Terminal) DeleteLines(n int) {
	if t.middleware != nil && t.middleware.DeleteLines != nil {
		t.middleware.DeleteLines(n, t.deleteLinesInternal)
		return
	}
	t.deleteLinesInternal(n)
}

func (t *Terminal) deleteLinesInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cursor.Row >= t.scrollTop && t.cursor.Row < t.scrollBottom {
		t.activeBuffer.DeleteLines(t.cursor.Row, n, t.scrollBottom)
	}
}

// DeviceStatus sends a device status report (DSR) response: ready (n=5) or cursor position (n=6).
func (t *Terminal) DeviceStatus(n int) {
	if t.middleware != nil && t.middleware.DeviceStatus != nil {
		t.middleware.DeviceStatus(n, t.deviceStatusInternal)
		return
	}
	t.deviceStatusInternal(n)
}

func (t *Terminal) deviceStatusInternal(n int) {
	t.mu.RLock()
	row := t.cursor.Row
	col := t.cursor.Col
	t.mu.RUnlock()

	var response string
	switch n {
	case 5:
		// Device status report - terminal is ready
		response = "\x1b[0n"
	case 6:
		// Cursor position report (1-based)
		response = fmt.Sprintf("\x1b[%d;%dR", row+1, col+1)
	}

	if response != "" {
		t.writeResponseString(response)
	}
}

// EraseChars resets n characters at the cursor to default state without shifting.
func (t *Terminal) EraseChars(n int) {
	if t.middleware != nil && t.middleware.EraseChars != nil {
		t.middleware.EraseChars(n, t.eraseCharsInternal)
		return
	}
	t.eraseCharsInternal(n)
}

func (t *Terminal) eraseCharsInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := 0; i < n && t.cursor.Col+i < t.cols; i++ {
		cell := t.activeBuffer.Cell(t.cursor.Row, t.cursor.Col+i)
		if cell != nil {
			cell.Reset()
		}
	}
}

// Goto moves the cursor to (row, col), adjusting for origin mode if enabled.
func (t *Terminal) Goto(row, col int) {
	if t.middleware != nil && t.middleware.Goto != nil {
		t.middleware.Goto(row, col, t.gotoInternal)
		return
	}
	t.gotoInternal(row, col)
}

func (t *Terminal) gotoInternal(row, col int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	row = t.effectiveRow(row)
	t.cursor.Row = clamp(row, 0, t.rows-1)
	t.cursor.Col = clamp(col, 0, t.cols-1)
}

// GotoCol moves the cursor to the specified column, keeping the current row.
func (t *Terminal) GotoCol(col int) {
	if t.middleware != nil && t.middleware.GotoCol != nil {
		t.middleware.GotoCol(col, t.gotoColInternal)
		return
	}
	t.gotoColInternal(col)
}

func (t *Terminal) gotoColInternal(col int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursor.Col = clamp(col, 0, t.cols-1)
}

// GotoLine moves the cursor to the specified row, adjusting for origin mode if enabled.
func (t *Terminal) GotoLine(row int) {
	if t.middleware != nil && t.middleware.GotoLine != nil {
		t.middleware.GotoLine(row, t.gotoLineInternal)
		return
	}
	t.gotoLineInternal(row)
}

func (t *Terminal) gotoLineInternal(row int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	row = t.effectiveRow(row)
	t.cursor.Row = clamp(row, 0, t.rows-1)
}

// HorizontalTabSet enables a tab stop at the current column.
func (t *Terminal) HorizontalTabSet() {
	if t.middleware != nil && t.middleware.HorizontalTabSet != nil {
		t.middleware.HorizontalTabSet(t.horizontalTabSetInternal)
		return
	}
	t.horizontalTabSetInternal()
}

func (t *Terminal) horizontalTabSetInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.activeBuffer.SetTabStop(t.cursor.Col)
}

// IdentifyTerminal sends a terminal identification response (default: VT220).
func (t *Terminal) IdentifyTerminal(b byte) {
	if t.middleware != nil && t.middleware.IdentifyTerminal != nil {
		t.middleware.IdentifyTerminal(b, t.identifyTerminalInternal)
		return
	}
	t.identifyTerminalInternal(b)
}

func (t *Terminal) identifyTerminalInternal(b byte) {
	// Default: identify as VT220
	response := "\x1b[?62;c"
	t.writeResponseString(response)
}

// Input writes a character to the buffer at the cursor position.
// Handles wide characters, line wrapping, insert mode, and charset translation.
func (t *Terminal) Input(r rune) {
	if t.middleware != nil && t.middleware.Input != nil {
		t.middleware.Input(r, t.inputInternal)
		return
	}
	t.inputInternal(r)
}

func (t *Terminal) inputInternal(r rune) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Handle line drawing charset (validate activeCharset to prevent index out of range)
	if t.activeCharset >= 0 && t.activeCharset < 4 && t.charsets[t.activeCharset] == CharsetLineDrawing {
		r = t.translateLineDrawing(r)
	}

	// Get the width of the character
	width := runeWidth(r)

	// Zero-width characters (combining marks) - attach to previous cell
	if width == 0 {
		// For now, just ignore zero-width characters
		// A full implementation would combine them with the previous character
		return
	}

	// Check if we need to wrap
	// For wide characters, we need 2 cells
	if t.cursor.Col+width > t.cols {
		if t.autoResize {
			// Grow the line instead of wrapping
			t.activeBuffer.GrowCols(t.cursor.Row, t.cursor.Col+width)
			t.cols = t.activeBuffer.Cols()
			// Revalidate cursor position after buffer growth
			if t.cursor.Col >= t.cols {
				t.cursor.Col = t.cols - 1
			}
		} else if t.modes&ModeLineWrap != 0 {
			// Mark the current line as wrapped (not explicit newline)
			t.activeBuffer.SetWrapped(t.cursor.Row, true)
			t.cursor.Col = 0
			t.cursor.Row++
			// Validate cursor row is within bounds before scrolling
			if t.cursor.Row >= t.rows {
				t.scrollIfNeeded()
			}
		} else {
			// Can't fit wide character at end of line
			if width == 2 {
				return
			}
			t.cursor.Col = t.cols - 1
		}
	}

	// Insert mode: shift characters to the right
	if t.modes&ModeInsert != 0 {
		t.activeBuffer.InsertBlanks(t.cursor.Row, t.cursor.Col, width)
	}

	// Validate cursor position is within bounds before writing
	if t.cursor.Row < 0 || t.cursor.Row >= t.rows || t.cursor.Col < 0 {
		return
	}

	// Write the character only if within column bounds
	if t.cursor.Col < t.cols {
		cell := t.activeBuffer.Cell(t.cursor.Row, t.cursor.Col)
		if cell != nil {
			cell.Char = r
			cell.Fg = t.template.Fg
			cell.Bg = t.template.Bg
			cell.UnderlineColor = t.template.UnderlineColor
			cell.Flags = t.template.Flags
			cell.Hyperlink = t.currentHyperlink

			// Mark as wide character if needed
			if width == 2 {
				cell.SetFlag(CellFlagWideChar)
			} else {
				cell.ClearFlag(CellFlagWideChar | CellFlagWideCharSpacer)
			}

			// Mark as dirty
			t.activeBuffer.MarkDirty(t.cursor.Row, t.cursor.Col)
		}
	}

	t.cursor.Col++

	// For wide characters, add a spacer cell
	if width == 2 && t.cursor.Col < t.cols {
		spacer := t.activeBuffer.Cell(t.cursor.Row, t.cursor.Col)
		if spacer != nil {
			spacer.Reset()
			spacer.Fg = t.template.Fg
			spacer.Bg = t.template.Bg
			spacer.SetFlag(CellFlagWideCharSpacer)
			t.activeBuffer.MarkDirty(t.cursor.Row, t.cursor.Col)
		}
		t.cursor.Col++
	}

	// Ensure cursor stays within bounds after all operations
	// Only clamp if we're not in a state that will handle overflow (wrap/scroll/auto-resize)
	if t.cursor.Col >= t.cols && !t.autoResize && t.modes&ModeLineWrap == 0 {
		t.cursor.Col = t.cols - 1
	}
	if t.cursor.Row >= t.rows && !t.autoResize {
		// scrollIfNeeded will handle this, but ensure we don't go beyond buffer
		if t.cursor.Row >= t.activeBuffer.Rows() {
			t.cursor.Row = t.activeBuffer.Rows() - 1
		}
	}
	if t.cursor.Col < 0 {
		t.cursor.Col = 0
	}
	if t.cursor.Row < 0 {
		t.cursor.Row = 0
	}
}

// translateLineDrawing translates characters for line drawing charset.
func (t *Terminal) translateLineDrawing(r rune) rune {
	switch r {
	case 'j':
		return '┘'
	case 'k':
		return '┐'
	case 'l':
		return '┌'
	case 'm':
		return '└'
	case 'n':
		return '┼'
	case 'q':
		return '─'
	case 't':
		return '├'
	case 'u':
		return '┤'
	case 'v':
		return '┴'
	case 'w':
		return '┬'
	case 'x':
		return '│'
	default:
		return r
	}
}

// InsertBlank inserts n blank cells at the cursor, shifting existing characters right.
func (t *Terminal) InsertBlank(n int) {
	if t.middleware != nil && t.middleware.InsertBlank != nil {
		t.middleware.InsertBlank(n, t.insertBlankInternal)
		return
	}
	t.insertBlankInternal(n)
}

func (t *Terminal) insertBlankInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.activeBuffer.InsertBlanks(t.cursor.Row, t.cursor.Col, n)
}

// InsertBlankLines inserts n blank lines at the cursor within the scroll region, shifting remaining lines down.
func (t *Terminal) InsertBlankLines(n int) {
	if t.middleware != nil && t.middleware.InsertBlankLines != nil {
		t.middleware.InsertBlankLines(n, t.insertBlankLinesInternal)
		return
	}
	t.insertBlankLinesInternal(n)
}

func (t *Terminal) insertBlankLinesInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cursor.Row >= t.scrollTop && t.cursor.Row < t.scrollBottom {
		t.activeBuffer.InsertLines(t.cursor.Row, n, t.scrollBottom)
	}
}

// LineFeed moves the cursor down one row. If ModeLineFeedNewLine is set, also moves to column 0.
// Clears the wrapped flag for the current line (indicates explicit newline).
func (t *Terminal) LineFeed() {
	if t.middleware != nil && t.middleware.LineFeed != nil {
		t.middleware.LineFeed(t.lineFeedInternal)
		return
	}
	t.lineFeedInternal()
}

func (t *Terminal) lineFeedInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Explicit newline clears the wrapped flag for this line
	t.activeBuffer.SetWrapped(t.cursor.Row, false)

	if t.modes&ModeLineFeedNewLine != 0 {
		t.cursor.Col = 0
	}

	t.cursor.Row++
	t.scrollIfNeeded()
}

// MoveBackward moves the cursor left n columns, stopping at column 0.
func (t *Terminal) MoveBackward(n int) {
	if t.middleware != nil && t.middleware.MoveBackward != nil {
		t.middleware.MoveBackward(n, t.moveBackwardInternal)
		return
	}
	t.moveBackwardInternal(n)
}

func (t *Terminal) moveBackwardInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursor.Col = clamp(t.cursor.Col-n, 0, t.cols-1)
}

// MoveBackwardTabs moves the cursor left to the previous n tab stops.
func (t *Terminal) MoveBackwardTabs(n int) {
	if t.middleware != nil && t.middleware.MoveBackwardTabs != nil {
		t.middleware.MoveBackwardTabs(n, t.moveBackwardTabsInternal)
		return
	}
	t.moveBackwardTabsInternal(n)
}

func (t *Terminal) moveBackwardTabsInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := 0; i < n; i++ {
		t.cursor.Col = t.activeBuffer.PrevTabStop(t.cursor.Col)
	}
}

// MoveDown moves the cursor down n rows, stopping at the last row.
func (t *Terminal) MoveDown(n int) {
	if t.middleware != nil && t.middleware.MoveDown != nil {
		t.middleware.MoveDown(n, t.moveDownInternal)
		return
	}
	t.moveDownInternal(n)
}

func (t *Terminal) moveDownInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursor.Row = clamp(t.cursor.Row+n, 0, t.rows-1)
}

// MoveDownCr moves the cursor down n rows and to column 0.
func (t *Terminal) MoveDownCr(n int) {
	if t.middleware != nil && t.middleware.MoveDownCr != nil {
		t.middleware.MoveDownCr(n, t.moveDownCrInternal)
		return
	}
	t.moveDownCrInternal(n)
}

func (t *Terminal) moveDownCrInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursor.Row = clamp(t.cursor.Row+n, 0, t.rows-1)
	t.cursor.Col = 0
}

// MoveForward moves the cursor right n columns, stopping at the last column.
func (t *Terminal) MoveForward(n int) {
	if t.middleware != nil && t.middleware.MoveForward != nil {
		t.middleware.MoveForward(n, t.moveForwardInternal)
		return
	}
	t.moveForwardInternal(n)
}

func (t *Terminal) moveForwardInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursor.Col = clamp(t.cursor.Col+n, 0, t.cols-1)
}

// MoveForwardTabs moves the cursor right to the next n tab stops.
func (t *Terminal) MoveForwardTabs(n int) {
	if t.middleware != nil && t.middleware.MoveForwardTabs != nil {
		t.middleware.MoveForwardTabs(n, t.moveForwardTabsInternal)
		return
	}
	t.moveForwardTabsInternal(n)
}

func (t *Terminal) moveForwardTabsInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := 0; i < n; i++ {
		t.cursor.Col = t.activeBuffer.NextTabStop(t.cursor.Col)
	}
}

// MoveUp moves the cursor up n rows, stopping at row 0.
func (t *Terminal) MoveUp(n int) {
	if t.middleware != nil && t.middleware.MoveUp != nil {
		t.middleware.MoveUp(n, t.moveUpInternal)
		return
	}
	t.moveUpInternal(n)
}

func (t *Terminal) moveUpInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursor.Row = clamp(t.cursor.Row-n, 0, t.rows-1)
}

// MoveUpCr moves the cursor up n rows and to column 0.
func (t *Terminal) MoveUpCr(n int) {
	if t.middleware != nil && t.middleware.MoveUpCr != nil {
		t.middleware.MoveUpCr(n, t.moveUpCrInternal)
		return
	}
	t.moveUpCrInternal(n)
}

func (t *Terminal) moveUpCrInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursor.Row = clamp(t.cursor.Row-n, 0, t.rows-1)
	t.cursor.Col = 0
}

// PopKeyboardMode removes n keyboard mode entries from the stack.
func (t *Terminal) PopKeyboardMode(n int) {
	if t.middleware != nil && t.middleware.PopKeyboardMode != nil {
		t.middleware.PopKeyboardMode(n, t.popKeyboardModeInternal)
		return
	}
	t.popKeyboardModeInternal(n)
}

func (t *Terminal) popKeyboardModeInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := 0; i < n && len(t.keyboardModes) > 0; i++ {
		t.keyboardModes = t.keyboardModes[:len(t.keyboardModes)-1]
	}
}

// PopTitle restores the previous title from the stack.
func (t *Terminal) PopTitle() {
	if t.middleware != nil && t.middleware.PopTitle != nil {
		t.middleware.PopTitle(t.popTitleInternal)
		return
	}
	t.popTitleInternal()
}

func (t *Terminal) popTitleInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if len(t.titleStack) > 0 {
		t.title = t.titleStack[len(t.titleStack)-1]
		t.titleStack = t.titleStack[:len(t.titleStack)-1]
	}
	if t.titleProvider != nil {
		t.titleProvider.PopTitle()
	}
}

// PrivacyMessageReceived processes a PM sequence and delegates to the configured provider.
func (t *Terminal) PrivacyMessageReceived(data []byte) {
	if t.middleware != nil && t.middleware.PrivacyMessageReceived != nil {
		t.middleware.PrivacyMessageReceived(data, t.privacyMessageReceivedInternal)
		return
	}
	t.privacyMessageReceivedInternal(data)
}

func (t *Terminal) privacyMessageReceivedInternal(data []byte) {
	if t.pmProvider != nil {
		t.pmProvider.Receive(data)
	}
}

// PushKeyboardMode adds a keyboard mode to the stack.
func (t *Terminal) PushKeyboardMode(mode ansicode.KeyboardMode) {
	if t.middleware != nil && t.middleware.PushKeyboardMode != nil {
		t.middleware.PushKeyboardMode(mode, t.pushKeyboardModeInternal)
		return
	}
	t.pushKeyboardModeInternal(mode)
}

func (t *Terminal) pushKeyboardModeInternal(mode ansicode.KeyboardMode) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.keyboardModes = append(t.keyboardModes, mode)
}

// PushTitle saves the current title to the stack.
func (t *Terminal) PushTitle() {
	if t.middleware != nil && t.middleware.PushTitle != nil {
		t.middleware.PushTitle(t.pushTitleInternal)
		return
	}
	t.pushTitleInternal()
}

func (t *Terminal) pushTitleInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.titleStack = append(t.titleStack, t.title)
	if t.titleProvider != nil {
		t.titleProvider.PushTitle()
	}
}

// ReportKeyboardMode sends the current keyboard mode via DSR response.
func (t *Terminal) ReportKeyboardMode() {
	if t.middleware != nil && t.middleware.ReportKeyboardMode != nil {
		t.middleware.ReportKeyboardMode(t.reportKeyboardModeInternal)
		return
	}
	t.reportKeyboardModeInternal()
}

func (t *Terminal) reportKeyboardModeInternal() {
	t.mu.RLock()
	var mode ansicode.KeyboardMode
	if len(t.keyboardModes) > 0 {
		mode = t.keyboardModes[len(t.keyboardModes)-1]
	}
	t.mu.RUnlock()

	response := fmt.Sprintf("\x1b[?%du", mode)
	t.writeResponseString(response)
}

// ReportModifyOtherKeys sends the current modify-other-keys mode via DSR response.
func (t *Terminal) ReportModifyOtherKeys() {
	if t.middleware != nil && t.middleware.ReportModifyOtherKeys != nil {
		t.middleware.ReportModifyOtherKeys(t.reportModifyOtherKeysInternal)
		return
	}
	t.reportModifyOtherKeysInternal()
}

func (t *Terminal) reportModifyOtherKeysInternal() {
	t.mu.RLock()
	modify := t.modifyOtherKeys
	t.mu.RUnlock()

	response := fmt.Sprintf("\x1b[>4;%dm", modify)
	t.writeResponseString(response)
}

// ResetColor removes a custom color from the palette at the given index.
func (t *Terminal) ResetColor(i int) {
	if t.middleware != nil && t.middleware.ResetColor != nil {
		t.middleware.ResetColor(i, t.resetColorInternal)
		return
	}
	t.resetColorInternal(i)
}

func (t *Terminal) resetColorInternal(i int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	delete(t.colors, i)
}

// ResetState clears the screen, resets cursor to (0,0), and restores default modes and attributes.
func (t *Terminal) ResetState() {
	if t.middleware != nil && t.middleware.ResetState != nil {
		t.middleware.ResetState(t.resetStateInternal)
		return
	}
	t.resetStateInternal()
}

func (t *Terminal) resetStateInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.activeBuffer.ClearAll()
	t.cursor.Row = 0
	t.cursor.Col = 0
	t.cursor.Visible = true
	t.cursor.Style = CursorStyleBlinkingBlock

	t.template = NewCellTemplate()
	t.scrollTop = 0
	t.scrollBottom = t.rows
	t.modes = ModeLineWrap | ModeShowCursor

	t.charsets = [4]Charset{CharsetASCII, CharsetASCII, CharsetASCII, CharsetASCII}
	t.activeCharset = 0

	t.colors = make(map[int]color.Color)
	t.keyboardModes = make([]ansicode.KeyboardMode, 0)
	t.currentHyperlink = nil
}

// RestoreCursorPosition restores cursor position, attributes, and charset state from the saved cursor.
func (t *Terminal) RestoreCursorPosition() {
	if t.middleware != nil && t.middleware.RestoreCursorPosition != nil {
		t.middleware.RestoreCursorPosition(t.restoreCursorPositionInternal)
		return
	}
	t.restoreCursorPositionInternal()
}

func (t *Terminal) restoreCursorPositionInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.restoreCursorPositionLocked()
}

// restoreCursorPositionLocked restores cursor without locking (caller must hold lock)
func (t *Terminal) restoreCursorPositionLocked() {
	if t.savedCursor != nil {
		t.cursor.Row = t.savedCursor.Row
		t.cursor.Col = t.savedCursor.Col
		t.template = t.savedCursor.Attrs

		if t.savedCursor.OriginMode {
			t.modes |= ModeOrigin
		} else {
			t.modes &^= ModeOrigin
		}

		t.activeCharset = t.savedCursor.CharsetIndex
		t.charsets = t.savedCursor.Charsets
	}
}

// ReverseIndex moves the cursor up one row. If at the top of the scroll region, scrolls down instead.
func (t *Terminal) ReverseIndex() {
	if t.middleware != nil && t.middleware.ReverseIndex != nil {
		t.middleware.ReverseIndex(t.reverseIndexInternal)
		return
	}
	t.reverseIndexInternal()
}

func (t *Terminal) reverseIndexInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cursor.Row == t.scrollTop {
		t.activeBuffer.ScrollDown(t.scrollTop, t.scrollBottom, 1)
	} else if t.cursor.Row > 0 {
		t.cursor.Row--
	}
}

// SaveCursorPosition saves cursor position, attributes, charset state, and origin mode for later restoration.
func (t *Terminal) SaveCursorPosition() {
	if t.middleware != nil && t.middleware.SaveCursorPosition != nil {
		t.middleware.SaveCursorPosition(t.saveCursorPositionInternal)
		return
	}
	t.saveCursorPositionInternal()
}

func (t *Terminal) saveCursorPositionInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.saveCursorPositionLocked()
}

// saveCursorPositionLocked saves cursor without locking (caller must hold lock)
func (t *Terminal) saveCursorPositionLocked() {
	t.savedCursor = &SavedCursor{
		Row:          t.cursor.Row,
		Col:          t.cursor.Col,
		Attrs:        t.template,
		OriginMode:   t.modes&ModeOrigin != 0,
		CharsetIndex: t.activeCharset,
		Charsets:     t.charsets,
	}
}

// ScrollDown shifts lines down within the scroll region, clearing top lines.
func (t *Terminal) ScrollDown(n int) {
	if t.middleware != nil && t.middleware.ScrollDown != nil {
		t.middleware.ScrollDown(n, t.scrollDownInternal)
		return
	}
	t.scrollDownInternal(n)
}

func (t *Terminal) scrollDownInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.activeBuffer.ScrollDown(t.scrollTop, t.scrollBottom, n)
}

// ScrollUp shifts lines up within the scroll region, pushing top lines to scrollback if enabled.
func (t *Terminal) ScrollUp(n int) {
	if t.middleware != nil && t.middleware.ScrollUp != nil {
		t.middleware.ScrollUp(n, t.scrollUpInternal)
		return
	}
	t.scrollUpInternal(n)
}

func (t *Terminal) scrollUpInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.activeBuffer.ScrollUp(t.scrollTop, t.scrollBottom, n)
}

// SetActiveCharset selects which charset slot (0-3, G0-G3) is currently active for character rendering.
func (t *Terminal) SetActiveCharset(n int) {
	if t.middleware != nil && t.middleware.SetActiveCharset != nil {
		t.middleware.SetActiveCharset(n, t.setActiveCharsetInternal)
		return
	}
	t.setActiveCharsetInternal(n)
}

func (t *Terminal) setActiveCharsetInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if n >= 0 && n < 4 {
		t.activeCharset = n
	}
}

// SetColor stores a custom color in the palette at the given index (used for indexed color resolution).
func (t *Terminal) SetColor(index int, c color.Color) {
	if t.middleware != nil && t.middleware.SetColor != nil {
		t.middleware.SetColor(index, c, t.setColorInternal)
		return
	}
	t.setColorInternal(index, c)
}

func (t *Terminal) setColorInternal(index int, c color.Color) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.colors[index] = c
}

// SetCursorStyle changes the cursor rendering style (block, underline, bar, blinking/steady).
func (t *Terminal) SetCursorStyle(style ansicode.CursorStyle) {
	if t.middleware != nil && t.middleware.SetCursorStyle != nil {
		t.middleware.SetCursorStyle(style, t.setCursorStyleInternal)
		return
	}
	t.setCursorStyleInternal(style)
}

func (t *Terminal) setCursorStyleInternal(style ansicode.CursorStyle) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.cursor.Style = CursorStyle(style)
}

// SetDynamicColor responds to a dynamic color query (OSC 10/11/12) with the current color value.
func (t *Terminal) SetDynamicColor(prefix string, index int, terminator string) {
	if t.middleware != nil && t.middleware.SetDynamicColor != nil {
		t.middleware.SetDynamicColor(prefix, index, terminator, t.setDynamicColorInternal)
		return
	}
	t.setDynamicColorInternal(prefix, index, terminator)
}

func (t *Terminal) setDynamicColorInternal(prefix string, index int, terminator string) {
	// Query the color from our color map or palette
	t.mu.RLock()
	c, ok := t.colors[index]
	t.mu.RUnlock()

	var response string
	if ok {
		rgba := resolveDefaultColor(c, true)
		response = fmt.Sprintf("\x1b]%s;rgb:%02x/%02x/%02x%s", prefix, rgba.R, rgba.G, rgba.B, terminator)
	} else if index >= 0 && index < 256 {
		rgba := DefaultPalette[index]
		response = fmt.Sprintf("\x1b]%s;rgb:%02x/%02x/%02x%s", prefix, rgba.R, rgba.G, rgba.B, terminator)
	}

	if response != "" {
		t.writeResponseString(response)
	}
}

// SetHyperlink sets the active hyperlink (OSC 8) for subsequently written characters.
// Pass nil to clear the hyperlink.
func (t *Terminal) SetHyperlink(hyperlink *ansicode.Hyperlink) {
	if t.middleware != nil && t.middleware.SetHyperlink != nil {
		t.middleware.SetHyperlink(hyperlink, t.setHyperlinkInternal)
		return
	}
	t.setHyperlinkInternal(hyperlink)
}

func (t *Terminal) setHyperlinkInternal(hyperlink *ansicode.Hyperlink) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if hyperlink == nil {
		t.currentHyperlink = nil
	} else {
		t.currentHyperlink = &Hyperlink{
			ID:  hyperlink.ID,
			URI: hyperlink.URI,
		}
	}
}

// SetKeyboardMode modifies the top keyboard mode on the stack using the specified behavior (replace, union, or difference).
func (t *Terminal) SetKeyboardMode(mode ansicode.KeyboardMode, behavior ansicode.KeyboardModeBehavior) {
	if t.middleware != nil && t.middleware.SetKeyboardMode != nil {
		t.middleware.SetKeyboardMode(mode, behavior, t.setKeyboardModeInternal)
		return
	}
	t.setKeyboardModeInternal(mode, behavior)
}

func (t *Terminal) setKeyboardModeInternal(mode ansicode.KeyboardMode, behavior ansicode.KeyboardModeBehavior) {
	t.mu.Lock()
	defer t.mu.Unlock()

	currentMode := ansicode.KeyboardModeNoMode
	if len(t.keyboardModes) > 0 {
		currentMode = t.keyboardModes[len(t.keyboardModes)-1]
	}

	var newMode ansicode.KeyboardMode
	switch behavior {
	case ansicode.KeyboardModeBehaviorReplace:
		newMode = mode
	case ansicode.KeyboardModeBehaviorUnion:
		newMode = currentMode | mode
	case ansicode.KeyboardModeBehaviorDifference:
		newMode = currentMode &^ mode
	}

	if len(t.keyboardModes) > 0 {
		t.keyboardModes[len(t.keyboardModes)-1] = newMode
	} else {
		t.keyboardModes = append(t.keyboardModes, newMode)
	}
}

// SetKeypadApplicationMode enables application keypad mode (numeric keypad sends escape sequences).
func (t *Terminal) SetKeypadApplicationMode() {
	if t.middleware != nil && t.middleware.SetKeypadApplicationMode != nil {
		t.middleware.SetKeypadApplicationMode(t.setKeypadApplicationModeInternal)
		return
	}
	t.setKeypadApplicationModeInternal()
}

func (t *Terminal) setKeypadApplicationModeInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.modes |= ModeKeypadApplication
}

// SetMode enables a terminal mode flag. Some modes have side effects (e.g., ModeOrigin moves cursor to scrollTop).
func (t *Terminal) SetMode(mode ansicode.TerminalMode) {
	if t.middleware != nil && t.middleware.SetMode != nil {
		t.middleware.SetMode(mode, t.setModeInternal)
		return
	}
	t.setModeInternal(mode)
}

func (t *Terminal) setModeInternal(mode ansicode.TerminalMode) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.setModeLocked(mode, true)
}

// setModeLocked sets or unsets a terminal mode (caller must hold lock).
func (t *Terminal) setModeLocked(mode ansicode.TerminalMode, set bool) {
	var m TerminalMode

	switch mode {
	case ansicode.TerminalModeCursorKeys:
		m = ModeCursorKeys
	case ansicode.TerminalModeColumnMode:
		m = ModeColumnMode
	case ansicode.TerminalModeInsert:
		m = ModeInsert
	case ansicode.TerminalModeOrigin:
		m = ModeOrigin
		if set {
			t.cursor.Row = t.scrollTop
			t.cursor.Col = 0
		}
	case ansicode.TerminalModeLineWrap:
		m = ModeLineWrap
	case ansicode.TerminalModeBlinkingCursor:
		m = ModeBlinkingCursor
	case ansicode.TerminalModeLineFeedNewLine:
		m = ModeLineFeedNewLine
	case ansicode.TerminalModeShowCursor:
		m = ModeShowCursor
		t.cursor.Visible = set
	case ansicode.TerminalModeReportMouseClicks:
		m = ModeReportMouseClicks
	case ansicode.TerminalModeReportCellMouseMotion:
		m = ModeReportCellMouseMotion
	case ansicode.TerminalModeReportAllMouseMotion:
		m = ModeReportAllMouseMotion
	case ansicode.TerminalModeReportFocusInOut:
		m = ModeReportFocusInOut
	case ansicode.TerminalModeUTF8Mouse:
		m = ModeUTF8Mouse
	case ansicode.TerminalModeSGRMouse:
		m = ModeSGRMouse
	case ansicode.TerminalModeAlternateScroll:
		m = ModeAlternateScroll
	case ansicode.TerminalModeUrgencyHints:
		m = ModeUrgencyHints
	case ansicode.TerminalModeSwapScreenAndSetRestoreCursor:
		m = ModeSwapScreenAndSetRestoreCursor
		if set {
			t.saveCursorPositionLocked()
			t.activeBuffer = t.alternateBuffer
			t.activeBuffer.ClearAll()
		} else {
			t.activeBuffer = t.primaryBuffer
			t.restoreCursorPositionLocked()
		}
	case ansicode.TerminalModeBracketedPaste:
		m = ModeBracketedPaste
	default:
		return
	}

	if set {
		t.modes |= m
	} else {
		t.modes &^= m
	}
}

// SetModifyOtherKeys sets how modifier keys are reported in keyboard input.
func (t *Terminal) SetModifyOtherKeys(modify ansicode.ModifyOtherKeys) {
	if t.middleware != nil && t.middleware.SetModifyOtherKeys != nil {
		t.middleware.SetModifyOtherKeys(modify, t.setModifyOtherKeysInternal)
		return
	}
	t.setModifyOtherKeysInternal(modify)
}

func (t *Terminal) setModifyOtherKeysInternal(modify ansicode.ModifyOtherKeys) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.modifyOtherKeys = modify
}

// SetScrollingRegion sets the scroll boundaries (1-based, converted to 0-based internally).
// Moves cursor to home position (top-left of region if origin mode, else absolute top-left).
func (t *Terminal) SetScrollingRegion(top, bottom int) {
	if t.middleware != nil && t.middleware.SetScrollingRegion != nil {
		t.middleware.SetScrollingRegion(top, bottom, t.setScrollingRegionInternal)
		return
	}
	t.setScrollingRegionInternal(top, bottom)
}

func (t *Terminal) setScrollingRegionInternal(top, bottom int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Convert from 1-based to 0-based
	top--
	bottom--

	if top < 0 {
		top = 0
	}
	if bottom <= 0 || bottom > t.rows {
		bottom = t.rows
	}
	if top >= bottom {
		return
	}

	t.scrollTop = top
	t.scrollBottom = bottom

	// Move cursor to home position (considering origin mode)
	if t.modes&ModeOrigin != 0 {
		t.cursor.Row = t.scrollTop
	} else {
		t.cursor.Row = 0
	}
	t.cursor.Col = 0
}

// StartOfStringReceived processes a SOS sequence and delegates to the configured provider.
func (t *Terminal) StartOfStringReceived(data []byte) {
	if t.middleware != nil && t.middleware.StartOfStringReceived != nil {
		t.middleware.StartOfStringReceived(data, t.startOfStringReceivedInternal)
		return
	}
	t.startOfStringReceivedInternal(data)
}

func (t *Terminal) startOfStringReceivedInternal(data []byte) {
	if t.sosProvider != nil {
		t.sosProvider.Receive(data)
	}
}

// SetTerminalCharAttribute applies SGR attributes to the cell template (colors, bold, underline, etc.).
func (t *Terminal) SetTerminalCharAttribute(attr ansicode.TerminalCharAttribute) {
	if t.middleware != nil && t.middleware.SetTerminalCharAttribute != nil {
		t.middleware.SetTerminalCharAttribute(attr, t.setTerminalCharAttributeInternal)
		return
	}
	t.setTerminalCharAttributeInternal(attr)
}

func (t *Terminal) setTerminalCharAttributeInternal(attr ansicode.TerminalCharAttribute) {
	t.mu.Lock()
	defer t.mu.Unlock()

	switch attr.Attr {
	case ansicode.CharAttributeReset:
		t.template = NewCellTemplate()

	case ansicode.CharAttributeBold:
		t.template.SetFlag(CellFlagBold)

	case ansicode.CharAttributeDim:
		t.template.SetFlag(CellFlagDim)

	case ansicode.CharAttributeItalic:
		t.template.SetFlag(CellFlagItalic)

	case ansicode.CharAttributeUnderline:
		t.template.SetFlag(CellFlagUnderline)
		t.template.ClearFlag(CellFlagDoubleUnderline | CellFlagCurlyUnderline | CellFlagDottedUnderline | CellFlagDashedUnderline)

	case ansicode.CharAttributeDoubleUnderline:
		t.template.SetFlag(CellFlagDoubleUnderline)
		t.template.ClearFlag(CellFlagUnderline | CellFlagCurlyUnderline | CellFlagDottedUnderline | CellFlagDashedUnderline)

	case ansicode.CharAttributeCurlyUnderline:
		t.template.SetFlag(CellFlagCurlyUnderline)
		t.template.ClearFlag(CellFlagUnderline | CellFlagDoubleUnderline | CellFlagDottedUnderline | CellFlagDashedUnderline)

	case ansicode.CharAttributeDottedUnderline:
		t.template.SetFlag(CellFlagDottedUnderline)
		t.template.ClearFlag(CellFlagUnderline | CellFlagDoubleUnderline | CellFlagCurlyUnderline | CellFlagDashedUnderline)

	case ansicode.CharAttributeDashedUnderline:
		t.template.SetFlag(CellFlagDashedUnderline)
		t.template.ClearFlag(CellFlagUnderline | CellFlagDoubleUnderline | CellFlagCurlyUnderline | CellFlagDottedUnderline)

	case ansicode.CharAttributeBlinkSlow:
		t.template.SetFlag(CellFlagBlinkSlow)

	case ansicode.CharAttributeBlinkFast:
		t.template.SetFlag(CellFlagBlinkFast)

	case ansicode.CharAttributeReverse:
		t.template.SetFlag(CellFlagReverse)

	case ansicode.CharAttributeHidden:
		t.template.SetFlag(CellFlagHidden)

	case ansicode.CharAttributeStrike:
		t.template.SetFlag(CellFlagStrike)

	case ansicode.CharAttributeCancelBold:
		t.template.ClearFlag(CellFlagBold)

	case ansicode.CharAttributeCancelBoldDim:
		t.template.ClearFlag(CellFlagBold | CellFlagDim)

	case ansicode.CharAttributeCancelItalic:
		t.template.ClearFlag(CellFlagItalic)

	case ansicode.CharAttributeCancelUnderline:
		t.template.ClearFlag(CellFlagUnderline | CellFlagDoubleUnderline | CellFlagCurlyUnderline | CellFlagDottedUnderline | CellFlagDashedUnderline)

	case ansicode.CharAttributeCancelBlink:
		t.template.ClearFlag(CellFlagBlinkSlow | CellFlagBlinkFast)

	case ansicode.CharAttributeCancelReverse:
		t.template.ClearFlag(CellFlagReverse)

	case ansicode.CharAttributeCancelHidden:
		t.template.ClearFlag(CellFlagHidden)

	case ansicode.CharAttributeCancelStrike:
		t.template.ClearFlag(CellFlagStrike)

	case ansicode.CharAttributeForeground:
		t.template.Fg = t.resolveColor(attr)

	case ansicode.CharAttributeBackground:
		t.template.Bg = t.resolveColor(attr)

	case ansicode.CharAttributeUnderlineColor:
		if attr.RGBColor == nil && attr.IndexedColor == nil && attr.NamedColor == nil {
			t.template.UnderlineColor = nil
		} else {
			t.template.UnderlineColor = t.resolveColor(attr)
		}
	}
}

// resolveColor resolves the color from the attribute.
// Returns a NamedColor default when no specific color is provided.
func (t *Terminal) resolveColor(attr ansicode.TerminalCharAttribute) color.Color {
	if attr.RGBColor != nil {
		return color.RGBA{
			R: attr.RGBColor.R,
			G: attr.RGBColor.G,
			B: attr.RGBColor.B,
			A: 255,
		}
	}

	if attr.IndexedColor != nil {
		return &IndexedColor{Index: int(attr.IndexedColor.Index)}
	}

	if attr.NamedColor != nil {
		return &NamedColor{Name: int(*attr.NamedColor)}
	}

	// Return appropriate default based on attribute type
	switch attr.Attr {
	case ansicode.CharAttributeForeground:
		return &NamedColor{Name: NamedColorForeground}
	case ansicode.CharAttributeBackground:
		return &NamedColor{Name: NamedColorBackground}
	default:
		return &NamedColor{Name: NamedColorForeground}
	}
}

// SetTitle updates the window title and notifies the title provider.
func (t *Terminal) SetTitle(title string) {
	if t.middleware != nil && t.middleware.SetTitle != nil {
		t.middleware.SetTitle(title, t.setTitleInternal)
		return
	}
	t.setTitleInternal(title)
}

func (t *Terminal) setTitleInternal(title string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.title = title
	if t.titleProvider != nil {
		t.titleProvider.SetTitle(title)
	}
}

// Substitute replaces the character at the cursor with '?' (used for error indication).
func (t *Terminal) Substitute() {
	if t.middleware != nil && t.middleware.Substitute != nil {
		t.middleware.Substitute(t.substituteInternal)
		return
	}
	t.substituteInternal()
}

func (t *Terminal) substituteInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	cell := t.activeBuffer.Cell(t.cursor.Row, t.cursor.Col)
	if cell != nil {
		cell.Char = '?'
	}
}

// Tab moves the cursor right to the next n tab stops.
func (t *Terminal) Tab(n int) {
	if t.middleware != nil && t.middleware.Tab != nil {
		t.middleware.Tab(n, t.tabInternal)
		return
	}
	t.tabInternal(n)
}

func (t *Terminal) tabInternal(n int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i := 0; i < n; i++ {
		t.cursor.Col = t.activeBuffer.NextTabStop(t.cursor.Col)
	}
}

// TextAreaSizeChars sends the terminal dimensions in characters via DSR response.
func (t *Terminal) TextAreaSizeChars() {
	if t.middleware != nil && t.middleware.TextAreaSizeChars != nil {
		t.middleware.TextAreaSizeChars(t.textAreaSizeCharsInternal)
		return
	}
	t.textAreaSizeCharsInternal()
}

func (t *Terminal) textAreaSizeCharsInternal() {
	t.mu.RLock()
	rows := t.rows
	cols := t.cols
	t.mu.RUnlock()

	// Default response: CSI 8 ; rows ; cols t
	response := fmt.Sprintf("\x1b[8;%d;%dt", rows, cols)
	t.writeResponseString(response)
}

// TextAreaSizePixels sends the terminal dimensions in pixels via DSR response (assumes 10x20 pixel cells).
func (t *Terminal) TextAreaSizePixels() {
	if t.middleware != nil && t.middleware.TextAreaSizePixels != nil {
		t.middleware.TextAreaSizePixels(t.textAreaSizePixelsInternal)
		return
	}
	t.textAreaSizePixelsInternal()
}

func (t *Terminal) textAreaSizePixelsInternal() {
	t.mu.RLock()
	rows := t.rows
	cols := t.cols
	t.mu.RUnlock()

	// Default response: CSI 4 ; height ; width t (assuming 10x20 pixel cells)
	response := fmt.Sprintf("\x1b[4;%d;%dt", rows*20, cols*10)
	t.writeResponseString(response)
}

// UnsetKeypadApplicationMode disables application keypad mode (numeric keypad sends digits).
func (t *Terminal) UnsetKeypadApplicationMode() {
	if t.middleware != nil && t.middleware.UnsetKeypadApplicationMode != nil {
		t.middleware.UnsetKeypadApplicationMode(t.unsetKeypadApplicationModeInternal)
		return
	}
	t.unsetKeypadApplicationModeInternal()
}

func (t *Terminal) unsetKeypadApplicationModeInternal() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.modes &^= ModeKeypadApplication
}

// UnsetMode disables a terminal mode flag. Some modes have side effects (e.g., ModeSwapScreenAndSetRestoreCursor restores primary buffer).
func (t *Terminal) UnsetMode(mode ansicode.TerminalMode) {
	if t.middleware != nil && t.middleware.UnsetMode != nil {
		t.middleware.UnsetMode(mode, t.unsetModeInternal)
		return
	}
	t.unsetModeInternal(mode)
}

func (t *Terminal) unsetModeInternal(mode ansicode.TerminalMode) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.setModeLocked(mode, false)
}

// IndexedColor references a color by palette index (0-255).
// Resolution to actual RGBA happens at render time using the palette.
type IndexedColor struct {
	Index int
}

// RGBA implements color.Color, returning a placeholder (actual resolution happens at render time).
func (c *IndexedColor) RGBA() (r, g, b, a uint32) {
	return 0, 0, 0, 0xffff
}

// NamedColor references a color by semantic name (foreground, background, cursor, etc.).
// Resolution to actual RGBA happens at render time using the palette and defaults.
type NamedColor struct {
	Name int
}

// RGBA implements color.Color, returning a placeholder (actual resolution happens at render time).
func (c *NamedColor) RGBA() (r, g, b, a uint32) {
	return 0, 0, 0, 0xffff
}

// SetWorkingDirectory stores the current working directory (OSC 7).
func (t *Terminal) SetWorkingDirectory(uri string) {
	if t.middleware != nil && t.middleware.SetWorkingDirectory != nil {
		t.middleware.SetWorkingDirectory(uri, t.setWorkingDirectoryInternal)
		return
	}
	t.setWorkingDirectoryInternal(uri)
}

func (t *Terminal) setWorkingDirectoryInternal(uri string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.workingDir = uri
}

// WorkingDirectory returns the current working directory URI (OSC 7).
func (t *Terminal) WorkingDirectory() string {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.workingDir
}

// WorkingDirectoryPath extracts the path from the working directory URI.
func (t *Terminal) WorkingDirectoryPath() string {
	t.mu.RLock()
	uri := t.workingDir
	t.mu.RUnlock()

	if uri == "" {
		return ""
	}

	// Parse file://hostname/path
	const prefix = "file://"
	if len(uri) <= len(prefix) {
		return ""
	}
	if uri[:len(prefix)] != prefix {
		return ""
	}
	rest := uri[len(prefix):]

	// Find the path after hostname
	slashIdx := -1
	for i := 0; i < len(rest); i++ {
		if rest[i] == '/' {
			slashIdx = i
			break
		}
	}
	if slashIdx < 0 {
		return ""
	}
	return rest[slashIdx:]
}

// CellSizePixels sends the cell size in pixels via DSR response.
func (t *Terminal) CellSizePixels() {
	t.mu.RLock()
	sizeProvider := t.sizeProvider
	t.mu.RUnlock()

	var cellWidth, cellHeight int
	if sizeProvider != nil {
		cellWidth, cellHeight = sizeProvider.CellSizePixels()
	} else {
		// Default cell size
		cellWidth = 10
		cellHeight = 20
	}

	// CSI 6 ; height ; width t
	response := fmt.Sprintf("\x1b[6;%d;%dt", cellHeight, cellWidth)
	t.writeResponseString(response)
}

// SixelReceived handles incoming Sixel graphics data.
// Currently a no-op stub - Sixel support is not implemented.
func (t *Terminal) SixelReceived(params [][]uint16, data []byte) {
	if t.middleware != nil && t.middleware.SixelReceived != nil {
		t.middleware.SixelReceived(params, data, t.sixelReceivedInternal)
		return
	}
	t.sixelReceivedInternal(params, data)
}

func (t *Terminal) sixelReceivedInternal(params [][]uint16, data []byte) {
	// Convert params to int64 slice
	var p []int64
	for _, param := range params {
		if len(param) > 0 {
			p = append(p, int64(param[0]))
		}
	}

	// Parse sixel data
	img, err := ParseSixel(p, data)
	if err != nil || img.Width == 0 || img.Height == 0 {
		return
	}

	// Store image
	imageID := t.images.Store(img.Width, img.Height, img.Data)

	// Calculate cell coverage
	cellWidth, cellHeight := t.getCellSizePixels()
	cols := int((img.Width + uint32(cellWidth) - 1) / uint32(cellWidth))
	rows := int((img.Height + uint32(cellHeight) - 1) / uint32(cellHeight))

	// Create placement at cursor position
	t.mu.Lock()
	curRow := t.cursor.Row
	curCol := t.cursor.Col
	t.mu.Unlock()

	placement := &ImagePlacement{
		ImageID: imageID,
		Row:     curRow,
		Col:     curCol,
		Cols:    cols,
		Rows:    rows,
		SrcX:    0,
		SrcY:    0,
		SrcW:    img.Width,
		SrcH:    img.Height,
		ZIndex:  0, // Sixel images are rendered in front of text
	}

	placementID := t.images.Place(placement)

	// Assign image references to cells
	t.assignImageToCells(imageID, placementID, placement, img.Width, img.Height, cellWidth, cellHeight)

	// Move cursor down by number of rows (Sixel behavior)
	t.mu.Lock()
	t.cursor.Row += rows
	if t.cursor.Row >= t.rows {
		t.cursor.Row = t.rows - 1
	}
	t.mu.Unlock()
}

// getCellSizePixels returns the cell size in pixels.
// Uses the SizeProvider if available, otherwise defaults to 10x20.
func (t *Terminal) getCellSizePixels() (width, height int) {
	if t.sizeProvider != nil {
		w, h := t.sizeProvider.CellSizePixels()
		if w > 0 && h > 0 {
			return w, h
		}
	}
	return 10, 20 // Default cell size
}

// assignImageToCells assigns image references to cells covered by a placement.
func (t *Terminal) assignImageToCells(imageID, placementID uint32, p *ImagePlacement, imgW, imgH uint32, cellW, cellH int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for row := 0; row < p.Rows; row++ {
		for col := 0; col < p.Cols; col++ {
			cellRow := p.Row + row
			cellCol := p.Col + col

			if cellRow < 0 || cellRow >= t.rows || cellCol < 0 || cellCol >= t.cols {
				continue
			}

			// Calculate UV coordinates for this cell
			u0 := float32(col*cellW) / float32(imgW)
			v0 := float32(row*cellH) / float32(imgH)
			u1 := float32((col+1)*cellW) / float32(imgW)
			v1 := float32((row+1)*cellH) / float32(imgH)

			// Clamp to [0, 1]
			if u1 > 1.0 {
				u1 = 1.0
			}
			if v1 > 1.0 {
				v1 = 1.0
			}

			cell := t.activeBuffer.Cell(cellRow, cellCol)
			if cell != nil {
				cell.Image = &CellImage{
					PlacementID: placementID,
					ImageID:     imageID,
					U0:          u0,
					V0:          v0,
					U1:          u1,
					V1:          v1,
					ZIndex:      p.ZIndex,
				}
				cell.MarkDirty()
			}
		}
	}
}
