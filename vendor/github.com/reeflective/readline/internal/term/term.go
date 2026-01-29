package term

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
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
	stdinTerm = os.Stdin
	stderrTerm = os.Stderr
}

// fallback terminal width when we can't get it through query.
var defaultTermWidth = 80

var (
	outputMu    sync.Mutex
	outputDepth int
	outputBuf   *bufio.Writer
)

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
	termFd := int(stdoutTerm.Fd())

	_, length, err := GetSize(termFd)
	if err != nil || length == 0 {
		return defaultTermWidth
	}

	return length
}

func printf(format string, a ...interface{}) {
	Printf(format, a...)
}

// BeginBuffer enables buffered terminal writes until EndBuffer is called.
func BeginBuffer() {
	outputMu.Lock()
	defer outputMu.Unlock()

	outputDepth++
	if outputDepth == 1 {
		outputBuf = bufio.NewWriterSize(stdoutTerm, 64*1024)
	}
}

// EndBuffer flushes buffered terminal writes when leaving the outermost buffer.
func EndBuffer() {
	outputMu.Lock()
	defer outputMu.Unlock()

	if outputDepth == 0 {
		return
	}

	outputDepth--
	if outputDepth == 0 && outputBuf != nil {
		_ = outputBuf.Flush()
		outputBuf = nil
	}
}

// WriteString writes to the terminal, using the buffer when enabled.
func WriteString(s string) {
	if s == "" {
		return
	}

	outputMu.Lock()
	defer outputMu.Unlock()

	if outputDepth > 0 && outputBuf != nil {
		_, _ = outputBuf.WriteString(s)
		return
	}

	_, _ = stdoutTerm.WriteString(s)
}

// Printf formats and writes to the terminal, using the buffer when enabled.
func Printf(format string, a ...interface{}) {
	WriteString(fmt.Sprintf(format, a...))
}

// ShouldQueryCursorPos reports whether querying the terminal cursor position is allowed.
// Some terminals respond slowly or cause visible flicker when queried frequently.
func ShouldQueryCursorPos() bool {
	if v := strings.TrimSpace(os.Getenv("READLINE_CURSOR_POS")); v != "" {
		switch strings.ToLower(v) {
		case "0", "false", "no", "off":
			return false
		case "1", "true", "yes", "on":
			return true
		}
	}

	if os.Getenv("TERM_PROGRAM") == "iTerm.app" {
		return false
	}

	return true
}
