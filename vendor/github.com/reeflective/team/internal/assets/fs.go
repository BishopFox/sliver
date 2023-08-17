package assets

/*
   team - Embedded teamserver for Go programs and CLI applications
   Copyright (C) 2023 Reeflective

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU General Public License as published by
   the Free Software Foundation, either version 3 of the License, or
   (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU General Public License for more details.

   You should have received a copy of the GNU General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/psanford/memfs"
)

const (
	// FileReadPerm is the permission bit given to the OS when reading files.
	FileReadPerm = 0o600
	// DirPerm is the permission bit given to teamserver/client directories.
	DirPerm = 0o700
	// FileWritePerm is the permission bit given to the OS when writing files.
	FileWritePerm = 0o644

	// FileWriteOpenMode is used when opening log files in append/create/write-only mode.
	FileWriteOpenMode = os.O_APPEND | os.O_CREATE | os.O_WRONLY
)

const (
	// Teamclient.

	// DirClient is the name of the teamclient subdirectory.
	DirClient = "teamclient"
	// DirLogs subdirectory name.
	DirLogs = "logs"
	// DirConfigs subdirectory name.
	DirConfigs = "configs"

	// Teamserver.

	// DirServer is the name of the teamserver subdirectory.
	DirServer = "teamserver"
	// DirCerts subdirectory name.
	DirCerts = "certs"
)

// FS is a filesystem abstraction for teamservers and teamclients.
// When either of them are configured to run in memory only, this
// filesystem is initialized accordingly, otherwise it will forward
// its calls to the on-disk filesystem.
//
// This type currently exists because the stdlib io/fs.FS type is read-only,
// and that in order to provide a unique abstraction to the teamclient/server
// filesystems, this filesystem type adds writing methods.
type FS struct {
	mem  *memfs.FS
	root string
}

// NewFileSystem returns a new filesystem configured to run on disk or in-memory.
func NewFileSystem(root string, inMemory bool) *FS {
	filesystem := &FS{
		root: root,
	}

	if inMemory {
		filesystem.mem = memfs.New()
	}

	return filesystem
}

// MkdirAll creates a directory named path, along with any necessary parents,
// and returns nil, or else returns an error.
// The permission bits perm (before umask) are used for all directories that MkdirAll creates.
// If path is already a directory, MkdirAll does nothing and returns nil.
//
// If the filesystem is in-memory, the teamclient/server application root
// is trimmed from this path, if the latter contains it.
func (f *FS) MkdirAll(path string, perm fs.FileMode) error {
	if f.mem == nil {
		return os.MkdirAll(path, perm)
	}

	path = strings.TrimPrefix(path, f.root)

	return f.mem.MkdirAll(path, perm)
}

// Sub returns a file system (an fs.FS) for the tree of files rooted at the directory dir,
// or an error if it failed. When the teamclient fs is on disk, os.Stat() and os.DirFS() are used.
//
// If the filesystem is in-memory, the teamclient/server application root
// is trimmed from this path, if the latter contains it.
func (f *FS) Sub(path string) (fs fs.FS, err error) {
	if f.mem == nil {
		_, err = os.Stat(path)

		return os.DirFS(path), err
	}

	path = strings.TrimPrefix(path, f.root)

	return f.mem.Sub(path)
}

// Open is like fs.Open().
//
// If the filesystem is in-memory, the teamclient/server application root
// is trimmed from this path, if the latter contains it.
func (f *FS) Open(name string) (fs.File, error) {
	if f.mem == nil {
		return os.Open(name)
	}

	name = strings.TrimPrefix(name, f.root)

	return f.mem.Open(name)
}

// OpenFile is like os.OpenFile(), but returns a custom *File type implementing
// the io.WriteCloser interface, so that it can be written to and closed more easily.
func (f *FS) OpenFile(name string, flag int, perm fs.FileMode) (*File, error) {
	inFile := &File{
		name: name,
	}

	if f.mem != nil {
		inFile.mem = f.mem

		return inFile, nil
	}

	file, err := os.OpenFile(name, flag, perm)
	if err != nil {
		return nil, err
	}

	inFile.file = file

	return inFile, nil
}

// WriteFile is like os.WriteFile().
func (f *FS) WriteFile(path string, data []byte, perm fs.FileMode) error {
	if f.mem == nil {
		return os.WriteFile(path, data, perm)
	}

	path = strings.TrimPrefix(path, f.root)

	return f.mem.WriteFile(path, data, perm)
}

// ReadFile is like os.ReadFile().
func (f *FS) ReadFile(path string) (b []byte, err error) {
	if f.mem == nil {
		return os.ReadFile(path)
	}

	_, err = f.mem.Open(path)
	if err != nil {
		return
	}

	return fs.ReadFile(f.mem, path)
}

// File wraps the *os.File type with some in-memory helpers,
// so that we can write/read to teamserver application files
// regardless of where they are.
// This should disappear if a Write() method set is added to the io/fs package.
type File struct {
	name string
	file *os.File
	mem  *memfs.FS
}

// Write implements the io.Writer interface by writing data either
// to the file on disk, or to an in-memory file.
func (f *File) Write(data []byte) (written int, err error) {
	if f.file != nil {
		return f.file.Write(data)
	}

	fileName := filepath.Base(f.name)

	return len(data), f.mem.WriteFile(fileName, data, FileWritePerm)
}

// Close implements io.Closer by closing the file on the filesystem.
func (f *File) Close() error {
	if f.file != nil {
		return f.file.Close()
	}

	return nil
}
