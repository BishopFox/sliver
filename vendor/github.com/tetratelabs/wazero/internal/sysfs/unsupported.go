package sysfs

import (
	"io/fs"
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

// UnimplementedFS is an FS that returns syscall.ENOSYS for all functions,
// This should be embedded to have forward compatible implementations.
type UnimplementedFS struct{}

// String implements fmt.Stringer
func (UnimplementedFS) String() string {
	return "Unimplemented:/"
}

// Open implements the same method as documented on fs.FS
func (UnimplementedFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: syscall.ENOSYS}
}

// OpenFile implements FS.OpenFile
func (UnimplementedFS) OpenFile(path string, flag int, perm fs.FileMode) (fs.File, syscall.Errno) {
	return nil, syscall.ENOSYS
}

// Lstat implements FS.Lstat
func (UnimplementedFS) Lstat(path string) (platform.Stat_t, syscall.Errno) {
	return platform.Stat_t{}, syscall.ENOSYS
}

// Stat implements FS.Stat
func (UnimplementedFS) Stat(path string) (platform.Stat_t, syscall.Errno) {
	return platform.Stat_t{}, syscall.ENOSYS
}

// Readlink implements FS.Readlink
func (UnimplementedFS) Readlink(path string) (string, syscall.Errno) {
	return "", syscall.ENOSYS
}

// Mkdir implements FS.Mkdir
func (UnimplementedFS) Mkdir(path string, perm fs.FileMode) syscall.Errno {
	return syscall.ENOSYS
}

// Chmod implements FS.Chmod
func (UnimplementedFS) Chmod(path string, perm fs.FileMode) syscall.Errno {
	return syscall.ENOSYS
}

// Chown implements FS.Chown
func (UnimplementedFS) Chown(path string, uid, gid int) syscall.Errno {
	return syscall.ENOSYS
}

// Lchown implements FS.Lchown
func (UnimplementedFS) Lchown(path string, uid, gid int) syscall.Errno {
	return syscall.ENOSYS
}

// Rename implements FS.Rename
func (UnimplementedFS) Rename(from, to string) syscall.Errno {
	return syscall.ENOSYS
}

// Rmdir implements FS.Rmdir
func (UnimplementedFS) Rmdir(path string) syscall.Errno {
	return syscall.ENOSYS
}

// Link implements FS.Link
func (UnimplementedFS) Link(_, _ string) syscall.Errno {
	return syscall.ENOSYS
}

// Symlink implements FS.Symlink
func (UnimplementedFS) Symlink(_, _ string) syscall.Errno {
	return syscall.ENOSYS
}

// Unlink implements FS.Unlink
func (UnimplementedFS) Unlink(path string) syscall.Errno {
	return syscall.ENOSYS
}

// Utimens implements FS.Utimens
func (UnimplementedFS) Utimens(path string, times *[2]syscall.Timespec, symlinkFollow bool) syscall.Errno {
	return syscall.ENOSYS
}

// Truncate implements FS.Truncate
func (UnimplementedFS) Truncate(string, int64) syscall.Errno {
	return syscall.ENOSYS
}
