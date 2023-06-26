//go:build !darwin && !linux && !windows

package sysfs

import (
	"syscall"
	"time"

	"github.com/tetratelabs/wazero/internal/platform"
)

func syscall_select(n int, r, w, e *platform.FdSet, timeout *time.Duration) (int, error) {
	return -1, syscall.ENOSYS
}
