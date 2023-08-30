package sysfs

import (
	"io/fs"
	"syscall"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/fsapi"
)

// NewReadFS is used to mask an existing fsapi.FS for reads. Notably, this allows
// the CLI to do read-only mounts of directories the host user can write, but
// doesn't want the guest wasm to. For example, Python libraries shouldn't be
// written to at runtime by the python wasm file.
func NewReadFS(fs fsapi.FS) fsapi.FS {
	if _, ok := fs.(*readFS); ok {
		return fs
	} else if _, ok = fs.(fsapi.UnimplementedFS); ok {
		return fs // unimplemented is read-only
	}
	return &readFS{fs}
}

type readFS struct {
	fsapi.FS
}

// OpenFile implements the same method as documented on fsapi.FS
func (r *readFS) OpenFile(path string, flag fsapi.Oflag, perm fs.FileMode) (fsapi.File, experimentalsys.Errno) {
	// TODO: Once the real implementation is complete, move the below to
	// /RATIONALE.md. Doing this while the type is unstable creates
	// documentation drift as we expect a lot of reshaping meanwhile.
	//
	// Callers of this function expect to either open a valid file handle, or
	// get an error, if they can't. We want to return ENOSYS if opened for
	// anything except reads.
	//
	// Instead, we could return a fake no-op file on O_WRONLY. However, this
	// hurts observability because a later write error to that file will be on
	// a different source code line than the root cause which is opening with
	// an unsupported flag.
	//
	// The tricky part is os.RD_ONLY is typically defined as zero, so while the
	// parameter is named flag, the part about opening read vs write isn't a
	// typical bitflag. We can't compare against zero anyway, because even if
	// there isn't a current flag to OR in with that, there may be in the
	// future. What we do instead is mask the flags about read/write mode and
	// check if they are the opposite of read or not.
	switch flag & (fsapi.O_RDONLY | fsapi.O_WRONLY | fsapi.O_RDWR) {
	case fsapi.O_WRONLY, fsapi.O_RDWR:
		if flag&fsapi.O_DIRECTORY != 0 {
			return nil, experimentalsys.EISDIR
		}
		return nil, experimentalsys.ENOSYS
	default: // fsapi.O_RDONLY (or no flag) so we are ok!
	}

	f, errno := r.FS.OpenFile(path, flag, perm)
	if errno != 0 {
		return nil, errno
	}
	return &readFile{f}, 0
}

// Mkdir implements the same method as documented on fsapi.FS
func (r *readFS) Mkdir(path string, perm fs.FileMode) experimentalsys.Errno {
	return experimentalsys.EROFS
}

// Chmod implements the same method as documented on fsapi.FS
func (r *readFS) Chmod(path string, perm fs.FileMode) experimentalsys.Errno {
	return experimentalsys.EROFS
}

// Rename implements the same method as documented on fsapi.FS
func (r *readFS) Rename(from, to string) experimentalsys.Errno {
	return experimentalsys.EROFS
}

// Rmdir implements the same method as documented on fsapi.FS
func (r *readFS) Rmdir(path string) experimentalsys.Errno {
	return experimentalsys.EROFS
}

// Link implements the same method as documented on fsapi.FS
func (r *readFS) Link(_, _ string) experimentalsys.Errno {
	return experimentalsys.EROFS
}

// Symlink implements the same method as documented on fsapi.FS
func (r *readFS) Symlink(_, _ string) experimentalsys.Errno {
	return experimentalsys.EROFS
}

// Unlink implements the same method as documented on fsapi.FS
func (r *readFS) Unlink(path string) experimentalsys.Errno {
	return experimentalsys.EROFS
}

// Utimens implements the same method as documented on fsapi.FS
func (r *readFS) Utimens(path string, times *[2]syscall.Timespec) experimentalsys.Errno {
	return experimentalsys.EROFS
}

// compile-time check to ensure readFile implements api.File.
var _ fsapi.File = (*readFile)(nil)

type readFile struct {
	fsapi.File
}

// Write implements the same method as documented on fsapi.File.
func (r *readFile) Write([]byte) (int, experimentalsys.Errno) {
	return 0, r.writeErr()
}

// Pwrite implements the same method as documented on fsapi.File.
func (r *readFile) Pwrite([]byte, int64) (n int, errno experimentalsys.Errno) {
	return 0, r.writeErr()
}

// Truncate implements the same method as documented on fsapi.File.
func (r *readFile) Truncate(int64) experimentalsys.Errno {
	return r.writeErr()
}

// Sync implements the same method as documented on fsapi.File.
func (r *readFile) Sync() experimentalsys.Errno {
	return experimentalsys.EBADF
}

// Datasync implements the same method as documented on fsapi.File.
func (r *readFile) Datasync() experimentalsys.Errno {
	return experimentalsys.EBADF
}

// Utimens implements the same method as documented on fsapi.File.
func (r *readFile) Utimens(*[2]syscall.Timespec) experimentalsys.Errno {
	return experimentalsys.EBADF
}

func (r *readFile) writeErr() experimentalsys.Errno {
	if isDir, errno := r.IsDir(); errno != 0 {
		return errno
	} else if isDir {
		return experimentalsys.EISDIR
	}
	return experimentalsys.EBADF
}
