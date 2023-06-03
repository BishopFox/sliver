//go:build !windows

package sysfs

import (
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

func fsync(f *os.File) syscall.Errno {
	return platform.UnwrapOSError(f.Sync())
}
