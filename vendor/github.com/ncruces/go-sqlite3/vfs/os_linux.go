//go:build !sqlite3_nosys

package vfs

import (
	"os"

	"golang.org/x/sys/unix"
)

func osSync(file *os.File, _ /*fullsync*/, _ /*dataonly*/ bool) error {
	// SQLite trusts Linux's fdatasync for all fsync's.
	return unix.Fdatasync(int(file.Fd()))
}

func osAllocate(file *os.File, size int64) error {
	if size == 0 {
		return nil
	}
	return unix.Fallocate(int(file.Fd()), 0, 0, size)
}
