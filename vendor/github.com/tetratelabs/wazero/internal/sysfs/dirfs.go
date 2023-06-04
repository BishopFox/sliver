package sysfs

import (
	"io/fs"
	"os"
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/internal/platform"
)

func NewDirFS(dir string) fsapi.FS {
	return &dirFS{
		dir:        dir,
		cleanedDir: ensureTrailingPathSeparator(dir),
	}
}

func ensureTrailingPathSeparator(dir string) string {
	if !os.IsPathSeparator(dir[len(dir)-1]) {
		return dir + string(os.PathSeparator)
	}
	return dir
}

type dirFS struct {
	fsapi.UnimplementedFS
	dir string
	// cleanedDir is for easier OS-specific concatenation, as it always has
	// a trailing path separator.
	cleanedDir string
}

// String implements fmt.Stringer
func (d *dirFS) String() string {
	return d.dir
}

// OpenFile implements the same method as documented on api.FS
func (d *dirFS) OpenFile(path string, flag int, perm fs.FileMode) (fsapi.File, syscall.Errno) {
	return OpenOSFile(d.join(path), flag, perm)
}

// Lstat implements the same method as documented on api.FS
func (d *dirFS) Lstat(path string) (fsapi.Stat_t, syscall.Errno) {
	return lstat(d.join(path))
}

// Stat implements the same method as documented on api.FS
func (d *dirFS) Stat(path string) (fsapi.Stat_t, syscall.Errno) {
	return stat(d.join(path))
}

// Mkdir implements the same method as documented on api.FS
func (d *dirFS) Mkdir(path string, perm fs.FileMode) (errno syscall.Errno) {
	err := os.Mkdir(d.join(path), perm)
	if errno = platform.UnwrapOSError(err); errno == syscall.ENOTDIR {
		errno = syscall.ENOENT
	}
	return
}

// Chmod implements the same method as documented on api.FS
func (d *dirFS) Chmod(path string, perm fs.FileMode) syscall.Errno {
	err := os.Chmod(d.join(path), perm)
	return platform.UnwrapOSError(err)
}

// Chown implements the same method as documented on api.FS
func (d *dirFS) Chown(path string, uid, gid int) syscall.Errno {
	return Chown(d.join(path), uid, gid)
}

// Lchown implements the same method as documented on api.FS
func (d *dirFS) Lchown(path string, uid, gid int) syscall.Errno {
	return Lchown(d.join(path), uid, gid)
}

// Rename implements the same method as documented on api.FS
func (d *dirFS) Rename(from, to string) syscall.Errno {
	from, to = d.join(from), d.join(to)
	return Rename(from, to)
}

// Readlink implements the same method as documented on api.FS
func (d *dirFS) Readlink(path string) (string, syscall.Errno) {
	// Note: do not use syscall.Readlink as that causes race on Windows.
	// In any case, syscall.Readlink does almost the same logic as os.Readlink.
	dst, err := os.Readlink(d.join(path))
	if err != nil {
		return "", platform.UnwrapOSError(err)
	}
	return platform.ToPosixPath(dst), 0
}

// Link implements the same method as documented on api.FS
func (d *dirFS) Link(oldName, newName string) syscall.Errno {
	err := os.Link(d.join(oldName), d.join(newName))
	return platform.UnwrapOSError(err)
}

// Rmdir implements the same method as documented on api.FS
func (d *dirFS) Rmdir(path string) syscall.Errno {
	err := syscall.Rmdir(d.join(path))
	return platform.UnwrapOSError(err)
}

// Unlink implements the same method as documented on api.FS
func (d *dirFS) Unlink(path string) (err syscall.Errno) {
	return Unlink(d.join(path))
}

// Symlink implements the same method as documented on api.FS
func (d *dirFS) Symlink(oldName, link string) syscall.Errno {
	// Note: do not resolve `oldName` relative to this dirFS. The link result is always resolved
	// when dereference the `link` on its usage (e.g. readlink, read, etc).
	// https://github.com/bytecodealliance/cap-std/blob/v1.0.4/cap-std/src/fs/dir.rs#L404-L409
	err := os.Symlink(oldName, d.join(link))
	return platform.UnwrapOSError(err)
}

// Utimens implements the same method as documented on api.FS
func (d *dirFS) Utimens(path string, times *[2]syscall.Timespec, symlinkFollow bool) syscall.Errno {
	return Utimens(d.join(path), times, symlinkFollow)
}

// Truncate implements the same method as documented on api.FS
func (d *dirFS) Truncate(path string, size int64) syscall.Errno {
	// Use os.Truncate as syscall.Truncate doesn't exist on Windows.
	err := os.Truncate(d.join(path), size)
	return platform.UnwrapOSError(err)
}

func (d *dirFS) join(path string) string {
	switch path {
	case "", ".", "/":
		if d.cleanedDir == "/" {
			return "/"
		}
		// cleanedDir includes an unnecessary delimiter for the root path.
		return d.cleanedDir[:len(d.cleanedDir)-1]
	}
	// TODO: Enforce similar to safefilepath.FromFS(path), but be careful as
	// relative path inputs are allowed. e.g. dir or path == ../
	return d.cleanedDir + path
}
