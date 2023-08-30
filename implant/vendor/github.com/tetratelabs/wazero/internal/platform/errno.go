//go:build !windows

package platform

import "syscall"

func adjustErrno(err syscall.Errno) syscall.Errno {
	return err
}
