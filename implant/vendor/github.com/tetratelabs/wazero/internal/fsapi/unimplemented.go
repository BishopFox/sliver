package fsapi

import (
	"io/fs"
	"syscall"
	"time"
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
func (UnimplementedFS) OpenFile(path string, flag int, perm fs.FileMode) (File, syscall.Errno) {
	return nil, syscall.ENOSYS
}

// Lstat implements FS.Lstat
func (UnimplementedFS) Lstat(path string) (Stat_t, syscall.Errno) {
	return Stat_t{}, syscall.ENOSYS
}

// Stat implements FS.Stat
func (UnimplementedFS) Stat(path string) (Stat_t, syscall.Errno) {
	return Stat_t{}, syscall.ENOSYS
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

// UnimplementedFile is a File that returns syscall.ENOSYS for all functions,
// except where no-op are otherwise documented.
//
// This should be embedded to have forward compatible implementations.
type UnimplementedFile struct{}

// Ino implements File.Ino
func (UnimplementedFile) Ino() (uint64, syscall.Errno) {
	return 0, 0
}

// IsAppend implements File.IsAppend
func (UnimplementedFile) IsAppend() bool {
	return false
}

// SetAppend implements File.SetAppend
func (UnimplementedFile) SetAppend(bool) syscall.Errno {
	return syscall.ENOSYS
}

// IsNonblock implements File.IsNonblock
func (UnimplementedFile) IsNonblock() bool {
	return false
}

// SetNonblock implements File.SetNonblock
func (UnimplementedFile) SetNonblock(bool) syscall.Errno {
	return syscall.ENOSYS
}

// Stat implements File.Stat
func (UnimplementedFile) Stat() (Stat_t, syscall.Errno) {
	return Stat_t{}, syscall.ENOSYS
}

// IsDir implements File.IsDir
func (UnimplementedFile) IsDir() (bool, syscall.Errno) {
	return false, syscall.ENOSYS
}

// Read implements File.Read
func (UnimplementedFile) Read([]byte) (int, syscall.Errno) {
	return 0, syscall.ENOSYS
}

// Pread implements File.Pread
func (UnimplementedFile) Pread([]byte, int64) (int, syscall.Errno) {
	return 0, syscall.ENOSYS
}

// Seek implements File.Seek
func (UnimplementedFile) Seek(int64, int) (int64, syscall.Errno) {
	return 0, syscall.ENOSYS
}

// Readdir implements File.Readdir
func (UnimplementedFile) Readdir(int) (dirents []Dirent, errno syscall.Errno) {
	return nil, syscall.ENOSYS
}

// PollRead implements File.PollRead
func (UnimplementedFile) PollRead(*time.Duration) (ready bool, errno syscall.Errno) {
	return false, syscall.ENOSYS
}

// Write implements File.Write
func (UnimplementedFile) Write([]byte) (int, syscall.Errno) {
	return 0, syscall.ENOSYS
}

// Pwrite implements File.Pwrite
func (UnimplementedFile) Pwrite([]byte, int64) (int, syscall.Errno) {
	return 0, syscall.ENOSYS
}

// Truncate implements File.Truncate
func (UnimplementedFile) Truncate(int64) syscall.Errno {
	return syscall.ENOSYS
}

// Sync implements File.Sync
func (UnimplementedFile) Sync() syscall.Errno {
	return 0 // not syscall.ENOSYS
}

// Datasync implements File.Datasync
func (UnimplementedFile) Datasync() syscall.Errno {
	return 0 // not syscall.ENOSYS
}

// Chmod implements File.Chmod
func (UnimplementedFile) Chmod(fs.FileMode) syscall.Errno {
	return syscall.ENOSYS
}

// Chown implements File.Chown
func (UnimplementedFile) Chown(int, int) syscall.Errno {
	return syscall.ENOSYS
}

// Utimens implements File.Utimens
func (UnimplementedFile) Utimens(*[2]syscall.Timespec) syscall.Errno {
	return syscall.ENOSYS
}

// Close implements File.Close
func (UnimplementedFile) Close() (errno syscall.Errno) { return }
