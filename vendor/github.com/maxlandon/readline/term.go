package readline

import (
	"fmt"
	"os"
	"regexp"
	"unicode/utf8"

	"github.com/olekukonko/ts"
)

// GetTermWidth returns the width of Stdout or 80 if the width cannot be established
func GetTermWidth() (termWidth int) {
	var err error
	fd := int(os.Stdout.Fd())
	termWidth, _, err = GetSize(fd)
	if err != nil {
		termWidth = 80 // The defacto standard on older terms
	}

	return
}

// GetTermLength returns the length of the terminal
// (Y length), or 80 if it cannot be established
func GetTermLength() (termLength int) {
	size, err := ts.GetSize()
	if err != nil || size.Row() == 0 {
		return 80
	}
	termLength = size.Row()

	return
}

func printf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	print(s)
}

func print(s string) {
	os.Stdout.WriteString(s)
}

var rxAnsiSgr = regexp.MustCompile("\x1b\\[[:;0-9]+m")

// Gets the number of runes in a string
func strLen(s string) int {
	s = rxAnsiSgr.ReplaceAllString(s, "")
	return utf8.RuneCountInString(s)
}
