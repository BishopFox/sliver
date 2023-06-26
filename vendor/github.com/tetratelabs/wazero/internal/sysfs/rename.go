//go:build !windows

package sysfs

import (
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

func Rename(from, to string) syscall.Errno {
	if from == to {
		return 0
	}
	return platform.UnwrapOSError(syscall.Rename(from, to))
}
