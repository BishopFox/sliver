//go:build illumos || solaris

package platform

import (
	"io/fs"
	"os"
	"syscall"
)

const (
	// See https://github.com/illumos/illumos-gate/blob/edd580643f2cf1434e252cd7779e83182ea84945/usr/src/uts/common/sys/fcntl.h#L90
	O_DIRECTORY = 0x1000000
	O_NOFOLLOW  = syscall.O_NOFOLLOW
)

func OpenFile(path string, flag int, perm fs.FileMode) (File, syscall.Errno) {
	f, err := os.OpenFile(path, flag, perm)
	return f, UnwrapOSError(err)
}
