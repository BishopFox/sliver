//go:build darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package shell

import (
	"fmt"

	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

func configureNoPTYTerminal(fd int, escapeByte byte) (func() error, error) {
	if !term.IsTerminal(fd) {
		return func() error { return nil }, nil
	}

	state, err := getTermios(fd)
	if err != nil {
		return nil, err
	}
	originalEOL, ok := replaceControlChar(state.Cc[:], unix.VEOL, escapeByte)
	if !ok {
		return nil, fmt.Errorf("VEOL index %d is outside terminal control characters", unix.VEOL)
	}
	if err := setTermios(fd, state); err != nil {
		return nil, err
	}

	return func() error {
		current, err := getTermios(fd)
		if err != nil {
			return err
		}
		if _, ok := replaceControlChar(current.Cc[:], unix.VEOL, originalEOL); !ok {
			return fmt.Errorf("VEOL index %d is outside terminal control characters", unix.VEOL)
		}
		return setTermios(fd, current)
	}, nil
}
