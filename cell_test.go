package headlessterm

import (
	"testing"
)

func TestNewCell(t *testing.T) {
	cell := NewCell()

	if cell.Char != ' ' {
		t.Errorf("expected space, got '%c'", cell.Char)
	}
	if cell.Fg != nil {
		t.Error("expected nil foreground")
	}
	if cell.Bg != nil {
		t.Error("expected nil background")
	}
	if cell.Flags != 0 {
		t.Error("expected no flags")
	}
}

func TestCellReset(t *testing.T) {
	cell := NewCell()
	cell.Char = 'A'
	cell.SetFlag(CellFlagBold)

	cell.Reset()

	if cell.Char != ' ' {
		t.Errorf("expected space after reset, got '%c'", cell.Char)
	}
	if cell.HasFlag(CellFlagBold) {
		t.Error("expected no flags after reset")
	}
}

func TestCellFlags(t *testing.T) {
	cell := NewCell()

	cell.SetFlag(CellFlagBold)
	if !cell.HasFlag(CellFlagBold) {
		t.Error("expected bold flag")
	}

	cell.SetFlag(CellFlagItalic)
	if !cell.HasFlag(CellFlagBold) || !cell.HasFlag(CellFlagItalic) {
		t.Error("expected both flags")
	}

	cell.ClearFlag(CellFlagBold)
	if cell.HasFlag(CellFlagBold) {
		t.Error("expected bold flag to be cleared")
	}
	if !cell.HasFlag(CellFlagItalic) {
		t.Error("expected italic flag to remain")
	}
}

func TestCellDirty(t *testing.T) {
	cell := NewCell()

	if cell.IsDirty() {
		t.Error("expected cell not dirty initially")
	}

	cell.MarkDirty()
	if !cell.IsDirty() {
		t.Error("expected cell to be dirty")
	}

	cell.ClearDirty()
	if cell.IsDirty() {
		t.Error("expected cell not dirty after clear")
	}
}

func TestCellWide(t *testing.T) {
	cell := NewCell()

	cell.SetFlag(CellFlagWideChar)
	if !cell.IsWide() {
		t.Error("expected cell to be wide")
	}

	spacer := NewCell()
	spacer.SetFlag(CellFlagWideCharSpacer)
	if !spacer.IsWideSpacer() {
		t.Error("expected cell to be spacer")
	}
}

func TestCellCopy(t *testing.T) {
	cell := NewCell()
	cell.Char = 'X'
	cell.SetFlag(CellFlagBold | CellFlagItalic)

	copied := cell.Copy()

	if copied.Char != 'X' {
		t.Errorf("expected 'X', got '%c'", copied.Char)
	}
	if !copied.HasFlag(CellFlagBold) || !copied.HasFlag(CellFlagItalic) {
		t.Error("expected flags to be copied")
	}

	// Modify original, copy should be unchanged
	cell.Char = 'Y'
	if copied.Char != 'X' {
		t.Error("copy should be independent")
	}
}
