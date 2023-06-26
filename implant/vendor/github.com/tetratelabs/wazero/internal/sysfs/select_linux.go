package sysfs

import (
	"syscall"
	"time"

	"github.com/tetratelabs/wazero/internal/platform"
)

// syscall_select invokes select on Unix (unless Darwin), with the given timeout Duration.
func syscall_select(n int, r, w, e *platform.FdSet, timeout *time.Duration) (int, error) {
	var t *syscall.Timeval
	if timeout != nil {
		tv := syscall.NsecToTimeval(timeout.Nanoseconds())
		t = &tv
	}
	return syscall.Select(n, (*syscall.FdSet)(r), (*syscall.FdSet)(w), (*syscall.FdSet)(e), t)
}
