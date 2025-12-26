package main

import (
	"image/png"
	"os"

	headlessterm "github.com/danielgatis/go-headless-term"
)

func main() {
	term := headlessterm.New(
    headlessterm.WithSize(1,1),
    headlessterm.WithAutoResize(),
  )

	// Write content with ANSI colors
	term.WriteString("\x1b[1;35m╔══════════════════════════════════════════════╗\x1b[0m\r\n")
	term.WriteString("\x1b[1;35m║\x1b[0m  \x1b[1;36mHeadless Terminal Screenshot Demo\x1b[0m         \x1b[1;35m║\x1b[0m\r\n")
	term.WriteString("\x1b[1;35m╚══════════════════════════════════════════════╝\x1b[0m\r\n")
	term.WriteString("\r\n")
	term.WriteString("\x1b[32m✓\x1b[0m Custom font support (TTF/OTF)\r\n")
	term.WriteString("\x1b[32m✓\x1b[0m Custom color palette (256 colors)\r\n")
	term.WriteString("\x1b[32m✓\x1b[0m \x1b[1mBold\x1b[0m, \x1b[4mUnderline\x1b[0m, \x1b[7mReverse\x1b[0m\r\n")
	term.WriteString("\x1b[31mRed \x1b[32mGreen \x1b[33mYellow \x1b[34mBlue \x1b[35mMagenta \x1b[36mCyan\x1b[0m\r\n")
	term.WriteString("\r\n")
	term.WriteString("$ \x1b[33m_\x1b[0m")

	img2 := term.Screenshot()
	f2, _ := os.Create("terminal_default.png")
	defer f2.Close()
	png.Encode(f2, img2)
	println("Saved terminal_default.png")
}
