//go:build !linux && !darwin && !freebsd && !dragonfly && !openbsd
// +build !linux,!darwin,!freebsd,!dragonfly,!openbsd

package pty

import (
	"os"
)

func open() (pty, tty *os.File, err error) {
	return nil, nil, ErrUnsupported
}
