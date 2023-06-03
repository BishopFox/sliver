package sysfs

import (
	"fmt"
	"io/fs"
	"path"
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
)

// Adapt adapts the input to api.FS unless it is already one. Use NewDirFS instead
// of os.DirFS as it handles interop issues such as windows support.
//
// Note: This performs no flag verification on OpenFile. fsapi.FS cannot read
// flags as there is no parameter to pass them through with. Moreover, fsapi.FS
// documentation does not require the file to be present. In summary, we can't
// enforce flag behavior.
func Adapt(fs fs.FS) fsapi.FS {
	if fs == nil {
		return fsapi.UnimplementedFS{}
	}
	if sys, ok := fs.(fsapi.FS); ok {
		return sys
	}
	return &adapter{fs: fs}
}

type adapter struct {
	fsapi.UnimplementedFS
	fs fs.FS
}

// String implements fmt.Stringer
func (a *adapter) String() string {
	return fmt.Sprintf("%v", a.fs)
}

// OpenFile implements the same method as documented on api.FS
func (a *adapter) OpenFile(path string, flag int, perm fs.FileMode) (fsapi.File, syscall.Errno) {
	return OpenFSFile(a.fs, cleanPath(path), flag, perm)
}

// Stat implements the same method as documented on api.FS
func (a *adapter) Stat(path string) (fsapi.Stat_t, syscall.Errno) {
	f, errno := a.OpenFile(path, syscall.O_RDONLY, 0)
	if errno != 0 {
		return fsapi.Stat_t{}, errno
	}
	defer f.Close()
	return f.Stat()
}

// Lstat implements the same method as documented on api.FS
func (a *adapter) Lstat(path string) (fsapi.Stat_t, syscall.Errno) {
	// At this time, we make the assumption that api.FS instances do not support
	// symbolic links, therefore Lstat is the same as Stat. This is obviously
	// not true but until api.FS has a solid story for how to handle symlinks we
	// are better off not making a decision that would be difficult to revert
	// later on.
	//
	// For further discussions on the topic, see:
	// https://github.com/golang/go/issues/49580
	return a.Stat(path)
}

func cleanPath(name string) string {
	if len(name) == 0 {
		return name
	}
	// fs.ValidFile cannot be rooted (start with '/')
	cleaned := name
	if name[0] == '/' {
		cleaned = name[1:]
	}
	cleaned = path.Clean(cleaned) // e.g. "sub/." -> "sub"
	return cleaned
}
