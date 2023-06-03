// Package sysfs includes a low-level filesystem interface and utilities needed
// for WebAssembly host functions (ABI) such as WASI and runtime.GOOS=js.
//
// The name sysfs was chosen because wazero's public API has a "sys" package,
// which was named after https://github.com/golang/sys.
package sysfs

import (
	"io"
	"io/fs"
	"syscall"

	"github.com/tetratelabs/wazero/internal/platform"
)

// FS is a writeable fs.FS bridge backed by syscall functions needed for ABI
// including WASI and runtime.GOOS=js.
//
// Implementations should embed UnimplementedFS for forward compatability. Any
// unsupported method or parameter should return syscall.ENOSYS.
//
// # Errors
//
// All methods that can return an error return a syscall.Errno, which is zero
// on success.
//
// Restricting to syscall.Errno matches current WebAssembly host functions,
// which are constrained to well-known error codes. For example, `GOOS=js` maps
// hard coded values and panics otherwise. More commonly, WASI maps syscall
// errors to u32 numeric values.
//
// # Notes
//
// A writable filesystem abstraction is not yet implemented as of Go 1.20. See
// https://github.com/golang/go/issues/45757
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
	// file system, and syscall.Errno are returned instead of an os.PathError.
	// A zero syscall.Errno is success.
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
	OpenFile(path string, flag int, perm fs.FileMode) (fs.File, syscall.Errno)
	// ^^ TODO: Consider syscall.Open, though this implies defining and
	// coercing flags and perms similar to what is done in os.OpenFile.

	// Lstat is similar to syscall.Lstat, except the path is relative to this
	// file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.ENOENT: `path` doesn't exist.
	//
	// # Notes
	//
	//   - An fs.FileInfo backed implementation sets atim, mtim and ctim to the
	//     same value.
	//   - When the path is a symbolic link, the stat returned is for the link,
	//     not the file it refers to.
	Lstat(path string) (platform.Stat_t, syscall.Errno)

	// Stat is similar to syscall.Stat, except the path is relative to this
	// file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.ENOENT: `path` doesn't exist.
	//
	// # Notes
	//
	//   - An fs.FileInfo backed implementation sets atim, mtim and ctim to the
	//     same value.
	//   - When the path is a symbolic link, the stat returned is for the file
	//     it refers to.
	Stat(path string) (platform.Stat_t, syscall.Errno)

	// Mkdir is similar to os.Mkdir, except the path is relative to this file
	// system, and syscall.Errno are returned instead of a os.PathError. A zero
	// syscall.Errno is success.
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
	Mkdir(path string, perm fs.FileMode) syscall.Errno
	// ^^ TODO: Consider syscall.Mkdir, though this implies defining and
	// coercing flags and perms similar to what is done in os.Mkdir.

	// Chmod is similar to os.Chmod, except the path is relative to this file
	// system, and syscall.Errno are returned instead of a os.PathError. A zero
	// syscall.Errno is success.
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
	Chmod(path string, perm fs.FileMode) syscall.Errno
	// ^^ TODO: Consider syscall.Chmod, though this implies defining and
	// coercing flags and perms similar to what is done in os.Chmod.

	// Chown is like os.Chown except the path is relative to this file
	// system, and syscall.Errno are returned instead of an os.PathError.
	// A zero syscall.Errno is success.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` does not exist.
	//
	// # Notes
	//
	//   - Windows will always return syscall.ENOSYS
	//   - This is similar to https://linux.die.net/man/3/chown
	Chown(path string, uid, gid int) syscall.Errno

	// Lchown is like os.Lchown except the path is relative to this file
	// system, and syscall.Errno are returned instead of an os.PathError. A
	// zero syscall.Errno is success.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` does not exist.
	//
	// # Notes
	//
	//   - Windows will always return syscall.ENOSYS
	//   - This is similar to https://linux.die.net/man/3/lchown
	Lchown(path string, uid, gid int) syscall.Errno

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
	Rename(from, to string) syscall.Errno

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
	Rmdir(path string) syscall.Errno

	// Unlink is similar to syscall.Unlink, except the path is relative to this
	// file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` doesn't exist.
	//   - syscall.EISDIR: `path` exists, but is a directory.
	//
	// # Notes
	//
	//   - On Windows, syscall.Unlink doesn't delete symlink to directory unlike other platforms. Implementations might
	//     want to combine syscall.RemoveDirectory with syscall.Unlink in order to delete such links on Windows.
	//     See https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-removedirectorya
	Unlink(path string) syscall.Errno

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
	Link(oldPath, newPath string) syscall.Errno

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
	Symlink(oldPath, linkName string) syscall.Errno

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
	Readlink(path string) (string, syscall.Errno)

	// Truncate is similar to syscall.Truncate, except the path is relative to
	// this file system.
	//
	// # Errors
	//
	// The following errors are expected:
	//   - syscall.EINVAL: `path` is invalid or size is negative.
	//   - syscall.ENOENT: `path` doesn't exist
	//   - syscall.EACCES: `path` doesn't have write access.
	Truncate(path string, size int64) syscall.Errno

	// Utimens set file access and modification times on a path relative to
	// this file system, at nanosecond precision.
	//
	// # Parameters
	//
	// The `times` parameter includes the access and modification timestamps to
	// assign. Special syscall.Timespec NSec values UTIME_NOW and UTIME_OMIT may be
	// specified instead of real timestamps. A nil `times` parameter behaves the
	// same as if both were set to UTIME_NOW.
	//
	// When the `symlinkFollow` parameter is true and the path is a symbolic link,
	// the target of expanding that link is updated.
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
	//   - This is similar to syscall.Utimens, except that doesn't have flags to
	//     control expansion of symbolic links. It also doesn't support special
	//     values UTIME_NOW or UTIME_NOW.
	//   - This is like `utimensat` with `AT_FDCWD` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/futimens.html
	Utimens(path string, times *[2]syscall.Timespec, symlinkFollow bool) syscall.Errno
}

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
func FileDatasync(f fs.File) (err syscall.Errno) {
	return platform.Fdatasync(f)
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
