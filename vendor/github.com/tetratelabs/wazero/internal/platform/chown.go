package platform

import (
	"io/fs"
	"os"
	"syscall"
)

// Chown is like os.Chown, except it returns a syscall.Errno, not a
// fs.PathError. For example, this returns syscall.ENOENT if the path doesn't
// exist. A syscall.Errno of zero is success.
//
// Note: This always returns syscall.ENOSYS on windows.
// See https://linux.die.net/man/3/chown
func Chown(path string, uid, gid int) syscall.Errno {
	err := os.Chown(path, uid, gid)
	return UnwrapOSError(err)
}

// Lchown is like os.Lchown, except it returns a syscall.Errno, not a
// fs.PathError. For example, this returns syscall.ENOENT if the path doesn't
// exist. A syscall.Errno of zero is success.
//
// Note: This always returns syscall.ENOSYS on windows.
// See https://linux.die.net/man/3/lchown
func Lchown(path string, uid, gid int) syscall.Errno {
	err := os.Lchown(path, uid, gid)
	return UnwrapOSError(err)
}

// ChownFile is like syscall.Fchown, but for nanosecond precision and
// fs.File instead of a file descriptor. This returns syscall.EBADF if the file
// or directory was closed. See https://linux.die.net/man/3/fchown
//
// Note: This always returns syscall.ENOSYS on windows.
func ChownFile(f fs.File, uid, gid int) syscall.Errno {
	if f, ok := f.(fdFile); ok {
		return fchown(f.Fd(), uid, gid)
	}
	return syscall.ENOSYS
}
