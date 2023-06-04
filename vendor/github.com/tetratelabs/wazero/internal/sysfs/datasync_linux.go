//go:build linux

package sysfs

import (
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

func datasync(f *os.File) syscall.Errno {
	return platform.UnwrapOSError(syscall.Fdatasync(int(f.Fd())))
}
