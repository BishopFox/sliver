package fsapi

import (
	"io/fs"
	"syscall"
)

// FS is a writeable fs.FS bridge backed by syscall functions needed for ABI
// including WASI and runtime.GOOS=js.
//
// Implementations should embed UnimplementedFS for forward compatability. Any
// unsupported method or parameter should return syscall.ENO
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

	// OpenFile opens a file. It should be closed via Close on File.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EINVAL: `path` or `flag` is invalid.
	//   - syscall.EISDIR: the path was a directory, but flag included
	//     syscall.O_RDWR or syscall.O_WRONLY
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
	//   - This is like os.OpenFile, except the path is relative to this file
	//     system, and syscall.Errno is returned instead of os.PathError.
	//   - flag are the same as os.OpenFile, for example, os.O_CREATE.
	//   - Implications of permissions when os.O_CREATE are described in Chmod
	//     notes.
	//   - This is like `open` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/open.html
	OpenFile(path string, flag int, perm fs.FileMode) (File, syscall.Errno)
	// ^^ TODO: Consider syscall.Open, though this implies defining and
	// coercing flags and perms similar to what is done in os.OpenFile.

	// Lstat gets file status without following symbolic links.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.ENOENT: `path` doesn't exist.
	//
	// # Notes
	//
	//   - This is like syscall.Lstat, except the `path` is relative to this
	//     file system.
	//   - This is like `lstat` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/lstat.html
	//   - An fs.FileInfo backed implementation sets atim, mtim and ctim to the
	//     same value.
	//   - When the path is a symbolic link, the stat returned is for the link,
	//     not the file it refers to.
	Lstat(path string) (Stat_t, syscall.Errno)

	// Stat gets file status.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.ENOENT: `path` doesn't exist.
	//
	// # Notes
	//
	//   - This is like syscall.Stat, except the `path` is relative to this
	//     file system.
	//   - This is like `stat` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/stat.html
	//   - An fs.FileInfo backed implementation sets atim, mtim and ctim to the
	//     same value.
	//   - When the path is a symbolic link, the stat returned is for the file
	//     it refers to.
	Stat(path string) (Stat_t, syscall.Errno)

	// Mkdir makes a directory.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.EEXIST: `path` exists and is a directory.
	//   - syscall.ENOTDIR: `path` exists and is a file.
	//
	// # Notes
	//
	//   - This is like syscall.Mkdir, except the `path` is relative to this
	//     file system.
	//   - This is like `mkdir` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/mkdir.html
	//   - Implications of permissions are described in Chmod notes.
	Mkdir(path string, perm fs.FileMode) syscall.Errno
	// ^^ TODO: Consider syscall.Mkdir, though this implies defining and
	// coercing flags and perms similar to what is done in os.Mkdir.

	// Chmod changes the mode of the file.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` does not exist.
	//
	// # Notes
	//
	//   - This is like syscall.Chmod, except the `path` is relative to this
	//     file system.
	//   - This is like `chmod` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/chmod.html
	//   - Windows ignores the execute bit, and any permissions come back as
	//     group and world. For example, chmod of 0400 reads back as 0444, and
	//     0700 0666. Also, permissions on directories aren't supported at all.
	Chmod(path string, perm fs.FileMode) syscall.Errno

	// Chown changes the owner and group of a file.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` does not exist.
	//
	// # Notes
	//
	//   - This is like syscall.Chown, except the `path` is relative to this
	//     file system.
	//   - This is like `chown` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/chown.html
	//   - This always returns syscall.ENOSYS on windows.
	Chown(path string, uid, gid int) syscall.Errno

	// Lchown changes the owner and group of a symbolic link.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` does not exist.
	//
	// # Notes
	//
	//   - This is like syscall.Lchown, except the `path` is relative to this
	//     file system.
	//   - This is like `lchown` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/lchown.html
	//   - Windows will always return syscall.ENOSYS
	Lchown(path string, uid, gid int) syscall.Errno

	// Rename renames file or directory.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EINVAL: `from` or `to` is invalid.
	//   - syscall.ENOENT: `from` or `to` don't exist.
	//   - syscall.ENOTDIR: `from` is a directory and `to` exists as a file.
	//   - syscall.EISDIR: `from` is a file and `to` exists as a directory.
	//   - syscall.ENOTEMPTY: `both from` and `to` are existing directory, but
	//    `to` is not empty.
	//
	// # Notes
	//
	//   - This is like syscall.Rename, except the paths are relative to this
	//     file system.
	//   - This is like `rename` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/rename.html
	//   -  Windows doesn't let you overwrite an existing directory.
	Rename(from, to string) syscall.Errno

	// Rmdir removes a directory.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` doesn't exist.
	//   - syscall.ENOTDIR: `path` exists, but isn't a directory.
	//   - syscall.ENOTEMPTY: `path` exists, but isn't empty.
	//
	// # Notes
	//
	//   - This is like syscall.Rmdir, except the `path` is relative to this
	//     file system.
	//   - This is like `rmdir` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/rmdir.html
	//   - As of Go 1.19, Windows maps syscall.ENOTDIR to syscall.ENOENT.
	Rmdir(path string) syscall.Errno

	// Unlink removes a directory entry.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.ENOENT: `path` doesn't exist.
	//   - syscall.EISDIR: `path` exists, but is a directory.
	//
	// # Notes
	//
	//   - This is like syscall.Unlink, except the `path` is relative to this
	//     file system.
	//   - This is like `unlink` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/unlink.html
	//   - On Windows, syscall.Unlink doesn't delete symlink to directory unlike other platforms. Implementations might
	//     want to combine syscall.RemoveDirectory with syscall.Unlink in order to delete such links on Windows.
	//     See https://learn.microsoft.com/en-us/windows/win32/api/fileapi/nf-fileapi-removedirectorya
	Unlink(path string) syscall.Errno

	// Link creates a "hard" link from oldPath to newPath, in contrast to a
	// soft link (via Symlink).
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EPERM: `oldPath` is invalid.
	//   - syscall.ENOENT: `oldPath` doesn't exist.
	//   - syscall.EISDIR: `newPath` exists, but is a directory.
	//
	// # Notes
	//
	//   - This is like syscall.Link, except the `oldPath` is relative to this
	//     file system.
	//   - This is like `link` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/link.html
	Link(oldPath, newPath string) syscall.Errno

	// Symlink creates a "soft" link from oldPath to newPath, in contrast to a
	// hard link (via Link).
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EPERM: `oldPath` or `newPath` is invalid.
	//   - syscall.EEXIST: `newPath` exists.
	//
	// # Notes
	//
	//   - This is like syscall.Symlink, except the `oldPath` is relative to
	//     this file system.
	//   - This is like `symlink` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/symlink.html
	//   - Only `newPath` is relative to this file system and `oldPath` is kept
	//     as-is. That is because the link is only resolved relative to the
	//     directory when dereferencing it (e.g. ReadLink).
	//     See https://github.com/bytecodealliance/cap-std/blob/v1.0.4/cap-std/src/fs/dir.rs#L404-L409
	//     for how others implement this.
	//   - Symlinks in Windows requires `SeCreateSymbolicLinkPrivilege`.
	//     Otherwise, syscall.EPERM results.
	//     See https://learn.microsoft.com/en-us/windows/security/threat-protection/security-policy-settings/create-symbolic-links
	Symlink(oldPath, linkName string) syscall.Errno

	// Readlink reads the contents of a symbolic link.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EINVAL: `path` is invalid.
	//
	// # Notes
	//
	//   - This is like syscall.Readlink, except the path is relative to this
	//     filesystem.
	//   - This is like `readlink` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/readlink.html
	//   - On Windows, the path separator is different from other platforms,
	//     but to provide consistent results to Wasm, this normalizes to a "/"
	//     separator.
	Readlink(path string) (string, syscall.Errno)

	// Truncate truncates a file to a specified length.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EINVAL: `path` is invalid or size is negative.
	//   - syscall.ENOENT: `path` doesn't exist.
	//   - syscall.EISDIR: `path` is a directory.
	//   - syscall.EACCES: `path` doesn't have write access.
	//
	// # Notes
	//
	//   - This is like syscall.Truncate, except the path is relative to this
	//     filesystem.
	//   - This is like `truncate` in POSIX. See
	//     https://pubs.opengroup.org/onlinepubs/9699919799/functions/truncate.html
	Truncate(path string, size int64) syscall.Errno

	// Utimens set file access and modification times on a path relative to
	// this file system, at nanosecond precision.
	//
	// # Parameters
	//
	// The `times` parameter includes the access and modification timestamps to
	// assign. Special syscall.Timespec NSec values platform.UTIME_NOW and
	// platform.UTIME_OMIT may be specified instead of real timestamps. A nil
	// `times` parameter behaves the same as if both were set to
	// platform.UTIME_NOW.
	//
	// When the `symlinkFollow` parameter is true and the path is a symbolic link,
	// the target of expanding that link is updated.
	//
	// # Errors
	//
	// A zero syscall.Errno is success. The below are expected otherwise:
	//   - syscall.ENOSYS: the implementation does not support this function.
	//   - syscall.EINVAL: `path` is invalid.
	//   - syscall.EEXIST: `path` exists and is a directory.
	//   - syscall.ENOTDIR: `path` exists and is a file.
	//
	// # Notes
	//
	//   - This is like syscall.UtimesNano and `utimensat` with `AT_FDCWD` in
	//     POSIX. See https://pubs.opengroup.org/onlinepubs/9699919799/functions/futimens.html
	Utimens(path string, times *[2]syscall.Timespec, symlinkFollow bool) syscall.Errno
}
