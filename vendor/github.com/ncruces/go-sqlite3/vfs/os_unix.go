//go:build unix && !sqlite3_nosys

package vfs

import (
	"os"
	"syscall"

	"golang.org/x/sys/unix"
)

func osAccess(path string, flags AccessFlag) error {
	var access uint32 // unix.F_OK
	switch flags {
	case ACCESS_READWRITE:
		access = unix.R_OK | unix.W_OK
	case ACCESS_READ:
		access = unix.R_OK
	}
	return unix.Access(path, access)
}

func osSetMode(file *os.File, modeof string) error {
	fi, err := os.Stat(modeof)
	if err != nil {
		return err
	}
	file.Chmod(fi.Mode())
	if sys, ok := fi.Sys().(*syscall.Stat_t); ok {
		file.Chown(int(sys.Uid), int(sys.Gid))
	}
	return nil
}
