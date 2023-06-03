package color

import (
	"os"
	"regexp"
)

// Base text effects.
var (
	Reset      = "\x1b[0m"
	Bold       = "\x1b[1m"
	Dim        = "\x1b[2m"
	Underscore = "\x1b[4m"
	Blink      = "\x1b[5m"
	Reverse    = "\x1b[7m"

	// Effects reset.
	BoldReset       = "\x1b[22m" // 21 actually causes underline instead
	DimReset        = "\x1b[22m"
	UnderscoreReset = "\x1b[24m"
	BlinkReset      = "\x1b[25m"
	ReverseReset    = "\x1b[27m"
)

// Text colours.
var (
	FgBlack   = "\x1b[30m"
	FgRed     = "\x1b[31m"
	FgGreen   = "\x1b[32m"
	FgYellow  = "\x1b[33m"
	FgBlue    = "\x1b[34m"
	FgMagenta = "\x1b[35m"
	FgCyan    = "\x1b[36m"
	FgWhite   = "\x1b[37m"
	FgDefault = "\x1b[39m"

	FgBlackBright   = "\x1b[1;30m"
	FgRedBright     = "\x1b[1;31m"
	FgGreenBright   = "\x1b[1;32m"
	FgYellowBright  = "\x1b[1;33m"
	FgBlueBright    = "\x1b[1;34m"
	FgMagentaBright = "\x1b[1;35m"
	FgCyanBright    = "\x1b[1;36m"
	FgWhiteBright   = "\x1b[1;37m"
)

// Background colours.
var (
	BgBlack   = "\x1b[40m"
	BgRed     = "\x1b[41m"
	BgGreen   = "\x1b[42m"
	BgYellow  = "\x1b[43m"
	BgBlue    = "\x1b[44m"
	BgMagenta = "\x1b[45m"
	BgCyan    = "\x1b[46m"
	BgWhite   = "\x1b[47m"
	BgDefault = "\x1b[49m"

	BgDarkGray  = "\x1b[100m"
	BgBlueLight = "\x1b[104m"

	BgBlackBright   = "\x1b[1;40m"
	BgRedBright     = "\x1b[1;41m"
	BgGreenBright   = "\x1b[1;42m"
	BgYellowBright  = "\x1b[1;43m"
	BgBlueBright    = "\x1b[1;44m"
	BgMagentaBright = "\x1b[1;45m"
	BgCyanBright    = "\x1b[1;46m"
	BgWhiteBright   = "\x1b[1;47m"
)

// Text effects.
const (
	SGRStart = "\x1b["
	Fg       = "38;05;"
	Bg       = "48;05;"
	SGREnd   = "m"
)

// Fmt formats a color code as an ANSI escaped color sequence.
func Fmt(color string) string {
	return SGRStart + color + SGREnd
}

// HasEffects returns true if colors and effects are supported
// on the current terminal.
func HasEffects() bool {
	if term := os.Getenv("TERM"); term == "" {
		return false
	} else if term == "dumb" {
		return false
	}

	return true
}

// DisableEffects will disable all colors and effects.
func DisableEffects() {
	// Effects
	Reset = ""
	Bold = ""
	Dim = ""
	Underscore = ""
	Blink = ""
	BoldReset = ""
	DimReset = ""
	UnderscoreReset = ""
	BlinkReset = ""

	// Foreground colors
	FgBlack = ""
	FgRed = ""
	FgGreen = ""
	FgYellow = ""
	FgBlue = ""
	FgMagenta = ""
	FgCyan = ""
	FgWhite = ""
	FgDefault = ""

	FgBlackBright = ""
	FgRedBright = ""
	FgGreenBright = ""
	FgYellowBright = ""
	FgBlueBright = ""
	FgMagentaBright = ""
	FgCyanBright = ""
	FgWhiteBright = ""

	// Background colours
	BgBlack = ""
	BgRed = ""
	BgGreen = ""
	BgYellow = ""
	BgBlue = ""
	BgMagenta = ""
	BgCyan = ""
	BgWhite = ""
	BgDefault = ""

	BgDarkGray = ""
	BgBlueLight = ""

	BgBlackBright = ""
	BgRedBright = ""
	BgGreenBright = ""
	BgYellowBright = ""
	BgBlueBright = ""
	BgMagentaBright = ""
	BgCyanBright = ""
	BgWhiteBright = ""
}

const ansi = "[\u001B\u009B][[\\]()#;?]*(?:(?:(?:[a-zA-Z\\d]*(?:;[a-zA-Z\\d]*)*)?\u0007)|(?:(?:\\d{1,4}(?:;\\d{0,4})*)?[\\dA-PRZcf-ntqry=><~]))"

var re = regexp.MustCompile(ansi)

// Strip removes all ANSI escaped color sequences in a string.
func Strip(str string) string {
	return re.ReplaceAllString(str, "")
}

// wrong: reapplies fg/bg escapes regardless of the string passed.
// Users should be in charge of applying any effect as they wish.
// func SGR(color string, fg bool) string {
// 	if fg {
// 		return SGRStart + FgColorStart + color + SGREnd
// 		// return SGRStart + color + SGREnd
// 	}
//
// 	return SGRStart + BgColorStart + color + SGREnd
// }
