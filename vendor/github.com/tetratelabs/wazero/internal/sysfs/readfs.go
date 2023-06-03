package sysfs

import (
	"io"
	"io/fs"
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

// NewReadFS is used to mask an existing FS for reads. Notably, this allows
// the CLI to do read-only mounts of directories the host user can write, but
// doesn't want the guest wasm to. For example, Python libraries shouldn't be
// written to at runtime by the python wasm file.
func NewReadFS(fs FS) FS {
	if _, ok := fs.(*readFS); ok {
		return fs
	} else if _, ok = fs.(UnimplementedFS); ok {
		return fs // unimplemented is read-only
	}
	return &readFS{fs: fs}
}

type readFS struct {
	fs FS
}

// String implements fmt.Stringer
func (r *readFS) String() string {
	return r.fs.String()
}

// Open implements the same method as documented on fs.FS
func (r *readFS) Open(name string) (fs.File, error) {
	return fsOpen(r, name)
}

// OpenFile implements FS.OpenFile
func (r *readFS) OpenFile(path string, flag int, perm fs.FileMode) (fs.File, syscall.Errno) {
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
	switch flag & (os.O_RDONLY | os.O_WRONLY | os.O_RDWR) {
	case os.O_WRONLY, os.O_RDWR:
		return nil, syscall.ENOSYS
	default: // os.O_RDONLY so we are ok!
	}

	f, errno := r.fs.OpenFile(path, flag, perm)
	if errno != 0 {
		return nil, errno
	}
	return maskForReads(f), 0
}

// maskForReads masks the file with read-only interfaces used by wazero.
//
// This technique was adapted from similar code in zipkin-go.
func maskForReads(f fs.File) fs.File {
	// Handle the most common types
	rf, ok := f.(platform.ReadFile)
	pf, pok := f.(platform.PathFile)
	switch {
	case ok && !pok:
		return struct {
			platform.ReadFile
		}{rf}
	case ok && pok:
		return struct {
			platform.ReadFile
			platform.PathFile
		}{rf, pf}
	}

	// The below are the types wazero casts into.
	// Note: os.File implements this even for normal files.
	d, i0 := f.(fs.ReadDirFile)
	ra, i1 := f.(io.ReaderAt)
	s, i2 := f.(io.Seeker)

	// Wrap any combination of the types above.
	switch {
	case !i0 && !i1 && !i2: // 0, 0, 0
		return struct{ fs.File }{f}
	case !i0 && !i1 && i2: // 0, 0, 1
		return struct {
			fs.File
			io.Seeker
		}{f, s}
	case !i0 && i1 && !i2: // 0, 1, 0
		return struct {
			fs.File
			io.ReaderAt
		}{f, ra}
	case !i0 && i1 && i2: // 0, 1, 1
		return struct {
			fs.File
			io.ReaderAt
			io.Seeker
		}{f, ra, s}
	case i0 && !i1 && !i2: // 1, 0, 0
		return struct {
			fs.ReadDirFile
		}{d}
	case i0 && !i1 && i2: // 1, 0, 1
		return struct {
			fs.ReadDirFile
			io.Seeker
		}{d, s}
	case i0 && i1 && !i2: // 1, 1, 0
		return struct {
			fs.ReadDirFile
			io.ReaderAt
		}{d, ra}
	case i0 && i1 && i2: // 1, 1, 1
		return struct {
			fs.ReadDirFile
			io.ReaderAt
			io.Seeker
		}{d, ra, s}
	default:
		panic("BUG: unhandled pattern")
	}
}

// Lstat implements FS.Lstat
func (r *readFS) Lstat(path string) (platform.Stat_t, syscall.Errno) {
	return r.fs.Lstat(path)
}

// Stat implements FS.Stat
func (r *readFS) Stat(path string) (platform.Stat_t, syscall.Errno) {
	return r.fs.Stat(path)
}

// Readlink implements FS.Readlink
func (r *readFS) Readlink(path string) (dst string, err syscall.Errno) {
	return r.fs.Readlink(path)
}

// Mkdir implements FS.Mkdir
func (r *readFS) Mkdir(path string, perm fs.FileMode) syscall.Errno {
	return syscall.EROFS
}

// Chmod implements FS.Chmod
func (r *readFS) Chmod(path string, perm fs.FileMode) syscall.Errno {
	return syscall.EROFS
}

// Chown implements FS.Chown
func (r *readFS) Chown(path string, uid, gid int) syscall.Errno {
	return syscall.EROFS
}

// Lchown implements FS.Lchown
func (r *readFS) Lchown(path string, uid, gid int) syscall.Errno {
	return syscall.EROFS
}

// Rename implements FS.Rename
func (r *readFS) Rename(from, to string) syscall.Errno {
	return syscall.EROFS
}

// Rmdir implements FS.Rmdir
func (r *readFS) Rmdir(path string) syscall.Errno {
	return syscall.EROFS
}

// Link implements FS.Link
func (r *readFS) Link(_, _ string) syscall.Errno {
	return syscall.EROFS
}

// Symlink implements FS.Symlink
func (r *readFS) Symlink(_, _ string) syscall.Errno {
	return syscall.EROFS
}

// Unlink implements FS.Unlink
func (r *readFS) Unlink(path string) syscall.Errno {
	return syscall.EROFS
}

// Utimens implements FS.Utimens
func (r *readFS) Utimens(path string, times *[2]syscall.Timespec, symlinkFollow bool) syscall.Errno {
	return syscall.EROFS
}

// Truncate implements FS.Truncate
func (r *readFS) Truncate(string, int64) syscall.Errno {
	return syscall.EROFS
}
