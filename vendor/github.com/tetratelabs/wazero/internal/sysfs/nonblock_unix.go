//go:build !windows

package sysfs

import (
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
)

func setNonblock(fd uintptr, enable bool) error {
	return syscall.SetNonblock(int(fd), enable)
}

func isNonblock(f *osFile) bool {
	return f.flag&fsapi.O_NONBLOCK == fsapi.O_NONBLOCK
}
