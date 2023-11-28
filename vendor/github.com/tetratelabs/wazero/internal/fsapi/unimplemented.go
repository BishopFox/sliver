package fsapi

import (
	"io/fs"
	"syscall"
	"time"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/sys"
)

// UnimplementedFS is an FS that returns ENOSYS for all functions,
// This should be embedded to have forward compatible implementations.
type UnimplementedFS struct{}

// String implements fmt.Stringer
func (UnimplementedFS) String() string {
	return "Unimplemented:/"
}

// Open implements the same method as documented on fs.FS
func (UnimplementedFS) Open(name string) (fs.File, error) {
	return nil, &fs.PathError{Op: "open", Path: name, Err: experimentalsys.ENOSYS}
}

// OpenFile implements FS.OpenFile
func (UnimplementedFS) OpenFile(path string, flag Oflag, perm fs.FileMode) (File, experimentalsys.Errno) {
	return nil, experimentalsys.ENOSYS
}

// Lstat implements FS.Lstat
func (UnimplementedFS) Lstat(path string) (sys.Stat_t, experimentalsys.Errno) {
	return sys.Stat_t{}, experimentalsys.ENOSYS
}

// Stat implements FS.Stat
func (UnimplementedFS) Stat(path string) (sys.Stat_t, experimentalsys.Errno) {
	return sys.Stat_t{}, experimentalsys.ENOSYS
}

// Readlink implements FS.Readlink
func (UnimplementedFS) Readlink(path string) (string, experimentalsys.Errno) {
	return "", experimentalsys.ENOSYS
}

// Mkdir implements FS.Mkdir
func (UnimplementedFS) Mkdir(path string, perm fs.FileMode) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// Chmod implements FS.Chmod
func (UnimplementedFS) Chmod(path string, perm fs.FileMode) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// Rename implements FS.Rename
func (UnimplementedFS) Rename(from, to string) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// Rmdir implements FS.Rmdir
func (UnimplementedFS) Rmdir(path string) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// Link implements FS.Link
func (UnimplementedFS) Link(_, _ string) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// Symlink implements FS.Symlink
func (UnimplementedFS) Symlink(_, _ string) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// Unlink implements FS.Unlink
func (UnimplementedFS) Unlink(path string) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// Utimens implements FS.Utimens
func (UnimplementedFS) Utimens(path string, times *[2]syscall.Timespec) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// Truncate implements FS.Truncate
func (UnimplementedFS) Truncate(string, int64) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// UnimplementedFile is a File that returns ENOSYS for all functions,
// except where no-op are otherwise documented.
//
// This should be embedded to have forward compatible implementations.
type UnimplementedFile struct{}

// Dev implements File.Dev
func (UnimplementedFile) Dev() (uint64, experimentalsys.Errno) {
	return 0, 0
}

// Ino implements File.Ino
func (UnimplementedFile) Ino() (sys.Inode, experimentalsys.Errno) {
	return 0, 0
}

// IsDir implements File.IsDir
func (UnimplementedFile) IsDir() (bool, experimentalsys.Errno) {
	return false, 0
}

// IsAppend implements File.IsAppend
func (UnimplementedFile) IsAppend() bool {
	return false
}

// SetAppend implements File.SetAppend
func (UnimplementedFile) SetAppend(bool) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// IsNonblock implements File.IsNonblock
func (UnimplementedFile) IsNonblock() bool {
	return false
}

// SetNonblock implements File.SetNonblock
func (UnimplementedFile) SetNonblock(bool) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// Stat implements File.Stat
func (UnimplementedFile) Stat() (sys.Stat_t, experimentalsys.Errno) {
	return sys.Stat_t{}, experimentalsys.ENOSYS
}

// Read implements File.Read
func (UnimplementedFile) Read([]byte) (int, experimentalsys.Errno) {
	return 0, experimentalsys.ENOSYS
}

// Pread implements File.Pread
func (UnimplementedFile) Pread([]byte, int64) (int, experimentalsys.Errno) {
	return 0, experimentalsys.ENOSYS
}

// Seek implements File.Seek
func (UnimplementedFile) Seek(int64, int) (int64, experimentalsys.Errno) {
	return 0, experimentalsys.ENOSYS
}

// Readdir implements File.Readdir
func (UnimplementedFile) Readdir(int) (dirents []Dirent, errno experimentalsys.Errno) {
	return nil, experimentalsys.ENOSYS
}

// PollRead implements File.PollRead
func (UnimplementedFile) PollRead(*time.Duration) (ready bool, errno experimentalsys.Errno) {
	return false, experimentalsys.ENOSYS
}

// Write implements File.Write
func (UnimplementedFile) Write([]byte) (int, experimentalsys.Errno) {
	return 0, experimentalsys.ENOSYS
}

// Pwrite implements File.Pwrite
func (UnimplementedFile) Pwrite([]byte, int64) (int, experimentalsys.Errno) {
	return 0, experimentalsys.ENOSYS
}

// Truncate implements File.Truncate
func (UnimplementedFile) Truncate(int64) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// Sync implements File.Sync
func (UnimplementedFile) Sync() experimentalsys.Errno {
	return 0 // not ENOSYS
}

// Datasync implements File.Datasync
func (UnimplementedFile) Datasync() experimentalsys.Errno {
	return 0 // not ENOSYS
}

// Utimens implements File.Utimens
func (UnimplementedFile) Utimens(*[2]syscall.Timespec) experimentalsys.Errno {
	return experimentalsys.ENOSYS
}

// Close implements File.Close
func (UnimplementedFile) Close() (errno experimentalsys.Errno) { return }
