package text

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// colorsEnabled is true if colors are enabled and supported by the terminal.
var colorsEnabled = areColorsOnInTheEnv() && areANSICodesSupported()

// DisableColors (forcefully) disables color coding globally.
func DisableColors() {
	colorsEnabled = false
}

// EnableColors (forcefully) enables color coding globally.
func EnableColors() {
	colorsEnabled = true
}

// areColorsOnInTheEnv returns true if colors are not disabled using
// well known environment variables.
func areColorsOnInTheEnv() bool {
	// FORCE_COLOR takes precedence - if set to a truthy value, enable colors
	forceColor := os.Getenv("FORCE_COLOR")
	if forceColor != "" && forceColor != "0" && forceColor != "false" {
		return true
	}

	// NO_COLOR: if set to any non-empty value (except "0"), disable colors
	// Note: "0" is treated as "not set" to allow explicit enabling via NO_COLOR=0
	noColor := os.Getenv("NO_COLOR")
	if noColor != "" && noColor != "0" {
		return false
	}

	// Default: check TERM - if not "dumb", assume colors are supported
	return os.Getenv("TERM") != "dumb"
}

// The logic here is inspired from github.com/fatih/color; the following is
// the bare minimum logic required to print Colored to the console.
// The differences:
// * This one caches the escape sequences for cases with multiple colors
// * This one handles cases where the incoming already has colors in the
//   form of escape sequences; in which case, text that does not have any
//   escape sequences are colored/escaped

// Color represents a single color to render with.
type Color int

// Base colors -- attributes in reality
const (
	Reset Color = iota
	Bold
	Faint
	Italic
	Underline
	BlinkSlow
	BlinkRapid
	ReverseVideo
	Concealed
	CrossedOut
)

// Foreground colors
const (
	FgBlack Color = iota + 30
	FgRed
	FgGreen
	FgYellow
	FgBlue
	FgMagenta
	FgCyan
	FgWhite
)

// Foreground Hi-Intensity colors
const (
	FgHiBlack Color = iota + 90
	FgHiRed
	FgHiGreen
	FgHiYellow
	FgHiBlue
	FgHiMagenta
	FgHiCyan
	FgHiWhite
)

// Background colors
const (
	BgBlack Color = iota + 40
	BgRed
	BgGreen
	BgYellow
	BgBlue
	BgMagenta
	BgCyan
	BgWhite
)

// Background Hi-Intensity colors
const (
	BgHiBlack Color = iota + 100
	BgHiRed
	BgHiGreen
	BgHiYellow
	BgHiBlue
	BgHiMagenta
	BgHiCyan
	BgHiWhite
)

// 256-color support
// Internal encoding for 256-color codes (used by escape_seq_parser.go):
// Foreground 256-color: fg256Start + colorIndex (1000-1255)
// Background 256-color: bg256Start + colorIndex (2000-2255)
const (
	// fg256Start is the base value for 256-color foreground colors.
	// Use Fg256Color(index) to create a 256-color foreground color.
	fg256Start Color = 1000
	// bg256Start is the base value for 256-color background colors.
	// Use Bg256Color(index) to create a 256-color background color.
	bg256Start Color = 2000
)

// CSSClasses returns the CSS class names for the color.
func (c Color) CSSClasses() string {
	// Check for 256-color and convert to RGB-based class
	if c >= fg256Start && c < fg256Start+256 {
		colorIndex := int(c - fg256Start)
		r, g, b := color256ToRGB(colorIndex)
		return fmt.Sprintf("fg-256-%d-%d-%d", r, g, b)
	}
	if c >= bg256Start && c < bg256Start+256 {
		colorIndex := int(c - bg256Start)
		r, g, b := color256ToRGB(colorIndex)
		return fmt.Sprintf("bg-256-%d-%d-%d", r, g, b)
	}
	// Existing behavior for standard colors
	if class, ok := colorCSSClassMap[c]; ok {
		return class
	}
	return ""
}

// EscapeSeq returns the ANSI escape sequence for the color.
func (c Color) EscapeSeq() string {
	// Check if it's a 256-color foreground (1000-1255)
	if c >= fg256Start && c < fg256Start+256 {
		colorIndex := int(c - fg256Start)
		return fmt.Sprintf("%s38;5;%d%s", EscapeStart, colorIndex, EscapeStop)
	}
	// Check if it's a 256-color background (2000-2255)
	if c >= bg256Start && c < bg256Start+256 {
		colorIndex := int(c - bg256Start)
		return fmt.Sprintf("%s48;5;%d%s", EscapeStart, colorIndex, EscapeStop)
	}
	// Regular color (existing behavior)
	return EscapeStart + strconv.Itoa(int(c)) + EscapeStop
}

// HTMLProperty returns the "class" attribute for the color.
func (c Color) HTMLProperty() string {
	classes := c.CSSClasses()
	if classes == "" {
		return ""
	}
	return fmt.Sprintf("class=\"%s\"", classes)
}

// Sprint colorizes and prints the given string(s).
func (c Color) Sprint(a ...interface{}) string {
	return colorize(fmt.Sprint(a...), c.EscapeSeq())
}

// Sprintf formats and colorizes and prints the given string(s).
func (c Color) Sprintf(format string, a ...interface{}) string {
	return colorize(fmt.Sprintf(format, a...), c.EscapeSeq())
}

// Colors represents an array of Color objects to render with.
// Example: Colors{FgCyan, BgBlack}
type Colors []Color

// colorsSeqMap caches the escape sequence for a set of colors
var colorsSeqMap = sync.Map{}

// CSSClasses returns the CSS class names for the colors.
func (c Colors) CSSClasses() string {
	if len(c) == 0 {
		return ""
	}

	var classes []string
	for _, color := range c {
		class := color.CSSClasses()
		if class != "" {
			classes = append(classes, class)
		}
	}
	if len(classes) > 1 {
		sort.Strings(classes)
	}
	return strings.Join(classes, " ")
}

// EscapeSeq returns the ANSI escape sequence for the colors set.
func (c Colors) EscapeSeq() string {
	if len(c) == 0 {
		return ""
	}

	colorsKey := fmt.Sprintf("%#v", c)
	escapeSeq, ok := colorsSeqMap.Load(colorsKey)
	if !ok || escapeSeq == "" {
		codes := make([]string, 0, len(c))
		for _, color := range c {
			codes = append(codes, c.colorToCode(color))
		}
		escapeSeq = EscapeStart + strings.Join(codes, ";") + EscapeStop
		colorsSeqMap.Store(colorsKey, escapeSeq)
	}
	return escapeSeq.(string)
}

// colorToCode converts a Color to its escape sequence code string.
func (c Colors) colorToCode(color Color) string {
	// Check if it's a 256-color foreground (1000-1255)
	if color >= fg256Start && color < fg256Start+256 {
		colorIndex := int(color - fg256Start)
		return fmt.Sprintf("38;5;%d", colorIndex)
	}
	// Check if it's a 256-color background (2000-2255)
	if color >= bg256Start && color < bg256Start+256 {
		colorIndex := int(color - bg256Start)
		return fmt.Sprintf("48;5;%d", colorIndex)
	}
	// Regular color
	return strconv.Itoa(int(color))
}

// HTMLProperty returns the "class" attribute for the colors.
func (c Colors) HTMLProperty() string {
	classes := c.CSSClasses()
	if classes == "" {
		return ""
	}
	return fmt.Sprintf("class=\"%s\"", classes)
}

// Sprint colorizes and prints the given string(s).
func (c Colors) Sprint(a ...interface{}) string {
	return colorize(fmt.Sprint(a...), c.EscapeSeq())
}

// Sprintf formats and colorizes and prints the given string(s).
func (c Colors) Sprintf(format string, a ...interface{}) string {
	return colorize(fmt.Sprintf(format, a...), c.EscapeSeq())
}

func colorize(s string, escapeSeq string) string {
	if !colorsEnabled || escapeSeq == "" {
		return s
	}
	return Escape(s, escapeSeq)
}

// Fg256Color returns a foreground 256-color Color value.
// The index must be in the range 0-255.
func Fg256Color(index int) Color {
	if index < 0 || index > 255 {
		return Reset
	}
	return fg256Start + Color(index)
}

// Bg256Color returns a background 256-color Color value.
// The index must be in the range 0-255.
func Bg256Color(index int) Color {
	if index < 0 || index > 255 {
		return Reset
	}
	return bg256Start + Color(index)
}

// Fg256RGB returns a foreground 256-color from RGB values in the 6x6x6 color cube.
// Each RGB component must be in the range 0-5.
// The resulting color index will be in the range 16-231.
func Fg256RGB(r, g, b int) Color {
	if r < 0 || r > 5 || g < 0 || g > 5 || b < 0 || b > 5 {
		return Reset
	}
	index := 16 + (r*36 + g*6 + b)
	return Fg256Color(index)
}

// Bg256RGB returns a background 256-color from RGB values in the 6x6x6 color cube.
// Each RGB component must be in the range 0-5.
// The resulting color index will be in the range 16-231.
func Bg256RGB(r, g, b int) Color {
	if r < 0 || r > 5 || g < 0 || g > 5 || b < 0 || b > 5 {
		return Reset
	}
	index := 16 + (r*36 + g*6 + b)
	return Bg256Color(index)
}

// color256ToRGB converts a 256-color index to RGB values.
// Returns (r, g, b) values in the range 0-255.
func color256ToRGB(index int) (r, g, b int) {
	if index < 16 {
		// Standard 16 colors - map to predefined RGB values
		standardColors := [16][3]int{
			{0, 0, 0},       // 0: black
			{128, 0, 0},     // 1: red
			{0, 128, 0},     // 2: green
			{128, 128, 0},   // 3: yellow
			{0, 0, 128},     // 4: blue
			{128, 0, 128},   // 5: magenta
			{0, 128, 128},   // 6: cyan
			{192, 192, 192}, // 7: light gray
			{128, 128, 128}, // 8: dark gray
			{255, 0, 0},     // 9: bright red
			{0, 255, 0},     // 10: bright green
			{255, 255, 0},   // 11: bright yellow
			{0, 0, 255},     // 12: bright blue
			{255, 0, 255},   // 13: bright magenta
			{0, 255, 255},   // 14: bright cyan
			{255, 255, 255}, // 15: white
		}
		return standardColors[index][0], standardColors[index][1], standardColors[index][2]
	} else if index < 232 {
		// 216-color RGB cube (16-231)
		index -= 16
		r = (index / 36) * 51
		g = ((index / 6) % 6) * 51
		b = (index % 6) * 51
	} else {
		// 24 grayscale colors (232-255)
		gray := 8 + (index-232)*10
		r, g, b = gray, gray, gray
	}
	return
}
