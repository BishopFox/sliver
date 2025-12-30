package assets

import (
	"fmt"
	"io"
	"os"
)

type logger struct {
	verbose bool
	out     io.Writer
	err     io.Writer
}

func newLogger(verbose bool) *logger {
	return &logger{
		verbose: verbose,
		out:     os.Stdout,
		err:     os.Stderr,
	}
}

func (l *logger) Logf(format string, args ...any) {
	fmt.Fprintf(l.out, format+"\n", args...)
}

func (l *logger) VLogf(format string, args ...any) {
	if !l.verbose {
		return
	}
	l.Logf(format, args...)
}

func (l *logger) Errorf(format string, args ...any) {
	fmt.Fprintf(l.err, format+"\n", args...)
}
