package headlessterm

import (
	"testing"

	"github.com/danielgatis/go-ansicode"
)

func TestShellIntegrationMark_PromptStart(t *testing.T) {
	term := New(WithSize(24, 80))

	// OSC 133 ; A BEL - Prompt start
	term.WriteString("\x1b]133;A\x07")

	marks := term.PromptMarks()
	if len(marks) != 1 {
		t.Fatalf("expected 1 mark, got %d", len(marks))
	}

	if marks[0].Type != ansicode.PromptStart {
		t.Errorf("expected PromptStart mark, got %d", marks[0].Type)
	}
	if marks[0].ExitCode != -1 {
		t.Errorf("expected exit code -1, got %d", marks[0].ExitCode)
	}
}

func TestShellIntegrationMark_CommandStart(t *testing.T) {
	term := New(WithSize(24, 80))

	// OSC 133 ; B BEL - Command start
	term.WriteString("\x1b]133;B\x07")

	marks := term.PromptMarks()
	if len(marks) != 1 {
		t.Fatalf("expected 1 mark, got %d", len(marks))
	}

	if marks[0].Type != ansicode.CommandStart {
		t.Errorf("expected CommandStart mark, got %d", marks[0].Type)
	}
}

func TestShellIntegrationMark_CommandExecuted(t *testing.T) {
	term := New(WithSize(24, 80))

	// OSC 133 ; C BEL - Command executed
	term.WriteString("\x1b]133;C\x07")

	marks := term.PromptMarks()
	if len(marks) != 1 {
		t.Fatalf("expected 1 mark, got %d", len(marks))
	}

	if marks[0].Type != ansicode.CommandExecuted {
		t.Errorf("expected CommandExecuted mark, got %d", marks[0].Type)
	}
}

func TestShellIntegrationMark_CommandFinished(t *testing.T) {
	term := New(WithSize(24, 80))

	// OSC 133 ; D BEL - Command finished (no exit code)
	term.WriteString("\x1b]133;D\x07")

	marks := term.PromptMarks()
	if len(marks) != 1 {
		t.Fatalf("expected 1 mark, got %d", len(marks))
	}

	if marks[0].Type != ansicode.CommandFinished {
		t.Errorf("expected CommandFinished mark, got %d", marks[0].Type)
	}
	if marks[0].ExitCode != -1 {
		t.Errorf("expected exit code -1, got %d", marks[0].ExitCode)
	}
}

func TestShellIntegrationMark_CommandFinishedWithExitCode(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		exitCode int
	}{
		{"exit code 0", "\x1b]133;D;0\x07", 0},
		{"exit code 1", "\x1b]133;D;1\x07", 1},
		{"exit code 127", "\x1b]133;D;127\x07", 127},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			term := New(WithSize(24, 80))
			term.WriteString(tt.input)

			marks := term.PromptMarks()
			if len(marks) != 1 {
				t.Fatalf("expected 1 mark, got %d", len(marks))
			}

			if marks[0].ExitCode != tt.exitCode {
				t.Errorf("expected exit code %d, got %d", tt.exitCode, marks[0].ExitCode)
			}
		})
	}
}

func TestShellIntegrationMark_FullSequence(t *testing.T) {
	term := New(WithSize(24, 80))

	// Simulate a full shell prompt cycle
	term.WriteString("\x1b]133;A\x07")          // Prompt start
	term.WriteString("$ ")                      // Prompt text
	term.WriteString("\x1b]133;B\x07")          // Command start
	term.WriteString("ls -la")                  // User input
	term.WriteString("\r\n")                    // Enter
	term.WriteString("\x1b]133;C\x07")          // Command executed
	term.WriteString("file1\r\nfile2\r\n")      // Command output
	term.WriteString("\x1b]133;D;0\x07")        // Command finished with exit code 0

	marks := term.PromptMarks()
	if len(marks) != 4 {
		t.Fatalf("expected 4 marks, got %d", len(marks))
	}

	// Check mark types in order
	expected := []ansicode.ShellIntegrationMark{
		ansicode.PromptStart,
		ansicode.CommandStart,
		ansicode.CommandExecuted,
		ansicode.CommandFinished,
	}

	for i, exp := range expected {
		if marks[i].Type != exp {
			t.Errorf("mark %d: expected type %d, got %d", i, exp, marks[i].Type)
		}
	}

	// Check exit code of the last mark
	if marks[3].ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", marks[3].ExitCode)
	}
}

func TestShellIntegrationMark_RowTracking(t *testing.T) {
	term := New(WithSize(24, 80))

	// Add marks at different rows
	term.WriteString("\x1b]133;A\x07")  // Row 0
	term.WriteString("prompt1\r\n")
	term.WriteString("\x1b]133;A\x07")  // Row 1
	term.WriteString("prompt2\r\n")
	term.WriteString("\x1b]133;A\x07")  // Row 2

	marks := term.PromptMarks()
	if len(marks) != 3 {
		t.Fatalf("expected 3 marks, got %d", len(marks))
	}

	// Rows should be tracked correctly
	if marks[0].Row != 0 {
		t.Errorf("mark 0: expected row 0, got %d", marks[0].Row)
	}
	if marks[1].Row != 1 {
		t.Errorf("mark 1: expected row 1, got %d", marks[1].Row)
	}
	if marks[2].Row != 2 {
		t.Errorf("mark 2: expected row 2, got %d", marks[2].Row)
	}
}

func TestShellIntegrationMark_NextPromptRow(t *testing.T) {
	term := New(WithSize(24, 80))

	// Add prompts at different rows
	term.WriteString("\x1b]133;A\x07")  // Row 0
	term.WriteString("prompt1\r\n")
	term.WriteString("\x1b]133;A\x07")  // Row 1
	term.WriteString("prompt2\r\n")
	term.WriteString("\x1b]133;A\x07")  // Row 2

	// Find next prompt from row -1 (before any content)
	next := term.NextPromptRow(-1, -1)
	if next != 0 {
		t.Errorf("expected next prompt at row 0, got %d", next)
	}

	// Find next prompt from row 0
	next = term.NextPromptRow(0, -1)
	if next != 1 {
		t.Errorf("expected next prompt at row 1, got %d", next)
	}

	// Find next prompt from row 1
	next = term.NextPromptRow(1, -1)
	if next != 2 {
		t.Errorf("expected next prompt at row 2, got %d", next)
	}

	// No next prompt from row 2
	next = term.NextPromptRow(2, -1)
	if next != -1 {
		t.Errorf("expected no next prompt (-1), got %d", next)
	}
}

func TestShellIntegrationMark_PrevPromptRow(t *testing.T) {
	term := New(WithSize(24, 80))

	// Add prompts at different rows
	term.WriteString("\x1b]133;A\x07")  // Row 0
	term.WriteString("prompt1\r\n")
	term.WriteString("\x1b]133;A\x07")  // Row 1
	term.WriteString("prompt2\r\n")
	term.WriteString("\x1b]133;A\x07")  // Row 2

	// Find previous prompt from row 3
	prev := term.PrevPromptRow(3, -1)
	if prev != 2 {
		t.Errorf("expected prev prompt at row 2, got %d", prev)
	}

	// Find previous prompt from row 2
	prev = term.PrevPromptRow(2, -1)
	if prev != 1 {
		t.Errorf("expected prev prompt at row 1, got %d", prev)
	}

	// Find previous prompt from row 1
	prev = term.PrevPromptRow(1, -1)
	if prev != 0 {
		t.Errorf("expected prev prompt at row 0, got %d", prev)
	}

	// No previous prompt from row 0
	prev = term.PrevPromptRow(0, -1)
	if prev != -1 {
		t.Errorf("expected no prev prompt (-1), got %d", prev)
	}
}

func TestShellIntegrationMark_FilterByType(t *testing.T) {
	term := New(WithSize(24, 80))

	// Add different mark types
	term.WriteString("\x1b]133;A\x07")  // PromptStart at row 0
	term.WriteString("prompt\r\n")
	term.WriteString("\x1b]133;B\x07")  // CommandStart at row 1
	term.WriteString("cmd\r\n")
	term.WriteString("\x1b]133;C\x07")  // CommandExecuted at row 2
	term.WriteString("output\r\n")
	term.WriteString("\x1b]133;A\x07")  // PromptStart at row 3

	// Find next PromptStart only
	next := term.NextPromptRow(-1, ansicode.PromptStart)
	if next != 0 {
		t.Errorf("expected next PromptStart at row 0, got %d", next)
	}

	next = term.NextPromptRow(0, ansicode.PromptStart)
	if next != 3 {
		t.Errorf("expected next PromptStart at row 3, got %d", next)
	}
}

func TestShellIntegrationMark_ClearMarks(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("\x1b]133;A\x07")
	term.WriteString("\x1b]133;B\x07")

	if term.PromptMarkCount() != 2 {
		t.Fatalf("expected 2 marks, got %d", term.PromptMarkCount())
	}

	term.ClearPromptMarks()

	if term.PromptMarkCount() != 0 {
		t.Errorf("expected 0 marks after clear, got %d", term.PromptMarkCount())
	}
}

func TestShellIntegrationMark_GetMarkAt(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("\x1b]133;A\x07")  // Row 0

	mark := term.GetPromptMarkAt(0)
	if mark == nil {
		t.Fatal("expected mark at row 0, got nil")
	}
	if mark.Type != ansicode.PromptStart {
		t.Errorf("expected PromptStart, got %d", mark.Type)
	}

	// No mark at row 1
	mark = term.GetPromptMarkAt(1)
	if mark != nil {
		t.Errorf("expected nil at row 1, got %v", mark)
	}
}

type testShellIntegrationProvider struct {
	marks []ansicode.ShellIntegrationMark
	codes []int
}

func (p *testShellIntegrationProvider) OnMark(mark ansicode.ShellIntegrationMark, exitCode int) {
	p.marks = append(p.marks, mark)
	p.codes = append(p.codes, exitCode)
}

func TestShellIntegrationMark_Provider(t *testing.T) {
	provider := &testShellIntegrationProvider{}
	term := New(WithSize(24, 80), WithShellIntegration(provider))

	term.WriteString("\x1b]133;A\x07")
	term.WriteString("\x1b]133;D;42\x07")

	if len(provider.marks) != 2 {
		t.Fatalf("expected provider to receive 2 marks, got %d", len(provider.marks))
	}

	if provider.marks[0] != ansicode.PromptStart {
		t.Errorf("expected PromptStart, got %d", provider.marks[0])
	}
	if provider.marks[1] != ansicode.CommandFinished {
		t.Errorf("expected CommandFinished, got %d", provider.marks[1])
	}
	if provider.codes[1] != 42 {
		t.Errorf("expected exit code 42, got %d", provider.codes[1])
	}
}

func TestShellIntegrationMark_ST_Terminator(t *testing.T) {
	term := New(WithSize(24, 80))

	// OSC 133 ; A ST (using ESC \ as string terminator)
	term.WriteString("\x1b]133;A\x1b\\")

	marks := term.PromptMarks()
	if len(marks) != 1 {
		t.Fatalf("expected 1 mark, got %d", len(marks))
	}

	if marks[0].Type != ansicode.PromptStart {
		t.Errorf("expected PromptStart mark, got %d", marks[0].Type)
	}
}

func TestShellIntegrationMark_Middleware(t *testing.T) {
	var middlewareCalled bool
	var receivedMark ansicode.ShellIntegrationMark
	var receivedExitCode int

	mw := &Middleware{
		ShellIntegrationMark: func(mark ansicode.ShellIntegrationMark, exitCode int, next func(ansicode.ShellIntegrationMark, int)) {
			middlewareCalled = true
			receivedMark = mark
			receivedExitCode = exitCode
			next(mark, exitCode)
		},
	}

	term := New(WithSize(24, 80), WithMiddleware(mw))

	term.WriteString("\x1b]133;D;123\x07")

	if !middlewareCalled {
		t.Error("expected middleware to be called")
	}
	if receivedMark != ansicode.CommandFinished {
		t.Errorf("expected CommandFinished, got %d", receivedMark)
	}
	if receivedExitCode != 123 {
		t.Errorf("expected exit code 123, got %d", receivedExitCode)
	}

	// Verify the mark was still stored
	if term.PromptMarkCount() != 1 {
		t.Errorf("expected 1 mark, got %d", term.PromptMarkCount())
	}
}

// --- GetLastCommandOutput Tests ---

func TestGetLastCommandOutput_Basic(t *testing.T) {
	term := New(WithSize(24, 80))

	// Simulate a command with output
	term.WriteString("\x1b]133;A\x07")       // Prompt start
	term.WriteString("$ ")
	term.WriteString("\x1b]133;B\x07")       // Command start
	term.WriteString("echo hello")
	term.WriteString("\r\n")
	term.WriteString("\x1b]133;C\x07")       // Command executed
	term.WriteString("hello\r\n")            // Output
	term.WriteString("\x1b]133;D;0\x07")     // Command finished

	output := term.GetLastCommandOutput()
	expected := "hello"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestGetLastCommandOutput_MultiLine(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("\x1b]133;C\x07")       // Command executed
	term.WriteString("line1\r\n")
	term.WriteString("line2\r\n")
	term.WriteString("line3\r\n")
	term.WriteString("\x1b]133;D;0\x07")     // Command finished

	output := term.GetLastCommandOutput()
	expected := "line1\nline2\nline3"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestGetLastCommandOutput_NoOutput(t *testing.T) {
	term := New(WithSize(24, 80))

	// Command with no output
	term.WriteString("\x1b]133;C\x07")       // Command executed
	term.WriteString("\x1b]133;D;0\x07")     // Command finished immediately

	output := term.GetLastCommandOutput()
	if output != "" {
		t.Errorf("expected empty string, got %q", output)
	}
}

func TestGetLastCommandOutput_NoMarks(t *testing.T) {
	term := New(WithSize(24, 80))

	// No marks at all
	output := term.GetLastCommandOutput()
	if output != "" {
		t.Errorf("expected empty string, got %q", output)
	}
}

func TestGetLastCommandOutput_OnlyExecutedNoFinished(t *testing.T) {
	term := New(WithSize(24, 80))

	// Only CommandExecuted, no CommandFinished
	term.WriteString("\x1b]133;C\x07")
	term.WriteString("output\r\n")

	output := term.GetLastCommandOutput()
	if output != "" {
		t.Errorf("expected empty string (no pair), got %q", output)
	}
}

func TestGetLastCommandOutput_MultipleCommands(t *testing.T) {
	term := New(WithSize(24, 80))

	// First command
	term.WriteString("\x1b]133;C\x07")
	term.WriteString("first output\r\n")
	term.WriteString("\x1b]133;D;0\x07")

	// Second command
	term.WriteString("\x1b]133;A\x07")
	term.WriteString("$ ")
	term.WriteString("\x1b]133;B\x07")
	term.WriteString("cmd2\r\n")
	term.WriteString("\x1b]133;C\x07")
	term.WriteString("second output\r\n")
	term.WriteString("\x1b]133;D;0\x07")

	// Should return the last command's output
	output := term.GetLastCommandOutput()
	expected := "second output"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestGetLastCommandOutput_WithExitCode(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("\x1b]133;C\x07")
	term.WriteString("error message\r\n")
	term.WriteString("\x1b]133;D;1\x07")     // Exit code 1

	output := term.GetLastCommandOutput()
	expected := "error message"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestGetLastCommandOutput_TrailingEmptyLines(t *testing.T) {
	term := New(WithSize(24, 80))

	term.WriteString("\x1b]133;C\x07")
	term.WriteString("content\r\n")
	term.WriteString("\r\n")                 // Empty line
	term.WriteString("\r\n")                 // Another empty line
	term.WriteString("\x1b]133;D;0\x07")

	output := term.GetLastCommandOutput()
	// Should trim trailing empty lines
	expected := "content"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}
