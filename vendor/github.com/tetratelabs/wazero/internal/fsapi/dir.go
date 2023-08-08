package fsapi

import (
	"fmt"
	"io/fs"
	"time"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/sys"
)

// FileType is fs.FileMode masked on fs.ModeType. For example, zero is a
// regular file, fs.ModeDir is a directory and fs.ModeIrregular is unknown.
//
// Note: This is defined by Linux, not POSIX.
type FileType = fs.FileMode

// Dirent is an entry read from a directory via File.Readdir.
//
// # Notes
//
//   - This extends `dirent` defined in POSIX with some fields defined by
//     Linux. See https://man7.org/linux/man-pages/man3/readdir.3.html and
//     https://pubs.opengroup.org/onlinepubs/9699919799/basedefs/dirent.h.html
//   - This has a subset of fields defined in Stat_t. Notably, there is no
//     field corresponding to Stat_t.Dev because that value will be constant
//     for all files in a directory. To get the Dev value, call File.Stat on
//     the directory File.Readdir was called on.
type Dirent struct {
	// Ino is the file serial number, or zero if not available. See Ino for
	// more details including impact returning a zero value.
	Ino sys.Inode

	// Name is the base name of the directory entry. Empty is invalid.
	Name string

	// Type is fs.FileMode masked on fs.ModeType. For example, zero is a
	// regular file, fs.ModeDir is a directory and fs.ModeIrregular is unknown.
	//
	// Note: This is defined by Linux, not POSIX.
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
func (DirFile) SetAppend(bool) experimentalsys.Errno {
	return experimentalsys.EISDIR
}

// IsNonblock implements File.IsNonblock
func (DirFile) IsNonblock() bool {
	return false
}

// SetNonblock implements File.SetNonblock
func (DirFile) SetNonblock(bool) experimentalsys.Errno {
	return experimentalsys.EISDIR
}

// IsDir implements File.IsDir
func (DirFile) IsDir() (bool, experimentalsys.Errno) {
	return true, 0
}

// Read implements File.Read
func (DirFile) Read([]byte) (int, experimentalsys.Errno) {
	return 0, experimentalsys.EISDIR
}

// Pread implements File.Pread
func (DirFile) Pread([]byte, int64) (int, experimentalsys.Errno) {
	return 0, experimentalsys.EISDIR
}

// PollRead implements File.PollRead
func (DirFile) PollRead(*time.Duration) (ready bool, errno experimentalsys.Errno) {
	return false, experimentalsys.ENOSYS
}

// Write implements File.Write
func (DirFile) Write([]byte) (int, experimentalsys.Errno) {
	return 0, experimentalsys.EISDIR
}

// Pwrite implements File.Pwrite
func (DirFile) Pwrite([]byte, int64) (int, experimentalsys.Errno) {
	return 0, experimentalsys.EISDIR
}

// Truncate implements File.Truncate
func (DirFile) Truncate(int64) experimentalsys.Errno {
	return experimentalsys.EISDIR
}
