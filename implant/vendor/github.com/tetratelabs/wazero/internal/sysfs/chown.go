package sysfs

import (
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

// Chown is like os.Chown, except it returns a syscall.Errno, not a
// fs.PathError. For example, this returns syscall.ENOENT if the path doesn't
// exist. A syscall.Errno of zero is success.
//
// Note: This always returns syscall.ENOSYS on windows.
// See https://linux.die.net/man/3/chown
func Chown(path string, uid, gid int) syscall.Errno {
	err := os.Chown(path, uid, gid)
	return platform.UnwrapOSError(err)
}

// Lchown is like os.Lchown, except it returns a syscall.Errno, not a
// fs.PathError. For example, this returns syscall.ENOENT if the path doesn't
// exist. A syscall.Errno of zero is success.
//
// Note: This always returns syscall.ENOSYS on windows.
// See https://linux.die.net/man/3/lchown
func Lchown(path string, uid, gid int) syscall.Errno {
	err := os.Lchown(path, uid, gid)
	return platform.UnwrapOSError(err)
}
