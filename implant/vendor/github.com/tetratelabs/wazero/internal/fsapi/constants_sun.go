//go:build illumos || solaris

package fsapi

import "syscall"

// See https://github.com/illumos/illumos-gate/blob/edd580643f2cf1434e252cd7779e83182ea84945/usr/src/uts/common/sys/fcntl.h#L90
const (
	O_DIRECTORY = 0x1000000
	O_NOFOLLOW  = syscall.O_NOFOLLOW
	O_NONBLOCK  = syscall.O_NONBLOCK
)
