//go:build !windows

package syscallfs

import "syscall"

func adjustMkdirError(err error) error {
	return err
}

func adjustRmdirError(err error) error {
	return err
}

func adjustUnlinkError(err error) error {
	if err == syscall.EPERM {
		return syscall.EISDIR
	}
	return err
}

func rename(old, new string) error {
	return syscall.Rename(old, new)
}
