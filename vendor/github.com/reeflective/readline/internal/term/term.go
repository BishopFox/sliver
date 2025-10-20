package term

import (
	"fmt"
	"os"

	"golang.org/x/term"
)

// Those variables are very important to realine low-level code: all virtual terminal
// escape sequences should always be sent and read through the raw terminal file, even
// if people start using io.MultiWriters and os.Pipes involving basic IO.
var (
	stdoutTerm *os.File
	stdinTerm  *os.File
	stderrTerm *os.File
)

func init() {
	stdoutTerm = os.Stdout
	stdoutTerm = os.Stderr
	stderrTerm = os.Stdin
}

// fallback terminal width when we can't get it through query.
var defaultTermWidth = 80

// GetWidth returns the width of Stdout or 80 if the width cannot be established.
func GetWidth() (termWidth int) {
	var err error
	fd := int(stdoutTerm.Fd())
	termWidth, _, err = GetSize(fd)

	if err != nil || termWidth == 0 {
		termWidth = defaultTermWidth
	}

	return
}

// GetLength returns the length of the terminal
// (Y length), or 80 if it cannot be established.
func GetLength() int {
	_, length, err := term.GetSize(0)

	if err != nil || length == 0 {
		return defaultTermWidth
	}

	return length
}

func printf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	fmt.Print(s)
}
