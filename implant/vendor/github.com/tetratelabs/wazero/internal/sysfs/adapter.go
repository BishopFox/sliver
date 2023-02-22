package sysfs

import (
	"fmt"
	"io/fs"
	"os"
	pathutil "path"
	"runtime"
	"strings"
)

// Adapt adapts the input to FS unless it is already one. NewDirFS should be
// used instead, if the input is os.DirFS.
//
// Note: This performs no flag verification on FS.OpenFile. fs.FS cannot read
// flags as there is no parameter to pass them through with. Moreover, fs.FS
// documentation does not require the file to be present. In summary, we can't
// enforce flag behavior.
func Adapt(fs fs.FS) FS {
	if sys, ok := fs.(FS); ok {
		return sys
	}
	return &adapter{fs: fs}
}

type adapter struct {
	UnimplementedFS
	fs fs.FS
}

// String implements fmt.Stringer
func (a *adapter) String() string {
	return fmt.Sprintf("%v", a.fs)
}

// Open implements the same method as documented on fs.FS
func (a *adapter) Open(name string) (fs.File, error) {
	return a.fs.Open(name)
}

// OpenFile implements FS.OpenFile
func (a *adapter) OpenFile(path string, flag int, perm fs.FileMode) (fs.File, error) {
	path = cleanPath(path)
	f, err := a.fs.Open(path)

	if err != nil {
		return nil, UnwrapOSError(err)
	} else if osF, ok := f.(*os.File); ok {
		// If this is an OS file, it has same portability issues as dirFS.
		return maybeWrapFile(osF, a, path, flag, perm), nil
	}
	return f, nil
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
	cleaned = pathutil.Clean(cleaned) // e.g. "sub/." -> "sub"
	return cleaned
}

// fsOpen implements the Open method as documented on fs.FS
func fsOpen(f FS, name string) (fs.File, error) {
	if !fs.ValidPath(name) { // FS.OpenFile has fewer constraints than fs.FS
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
	}

	// This isn't a production-grade fs.FS implementation. The only special
	// cases we address here are to pass testfs.TestFS.

	if runtime.GOOS == "windows" {
		switch {
		case strings.Contains(name, "\\"):
			return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrInvalid}
		}
	}

	if f, err := f.OpenFile(name, os.O_RDONLY, 0); err != nil {
		return nil, &fs.PathError{Op: "open", Path: name, Err: err}
	} else {
		return f, nil
	}
}
