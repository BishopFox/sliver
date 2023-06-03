//go:build !linux && (!darwin || sqlite3_bsd)

package sqlite3

import (
	"io"
	"os"
)

func (vfsOSMethods) Sync(file *os.File, fullsync, dataonly bool) error {
	return file.Sync()
}

func (vfsOSMethods) Allocate(file *os.File, size int64) error {
	off, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	if size <= off {
		return nil
	}
	return file.Truncate(size)
}
