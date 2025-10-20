// Package style provide display coloring
package style

import (
	"strings"

	"github.com/rsteube/carapace/third_party/github.com/elves/elvish/pkg/ui"
)

var (
	Default = ""

	Black   = "black"
	Red     = "red"
	Green   = "green"
	Yellow  = "yellow"
	Blue    = "blue"
	Magenta = "magenta"
	Cyan    = "cyan"
	White   = "white"

	BrightBlack   = "bright-black"
	BrightRed     = "bright-red"
	BrightGreen   = "bright-green"
	BrightYellow  = "bright-yellow"
	BrightBlue    = "bright-blue"
	BrightMagenta = "bright-magenta"
	BrightCyan    = "bright-cyan"
	BrightWhite   = "bright-white"

	BgBlack   = "bg-black"
	BgRed     = "bg-red"
	BgGreen   = "bg-green"
	BgYellow  = "bg-yellow"
	BgBlue    = "bg-blue"
	BgMagenta = "bg-magenta"
	BgCyan    = "bg-cyan"
	BgWhite   = "bg-white"

	BgBrightBlack   = "bg-bright-black"
	BgBrightRed     = "bg-bright-red"
	BgBrightGreen   = "bg-bright-green"
	BgBrightYellow  = "bg-bright-yellow"
	BgBrightBlue    = "bg-bright-blue"
	BgBrightMagenta = "bg-bright-magenta"
	BgBrightCyan    = "bg-bright-cyan"
	BgBrightWhite   = "bg-bright-white"

	Bold       = "bold"
	Dim        = "dim"
	Italic     = "italic"
	Underlined = "underlined"
	Blink      = "blink"
	Inverse    = "inverse"
)

// Of combines different styles.
func Of(s ...string) string { return strings.Join(s, " ") }

// XTerm256Color returns a color from the xterm 256-color palette.
func XTerm256Color(i uint8) string { return ui.XTerm256Color(i).String() }

// TrueColor returns a 24-bit true color.
func TrueColor(r, g, b uint8) string { return ui.TrueColor(r, g, b).String() }

// SGR returns the SGR sequence for given style.
func SGR(s string) string { return parseStyle(s).SGR() }

func parseStyle(s string) ui.Style {
	stylings := make([]ui.Styling, 0)
	for _, word := range strings.Split(s, " ") {
		if styling := ui.ParseStyling(word); styling != nil {
			stylings = append(stylings, styling)
		}
	}
	return ui.ApplyStyling(ui.Style{}, stylings...)
}
