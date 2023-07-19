package memfs

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	syspath "path"
	"strings"
	"sync"
	"time"
)

// FS is an in-memory filesystem that implements
// io/fs.FS
type FS struct {
	dir *dir
}

// New creates a new in-memory FileSystem.
func New() *FS {
	return &FS{
		dir: &dir{
			children: make(map[string]childI),
		},
	}
}

// MkdirAll creates a directory named path,
// along with any necessary parents, and returns nil,
// or else returns an error.
// The permission bits perm (before umask) are used for all
// directories that MkdirAll creates.
// If path is already a directory, MkdirAll does nothing
// and returns nil.
func (rootFS *FS) MkdirAll(path string, perm os.FileMode) error {
	if !fs.ValidPath(path) {
		return fmt.Errorf("invalid path: %s: %w", path, fs.ErrInvalid)
	}

	if path == "." {
		// root dir always exists
		return nil
	}

	parts := strings.Split(path, "/")

	next := rootFS.dir
	for _, part := range parts {
		cur := next
		cur.mu.Lock()
		child := cur.children[part]
		if child == nil {
			newDir := &dir{
				name:     part,
				perm:     perm,
				children: make(map[string]childI),
			}
			cur.children[part] = newDir
			next = newDir
		} else {
			childDir, ok := child.(*dir)
			if !ok {
				return fmt.Errorf("not a directory: %s: %w", part, fs.ErrInvalid)
			}
			next = childDir
		}
		cur.mu.Unlock()
	}

	return nil
}

func (rootFS *FS) getDir(path string) (*dir, error) {
	if path == "" {
		return rootFS.dir, nil
	}
	parts := strings.Split(path, "/")

	cur := rootFS.dir
	for _, part := range parts {
		err := func() error {
			cur.mu.Lock()
			defer cur.mu.Unlock()
			child := cur.children[part]
			if child == nil {
				return fmt.Errorf("not a directory: %s: %w", part, fs.ErrNotExist)
			} else {
				childDir, ok := child.(*dir)
				if !ok {
					return fmt.Errorf("no such file or directory: %s: %w", part, fs.ErrNotExist)
				}
				cur = childDir
			}
			return nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return cur, nil
}

func (rootFS *FS) get(path string) (childI, error) {
	if path == "" {
		return rootFS.dir, nil
	}

	parts := strings.Split(path, "/")

	var (
		cur = rootFS.dir

		chld childI
		err  error
	)
	for i, part := range parts {
		chld, err = func() (childI, error) {
			cur.mu.Lock()
			defer cur.mu.Unlock()
			child := cur.children[part]
			if child == nil {
				return nil, fmt.Errorf("not a directory: %s: %w", part, fs.ErrNotExist)
			} else {
				_, isFile := child.(*File)
				if isFile {
					if i == len(parts)-1 {
						return child, nil
					} else {
						return nil, fmt.Errorf("no such file or directory: %s: %w", part, fs.ErrNotExist)
					}
				}

				childDir, ok := child.(*dir)
				if !ok {
					return nil, errors.New("not a directory")
				}
				cur = childDir
			}
			return child, nil
		}()
		if err != nil {
			return nil, err
		}
	}

	return chld, nil
}

func (rootFS *FS) create(path string) (*File, error) {
	if !fs.ValidPath(path) {
		return nil, fmt.Errorf("invalid path: %s: %w", path, fs.ErrInvalid)
	}

	if path == "." {
		// root dir
		path = ""
	}

	dirPart, filePart := syspath.Split(path)

	dirPart = strings.TrimSuffix(dirPart, "/")
	dir, err := rootFS.getDir(dirPart)
	if err != nil {
		return nil, err
	}

	dir.mu.Lock()
	defer dir.mu.Unlock()
	existing := dir.children[filePart]
	if existing != nil {
		_, ok := existing.(*File)
		if !ok {
			return nil, fmt.Errorf("path is a directory: %s: %w", path, fs.ErrExist)
		}
	}

	newFile := &File{
		name:    filePart,
		perm:    0666,
		content: &bytes.Buffer{},
	}
	dir.children[filePart] = newFile

	return newFile, nil
}

// WriteFile writes data to a file named by filename.
// If the file does not exist, WriteFile creates it with permissions perm
// (before umask); otherwise WriteFile truncates it before writing, without changing permissions.
func (rootFS *FS) WriteFile(path string, data []byte, perm os.FileMode) error {
	if !fs.ValidPath(path) {
		return fmt.Errorf("invalid path: %s: %w", path, fs.ErrInvalid)
	}

	if path == "." {
		// root dir
		path = ""
	}

	f, err := rootFS.create(path)
	if err != nil {
		return err
	}
	f.content = bytes.NewBuffer(data)
	f.perm = perm
	return nil
}

// Open opens the named file.
func (rootFS *FS) Open(name string) (fs.File, error) {
	if !fs.ValidPath(name) {
		return nil, &fs.PathError{
			Op:   "open",
			Path: name,
			Err:  fs.ErrInvalid,
		}
	}

	if name == "." {
		// root dir
		name = ""
	}

	child, err := rootFS.get(name)
	if err != nil {
		return nil, err
	}

	switch cc := child.(type) {
	case *File:
		handle := &File{
			name:    cc.name,
			perm:    cc.perm,
			content: bytes.NewBuffer(cc.content.Bytes()),
		}
		return handle, nil
	case *dir:
		handle := &fhDir{
			dir: cc,
		}
		return handle, nil
	}

	return nil, fmt.Errorf("unexpected file type in fs: %s: %w", name, fs.ErrInvalid)
}

// Sub returns an FS corresponding to the subtree rooted at path.
func (rootFS *FS) Sub(path string) (fs.FS, error) {
	dir, err := rootFS.getDir(path)
	if err != nil {
		return nil, err
	}
	return &FS{dir: dir}, nil
}

type dir struct {
	mu       sync.Mutex
	name     string
	perm     os.FileMode
	modTime  time.Time
	children map[string]childI
}

type fhDir struct {
	dir *dir
	idx int
}

func (d *fhDir) Stat() (fs.FileInfo, error) {
	fi := fileInfo{
		name:    d.dir.name,
		size:    4096,
		modTime: d.dir.modTime,
		mode:    d.dir.perm | fs.ModeDir,
	}
	return &fi, nil
}

func (d *fhDir) Read(b []byte) (int, error) {
	return 0, errors.New("is a directory")
}

func (d *fhDir) Close() error {
	return nil
}

func (d *fhDir) ReadDir(n int) ([]fs.DirEntry, error) {
	d.dir.mu.Lock()
	defer d.dir.mu.Unlock()

	names := make([]string, 0, len(d.dir.children))
	for name := range d.dir.children {
		names = append(names, name)
	}

	if n <= 0 {
		n = len(names)
	}

	out := make([]fs.DirEntry, 0, n)

	for i := d.idx; i < n && i < len(names); i++ {
		name := names[i]
		child := d.dir.children[name]

		f, isFile := child.(*File)
		if isFile {
			stat, _ := f.Stat()
			out = append(out, &dirEntry{
				info: stat,
			})
		} else {
			d := child.(*dir)
			fi := fileInfo{
				name:    d.name,
				size:    4096,
				modTime: d.modTime,
				mode:    d.perm | fs.ModeDir,
			}
			out = append(out, &dirEntry{
				info: &fi,
			})
		}

		d.idx = i
	}
	return out, nil
}

type File struct {
	name    string
	perm    os.FileMode
	content *bytes.Buffer
	modTime time.Time
	closed  bool
}

func (f *File) Stat() (fs.FileInfo, error) {
	if f.closed {
		return nil, fs.ErrClosed
	}
	fi := fileInfo{
		name:    f.name,
		size:    int64(f.content.Len()),
		modTime: f.modTime,
		mode:    f.perm,
	}
	return &fi, nil
}

func (f *File) Read(b []byte) (int, error) {
	if f.closed {
		return 0, fs.ErrClosed
	}
	return f.content.Read(b)
}

func (f *File) Close() error {
	if f.closed {
		return fs.ErrClosed
	}
	f.closed = true
	return nil
}

type childI interface {
}

type fileInfo struct {
	name    string
	size    int64
	modTime time.Time
	mode    fs.FileMode
}

// base name of the file
func (fi *fileInfo) Name() string {
	return fi.name
}

// length in bytes for regular files; system-dependent for others
func (fi *fileInfo) Size() int64 {
	return fi.size
}

// file mode bits
func (fi *fileInfo) Mode() fs.FileMode {
	return fi.mode
}

// modification time
func (fi *fileInfo) ModTime() time.Time {
	return fi.modTime
}

// abbreviation for Mode().IsDir()
func (fi *fileInfo) IsDir() bool {
	return fi.mode&fs.ModeDir > 0
}

// underlying data source (can return nil)
func (fi *fileInfo) Sys() interface{} {
	return nil
}

type dirEntry struct {
	info fs.FileInfo
}

func (de *dirEntry) Name() string {
	return de.info.Name()
}

func (de *dirEntry) IsDir() bool {
	return de.info.IsDir()
}

func (de *dirEntry) Type() fs.FileMode {
	return de.info.Mode()
}

func (de *dirEntry) Info() (fs.FileInfo, error) {
	return de.info, nil
}
