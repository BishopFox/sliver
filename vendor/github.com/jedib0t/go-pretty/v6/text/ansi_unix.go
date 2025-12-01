//go:build !windows
// +build !windows

package text

import "os"

func areANSICodesSupported() bool {
	// On Unix systems, ANSI codes are generally supported unless TERM is "dumb"
	// This is a basic check; 256-color sequences are ANSI sequences and will
	// be handled by terminals that support them (or ignored by those that don't)
	term := os.Getenv("TERM")
	return term != "dumb"
}
