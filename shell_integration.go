package headlessterm

import (
	"github.com/danielgatis/go-ansicode"
)

// PromptMark stores information about a shell integration mark (OSC 133).
// Used for prompt-based navigation in scrollback.
type PromptMark struct {
	// Type is the mark type (PromptStart, CommandStart, CommandExecuted, CommandFinished).
	Type ansicode.ShellIntegrationMark
	// Row is the absolute row position (including scrollback offset).
	// Negative values indicate scrollback lines (-1 is most recent scrollback line).
	Row int
	// ExitCode is the command exit code (only valid for CommandFinished marks, -1 otherwise).
	ExitCode int
}

// ShellIntegrationProvider handles shell integration events (OSC 133).
type ShellIntegrationProvider interface {
	// OnMark is called when a shell integration mark is received.
	OnMark(mark ansicode.ShellIntegrationMark, exitCode int)
}

// NoopShellIntegration ignores all shell integration events.
type NoopShellIntegration struct{}

func (NoopShellIntegration) OnMark(mark ansicode.ShellIntegrationMark, exitCode int) {}

// Ensure NoopShellIntegration satisfies the interface
var _ ShellIntegrationProvider = (*NoopShellIntegration)(nil)

// ShellIntegrationMark processes a shell integration mark (OSC 133).
// Records the mark position for prompt-based navigation.
func (t *Terminal) ShellIntegrationMark(mark ansicode.ShellIntegrationMark, exitCode int) {
	if t.middleware != nil && t.middleware.ShellIntegrationMark != nil {
		t.middleware.ShellIntegrationMark(mark, exitCode, t.shellIntegrationMarkInternal)
		return
	}
	t.shellIntegrationMarkInternal(mark, exitCode)
}

func (t *Terminal) shellIntegrationMarkInternal(mark ansicode.ShellIntegrationMark, exitCode int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Calculate absolute row (accounting for scrollback)
	scrollbackLen := t.primaryBuffer.ScrollbackLen()
	absoluteRow := t.cursor.Row + scrollbackLen

	// Store the mark
	t.promptMarks = append(t.promptMarks, PromptMark{
		Type:     mark,
		Row:      absoluteRow,
		ExitCode: exitCode,
	})

	// Notify provider if set
	if t.shellIntegrationProvider != nil {
		t.shellIntegrationProvider.OnMark(mark, exitCode)
	}
}

// PromptMarks returns all recorded prompt marks.
func (t *Terminal) PromptMarks() []PromptMark {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Return a copy to prevent external modification
	marks := make([]PromptMark, len(t.promptMarks))
	copy(marks, t.promptMarks)
	return marks
}

// PromptMarkCount returns the number of recorded prompt marks.
func (t *Terminal) PromptMarkCount() int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return len(t.promptMarks)
}

// ClearPromptMarks removes all recorded prompt marks.
func (t *Terminal) ClearPromptMarks() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.promptMarks = nil
}

// NextPromptRow returns the absolute row of the next prompt mark after the given absolute row.
// Returns -1 if no next prompt exists.
// If markType is specified (not -1), only returns marks of that type.
func (t *Terminal) NextPromptRow(currentAbsRow int, markType ansicode.ShellIntegrationMark) int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for _, mark := range t.promptMarks {
		if mark.Row > currentAbsRow {
			if markType == -1 || mark.Type == markType {
				return mark.Row
			}
		}
	}
	return -1
}

// PrevPromptRow returns the absolute row of the previous prompt mark before the given absolute row.
// Returns -1 if no previous prompt exists.
// If markType is specified (not -1), only returns marks of that type.
func (t *Terminal) PrevPromptRow(currentAbsRow int, markType ansicode.ShellIntegrationMark) int {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Search backwards
	for i := len(t.promptMarks) - 1; i >= 0; i-- {
		mark := t.promptMarks[i]
		if mark.Row < currentAbsRow {
			if markType == -1 || mark.Type == markType {
				return mark.Row
			}
		}
	}
	return -1
}

// GetPromptMarkAt returns the prompt mark at the given absolute row, or nil if none exists.
func (t *Terminal) GetPromptMarkAt(absRow int) *PromptMark {
	t.mu.RLock()
	defer t.mu.RUnlock()

	for i := range t.promptMarks {
		if t.promptMarks[i].Row == absRow {
			mark := t.promptMarks[i]
			return &mark
		}
	}
	return nil
}

// SetShellIntegrationProvider sets the shell integration provider at runtime.
func (t *Terminal) SetShellIntegrationProvider(p ShellIntegrationProvider) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.shellIntegrationProvider = p
}

// ShellIntegrationProviderValue returns the current shell integration provider.
func (t *Terminal) ShellIntegrationProviderValue() ShellIntegrationProvider {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.shellIntegrationProvider
}

// GetLastCommandOutput returns the output of the last executed command.
// It finds the text between the last CommandExecuted (C) mark and the last CommandFinished (D) mark.
// Returns empty string if no complete command output is available.
func (t *Terminal) GetLastCommandOutput() string {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.promptMarks) == 0 {
		return ""
	}

	// Find the last CommandExecuted and CommandFinished marks
	var lastExecuted, lastFinished *PromptMark
	for i := len(t.promptMarks) - 1; i >= 0; i-- {
		mark := &t.promptMarks[i]
		if lastFinished == nil && mark.Type == ansicode.CommandFinished {
			lastFinished = mark
		}
		if lastExecuted == nil && mark.Type == ansicode.CommandExecuted {
			lastExecuted = mark
		}
		// Once we have both, check if they form a valid pair
		if lastExecuted != nil && lastFinished != nil {
			// CommandExecuted must come before CommandFinished
			if lastExecuted.Row < lastFinished.Row {
				break
			}
			// Invalid pair, continue searching
			lastFinished = nil
			lastExecuted = nil
		}
	}

	if lastExecuted == nil || lastFinished == nil {
		return ""
	}

	// Extract text between the two marks
	return t.extractTextBetweenRows(lastExecuted.Row, lastFinished.Row)
}

// extractTextBetweenRows extracts text from startRow (inclusive) to endRow (exclusive).
// Rows are absolute (including scrollback offset).
func (t *Terminal) extractTextBetweenRows(startRow, endRow int) string {
	scrollbackLen := t.primaryBuffer.ScrollbackLen()

	var lines []string
	// Start from the CommandExecuted row (inclusive) to CommandFinished row (exclusive)
	for absRow := startRow; absRow < endRow; absRow++ {
		var lineContent string

		if absRow < scrollbackLen {
			// Row is in scrollback
			scrollbackLine := t.primaryBuffer.ScrollbackLine(absRow)
			if scrollbackLine != nil {
				lineContent = t.cellsToString(scrollbackLine)
			}
		} else {
			// Row is in visible buffer
			bufferRow := absRow - scrollbackLen
			if bufferRow >= 0 && bufferRow < t.rows {
				lineContent = t.activeBuffer.LineContent(bufferRow)
			}
		}

		lines = append(lines, lineContent)
	}

	// Join lines, trimming trailing empty lines
	result := ""
	lastNonEmpty := -1
	for i, line := range lines {
		if line != "" {
			lastNonEmpty = i
		}
	}

	if lastNonEmpty < 0 {
		return ""
	}

	for i := 0; i <= lastNonEmpty; i++ {
		if i > 0 {
			result += "\n"
		}
		result += lines[i]
	}

	return result
}

// cellsToString converts a slice of cells to a string.
func (t *Terminal) cellsToString(cells []Cell) string {
	// Find the last non-space character
	lastNonSpace := -1
	for i := len(cells) - 1; i >= 0; i-- {
		cell := &cells[i]
		if cell.Char != ' ' && cell.Char != 0 && !cell.IsWideSpacer() {
			lastNonSpace = i
			break
		}
	}

	if lastNonSpace < 0 {
		return ""
	}

	runes := make([]rune, 0, lastNonSpace+1)
	for i := 0; i <= lastNonSpace; i++ {
		cell := &cells[i]
		if cell.IsWideSpacer() {
			continue
		}
		if cell.Char == 0 {
			runes = append(runes, ' ')
		} else {
			runes = append(runes, cell.Char)
		}
	}

	return string(runes)
}
