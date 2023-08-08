//go:build windows

package sysfs

import (
	"io/fs"
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
)

func setNonblock(fd uintptr, enable bool) error {
	// We invoke the syscall, but this is currently no-op.
	return syscall.SetNonblock(syscall.Handle(fd), enable)
}

func isNonblock(f *osFile) bool {
	// On Windows, we support non-blocking reads only on named pipes.
	isValid := false
	st, errno := f.Stat()
	if errno == 0 {
		isValid = st.Mode&fs.ModeNamedPipe != 0
	}
	return isValid && f.flag&fsapi.O_NONBLOCK == fsapi.O_NONBLOCK
}
