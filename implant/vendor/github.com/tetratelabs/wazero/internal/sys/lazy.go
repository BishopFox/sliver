package sys

import (
	"io/fs"
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
)

// compile-time check to ensure lazyDir implements internalapi.File.
var _ fsapi.File = (*lazyDir)(nil)

type lazyDir struct {
	fsapi.DirFile

	fs fsapi.FS
	f  fsapi.File
}

// Ino implements the same method as documented on internalapi.File
func (r *lazyDir) Ino() (uint64, syscall.Errno) {
	if f, ok := r.file(); !ok {
		return 0, syscall.EBADF
	} else {
		return f.Ino()
	}
}

// IsAppend implements the same method as documented on internalapi.File
func (r *lazyDir) IsAppend() bool {
	return false
}

// SetAppend implements the same method as documented on internalapi.File
func (r *lazyDir) SetAppend(bool) syscall.Errno {
	return syscall.EISDIR
}

// Seek implements the same method as documented on internalapi.File
func (r *lazyDir) Seek(offset int64, whence int) (newOffset int64, errno syscall.Errno) {
	if f, ok := r.file(); !ok {
		return 0, syscall.EBADF
	} else {
		return f.Seek(offset, whence)
	}
}

// Stat implements the same method as documented on internalapi.File
func (r *lazyDir) Stat() (fsapi.Stat_t, syscall.Errno) {
	if f, ok := r.file(); !ok {
		return fsapi.Stat_t{}, syscall.EBADF
	} else {
		return f.Stat()
	}
}

// Readdir implements the same method as documented on internalapi.File
func (r *lazyDir) Readdir(n int) (dirents []fsapi.Dirent, errno syscall.Errno) {
	if f, ok := r.file(); !ok {
		return nil, syscall.EBADF
	} else {
		return f.Readdir(n)
	}
}

// Sync implements the same method as documented on internalapi.File
func (r *lazyDir) Sync() syscall.Errno {
	if f, ok := r.file(); !ok {
		return syscall.EBADF
	} else {
		return f.Sync()
	}
}

// Datasync implements the same method as documented on internalapi.File
func (r *lazyDir) Datasync() syscall.Errno {
	if f, ok := r.file(); !ok {
		return syscall.EBADF
	} else {
		return f.Datasync()
	}
}

// Chmod implements the same method as documented on internalapi.File
func (r *lazyDir) Chmod(mode fs.FileMode) syscall.Errno {
	if f, ok := r.file(); !ok {
		return syscall.EBADF
	} else {
		return f.Chmod(mode)
	}
}

// Chown implements the same method as documented on internalapi.File
func (r *lazyDir) Chown(uid, gid int) syscall.Errno {
	if f, ok := r.file(); !ok {
		return syscall.EBADF
	} else {
		return f.Chown(uid, gid)
	}
}

// Utimens implements the same method as documented on internalapi.File
func (r *lazyDir) Utimens(times *[2]syscall.Timespec) syscall.Errno {
	if f, ok := r.file(); !ok {
		return syscall.EBADF
	} else {
		return f.Utimens(times)
	}
}

// file returns the underlying file or false if it doesn't exist.
func (r *lazyDir) file() (fsapi.File, bool) {
	if f := r.f; r.f != nil {
		return f, true
	}
	var errno syscall.Errno
	r.f, errno = r.fs.OpenFile(".", os.O_RDONLY, 0)
	switch errno {
	case 0:
		return r.f, true
	case syscall.ENOENT:
		return nil, false
	default:
		panic(errno) // unexpected
	}
}

// Close implements fs.File
func (r *lazyDir) Close() syscall.Errno {
	f := r.f
	if f == nil {
		return 0 // never opened
	}
	return f.Close()
}
