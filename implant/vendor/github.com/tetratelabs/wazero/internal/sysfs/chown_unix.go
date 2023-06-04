//go:build !windows

package sysfs

import (
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

func fchown(fd uintptr, uid, gid int) syscall.Errno {
	return platform.UnwrapOSError(syscall.Fchown(int(fd), uid, gid))
}
