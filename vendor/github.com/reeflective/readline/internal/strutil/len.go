package strutil

import (
	"strings"

	"github.com/rivo/uniseg"

	"github.com/reeflective/readline/internal/color"
	"github.com/reeflective/readline/internal/term"
)

// FormatTabs replaces all '\t' occurrences in a string with 6 spaces each.
func FormatTabs(s string) string {
	return strings.ReplaceAll(s, "\t", "     ")
}

// RealLength returns the real length of a string (the number of terminal
// columns used to render the line, which may contain special graphemes).
// Before computing the width, it replaces tabs with (4) spaces, and strips colors.
func RealLength(s string) int {
	colors := color.Strip(s)
	tabs := strings.ReplaceAll(colors, "\t", "     ")

	return uniseg.StringWidth(tabs)
}

// LineSpan computes the number of columns and lines that are needed for a given line,
// accounting for any ANSI escapes/color codes, and tabulations replaced with 4 spaces.
func LineSpan(line []rune, idx, indent int) (x, y int) {
	termWidth := term.GetWidth()
	lineLen := RealLength(string(line))
	lineLen += indent

	cursorY := lineLen / termWidth
	cursorX := lineLen % termWidth

	// Empty lines are still considered a line.
	if idx != 0 {
		cursorY++
	}

	return cursorX, cursorY
}
