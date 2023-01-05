package syscallfs

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"syscall"
)

func NewDirFS(dir string) (FS, error) {
	if stat, err := os.Stat(dir); err != nil {
		return nil, syscall.ENOENT
	} else if !stat.IsDir() {
		return nil, syscall.ENOTDIR
	}
	return dirFS(dir), nil
}

// dirFS currently validates each path, which means that input paths cannot
// escape the directory, except via symlink. We may want to relax this in the
// future, especially as we decoupled from fs.FS which has this requirement.
type dirFS string

// Open implements the same method as documented on fs.FS
func (dir dirFS) Open(name string) (fs.File, error) {
	panic(fmt.Errorf("unexpected to call fs.FS.Open(%s)", name))
}

// OpenFile implements FS.OpenFile
func (dir dirFS) OpenFile(name string, flag int, perm fs.FileMode) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}
	return os.OpenFile(path.Join(string(dir), name), flag, perm)
}

// Mkdir implements FS.Mkdir
func (dir dirFS) Mkdir(name string, perm fs.FileMode) error {
	if !fs.ValidPath(name) {
		return &fs.PathError{Op: "mkdir", Path: name, Err: fs.ErrInvalid}
	}

	err := os.Mkdir(path.Join(string(dir), name), perm)

	return adjustMkdirError(err)
}

// Rename implements FS.Rename
func (dir dirFS) Rename(from, to string) error {
	if !fs.ValidPath(from) {
		return syscall.EINVAL
	}
	if !fs.ValidPath(to) {
		return syscall.EINVAL
	}
	if from == to {
		return nil
	}
	return rename(path.Join(string(dir), from), path.Join(string(dir), to))
}

// Rmdir implements FS.Rmdir
func (dir dirFS) Rmdir(name string) error {
	if !fs.ValidPath(name) {
		return syscall.EINVAL
	}

	err := syscall.Rmdir(path.Join(string(dir), name))

	return adjustRmdirError(err)
}

// Unlink implements FS.Unlink
func (dir dirFS) Unlink(name string) error {
	if !fs.ValidPath(name) {
		return syscall.EINVAL
	}

	err := syscall.Unlink(path.Join(string(dir), name))

	return adjustUnlinkError(err)
}

// Utimes implements FS.Utimes
func (dir dirFS) Utimes(name string, atimeNsec, mtimeNsec int64) error {
	if !fs.ValidPath(name) {
		return syscall.EINVAL
	}
	return syscall.UtimesNano(path.Join(string(dir), name), []syscall.Timespec{
		syscall.NsecToTimespec(atimeNsec),
		syscall.NsecToTimespec(mtimeNsec),
	})
}
