package term

// Terminal control sequences.
const (
	NewlineReturn = "\r\n"

	ClearLineAfter   = "\x1b[0K"
	ClearLineBefore  = "\x1b[1K"
	ClearLine        = "\x1b[2K"
	ClearScreenBelow = "\x1b[0J"
	ClearScreen      = "\x1b[2J" // Clears screen, preserving scroll buffer
	ClearDisplay     = "\x1b[3J" // Clears screen fully, wipes the scroll buffer

	CursorTopLeft    = "\x1b[H"
	SaveCursorPos    = "\x1b7"
	RestoreCursorPos = "\x1b8"
	HideCursor       = "\x1b[?25l"
	ShowCursor       = "\x1b[?25h"
)

// Some core keys needed by some stuff.
var (
	ArrowUp    = string([]byte{27, 91, 65}) // ^[[A
	ArrowDown  = string([]byte{27, 91, 66}) // ^[[B
	ArrowRight = string([]byte{27, 91, 67}) // ^[[C
	ArrowLeft  = string([]byte{27, 91, 68}) // ^[[D
)
