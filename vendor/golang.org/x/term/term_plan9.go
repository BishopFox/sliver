// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package term

import (
	"fmt"
	"runtime"

	"golang.org/x/sys/plan9"
)

type State struct{}

func isTerminal(fd int) bool {
	path, err := plan9.Fd2path(fd)
	if err != nil {
		return false
	}
	return path == "/dev/cons" || path == "/mnt/term/dev/cons"
}

// MakeRaw put the terminal connected to the given file descriptor into raw
// mode and returns the previous state of the terminal so that it can be
// restored.
func MakeRaw(fd int) (*State, error) {
	return nil, fmt.Errorf("terminal: MakeRaw not implemented on %s/%s", runtime.GOOS, runtime.GOARCH)
}

// GetState returns the current state of a terminal which may be useful to
// restore the terminal after a signal.
func GetState(fd int) (*State, error) {
	return nil, fmt.Errorf("terminal: GetState not implemented on %s/%s", runtime.GOOS, runtime.GOARCH)
}

// Restore restores the terminal connected to the given file descriptor to a
// previous state.
func Restore(fd int, state *State) error {
	return fmt.Errorf("terminal: Restore not implemented on %s/%s", runtime.GOOS, runtime.GOARCH)
}

// GetSize returns the dimensions of the given terminal.
func GetSize(fd int) (width, height int, err error) {
	return 0, 0, fmt.Errorf("terminal: GetSize not implemented on %s/%s", runtime.GOOS, runtime.GOARCH)
}

// ReadPassword reads a line of input from a terminal without local echo.  This
// is commonly used for inputting passwords and other sensitive data. The slice
// returned does not include the \n.
func ReadPassword(fd int) ([]byte, error) {
	return nil, fmt.Errorf("terminal: ReadPassword not implemented on %s/%s", runtime.GOOS, runtime.GOARCH)
}
