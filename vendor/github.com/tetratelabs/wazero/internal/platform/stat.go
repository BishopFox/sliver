package platform

import (
	"io/fs"
	"syscall"
)

// Stat_t is similar to syscall.Stat_t, and fields frequently used by
// WebAssembly ABI including WASI snapshot-01, GOOS=js and wasi-filesystem.
//
// # Note
//
// Zero values may be returned where not available. For example, fs.FileInfo
// implementations may not be able to provide Ino values.
type Stat_t struct {
	// Dev is the device ID of device containing the file.
	Dev uint64

	// Ino is the file serial number.
	Ino uint64

	// Uid is the user ID that owns the file, or zero if unsupported.
	// For example, this is unsupported on some virtual filesystems or windows.
	Uid uint32

	// Gid is the group ID that owns the file, or zero if unsupported.
	// For example, this is unsupported on some virtual filesystems or windows.
	Gid uint32

	// Mode is the same as Mode on fs.FileInfo containing bits to identify the
	// type of the file (fs.ModeType) and its permissions (fs.ModePerm).
	Mode fs.FileMode

	/// Nlink is the number of hard links to the file.
	Nlink uint64
	// ^^ uint64 not uint16 to accept widest syscall.Stat_t.Nlink

	// Size is the length in bytes for regular files. For symbolic links, this
	// is length in bytes of the pathname contained in the symbolic link.
	Size int64
	// ^^ int64 not uint64 to defer to fs.FileInfo

	// Atim is the last data access timestamp in epoch nanoseconds.
	Atim int64

	// Mtim is the last data modification timestamp in epoch nanoseconds.
	Mtim int64

	// Ctim is the last file status change timestamp in epoch nanoseconds.
	Ctim int64
}

// Lstat is like syscall.Lstat. This returns syscall.ENOENT if the path doesn't
// exist.
//
// # Notes
//
// The primary difference between this and Stat is, when the path is a
// symbolic link, the stat is about the link, not its target, such as directory
// listings.
func Lstat(path string) (Stat_t, syscall.Errno) {
	return lstat(path) // extracted to override more expensively in windows
}

// Stat is like syscall.Stat. This returns syscall.ENOENT if the path doesn't
// exist.
func Stat(path string) (Stat_t, syscall.Errno) {
	return stat(path) // extracted to override more expensively in windows
}

// StatFile is like syscall.Fstat, but for fs.File instead of a file
// descriptor. This returns syscall.EBADF if the file or directory was closed.
// Note: windows allows you to stat a closed directory.
func StatFile(f fs.File) (Stat_t, syscall.Errno) {
	st, errno := statFile(f)
	if errno == syscall.EIO {
		errno = syscall.EBADF
	}
	return st, errno
}

func defaultStatFile(f fs.File) (Stat_t, syscall.Errno) {
	if t, err := f.Stat(); err != nil {
		return Stat_t{}, UnwrapOSError(err)
	} else {
		return statFromFileInfo(t), 0
	}
}

func statFromDefaultFileInfo(t fs.FileInfo) Stat_t {
	st := Stat_t{}
	st.Ino = 0
	st.Dev = 0
	st.Mode = t.Mode()
	st.Nlink = 1
	st.Size = t.Size()
	mtim := t.ModTime().UnixNano() // Set all times to the mod time
	st.Atim = mtim
	st.Mtim = mtim
	st.Ctim = mtim
	return st
}
