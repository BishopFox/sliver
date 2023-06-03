package fsapi

import (
	"fmt"
	"io/fs"
	"syscall"
	"time"
)

// Dirent is an entry read from a directory.
//
// This is a portable variant of syscall.Dirent containing fields needed for
// WebAssembly ABI including WASI snapshot-01 and wasi-filesystem. Unlike
// fs.DirEntry, this may include the Ino.
type Dirent struct {
	// ^^ Dirent name matches syscall.Dirent

	// Name is the base name of the directory entry.
	Name string

	// Ino is the file serial number, or zero if not available.
	Ino uint64

	// Type is fs.FileMode masked on fs.ModeType. For example, zero is a
	// regular file, fs.ModeDir is a directory and fs.ModeIrregular is unknown.
	Type fs.FileMode
}

func (d *Dirent) String() string {
	return fmt.Sprintf("name=%s, type=%v, ino=%d", d.Name, d.Type, d.Ino)
}

// IsDir returns true if the Type is fs.ModeDir.
func (d *Dirent) IsDir() bool {
	return d.Type == fs.ModeDir
}

// DirFile is embeddable to reduce the amount of functions to implement a file.
type DirFile struct{}

// IsAppend implements File.IsAppend
func (DirFile) IsAppend() bool {
	return false
}

// SetAppend implements File.SetAppend
func (DirFile) SetAppend(bool) syscall.Errno {
	return syscall.EISDIR
}

// IsNonblock implements File.IsNonblock
func (DirFile) IsNonblock() bool {
	return false
}

// SetNonblock implements File.SetNonblock
func (DirFile) SetNonblock(bool) syscall.Errno {
	return syscall.EISDIR
}

// IsDir implements File.IsDir
func (DirFile) IsDir() (bool, syscall.Errno) {
	return true, 0
}

// Read implements File.Read
func (DirFile) Read([]byte) (int, syscall.Errno) {
	return 0, syscall.EISDIR
}

// Pread implements File.Pread
func (DirFile) Pread([]byte, int64) (int, syscall.Errno) {
	return 0, syscall.EISDIR
}

// Seek implements File.Seek
func (DirFile) Seek(int64, int) (int64, syscall.Errno) {
	return 0, syscall.EISDIR
}

// PollRead implements File.PollRead
func (DirFile) PollRead(*time.Duration) (ready bool, errno syscall.Errno) {
	return false, syscall.ENOSYS
}

// Write implements File.Write
func (DirFile) Write([]byte) (int, syscall.Errno) {
	return 0, syscall.EISDIR
}

// Pwrite implements File.Pwrite
func (DirFile) Pwrite([]byte, int64) (int, syscall.Errno) {
	return 0, syscall.EISDIR
}

// Truncate implements File.Truncate
func (DirFile) Truncate(int64) syscall.Errno {
	return syscall.EISDIR
}
