package headlessterm

import "github.com/unilibs/uniwidth"

// runeWidth returns the display width: 2 for wide characters (CJK, emoji), 1 for normal, 0 for zero-width (combining marks, control chars).
func runeWidth(r rune) int {
	return uniwidth.RuneWidth(r)
}

// isWideRune returns true if the rune occupies 2 columns (CJK ideographs, fullwidth forms, emoji).
func isWideRune(r rune) bool {
	return uniwidth.RuneWidth(r) == 2
}

// StringWidth returns the total display width of a string (sum of rune widths).
func StringWidth(s string) int {
	return uniwidth.StringWidth(s)
}
