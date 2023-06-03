package sqlite3

import (
	"os"

	"golang.org/x/sys/unix"
)

func (vfsOSMethods) Sync(file *os.File, fullsync, dataonly bool) error {
	if dataonly {
		_, _, err := unix.Syscall(unix.SYS_FDATASYNC, file.Fd(), 0, 0)
		if err != 0 {
			return err
		}
		return nil
	}
	return file.Sync()
}

func (vfsOSMethods) Allocate(file *os.File, size int64) error {
	if size == 0 {
		return nil
	}
	return unix.Fallocate(int(file.Fd()), 0, 0, size)
}
