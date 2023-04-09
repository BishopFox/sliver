package platform

import (
	"io/fs"
	"syscall"
)

// Fdatasync is like syscall.Fdatasync except that's only defined in linux.
//
// Note: This returns with no error instead of syscall.ENOSYS when
// unimplemented. This prevents fake filesystems from erring.
func Fdatasync(f fs.File) syscall.Errno {
	return fdatasync(f)
}
