package sysfs

import (
	"io"
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/internal/platform"
)

func adjustReaddirErr(f fsapi.File, isClosed bool, err error) syscall.Errno {
	if err == io.EOF {
		return 0 // e.g. Readdir on darwin returns io.EOF, but linux doesn't.
	} else if errno := platform.UnwrapOSError(err); errno != 0 {
		errno = dirError(f, isClosed, errno)
		// Ignore errors when the file was closed or removed.
		switch errno {
		case syscall.EIO, syscall.EBADF: // closed while open
			return 0
		case syscall.ENOENT: // Linux error when removed while open
			return 0
		}
		return errno
	}
	return 0
}
