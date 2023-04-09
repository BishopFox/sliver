package platform

import (
	"fmt"
	"io"
	"io/fs"
	"syscall"
)

// Readdirnames reads the names of the directory associated with file and
// returns a slice of up to n strings in an arbitrary order. This is a stateful
// function, so subsequent calls return any next values.
//
// If n > 0, Readdirnames returns at most n entries or an error.
// If n <= 0, Readdirnames returns all remaining entries or an error.
//
// # Errors
//
// A zero syscall.Errno is success.
//
// For portability reasons, no error is returned on io.EOF, when the file is
// closed or removed while open.
// See https://github.com/ziglang/zig/blob/0.10.1/lib/std/fs.zig#L635-L637
func Readdirnames(f fs.File, n int) (names []string, errno syscall.Errno) {
	switch f := f.(type) {
	case readdirnamesFile:
		var err error
		names, err = f.Readdirnames(n)
		if errno = adjustReaddirErr(err); errno != 0 {
			return
		}
	case fs.ReadDirFile:
		entries, err := f.ReadDir(n)
		if errno = adjustReaddirErr(err); errno != 0 {
			return
		}
		names = make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
	default:
		errno = syscall.ENOTDIR
	}
	return
}

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

// Readdir reads the contents of the directory associated with file and returns
// a slice of up to n Dirent values in an arbitrary order. This is a stateful
// function, so subsequent calls return any next values.
//
// If n > 0, Readdir returns at most n entries or an error.
// If n <= 0, Readdir returns all remaining entries or an error.
//
// # Errors
//
// A zero syscall.Errno is success.
//
// For portability reasons, no error is returned on io.EOF, when the file is
// closed or removed while open.
// See https://github.com/ziglang/zig/blob/0.10.1/lib/std/fs.zig#L635-L637
func Readdir(f fs.File, n int) (dirents []*Dirent, errno syscall.Errno) {
	// ^^ case format is to match POSIX and similar to os.File.Readdir

	switch f := f.(type) {
	case readdirFile:
		fis, e := f.Readdir(n)
		if errno = adjustReaddirErr(e); errno != 0 {
			return
		}
		dirents = make([]*Dirent, 0, len(fis))

		// linux/darwin won't have to fan out to lstat, but windows will.
		var ino uint64
		for _, t := range fis {
			if ino, errno = inoFromFileInfo(f, t); errno != 0 {
				return
			}
			dirents = append(dirents, &Dirent{Name: t.Name(), Ino: ino, Type: t.Mode().Type()})
		}
	case fs.ReadDirFile:
		entries, e := f.ReadDir(n)
		if errno = adjustReaddirErr(e); errno != 0 {
			return
		}
		dirents = make([]*Dirent, 0, len(entries))
		for _, e := range entries {
			// By default, we don't attempt to read inode data
			dirents = append(dirents, &Dirent{Name: e.Name(), Type: e.Type()})
		}
	default:
		errno = syscall.ENOTDIR
	}
	return
}

func adjustReaddirErr(err error) syscall.Errno {
	if err == io.EOF {
		return 0 // e.g. Readdir on darwin returns io.EOF, but linux doesn't.
	} else if errno := UnwrapOSError(err); errno != 0 {
		// Ignore errors when the file was closed or removed.
		switch errno {
		case syscall.EIO, syscall.EBADF: // closed while open
			return 0
		case syscall.ENOENT: // Linux error when removed while open
			return 0
		}
		return errno
	}
	return 0
}
