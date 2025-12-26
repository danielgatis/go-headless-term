package main

import (
	"fmt"

	headlessterm "github.com/danielgatis/go-headless-term"
)

func main() {
	term := headlessterm.New()

	// Write some ANSI sequences directly to the terminal
	term.WriteString("\x1b]0;My Terminal Title\x07") // Set window title
	term.WriteString("\x1b[31mHello ")               // Red text
	term.WriteString("\x1b[32mWorld")                // Green text
	term.WriteString("\x1b[0m!\r\n")                 // Reset and newline
	term.WriteString("\x1b[1;4mBold and Underlined\x1b[0m\r\n")
	term.WriteString("Normal text\r\n")
	term.WriteString("\x1b[2J") // Clear screen
	term.WriteString("\x1b[H")  // Move cursor to home
	term.WriteString("After clear")

	// Read the terminal content
	fmt.Println("\n=== Terminal Content ===")
	fmt.Printf("Content: %s\n", term)

	// Show cursor position
	cursorRow, cursorCol := term.CursorPos()
	fmt.Printf("Cursor position: row=%d, col=%d\n", cursorRow, cursorCol)
}
