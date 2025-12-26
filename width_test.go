package headlessterm

import (
	"testing"
)

func TestRuneWidth(t *testing.T) {
	tests := []struct {
		r        rune
		expected int
	}{
		{'A', 1},
		{'a', 1},
		{'1', 1},
		{' ', 1},
		{'中', 2},
		{'日', 2},
		{'本', 2},
		{'한', 2},
		{'글', 2},
		{'가', 2},
		{'Ａ', 2}, // Fullwidth A
		{0, 0},
	}

	for _, tt := range tests {
		got := runeWidth(tt.r)
		if got != tt.expected {
			t.Errorf("runeWidth(%q) = %d, want %d", tt.r, got, tt.expected)
		}
	}
}

func TestIsWideRune(t *testing.T) {
	tests := []struct {
		r        rune
		expected bool
	}{
		{'A', false},
		{'a', false},
		{' ', false},
		{'中', true},
		{'日', true},
		{'한', true},
		{'가', true},
		{'Ａ', true}, // Fullwidth A
		{'0', false},
	}

	for _, tt := range tests {
		got := isWideRune(tt.r)
		if got != tt.expected {
			t.Errorf("isWideRune(%q) = %v, want %v", tt.r, got, tt.expected)
		}
	}
}

func TestStringWidth(t *testing.T) {
	tests := []struct {
		s        string
		expected int
	}{
		{"Hello", 5},
		{"中文", 4},
		{"Hello中文", 9},
		{"", 0},
		{"한글", 4},
	}

	for _, tt := range tests {
		got := StringWidth(tt.s)
		if got != tt.expected {
			t.Errorf("StringWidth(%q) = %d, want %d", tt.s, got, tt.expected)
		}
	}
}
