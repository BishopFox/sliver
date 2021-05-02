package readline

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
	seqAltQuote  = string([]byte{27, 34})  // Added for showing registers ^["
	seqAltR      = string([]byte{27, 114}) // Used for alternative history
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

	seqCtrlLeftArrow  = "\x1b[1;5D"
	seqCtrlRightArrow = "\x1b[1;5C"

	// seqAltQuote = "\x1b\"" // trigger registers list
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
