//go:build !darwin && !linux && !windows

package sysfs

import (
	"time"

	"github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/platform"
)

func syscall_select(n int, r, w, e *platform.FdSet, timeout *time.Duration) (int, error) {
	return -1, sys.ENOSYS
}
