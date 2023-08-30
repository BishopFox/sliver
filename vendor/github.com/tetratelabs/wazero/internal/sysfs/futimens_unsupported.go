//go:build !windows && !linux && !darwin

package sysfs

import (
	"syscall"

	"github.com/tetratelabs/wazero/experimental/sys"
)

// Define values even if not used except as sentinels.
const (
	_UTIME_NOW  = -1
	_UTIME_OMIT = -2
)

func utimens(path string, times *[2]syscall.Timespec) error {
	return utimensPortable(path, times)
}

func futimens(fd uintptr, times *[2]syscall.Timespec) error {
	// Go exports syscall.Futimes, which is microsecond granularity, and
	// WASI tests expect nanosecond. We don't yet have a way to invoke the
	// futimens syscall portably.
	return sys.ENOSYS
}
