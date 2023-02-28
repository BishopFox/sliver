// Package sysfs includes a low-level filesystem interface and utilities needed
// for WebAssembly host functions (ABI) such as WASI and runtime.GOOS=js.
//
// The name sysfs was chosen because wazero's public API has a "sys" package,
// which was named after https://github.com/golang/sys.
package sysfs

import (
	"io"
	"io/fs"
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

// FS is a writeable fs.FS bridge backed by syscall functions needed for ABI
// including WASI and runtime.GOOS=js.
//
// Implementations should embed UnimplementedFS for forward compatability. Any
// unsupported method or parameter should return syscall.ENOSYS.
//
// See https://github.com/golang/go/issues/45757
type FS interface {
	// String should return a human-readable format of the filesystem
	//
	// For example, if this filesystem is backed by the real directory
	// "/tmp/wasm", the expected value is "/tmp/wasm".
	//
	// When the host filesystem isn't a real filesystem, substitute a symbolic,
	// human-readable name. e.g. "virtual"
	String() string

	// OpenFile is similar to os.OpenFile, except the path is relative to this
	// file system, and syscall.Errno are returned instead of a os.PathError.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` or `flag` is invalid.
	//   - syscall.ENOENT: `path` doesn't exist and `flag` doesn't contain
	//     os.O_CREATE.
	//
	// # Constraints on the returned file
	//
	// Implementations that can read flags should enforce them regardless of
	// the type returned. For example, while os.File implements io.Writer,
	// attempts to write to a directory or a file opened with os.O_RDONLY fail
	// with a syscall.EBADF.
	//
	// Some implementations choose whether to enforce read-only opens, namely
	// fs.FS. While fs.FS is supported (Adapt), wazero cannot runtime enforce
	// open flags. Instead, we encourage good behavior and test our built-in
	// implementations.
	//
	// # Notes
	//
	//   - flag are the same as OpenFile, for example, os.O_CREATE.
	//   - Implications of permissions when os.O_CREATE are described in Chmod
	//     notes.
	OpenFile(path string, flag int, perm fs.FileMode) (fs.File, error)
	// ^^ TODO: Consider syscall.Open, though this implies defining and
	// coercing flags and perms similar to what is done in os.OpenFile.

	// Mkdir is similar to os.Mkdir, except the path is relative to this file
	// system, and syscall.Errno are returned instead of a os.PathError.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.EEXIST: `path` exists and is a directory.
	//   - syscall.ENOTDIR: `path` exists and is a file.
	//
	// # Notes
	//
	//   - Implications of permissions are described in Chmod notes.
	Mkdir(path string, perm fs.FileMode) error
	// ^^ TODO: Consider syscall.Mkdir, though this implies defining and
	// coercing flags and perms similar to what is done in os.Mkdir.

	// Chmod is similar to os.Chmod, except the path is relative to this file
	// system, and syscall.Errno are returned instead of a os.PathError.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` does not exist.
	//
	// # Notes
	//
	//   - Windows ignores the execute bit, and any permissions come back as
	//     group and world. For example, chmod of 0400 reads back as 0444, and
	//     0700 0666. Also, permissions on directories aren't supported at all.
	Chmod(path string, perm fs.FileMode) error
	// ^^ TODO: Consider syscall.Chmod, though this implies defining and
	// coercing flags and perms similar to what is done in os.Chmod.

	// Rename is similar to syscall.Rename, except the path is relative to this
	// file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `from` or `to` is invalid.
	//   - syscall.ENOENT: `from` or `to` don't exist.
	//   - syscall.ENOTDIR: `from` is a directory and `to` exists as a file.
	//   - syscall.EISDIR: `from` is a file and `to` exists as a directory.
	//   - syscall.ENOTEMPTY: `both from` and `to` are existing directory, but
	//    `to` is not empty.
	//
	// # Notes
	//
	//   -  Windows doesn't let you overwrite an existing directory.
	Rename(from, to string) error

	// Rmdir is similar to syscall.Rmdir, except the path is relative to this
	// file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` doesn't exist.
	//   - syscall.ENOTDIR: `path` exists, but isn't a directory.
	//   - syscall.ENOTEMPTY: `path` exists, but isn't empty.
	//
	// # Notes
	//
	//   - As of Go 1.19, Windows maps syscall.ENOTDIR to syscall.ENOENT.
	Rmdir(path string) error

	// Unlink is similar to syscall.Unlink, except the path is relative to this
	// file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` doesn't exist.
	//   - syscall.EISDIR: `path` exists, but is a directory.
	Unlink(path string) error

	// Link is similar to syscall.Link, except the path is relative to this
	// file system. This creates "hard" link from oldPath to newPath, in
	// contrast to soft link as in Symlink.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EPERM: `oldPath` is invalid.
	//   - syscall.ENOENT: `oldPath` doesn't exist.
	//   - syscall.EISDIR: `newPath` exists, but is a directory.
	Link(oldPath, newPath string) error

	// Symlink is similar to syscall.Symlink, except the `oldPath` is relative
	// to this file system. This creates "soft" link from oldPath to newPath,
	// in contrast to hard link as in Link.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EPERM: `oldPath` or `newPath` is invalid.
	//   - syscall.EEXIST: `newPath` exists.
	//
	// # Notes
	//
	//   - Only `newPath` is relative to this file system and `oldPath` is kept
	//     as-is. That is because the link is only resolved relative to the
	//     directory when dereferencing it (e.g. ReadLink).
	//     See https://github.com/bytecodealliance/cap-std/blob/v1.0.4/cap-std/src/fs/dir.rs#L404-L409
	//     for how others implement this.
	//   - Symlinks in Windows requires `SeCreateSymbolicLinkPrivilege`.
	//     Otherwise, syscall.EPERM results.
	//     See https://learn.microsoft.com/en-us/windows/security/threat-protection/security-policy-settings/create-symbolic-links
	Symlink(oldPath, linkName string) error

	// Readlink is similar to syscall.Readlink, except the path is relative to
	// this file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//
	// # Notes
	//   - On Windows, the path separator is different from other platforms,
	//     but to provide consistent results to Wasm, this normalizes to a "/"
	//     separator.
	Readlink(path string, buf []byte) (n int, err error)

	// Truncate is similar to syscall.Truncate, except the path is relative to
	// this file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid or size is negative.
	//   - syscall.ENOENT: `path` doesn't exist
	//   - syscall.EACCES: `path` doesn't have write access.
	Truncate(path string, size int64) error

	// Utimes is similar to syscall.UtimesNano, except the path is relative to
	// this file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` doesn't exist
	//
	// # Notes
	//
	//   - To set wall clock time, retrieve it first from sys.Walltime.
	//   - syscall.UtimesNano cannot change the ctime. Also, neither WASI nor
	//     runtime.GOOS=js support changing it. Hence, ctime it is absent here.
	Utimes(path string, atimeNsec, mtimeNsec int64) error
}

// StatPath is a convenience that calls FS.OpenFile, then StatFile, until there
// is a stat method.
func StatPath(fs FS, path string) (s fs.FileInfo, err error) {
	f, err := fs.OpenFile(path, os.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return StatFile(f)
}

// StatFile is like the same method on fs.File, except the returned error is
// nil or syscall.Errno.
func StatFile(f fs.File) (s fs.FileInfo, err error) {
	s, err = f.Stat()
	err = UnwrapOSError(err)
	return
}

// readFile declares all read interfaces defined on os.File used by wazero.
type readFile interface {
	fs.ReadDirFile
	io.ReaderAt // for pread
	io.Seeker   // fallback for ReaderAt for embed:fs
}

// file declares all interfaces defined on os.File used by wazero.
type file interface {
	readFile
	io.Writer
	io.WriterAt // for pwrite
	chmoder
	syncer
	truncater
	fder // for the number of links.
}

// The following interfaces are used until we finalize our own FD-scoped file.
type (
	// chmoder is implemented by os.File in file_posix.go
	chmoder interface{ Chmod(fs.FileMode) error }
	// syncer is implemented by os.File in file_posix.go
	syncer interface{ Sync() error }
	// truncater is implemented by os.File in file_posix.go
	truncater interface{ Truncate(size int64) error }
	// fder is implemented by os.File in file_unix.go and file_windows.go
	fder interface{ Fd() (fd uintptr) }
)

// ReaderAtOffset gets an io.Reader from a fs.File that reads from an offset,
// yet doesn't affect the underlying position. This is used to implement
// syscall.Pread.
//
// Note: The file accessed shouldn't be used concurrently, but wasm isn't safe
// to use concurrently anyway. Hence, we don't do any locking against parallel
// reads.
func ReaderAtOffset(f fs.File, offset int64) io.Reader {
	if ret, ok := f.(io.ReaderAt); ok {
		return &readerAtOffset{ret, offset}
	} else if ret, ok := f.(io.ReadSeeker); ok {
		return &seekToOffsetReader{ret, offset}
	} else {
		return enosysReader{}
	}
}

// FileDatasync is like syscall.Fdatasync except that's only defined in linux.
func FileDatasync(f fs.File) (err error) {
	if fd, ok := f.(fder); ok {
		if err := platform.Fdatasync(fd.Fd()); err != syscall.ENOSYS {
			return err
		}
	}

	// Attempt to sync everything, even if we only need to sync the data.
	if s, ok := f.(syncer); ok {
		err = s.Sync()
	}
	return
}

type enosysReader struct{}

// enosysReader implements io.Reader
func (rs enosysReader) Read([]byte) (n int, err error) {
	return 0, syscall.ENOSYS
}

type readerAtOffset struct {
	r      io.ReaderAt
	offset int64
}

// Read implements io.Reader
func (r *readerAtOffset) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil // less overhead on zero-length reads.
	}

	n, err := r.r.ReadAt(p, r.offset)
	r.offset += int64(n)
	return n, err
}

// seekToOffsetReader implements io.Reader that seeks to an offset and reverts
// to its initial offset after each call to Read.
//
// See /RATIONALE.md "fd_pread: io.Seeker fallback when io.ReaderAt is not supported"
type seekToOffsetReader struct {
	s      io.ReadSeeker
	offset int64
}

// Read implements io.Reader
func (rs *seekToOffsetReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil // less overhead on zero-length reads.
	}

	// Determine the current position in the file, as we need to revert it.
	currentOffset, err := rs.s.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, err
	}

	// Put the read position back when complete.
	defer func() { _, _ = rs.s.Seek(currentOffset, io.SeekStart) }()

	// If the current offset isn't in sync with this reader, move it.
	if rs.offset != currentOffset {
		_, err := rs.s.Seek(rs.offset, io.SeekStart)
		if err != nil {
			return 0, err
		}
	}

	// Perform the read, updating the offset.
	n, err := rs.s.Read(p)
	rs.offset += int64(n)
	return n, err
}

// WriterAtOffset gets an io.Writer from a fs.File that writes to an offset,
// yet doesn't affect the underlying position. This is used to implement
// syscall.Pwrite.
func WriterAtOffset(f fs.File, offset int64) io.Writer {
	if ret, ok := f.(io.WriterAt); ok {
		return &writerAtOffset{ret, offset}
	} else {
		return enosysWriter{}
	}
}

type enosysWriter struct{}

// enosysWriter implements io.Writer
func (rs enosysWriter) Write([]byte) (n int, err error) {
	return 0, syscall.ENOSYS
}

type writerAtOffset struct {
	r      io.WriterAt
	offset int64
}

// Write implements io.Writer
func (r *writerAtOffset) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil // less overhead on zero-length writes.
	}

	n, err := r.r.WriteAt(p, r.offset)
	r.offset += int64(n)
	return n, err
}

// UnwrapOSError returns a syscall.Errno or nil if the input is nil.
func UnwrapOSError(err error) error {
	if err == nil {
		return nil
	}
	err = underlyingError(err)
	if se, ok := err.(syscall.Errno); ok {
		return adjustErrno(se)
	}
	// Below are all the fs.ErrXXX in fs.go.
	//
	// Note: Once we have our own file type, we should never see these.
	switch err {
	case nil:
	case fs.ErrInvalid:
		return syscall.EINVAL
	case fs.ErrPermission:
		return syscall.EPERM
	case fs.ErrExist:
		return syscall.EEXIST
	case fs.ErrNotExist:
		return syscall.ENOENT
	case fs.ErrClosed:
		return syscall.EBADF
	}
	return syscall.EIO
}

// underlyingError returns the underlying error if a well-known OS error type.
//
// This impl is basically the same as os.underlyingError in os/error.go
func underlyingError(err error) error {
	switch err := err.(type) {
	case *os.PathError:
		return err.Err
	case *os.LinkError:
		return err.Err
	case *os.SyscallError:
		return err.Err
	}
	return err
}
