package readline

import (
	"os"
	"strings"
)

// Character codes
const (
	charCtrlA = iota + 1
	charCtrlB
	charCtrlC
	charEOF
	charCtrlE
	charCtrlF
	charCtrlG
	charBackspace // ISO 646
	charTab
	charCtrlJ
	charCtrlK
	charCtrlL
	charCtrlM
	charCtrlN
	charCtrlO
	charCtrlP
	charCtrlQ
	charCtrlR
	charCtrlS
	charCtrlT
	charCtrlU
	charCtrlV
	charCtrlW
	charCtrlX
	charCtrlY
	charCtrlZ
	charEscape
	charCtrlSlash             // ^\
	charCtrlCloseSquare       // ^]
	charCtrlHat               // ^^
	charCtrlUnderscore        // ^_
	charBackspace2      = 127 // ASCII 1963
)

// Escape sequences
var (
	seqUp        = string([]byte{27, 91, 65})
	seqDown      = string([]byte{27, 91, 66})
	seqForwards  = string([]byte{27, 91, 67})
	seqBackwards = string([]byte{27, 91, 68})
	seqHome      = string([]byte{27, 91, 72})
	seqHomeSc    = string([]byte{27, 91, 49, 126})
	seqEnd       = string([]byte{27, 91, 70})
	seqEndSc     = string([]byte{27, 91, 52, 126})
	seqDelete    = string([]byte{27, 91, 51, 126})
	seqShiftTab  = string([]byte{27, 91, 90})
)

const (
	seqPosSave    = "\x1b[s"
	seqPosRestore = "\x1b[u"

	seqClearLineAfer    = "\x1b[0k"
	seqClearLineBefore  = "\x1b[1k"
	seqClearLine        = "\x1b[2k"
	seqClearScreenBelow = "\x1b[0J"
	seqClearScreen      = "\x1b[2J" // Clears screen fully
	seqCursorTopLeft    = "\x1b[H"  // Clears screen and places cursor on top-left

	seqGetCursorPos = "\x1b6n" // response: "\x1b{Line};{Column}R"
)

// Text effects
const (
	seqReset      = "\x1b[0m"
	seqBold       = "\x1b[1m"
	seqUnderscore = "\x1b[4m"
	seqBlink      = "\x1b[5m"
)

// Text colours
const (
	seqFgBlack   = "\x1b[30m"
	seqFgRed     = "\x1b[31m"
	seqFgGreen   = "\x1b[32m"
	seqFgYellow  = "\x1b[33m"
	seqFgBlue    = "\x1b[34m"
	seqFgMagenta = "\x1b[35m"
	seqFgCyan    = "\x1b[36m"
	seqFgWhite   = "\x1b[37m"

	seqFgBlackBright   = "\x1b[1;30m"
	seqFgRedBright     = "\x1b[1;31m"
	seqFgGreenBright   = "\x1b[1;32m"
	seqFgYellowBright  = "\x1b[1;33m"
	seqFgBlueBright    = "\x1b[1;34m"
	seqFgMagentaBright = "\x1b[1;35m"
	seqFgCyanBright    = "\x1b[1;36m"
	seqFgWhiteBright   = "\x1b[1;37m"
)

// Background colours
const (
	seqBgBlack   = "\x1b[40m"
	seqBgRed     = "\x1b[41m"
	seqBgGreen   = "\x1b[42m"
	seqBgYellow  = "\x1b[43m"
	seqBgBlue    = "\x1b[44m"
	seqBgMagenta = "\x1b[45m"
	seqBgCyan    = "\x1b[46m"
	seqBgWhite   = "\x1b[47m"

	seqBgBlackBright   = "\x1b[1;40m"
	seqBgRedBright     = "\x1b[1;41m"
	seqBgGreenBright   = "\x1b[1;42m"
	seqBgYellowBright  = "\x1b[1;43m"
	seqBgBlueBright    = "\x1b[1;44m"
	seqBgMagentaBright = "\x1b[1;45m"
	seqBgCyanBright    = "\x1b[1;46m"
	seqBgWhiteBright   = "\x1b[1;47m"
)

// Xterm 256 colors
const (
	seqCtermFg255 = "\033[48;5;255m"
)

// TUI package colors & effects
// ---------------------------------------------------------------------------------

// https://misc.flogisoft.com/bash/tip_colors_and_formatting
var (
	// effects
	BOLD  = "\033[1m"
	DIM   = "\033[2m"
	RESET = "\033[0m"
	// colors
	RED    = "\033[31m"
	GREEN  = "\033[32m"
	BLUE   = "\033[34m"
	YELLOW = "\033[33m"
	// foreground colors
	FOREBLACK = "\033[30m"
	FOREWHITE = "\033[97m"
	// background colors
	BACKDARKGRAY  = "\033[100m"
	BACKRED       = "\033[41m"
	BACKGREEN     = "\033[42m"
	BACKYELLOW    = "\033[43m"
	BACKLIGHTBLUE = "\033[104m"

	ctrl = []string{"\x033", "\\e", "\x1b"}
)

// Effects returns true if colors and effects are supported
// on the current terminal.
func Effects() bool {
	if term := os.Getenv("TERM"); term == "" {
		return false
	} else if term == "dumb" {
		return false
	}
	return true
}

// Disable will disable all colors and effects.
func Disable() {
	BOLD = ""
	DIM = ""
	RESET = ""
	RED = ""
	GREEN = ""
	BLUE = ""
	YELLOW = ""
	FOREBLACK = ""
	FOREWHITE = ""
	BACKDARKGRAY = ""
	BACKRED = ""
	BACKGREEN = ""
	BACKYELLOW = ""
	BACKLIGHTBLUE = ""
}

// HasEffect returns true if the string has any shell control codes in it.
func HasEffect(s string) bool {
	for _, ch := range ctrl {
		if strings.Contains(s, ch) {
			return true
		}
	}
	return false
}

// Wrap wraps a string with an effect or color and appends a reset control code.
func Wrap(e, s string) string {
	return e + s + RESET
}

// Bold makes the string Bold.
func Bold(s string) string {
	return Wrap(BOLD, s)
}

// Dim makes the string Diminished.
func Dim(s string) string {
	return Wrap(DIM, s)
}

// Red makes the string Red.
func Red(s string) string {
	return Wrap(RED, s)
}

// Green makes the string Green.
func Green(s string) string {
	return Wrap(GREEN, s)
}

// Blue makes the string Green.
func Blue(s string) string {
	return Wrap(BLUE, s)
}

// Yellow makes the string Green.
func Yellow(s string) string {
	return Wrap(YELLOW, s)
}
