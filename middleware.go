package headlessterm

import (
	"image/color"

	"github.com/danielgatis/go-ansicode"
)

// Middleware intercepts ANSI handler calls, allowing custom behavior before/after execution.
// Each field wraps one handler: receive original parameters and a next function to call the default implementation.
type Middleware struct {
	// Input wraps the Input handler
	Input func(r rune, next func(rune))

	// Bell wraps the Bell handler
	Bell func(next func())

	// Backspace wraps the Backspace handler
	Backspace func(next func())

	// CarriageReturn wraps the CarriageReturn handler
	CarriageReturn func(next func())

	// LineFeed wraps the LineFeed handler
	LineFeed func(next func())

	// Tab wraps the Tab handler
	Tab func(n int, next func(int))

	// ClearLine wraps the ClearLine handler
	ClearLine func(mode ansicode.LineClearMode, next func(ansicode.LineClearMode))

	// ClearScreen wraps the ClearScreen handler
	ClearScreen func(mode ansicode.ClearMode, next func(ansicode.ClearMode))

	// ClearTabs wraps the ClearTabs handler
	ClearTabs func(mode ansicode.TabulationClearMode, next func(ansicode.TabulationClearMode))

	// Goto wraps the Goto handler
	Goto func(row, col int, next func(int, int))

	// GotoLine wraps the GotoLine handler
	GotoLine func(row int, next func(int))

	// GotoCol wraps the GotoCol handler
	GotoCol func(col int, next func(int))

	// MoveUp wraps the MoveUp handler
	MoveUp func(n int, next func(int))

	// MoveDown wraps the MoveDown handler
	MoveDown func(n int, next func(int))

	// MoveForward wraps the MoveForward handler
	MoveForward func(n int, next func(int))

	// MoveBackward wraps the MoveBackward handler
	MoveBackward func(n int, next func(int))

	// MoveUpCr wraps the MoveUpCr handler
	MoveUpCr func(n int, next func(int))

	// MoveDownCr wraps the MoveDownCr handler
	MoveDownCr func(n int, next func(int))

	// MoveForwardTabs wraps the MoveForwardTabs handler
	MoveForwardTabs func(n int, next func(int))

	// MoveBackwardTabs wraps the MoveBackwardTabs handler
	MoveBackwardTabs func(n int, next func(int))

	// InsertBlank wraps the InsertBlank handler
	InsertBlank func(n int, next func(int))

	// InsertBlankLines wraps the InsertBlankLines handler
	InsertBlankLines func(n int, next func(int))

	// DeleteChars wraps the DeleteChars handler
	DeleteChars func(n int, next func(int))

	// DeleteLines wraps the DeleteLines handler
	DeleteLines func(n int, next func(int))

	// EraseChars wraps the EraseChars handler
	EraseChars func(n int, next func(int))

	// ScrollUp wraps the ScrollUp handler
	ScrollUp func(n int, next func(int))

	// ScrollDown wraps the ScrollDown handler
	ScrollDown func(n int, next func(int))

	// SetScrollingRegion wraps the SetScrollingRegion handler
	SetScrollingRegion func(top, bottom int, next func(int, int))

	// SetMode wraps the SetMode handler
	SetMode func(mode ansicode.TerminalMode, next func(ansicode.TerminalMode))

	// UnsetMode wraps the UnsetMode handler
	UnsetMode func(mode ansicode.TerminalMode, next func(ansicode.TerminalMode))

	// SetTerminalCharAttribute wraps the SetTerminalCharAttribute handler
	SetTerminalCharAttribute func(attr ansicode.TerminalCharAttribute, next func(ansicode.TerminalCharAttribute))

	// SetTitle wraps the SetTitle handler
	SetTitle func(title string, next func(string))

	// SetCursorStyle wraps the SetCursorStyle handler
	SetCursorStyle func(style ansicode.CursorStyle, next func(ansicode.CursorStyle))

	// SaveCursorPosition wraps the SaveCursorPosition handler
	SaveCursorPosition func(next func())

	// RestoreCursorPosition wraps the RestoreCursorPosition handler
	RestoreCursorPosition func(next func())

	// ReverseIndex wraps the ReverseIndex handler
	ReverseIndex func(next func())

	// ResetState wraps the ResetState handler
	ResetState func(next func())

	// Substitute wraps the Substitute handler
	Substitute func(next func())

	// Decaln wraps the Decaln handler
	Decaln func(next func())

	// DeviceStatus wraps the DeviceStatus handler
	DeviceStatus func(n int, next func(int))

	// IdentifyTerminal wraps the IdentifyTerminal handler
	IdentifyTerminal func(b byte, next func(byte))

	// ConfigureCharset wraps the ConfigureCharset handler
	ConfigureCharset func(index ansicode.CharsetIndex, charset ansicode.Charset, next func(ansicode.CharsetIndex, ansicode.Charset))

	// SetActiveCharset wraps the SetActiveCharset handler
	SetActiveCharset func(n int, next func(int))

	// SetKeypadApplicationMode wraps the SetKeypadApplicationMode handler
	SetKeypadApplicationMode func(next func())

	// UnsetKeypadApplicationMode wraps the UnsetKeypadApplicationMode handler
	UnsetKeypadApplicationMode func(next func())

	// SetColor wraps the SetColor handler
	SetColor func(index int, c color.Color, next func(int, color.Color))

	// ResetColor wraps the ResetColor handler
	ResetColor func(i int, next func(int))

	// SetDynamicColor wraps the SetDynamicColor handler
	SetDynamicColor func(prefix string, index int, terminator string, next func(string, int, string))

	// ClipboardLoad wraps the ClipboardLoad handler
	ClipboardLoad func(clipboard byte, terminator string, next func(byte, string))

	// ClipboardStore wraps the ClipboardStore handler
	ClipboardStore func(clipboard byte, data []byte, next func(byte, []byte))

	// SetHyperlink wraps the SetHyperlink handler
	SetHyperlink func(hyperlink *ansicode.Hyperlink, next func(*ansicode.Hyperlink))

	// PushTitle wraps the PushTitle handler
	PushTitle func(next func())

	// PopTitle wraps the PopTitle handler
	PopTitle func(next func())

	// TextAreaSizeChars wraps the TextAreaSizeChars handler
	TextAreaSizeChars func(next func())

	// TextAreaSizePixels wraps the TextAreaSizePixels handler
	TextAreaSizePixels func(next func())

	// HorizontalTabSet wraps the HorizontalTabSet handler
	HorizontalTabSet func(next func())

	// SetKeyboardMode wraps the SetKeyboardMode handler
	SetKeyboardMode func(mode ansicode.KeyboardMode, behavior ansicode.KeyboardModeBehavior, next func(ansicode.KeyboardMode, ansicode.KeyboardModeBehavior))

	// PushKeyboardMode wraps the PushKeyboardMode handler
	PushKeyboardMode func(mode ansicode.KeyboardMode, next func(ansicode.KeyboardMode))

	// PopKeyboardMode wraps the PopKeyboardMode handler
	PopKeyboardMode func(n int, next func(int))

	// ReportKeyboardMode wraps the ReportKeyboardMode handler
	ReportKeyboardMode func(next func())

	// SetModifyOtherKeys wraps the SetModifyOtherKeys handler
	SetModifyOtherKeys func(modify ansicode.ModifyOtherKeys, next func(ansicode.ModifyOtherKeys))

	// ReportModifyOtherKeys wraps the ReportModifyOtherKeys handler
	ReportModifyOtherKeys func(next func())

	// ApplicationCommandReceived wraps the ApplicationCommandReceived handler
	ApplicationCommandReceived func(data []byte, next func([]byte))

	// PrivacyMessageReceived wraps the PrivacyMessageReceived handler
	PrivacyMessageReceived func(data []byte, next func([]byte))

	// StartOfStringReceived wraps the StartOfStringReceived handler
	StartOfStringReceived func(data []byte, next func([]byte))

	// SemanticPromptMark wraps the SemanticPromptMark handler
	SemanticPromptMark func(mark ansicode.ShellIntegrationMark, exitCode int, next func(ansicode.ShellIntegrationMark, int))

	// SetWorkingDirectory wraps the SetWorkingDirectory handler
	SetWorkingDirectory func(uri string, next func(string))

	// SixelReceived wraps the SixelReceived handler
	SixelReceived func(params [][]uint16, data []byte, next func([][]uint16, []byte))

	// DesktopNotification wraps the DesktopNotification handler (OSC 99)
	DesktopNotification func(payload *NotificationPayload, next func(*NotificationPayload))

	// SetUserVar wraps the SetUserVar handler (OSC 1337)
	SetUserVar func(name, value string, next func(string, string))
}

// Merge copies non-nil middleware functions from other into this, overwriting existing values.
func (m *Middleware) Merge(other *Middleware) {
	if other == nil {
		return
	}

	if other.Input != nil {
		m.Input = other.Input
	}
	if other.Bell != nil {
		m.Bell = other.Bell
	}
	if other.Backspace != nil {
		m.Backspace = other.Backspace
	}
	if other.CarriageReturn != nil {
		m.CarriageReturn = other.CarriageReturn
	}
	if other.LineFeed != nil {
		m.LineFeed = other.LineFeed
	}
	if other.Tab != nil {
		m.Tab = other.Tab
	}
	if other.ClearLine != nil {
		m.ClearLine = other.ClearLine
	}
	if other.ClearScreen != nil {
		m.ClearScreen = other.ClearScreen
	}
	if other.ClearTabs != nil {
		m.ClearTabs = other.ClearTabs
	}
	if other.Goto != nil {
		m.Goto = other.Goto
	}
	if other.GotoLine != nil {
		m.GotoLine = other.GotoLine
	}
	if other.GotoCol != nil {
		m.GotoCol = other.GotoCol
	}
	if other.MoveUp != nil {
		m.MoveUp = other.MoveUp
	}
	if other.MoveDown != nil {
		m.MoveDown = other.MoveDown
	}
	if other.MoveForward != nil {
		m.MoveForward = other.MoveForward
	}
	if other.MoveBackward != nil {
		m.MoveBackward = other.MoveBackward
	}
	if other.MoveUpCr != nil {
		m.MoveUpCr = other.MoveUpCr
	}
	if other.MoveDownCr != nil {
		m.MoveDownCr = other.MoveDownCr
	}
	if other.MoveForwardTabs != nil {
		m.MoveForwardTabs = other.MoveForwardTabs
	}
	if other.MoveBackwardTabs != nil {
		m.MoveBackwardTabs = other.MoveBackwardTabs
	}
	if other.InsertBlank != nil {
		m.InsertBlank = other.InsertBlank
	}
	if other.InsertBlankLines != nil {
		m.InsertBlankLines = other.InsertBlankLines
	}
	if other.DeleteChars != nil {
		m.DeleteChars = other.DeleteChars
	}
	if other.DeleteLines != nil {
		m.DeleteLines = other.DeleteLines
	}
	if other.EraseChars != nil {
		m.EraseChars = other.EraseChars
	}
	if other.ScrollUp != nil {
		m.ScrollUp = other.ScrollUp
	}
	if other.ScrollDown != nil {
		m.ScrollDown = other.ScrollDown
	}
	if other.SetScrollingRegion != nil {
		m.SetScrollingRegion = other.SetScrollingRegion
	}
	if other.SetMode != nil {
		m.SetMode = other.SetMode
	}
	if other.UnsetMode != nil {
		m.UnsetMode = other.UnsetMode
	}
	if other.SetTerminalCharAttribute != nil {
		m.SetTerminalCharAttribute = other.SetTerminalCharAttribute
	}
	if other.SetTitle != nil {
		m.SetTitle = other.SetTitle
	}
	if other.SetCursorStyle != nil {
		m.SetCursorStyle = other.SetCursorStyle
	}
	if other.SaveCursorPosition != nil {
		m.SaveCursorPosition = other.SaveCursorPosition
	}
	if other.RestoreCursorPosition != nil {
		m.RestoreCursorPosition = other.RestoreCursorPosition
	}
	if other.ReverseIndex != nil {
		m.ReverseIndex = other.ReverseIndex
	}
	if other.ResetState != nil {
		m.ResetState = other.ResetState
	}
	if other.Substitute != nil {
		m.Substitute = other.Substitute
	}
	if other.Decaln != nil {
		m.Decaln = other.Decaln
	}
	if other.DeviceStatus != nil {
		m.DeviceStatus = other.DeviceStatus
	}
	if other.IdentifyTerminal != nil {
		m.IdentifyTerminal = other.IdentifyTerminal
	}
	if other.ConfigureCharset != nil {
		m.ConfigureCharset = other.ConfigureCharset
	}
	if other.SetActiveCharset != nil {
		m.SetActiveCharset = other.SetActiveCharset
	}
	if other.SetKeypadApplicationMode != nil {
		m.SetKeypadApplicationMode = other.SetKeypadApplicationMode
	}
	if other.UnsetKeypadApplicationMode != nil {
		m.UnsetKeypadApplicationMode = other.UnsetKeypadApplicationMode
	}
	if other.SetColor != nil {
		m.SetColor = other.SetColor
	}
	if other.ResetColor != nil {
		m.ResetColor = other.ResetColor
	}
	if other.SetDynamicColor != nil {
		m.SetDynamicColor = other.SetDynamicColor
	}
	if other.ClipboardLoad != nil {
		m.ClipboardLoad = other.ClipboardLoad
	}
	if other.ClipboardStore != nil {
		m.ClipboardStore = other.ClipboardStore
	}
	if other.SetHyperlink != nil {
		m.SetHyperlink = other.SetHyperlink
	}
	if other.PushTitle != nil {
		m.PushTitle = other.PushTitle
	}
	if other.PopTitle != nil {
		m.PopTitle = other.PopTitle
	}
	if other.TextAreaSizeChars != nil {
		m.TextAreaSizeChars = other.TextAreaSizeChars
	}
	if other.TextAreaSizePixels != nil {
		m.TextAreaSizePixels = other.TextAreaSizePixels
	}
	if other.HorizontalTabSet != nil {
		m.HorizontalTabSet = other.HorizontalTabSet
	}
	if other.SetKeyboardMode != nil {
		m.SetKeyboardMode = other.SetKeyboardMode
	}
	if other.PushKeyboardMode != nil {
		m.PushKeyboardMode = other.PushKeyboardMode
	}
	if other.PopKeyboardMode != nil {
		m.PopKeyboardMode = other.PopKeyboardMode
	}
	if other.ReportKeyboardMode != nil {
		m.ReportKeyboardMode = other.ReportKeyboardMode
	}
	if other.SetModifyOtherKeys != nil {
		m.SetModifyOtherKeys = other.SetModifyOtherKeys
	}
	if other.ReportModifyOtherKeys != nil {
		m.ReportModifyOtherKeys = other.ReportModifyOtherKeys
	}
	if other.ApplicationCommandReceived != nil {
		m.ApplicationCommandReceived = other.ApplicationCommandReceived
	}
	if other.PrivacyMessageReceived != nil {
		m.PrivacyMessageReceived = other.PrivacyMessageReceived
	}
	if other.StartOfStringReceived != nil {
		m.StartOfStringReceived = other.StartOfStringReceived
	}
	if other.SemanticPromptMark != nil {
		m.SemanticPromptMark = other.SemanticPromptMark
	}
	if other.SetWorkingDirectory != nil {
		m.SetWorkingDirectory = other.SetWorkingDirectory
	}
	if other.SixelReceived != nil {
		m.SixelReceived = other.SixelReceived
	}
	if other.DesktopNotification != nil {
		m.DesktopNotification = other.DesktopNotification
	}
	if other.SetUserVar != nil {
		m.SetUserVar = other.SetUserVar
	}
}
