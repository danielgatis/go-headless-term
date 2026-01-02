//go:build js && wasm

package main

import (
	"syscall/js"

	"github.com/danielgatis/go-ansicode"
	headlessterm "github.com/danielgatis/go-headless-term"
)

// jsHandlers holds all JavaScript callback handlers for a terminal instance
type jsHandlers struct {
	bell           *jsBellProvider
	title          *jsTitleProvider
	clipboard      *jsClipboardProvider
	notification   *jsNotificationProvider
	ptyWriter      *jsPTYWriter
	size           *jsSizeProvider
	semanticPrompt *jsSemanticPromptHandler
	apc            *jsAPCProvider
	pm             *jsPMProvider
	sos            *jsSOSProvider
	scrollback     *jsScrollbackProvider
	recording      *jsRecordingProvider
}

func newJSHandlers() *jsHandlers {
	return &jsHandlers{
		bell:           &jsBellProvider{},
		title:          &jsTitleProvider{},
		clipboard:      &jsClipboardProvider{},
		notification:   &jsNotificationProvider{},
		ptyWriter:      &jsPTYWriter{},
		size:           &jsSizeProvider{},
		semanticPrompt: &jsSemanticPromptHandler{},
		apc:            &jsAPCProvider{},
		pm:             &jsPMProvider{},
		sos:            &jsSOSProvider{},
		scrollback:     &jsScrollbackProvider{},
		recording:      &jsRecordingProvider{},
	}
}

// ============================================================================
// Bell Provider - calls onBell()
// ============================================================================

type jsBellProvider struct {
	callback js.Value
}

func (p *jsBellProvider) Ring() {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}
	p.callback.Invoke()
}

var _ headlessterm.BellProvider = (*jsBellProvider)(nil)

// ============================================================================
// Title Provider - calls onTitle(event, title)
// event: "set", "push", "pop"
// ============================================================================

type jsTitleProvider struct {
	callback js.Value
}

func (p *jsTitleProvider) SetTitle(title string) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}
	p.callback.Invoke("set", title)
}

func (p *jsTitleProvider) PushTitle() {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}
	p.callback.Invoke("push", "")
}

func (p *jsTitleProvider) PopTitle() {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}
	p.callback.Invoke("pop", "")
}

var _ headlessterm.TitleProvider = (*jsTitleProvider)(nil)

// ============================================================================
// Clipboard Provider - calls onClipboard(event, clipboard, data)
// event: "read", "write"
// Returns string for read operations
// ============================================================================

type jsClipboardProvider struct {
	callback js.Value
}

func (p *jsClipboardProvider) Read(clipboard byte) string {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return ""
	}
	result := p.callback.Invoke("read", string(clipboard), "")
	if result.IsUndefined() || result.IsNull() {
		return ""
	}
	return result.String()
}

func (p *jsClipboardProvider) Write(clipboard byte, data []byte) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}
	p.callback.Invoke("write", string(clipboard), string(data))
}

var _ headlessterm.ClipboardProvider = (*jsClipboardProvider)(nil)

// ============================================================================
// Notification Provider - calls onNotification(payload)
// payload: { title, body, urgency, ... }
// ============================================================================

type jsNotificationProvider struct {
	callback js.Value
}

func (p *jsNotificationProvider) Notify(payload *headlessterm.NotificationPayload) string {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return ""
	}

	jsPayload := map[string]interface{}{
		"id":          payload.ID,
		"payloadType": payload.PayloadType,
		"data":        string(payload.Data),
	}

	if payload.AppName != "" {
		jsPayload["appName"] = payload.AppName
	}
	if payload.Type != "" {
		jsPayload["type"] = payload.Type
	}
	if payload.Urgency != 0 {
		jsPayload["urgency"] = int(payload.Urgency)
	}
	if payload.IconName != "" {
		jsPayload["iconName"] = payload.IconName
	}
	if payload.Sound != "" {
		jsPayload["sound"] = payload.Sound
	}
	if payload.Timeout != 0 {
		jsPayload["timeout"] = payload.Timeout
	}

	result := p.callback.Invoke(js.ValueOf(jsPayload))
	if result.IsUndefined() || result.IsNull() {
		return ""
	}
	return result.String()
}

var _ headlessterm.NotificationProvider = (*jsNotificationProvider)(nil)

// ============================================================================
// PTY Writer - calls onPTYWrite(data)
// data: Uint8Array of response bytes
// ============================================================================

type jsPTYWriter struct {
	callback js.Value
}

func (p *jsPTYWriter) Write(data []byte) (int, error) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return len(data), nil
	}

	// Create Uint8Array and copy data
	jsArray := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(jsArray, data)

	p.callback.Invoke(jsArray)
	return len(data), nil
}

var _ headlessterm.PTYWriter = (*jsPTYWriter)(nil)

// ============================================================================
// Size Provider - calls onSize(query)
// query: "window" or "cell"
// Returns { width, height }
// ============================================================================

type jsSizeProvider struct {
	callback js.Value
}

func (p *jsSizeProvider) WindowSizePixels() (width, height int) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return 800, 600 // defaults
	}

	result := p.callback.Invoke("window")
	if result.IsUndefined() || result.IsNull() {
		return 800, 600
	}

	w := result.Get("width")
	h := result.Get("height")
	if w.IsUndefined() || h.IsUndefined() {
		return 800, 600
	}

	return w.Int(), h.Int()
}

func (p *jsSizeProvider) CellSizePixels() (width, height int) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return 10, 20 // defaults
	}

	result := p.callback.Invoke("cell")
	if result.IsUndefined() || result.IsNull() {
		return 10, 20
	}

	w := result.Get("width")
	h := result.Get("height")
	if w.IsUndefined() || h.IsUndefined() {
		return 10, 20
	}

	return w.Int(), h.Int()
}

var _ headlessterm.SizeProvider = (*jsSizeProvider)(nil)

// ============================================================================
// Semantic Prompt Handler - calls onSemanticPrompt(mark, exitCode)
// mark: 0=PromptStart, 1=PromptEnd, 2=CommandStart, 3=CommandEnd
// ============================================================================

type jsSemanticPromptHandler struct {
	callback js.Value
}

func (p *jsSemanticPromptHandler) OnMark(mark ansicode.ShellIntegrationMark, exitCode int) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}
	p.callback.Invoke(int(mark), exitCode)
}

var _ headlessterm.SemanticPromptHandler = (*jsSemanticPromptHandler)(nil)

// ============================================================================
// APC Provider - calls onAPC(data)
// data: Uint8Array
// ============================================================================

type jsAPCProvider struct {
	callback js.Value
}

func (p *jsAPCProvider) Receive(data []byte) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}

	jsArray := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(jsArray, data)
	p.callback.Invoke(jsArray)
}

var _ headlessterm.APCProvider = (*jsAPCProvider)(nil)

// ============================================================================
// PM Provider - calls onPM(data)
// data: Uint8Array
// ============================================================================

type jsPMProvider struct {
	callback js.Value
}

func (p *jsPMProvider) Receive(data []byte) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}

	jsArray := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(jsArray, data)
	p.callback.Invoke(jsArray)
}

var _ headlessterm.PMProvider = (*jsPMProvider)(nil)

// ============================================================================
// SOS Provider - calls onSOS(data)
// data: Uint8Array
// ============================================================================

type jsSOSProvider struct {
	callback js.Value
}

func (p *jsSOSProvider) Receive(data []byte) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}

	jsArray := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(jsArray, data)
	p.callback.Invoke(jsArray)
}

var _ headlessterm.SOSProvider = (*jsSOSProvider)(nil)

// ============================================================================
// Scrollback Provider - calls onScrollback(event, data)
// event: "push", "pop", "len", "line", "clear", "setMaxLines", "maxLines"
// ============================================================================

type jsScrollbackProvider struct {
	callback js.Value
}

func (p *jsScrollbackProvider) Push(line []headlessterm.Cell) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}
	jsLine := cellsToJS(line)
	p.callback.Invoke("push", jsLine)
}

func (p *jsScrollbackProvider) Pop() []headlessterm.Cell {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return nil
	}
	result := p.callback.Invoke("pop")
	if result.IsUndefined() || result.IsNull() {
		return nil
	}
	return jsToCells(result)
}

func (p *jsScrollbackProvider) Len() int {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return 0
	}
	result := p.callback.Invoke("len")
	if result.IsUndefined() || result.IsNull() {
		return 0
	}
	return result.Int()
}

func (p *jsScrollbackProvider) Line(index int) []headlessterm.Cell {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return nil
	}
	result := p.callback.Invoke("line", index)
	if result.IsUndefined() || result.IsNull() {
		return nil
	}
	return jsToCells(result)
}

func (p *jsScrollbackProvider) Clear() {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}
	p.callback.Invoke("clear")
}

func (p *jsScrollbackProvider) SetMaxLines(max int) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}
	p.callback.Invoke("setMaxLines", max)
}

func (p *jsScrollbackProvider) MaxLines() int {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return 0
	}
	result := p.callback.Invoke("maxLines")
	if result.IsUndefined() || result.IsNull() {
		return 0
	}
	return result.Int()
}

var _ headlessterm.ScrollbackProvider = (*jsScrollbackProvider)(nil)

// ============================================================================
// Recording Provider - calls onRecording(event, data)
// event: "record", "data", "clear"
// ============================================================================

type jsRecordingProvider struct {
	callback js.Value
}

func (p *jsRecordingProvider) Record(data []byte) {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}
	jsArray := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(jsArray, data)
	p.callback.Invoke("record", jsArray)
}

func (p *jsRecordingProvider) Data() []byte {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return nil
	}
	result := p.callback.Invoke("data")
	if result.IsUndefined() || result.IsNull() {
		return nil
	}
	data := make([]byte, result.Length())
	js.CopyBytesToGo(data, result)
	return data
}

func (p *jsRecordingProvider) Clear() {
	if p.callback.IsUndefined() || p.callback.IsNull() {
		return
	}
	p.callback.Invoke("clear")
}

var _ headlessterm.RecordingProvider = (*jsRecordingProvider)(nil)

// ============================================================================
// Helper functions for cell conversion
// ============================================================================

func cellsToJS(cells []headlessterm.Cell) js.Value {
	arr := make([]interface{}, len(cells))
	for i, c := range cells {
		arr[i] = map[string]interface{}{
			"char":  string(c.Char),
			"flags": int(c.Flags),
		}
	}
	return js.ValueOf(arr)
}

func jsToCells(v js.Value) []headlessterm.Cell {
	if !v.Truthy() {
		return nil
	}
	length := v.Length()
	cells := make([]headlessterm.Cell, length)
	for i := 0; i < length; i++ {
		item := v.Index(i)
		charStr := item.Get("char").String()
		if len(charStr) > 0 {
			cells[i].Char = []rune(charStr)[0]
		}
		cells[i].Flags = headlessterm.CellFlags(item.Get("flags").Int())
	}
	return cells
}
