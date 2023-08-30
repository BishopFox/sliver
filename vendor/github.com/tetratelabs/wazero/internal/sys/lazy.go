package sys

import (
	"syscall"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/sys"
)

// compile-time check to ensure lazyDir implements fsapi.File.
var _ fsapi.File = (*lazyDir)(nil)

type lazyDir struct {
	fsapi.DirFile

	fs fsapi.FS
	f  fsapi.File
}

// Dev implements the same method as documented on fsapi.File
func (r *lazyDir) Dev() (uint64, experimentalsys.Errno) {
	if f, ok := r.file(); !ok {
		return 0, experimentalsys.EBADF
	} else {
		return f.Dev()
	}
}

// Ino implements the same method as documented on fsapi.File
func (r *lazyDir) Ino() (sys.Inode, experimentalsys.Errno) {
	if f, ok := r.file(); !ok {
		return 0, experimentalsys.EBADF
	} else {
		return f.Ino()
	}
}

// IsDir implements the same method as documented on fsapi.File
func (r *lazyDir) IsDir() (bool, experimentalsys.Errno) {
	// Note: we don't return a constant because we don't know if this is really
	// backed by a dir, until the first call.
	if f, ok := r.file(); !ok {
		return false, experimentalsys.EBADF
	} else {
		return f.IsDir()
	}
}

// IsAppend implements the same method as documented on fsapi.File
func (r *lazyDir) IsAppend() bool {
	return false
}

// SetAppend implements the same method as documented on fsapi.File
func (r *lazyDir) SetAppend(bool) experimentalsys.Errno {
	return experimentalsys.EISDIR
}

// Seek implements the same method as documented on fsapi.File
func (r *lazyDir) Seek(offset int64, whence int) (newOffset int64, errno experimentalsys.Errno) {
	if f, ok := r.file(); !ok {
		return 0, experimentalsys.EBADF
	} else {
		return f.Seek(offset, whence)
	}
}

// Stat implements the same method as documented on fsapi.File
func (r *lazyDir) Stat() (sys.Stat_t, experimentalsys.Errno) {
	if f, ok := r.file(); !ok {
		return sys.Stat_t{}, experimentalsys.EBADF
	} else {
		return f.Stat()
	}
}

// Readdir implements the same method as documented on fsapi.File
func (r *lazyDir) Readdir(n int) (dirents []fsapi.Dirent, errno experimentalsys.Errno) {
	if f, ok := r.file(); !ok {
		return nil, experimentalsys.EBADF
	} else {
		return f.Readdir(n)
	}
}

// Sync implements the same method as documented on fsapi.File
func (r *lazyDir) Sync() experimentalsys.Errno {
	if f, ok := r.file(); !ok {
		return experimentalsys.EBADF
	} else {
		return f.Sync()
	}
}

// Datasync implements the same method as documented on fsapi.File
func (r *lazyDir) Datasync() experimentalsys.Errno {
	if f, ok := r.file(); !ok {
		return experimentalsys.EBADF
	} else {
		return f.Datasync()
	}
}

// Utimens implements the same method as documented on fsapi.File
func (r *lazyDir) Utimens(times *[2]syscall.Timespec) experimentalsys.Errno {
	if f, ok := r.file(); !ok {
		return experimentalsys.EBADF
	} else {
		return f.Utimens(times)
	}
}

// file returns the underlying file or false if it doesn't exist.
func (r *lazyDir) file() (fsapi.File, bool) {
	if f := r.f; r.f != nil {
		return f, true
	}
	var errno experimentalsys.Errno
	r.f, errno = r.fs.OpenFile(".", fsapi.O_RDONLY, 0)
	switch errno {
	case 0:
		return r.f, true
	case experimentalsys.ENOENT:
		return nil, false
	default:
		panic(errno) // unexpected
	}
}

// Close implements fs.File
func (r *lazyDir) Close() experimentalsys.Errno {
	f := r.f
	if f == nil {
		return 0 // never opened
	}
	return f.Close()
}
