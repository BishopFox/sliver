package sysfs

import (
	"os"

	"github.com/tetratelabs/wazero/internal/fsapi"
)

// toOsOpenFlag converts the input to the flag parameter of os.OpenFile
func toOsOpenFlag(oflag fsapi.Oflag) (flag int) {
	// First flags are exclusive
	switch oflag & (fsapi.O_RDONLY | fsapi.O_RDWR | fsapi.O_WRONLY) {
	case fsapi.O_RDONLY:
		flag |= os.O_RDONLY
	case fsapi.O_RDWR:
		flag |= os.O_RDWR
	case fsapi.O_WRONLY:
		flag |= os.O_WRONLY
	}

	// Run down the flags defined in the os package
	if oflag&fsapi.O_APPEND != 0 {
		flag |= os.O_APPEND
	}
	if oflag&fsapi.O_CREAT != 0 {
		flag |= os.O_CREATE
	}
	if oflag&fsapi.O_EXCL != 0 {
		flag |= os.O_EXCL
	}
	if oflag&fsapi.O_SYNC != 0 {
		flag |= os.O_SYNC
	}
	if oflag&fsapi.O_TRUNC != 0 {
		flag |= os.O_TRUNC
	}
	return withSyscallOflag(oflag, flag)
}
