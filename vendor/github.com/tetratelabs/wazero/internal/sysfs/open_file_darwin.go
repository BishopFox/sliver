package sysfs

import (
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
)

const supportedSyscallOflag = fsapi.O_DIRECTORY | fsapi.O_DSYNC | fsapi.O_NOFOLLOW | fsapi.O_NONBLOCK

func withSyscallOflag(oflag fsapi.Oflag, flag int) int {
	if oflag&fsapi.O_DIRECTORY != 0 {
		flag |= syscall.O_DIRECTORY
	}
	if oflag&fsapi.O_DSYNC != 0 {
		flag |= syscall.O_DSYNC
	}
	if oflag&fsapi.O_NOFOLLOW != 0 {
		flag |= syscall.O_NOFOLLOW
	}
	if oflag&fsapi.O_NONBLOCK != 0 {
		flag |= syscall.O_NONBLOCK
	}
	// syscall.O_RSYNC not defined on darwin
	return flag
}
