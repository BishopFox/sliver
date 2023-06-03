package term

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// fallback terminal width when we can't get it through query.
var defaultTermWidth = 80

// GetWidth returns the width of Stdout or 80 if the width cannot be established.
func GetWidth() (termWidth int) {
	var err error
	fd := int(os.Stdout.Fd())
	termWidth, _, err = GetSize(fd)

	if err != nil {
		termWidth = defaultTermWidth
	}

	return
}

// GetLength returns the length of the terminal
// (Y length), or 80 if it cannot be established.
func GetLength() int {
	width, _, err := term.GetSize(0)

	if err != nil || width == 0 {
		return defaultTermWidth
	}

	return width
}

func printf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Print(s)
}
