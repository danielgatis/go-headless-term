//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	headlessterm "github.com/danielgatis/go-headless-term"
)

// Global terminal registry
var terminals = make(map[int]*terminalInstance)
var nextTerminalID = 1

// terminalInstance wraps a terminal with its JS handlers
type terminalInstance struct {
	term     *headlessterm.Terminal
	handlers *jsHandlers
}

func main() {
	// Register all exported functions
	js.Global().Set("HeadlessTerm", js.ValueOf(map[string]interface{}{
		// Terminal lifecycle
		"create":  js.FuncOf(createTerminal),
		"destroy": js.FuncOf(destroyTerminal),

		// Input processing
		"write":       js.FuncOf(write),
		"writeString": js.FuncOf(writeString),

		// Dimensions
		"resize": js.FuncOf(resize),
		"rows":   js.FuncOf(rows),
		"cols":   js.FuncOf(cols),

		// Cursor
		"cursorPos":     js.FuncOf(cursorPos),
		"cursorVisible": js.FuncOf(cursorVisible),
		"cursorStyle":   js.FuncOf(cursorStyle),

		// Content
		"getString":    js.FuncOf(getString),
		"lineContent":  js.FuncOf(lineContent),
		"cell":         js.FuncOf(cell),
		"snapshot":     js.FuncOf(snapshot),
		"snapshotJSON": js.FuncOf(snapshotJSON),

		// State inspection
		"title":             js.FuncOf(title),
		"hasMode":           js.FuncOf(hasMode),
		"isAlternateScreen": js.FuncOf(isAlternateScreen),
		"scrollRegion":      js.FuncOf(scrollRegion),

		// Scrollback
		"scrollbackLen":   js.FuncOf(scrollbackLen),
		"scrollbackLine":  js.FuncOf(scrollbackLine),
		"clearScrollback": js.FuncOf(clearScrollback),

		// Selection
		"setSelection":    js.FuncOf(setSelection),
		"clearSelection":  js.FuncOf(clearSelection),
		"hasSelection":    js.FuncOf(hasSelection),
		"getSelectedText": js.FuncOf(getSelectedText),

		// Dirty tracking
		"hasDirty":   js.FuncOf(hasDirty),
		"dirtyCells": js.FuncOf(dirtyCells),
		"clearDirty": js.FuncOf(clearDirty),

		// Search
		"search":           js.FuncOf(search),
		"searchScrollback": js.FuncOf(searchScrollback),

		// Working directory & user vars
		"workingDirectory":     js.FuncOf(workingDirectory),
		"workingDirectoryPath": js.FuncOf(workingDirectoryPath),
		"getUserVar":           js.FuncOf(getUserVar),
		"getUserVars":          js.FuncOf(getUserVars),

		// Semantic prompts
		"promptMarks":          js.FuncOf(promptMarks),
		"getLastCommandOutput": js.FuncOf(getLastCommandOutput),

		// Images
		"imageCount":          js.FuncOf(imageCount),
		"imagePlacementCount": js.FuncOf(imagePlacementCount),
		"imageUsedMemory":     js.FuncOf(imageUsedMemory),
		"sixelEnabled":        js.FuncOf(sixelEnabled),
		"kittyEnabled":        js.FuncOf(kittyEnabled),

		// Handler registration
		"onBell":           js.FuncOf(onBell),
		"onTitle":          js.FuncOf(onTitle),
		"onClipboard":      js.FuncOf(onClipboard),
		"onNotification":   js.FuncOf(onNotification),
		"onPTYWrite":       js.FuncOf(onPTYWrite),
		"onSize":           js.FuncOf(onSize),
		"onSemanticPrompt": js.FuncOf(onSemanticPrompt),
		"onAPC":            js.FuncOf(onAPC),
		"onPM":             js.FuncOf(onPM),
		"onSOS":            js.FuncOf(onSOS),
		"onScrollback":     js.FuncOf(onScrollback),
		"onRecording":      js.FuncOf(onRecording),
	}))

	// Keep the program running
	select {}
}

// ============================================================================
// Terminal Lifecycle
// ============================================================================

func createTerminal(_ js.Value, args []js.Value) interface{} {
	rows := 24
	cols := 80
	if len(args) >= 2 {
		rows = args[0].Int()
		cols = args[1].Int()
	}

	handlers := newJSHandlers()

	opts := []headlessterm.Option{
		headlessterm.WithSize(rows, cols),
		headlessterm.WithScrollback(handlers.scrollback),
		headlessterm.WithBell(handlers.bell),
		headlessterm.WithTitle(handlers.title),
		headlessterm.WithClipboard(handlers.clipboard),
		headlessterm.WithNotification(handlers.notification),
		headlessterm.WithPTYWriter(handlers.ptyWriter),
		headlessterm.WithSizeProvider(handlers.size),
		headlessterm.WithSemanticPromptHandler(handlers.semanticPrompt),
		headlessterm.WithAPC(handlers.apc),
		headlessterm.WithPM(handlers.pm),
		headlessterm.WithSOS(handlers.sos),
		headlessterm.WithRecording(handlers.recording),
	}

	term := headlessterm.New(opts...)
	id := nextTerminalID
	nextTerminalID++
	terminals[id] = &terminalInstance{
		term:     term,
		handlers: handlers,
	}

	return id
}

func destroyTerminal(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	id := args[0].Int()
	delete(terminals, id)
	return nil
}

func getInstance(id int) *terminalInstance {
	return terminals[id]
}

func getTerminal(id int) *headlessterm.Terminal {
	inst := getInstance(id)
	if inst == nil {
		return nil
	}
	return inst.term
}

// ============================================================================
// Input Processing
// ============================================================================

func write(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return -1
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return -1
	}

	// Get Uint8Array from JS
	data := make([]byte, args[1].Length())
	js.CopyBytesToGo(data, args[1])

	n, _ := term.Write(data)
	return n
}

func writeString(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return -1
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return -1
	}

	n, _ := term.WriteString(args[1].String())
	return n
}

// ============================================================================
// Dimensions
// ============================================================================

func resize(_ js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	term.Resize(args[1].Int(), args[2].Int())
	return nil
}

func rows(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return 0
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return 0
	}
	return term.Rows()
}

func cols(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return 0
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return 0
	}
	return term.Cols()
}

// ============================================================================
// Cursor
// ============================================================================

func cursorPos(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	row, col := term.CursorPos()
	return map[string]interface{}{
		"row": row,
		"col": col,
	}
}

func cursorVisible(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return false
	}
	return term.CursorVisible()
}

func cursorStyle(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return 0
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return 0
	}
	return int(term.CursorStyle())
}

// ============================================================================
// Content
// ============================================================================

func getString(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return ""
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return ""
	}
	return term.String()
}

func lineContent(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return ""
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return ""
	}
	return term.LineContent(args[1].Int())
}

func cell(_ js.Value, args []js.Value) interface{} {
	if len(args) < 3 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	c := term.Cell(args[1].Int(), args[2].Int())
	if c == nil {
		return nil
	}
	return cellToJS(c)
}

func cellToJS(c *headlessterm.Cell) map[string]interface{} {
	fg := headlessterm.ResolveDefaultColor(c.Fg, true)
	bg := headlessterm.ResolveDefaultColor(c.Bg, false)

	result := map[string]interface{}{
		"char": string(c.Char),
		"fg": map[string]interface{}{
			"r": fg.R,
			"g": fg.G,
			"b": fg.B,
			"a": fg.A,
		},
		"bg": map[string]interface{}{
			"r": bg.R,
			"g": bg.G,
			"b": bg.B,
			"a": bg.A,
		},
		"bold":      c.Flags&headlessterm.CellFlagBold != 0,
		"dim":       c.Flags&headlessterm.CellFlagDim != 0,
		"italic":    c.Flags&headlessterm.CellFlagItalic != 0,
		"underline": c.Flags&headlessterm.CellFlagUnderline != 0,
		"blink":     c.Flags&headlessterm.CellFlagBlinkSlow != 0 || c.Flags&headlessterm.CellFlagBlinkFast != 0,
		"reverse":   c.Flags&headlessterm.CellFlagReverse != 0,
		"hidden":    c.Flags&headlessterm.CellFlagHidden != 0,
		"strike":    c.Flags&headlessterm.CellFlagStrike != 0,
		"wideChar":  c.Flags&headlessterm.CellFlagWideChar != 0,
	}

	if c.Hyperlink != nil {
		result["hyperlink"] = map[string]interface{}{
			"id":  c.Hyperlink.ID,
			"uri": c.Hyperlink.URI,
		}
	}

	return result
}

func snapshot(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}

	detail := headlessterm.SnapshotDetailStyled
	if len(args) >= 2 {
		switch args[1].String() {
		case "text":
			detail = headlessterm.SnapshotDetailText
		case "styled":
			detail = headlessterm.SnapshotDetailStyled
		case "full":
			detail = headlessterm.SnapshotDetailFull
		}
	}

	snap := term.Snapshot(detail)
	return snapshotToJS(snap)
}

func snapshotJSON(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return ""
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return ""
	}

	detail := headlessterm.SnapshotDetailStyled
	if len(args) >= 2 {
		switch args[1].String() {
		case "text":
			detail = headlessterm.SnapshotDetailText
		case "styled":
			detail = headlessterm.SnapshotDetailStyled
		case "full":
			detail = headlessterm.SnapshotDetailFull
		}
	}

	snap := term.Snapshot(detail)
	data, err := json.Marshal(snap)
	if err != nil {
		return ""
	}
	return string(data)
}

func snapshotToJS(snap *headlessterm.Snapshot) map[string]interface{} {
	lines := make([]interface{}, len(snap.Lines))
	for i, line := range snap.Lines {
		lines[i] = map[string]interface{}{
			"text": line.Text,
		}
	}

	return map[string]interface{}{
		"size": map[string]interface{}{
			"rows": snap.Size.Rows,
			"cols": snap.Size.Cols,
		},
		"cursor": map[string]interface{}{
			"row":     snap.Cursor.Row,
			"col":     snap.Cursor.Col,
			"visible": snap.Cursor.Visible,
			"style":   snap.Cursor.Style,
		},
		"lines": lines,
	}
}

// ============================================================================
// State Inspection
// ============================================================================

func title(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return ""
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return ""
	}
	return term.Title()
}

func hasMode(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return false
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return false
	}
	return term.HasMode(headlessterm.TerminalMode(args[1].Int()))
}

func isAlternateScreen(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return false
	}
	return term.IsAlternateScreen()
}

func scrollRegion(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	top, bottom := term.ScrollRegion()
	return map[string]interface{}{
		"top":    top,
		"bottom": bottom,
	}
}

// ============================================================================
// Scrollback
// ============================================================================

func scrollbackLen(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return 0
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return 0
	}
	return term.ScrollbackLen()
}

func scrollbackLine(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	cells := term.ScrollbackLine(args[1].Int())
	if cells == nil {
		return nil
	}

	result := make([]interface{}, len(cells))
	for i, c := range cells {
		result[i] = cellToJS(&c)
	}
	return result
}

func clearScrollback(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	term.ClearScrollback()
	return nil
}

// ============================================================================
// Selection
// ============================================================================

func setSelection(_ js.Value, args []js.Value) interface{} {
	if len(args) < 5 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	term.SetSelection(
		headlessterm.Position{Row: args[1].Int(), Col: args[2].Int()},
		headlessterm.Position{Row: args[3].Int(), Col: args[4].Int()},
	)
	return nil
}

func clearSelection(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	term.ClearSelection()
	return nil
}

func hasSelection(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return false
	}
	return term.HasSelection()
}

func getSelectedText(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return ""
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return ""
	}
	return term.GetSelectedText()
}

// ============================================================================
// Dirty Tracking
// ============================================================================

func hasDirty(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return false
	}
	return term.HasDirty()
}

func dirtyCells(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	cells := term.DirtyCells()
	result := make([]interface{}, len(cells))
	for i, pos := range cells {
		result[i] = map[string]interface{}{
			"row": pos.Row,
			"col": pos.Col,
		}
	}
	return result
}

func clearDirty(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	term.ClearDirty()
	return nil
}

// ============================================================================
// Search
// ============================================================================

func search(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	positions := term.Search(args[1].String())
	result := make([]interface{}, len(positions))
	for i, pos := range positions {
		result[i] = map[string]interface{}{
			"row": pos.Row,
			"col": pos.Col,
		}
	}
	return result
}

func searchScrollback(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	positions := term.SearchScrollback(args[1].String())
	result := make([]interface{}, len(positions))
	for i, pos := range positions {
		result[i] = map[string]interface{}{
			"row": pos.Row,
			"col": pos.Col,
		}
	}
	return result
}

// ============================================================================
// Working Directory & User Vars
// ============================================================================

func workingDirectory(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return ""
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return ""
	}
	return term.WorkingDirectory()
}

func workingDirectoryPath(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return ""
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return ""
	}
	return term.WorkingDirectoryPath()
}

func getUserVar(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return ""
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return ""
	}
	return term.GetUserVar(args[1].String())
}

func getUserVars(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	vars := term.GetUserVars()
	result := make(map[string]interface{})
	for k, v := range vars {
		result[k] = v
	}
	return result
}

// ============================================================================
// Semantic Prompts
// ============================================================================

func promptMarks(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return nil
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return nil
	}
	marks := term.PromptMarks()
	result := make([]interface{}, len(marks))
	for i, mark := range marks {
		result[i] = map[string]interface{}{
			"type":     int(mark.Type),
			"row":      mark.Row,
			"exitCode": mark.ExitCode,
		}
	}
	return result
}

func getLastCommandOutput(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return ""
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return ""
	}
	return term.GetLastCommandOutput()
}

// ============================================================================
// Images
// ============================================================================

func imageCount(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return 0
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return 0
	}
	return term.ImageCount()
}

func imagePlacementCount(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return 0
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return 0
	}
	return term.ImagePlacementCount()
}

func imageUsedMemory(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return 0
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return 0
	}
	return term.ImageUsedMemory()
}

func sixelEnabled(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return false
	}
	return term.SixelEnabled()
}

func kittyEnabled(_ js.Value, args []js.Value) interface{} {
	if len(args) < 1 {
		return false
	}
	term := getTerminal(args[0].Int())
	if term == nil {
		return false
	}
	return term.KittyEnabled()
}

// ============================================================================
// Handler Registration
// ============================================================================

func onBell(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.bell.callback = args[1]
	return nil
}

func onTitle(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.title.callback = args[1]
	return nil
}

func onClipboard(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.clipboard.callback = args[1]
	return nil
}

func onNotification(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.notification.callback = args[1]
	return nil
}

func onPTYWrite(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.ptyWriter.callback = args[1]
	return nil
}

func onSize(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.size.callback = args[1]
	return nil
}

func onSemanticPrompt(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.semanticPrompt.callback = args[1]
	return nil
}

func onAPC(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.apc.callback = args[1]
	return nil
}

func onPM(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.pm.callback = args[1]
	return nil
}

func onSOS(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.sos.callback = args[1]
	return nil
}

func onScrollback(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.scrollback.callback = args[1]
	return nil
}

func onRecording(_ js.Value, args []js.Value) interface{} {
	if len(args) < 2 {
		return nil
	}
	inst := getInstance(args[0].Int())
	if inst == nil {
		return nil
	}
	inst.handlers.recording.callback = args[1]
	return nil
}
