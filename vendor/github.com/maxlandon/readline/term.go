package readline

import (
	"fmt"
	"os"
	"regexp"
	"unicode/utf8"
)

// GetTermWidth returns the width of Stdout or 80 if the width cannot be established
func GetTermWidth() (termWidth int) {
	var err error
	fd := int(os.Stdout.Fd())
	termWidth, _, err = GetSize(fd)
	if err != nil {
		termWidth = 80
	}

	return
}

func printf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	print(s)
}

func print(s string) {
	os.Stdout.WriteString(s)
}

/*func rLen(r []rune) (length int) {
	for _, i := range r {
		length += utf8.RuneLen(i)
	}
	return
}*/

var rxAnsiSgr = regexp.MustCompile("\x1b\\[[:;0-9]+m")

// Gets the number of runes in a string and
func strLen(s string) int {
	s = rxAnsiSgr.ReplaceAllString(s, "")
	return utf8.RuneCountInString(s)
}
