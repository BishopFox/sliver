package sysfs

import (
	"fmt"
	"io"
	"io/fs"
	"strings"
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
)

func NewRootFS(fs []fsapi.FS, guestPaths []string) (fsapi.FS, error) {
	switch len(fs) {
	case 0:
		return fsapi.UnimplementedFS{}, nil
	case 1:
		if StripPrefixesAndTrailingSlash(guestPaths[0]) == "" {
			return fs[0], nil
		}
	}

	ret := &CompositeFS{
		string:            stringFS(fs, guestPaths),
		fs:                make([]fsapi.FS, len(fs)),
		guestPaths:        make([]string, len(fs)),
		cleanedGuestPaths: make([]string, len(fs)),
		rootGuestPaths:    map[string]int{},
		rootIndex:         -1,
	}

	copy(ret.guestPaths, guestPaths)
	copy(ret.fs, fs)

	for i, guestPath := range guestPaths {
		// Clean the prefix in the same way path matches will.
		cleaned := StripPrefixesAndTrailingSlash(guestPath)
		if cleaned == "" {
			if ret.rootIndex != -1 {
				return nil, fmt.Errorf("multiple root filesystems are invalid: %s", ret.string)
			}
			ret.rootIndex = i
		} else if strings.HasPrefix(cleaned, "..") {
			// ../ mounts are special cased and aren't returned in a directory
			// listing, so we can ignore them for now.
		} else if strings.Contains(cleaned, "/") {
			return nil, fmt.Errorf("only single-level guest paths allowed: %s", ret.string)
		} else {
			ret.rootGuestPaths[cleaned] = i
		}
		ret.cleanedGuestPaths[i] = cleaned
	}

	// Ensure there is always a root match to keep runtime logic simpler.
	if ret.rootIndex == -1 {
		ret.rootIndex = len(fs)
		ret.cleanedGuestPaths = append(ret.cleanedGuestPaths, "")
		ret.fs = append(ret.fs, &fakeRootFS{})
	}
	return ret, nil
}

type CompositeFS struct {
	fsapi.UnimplementedFS
	// string is cached for convenience.
	string string
	// fs is index-correlated with cleanedGuestPaths
	fs []fsapi.FS
	// guestPaths are the original paths supplied by the end user, cleaned as
	// cleanedGuestPaths.
	guestPaths []string
	// cleanedGuestPaths to match in precedence order, ascending.
	cleanedGuestPaths []string
	// rootGuestPaths are cleanedGuestPaths that exist directly under root, such as
	// "tmp".
	rootGuestPaths map[string]int
	// rootIndex is the index in fs that is the root filesystem
	rootIndex int
}

// String implements fmt.Stringer
func (c *CompositeFS) String() string {
	return c.string
}

func stringFS(fs []fsapi.FS, guestPaths []string) string {
	var ret strings.Builder
	ret.WriteString("[")
	writeMount(&ret, fs[0], guestPaths[0])
	for i, f := range fs[1:] {
		ret.WriteString(" ")
		writeMount(&ret, f, guestPaths[i+1])
	}
	ret.WriteString("]")
	return ret.String()
}

func writeMount(ret *strings.Builder, f fsapi.FS, guestPath string) {
	ret.WriteString(f.String())
	ret.WriteString(":")
	ret.WriteString(guestPath)
	if _, ok := f.(*readFS); ok {
		ret.WriteString(":ro")
	}
}

// GuestPaths returns the underlying pre-open paths in original order.
func (c *CompositeFS) GuestPaths() (guestPaths []string) {
	return c.guestPaths
}

// FS returns the underlying filesystems in original order.
func (c *CompositeFS) FS() (fs []fsapi.FS) {
	fs = make([]fsapi.FS, len(c.guestPaths))
	copy(fs, c.fs)
	return
}

// OpenFile implements the same method as documented on api.FS
func (c *CompositeFS) OpenFile(path string, flag int, perm fs.FileMode) (f fsapi.File, err syscall.Errno) {
	matchIndex, relativePath := c.chooseFS(path)

	f, err = c.fs[matchIndex].OpenFile(relativePath, flag, perm)
	if err != 0 {
		return
	}

	// Ensure the root directory listing includes any prefix mounts.
	if matchIndex == c.rootIndex {
		switch path {
		case ".", "/", "":
			if len(c.rootGuestPaths) > 0 {
				f = &openRootDir{path: path, c: c, f: f}
			}
		}
	}
	return
}

// An openRootDir is a root directory open for reading, which has mounts inside
// of it.
type openRootDir struct {
	fsapi.DirFile

	path     string
	c        *CompositeFS
	f        fsapi.File     // the directory file itself
	dirents  []fsapi.Dirent // the directory contents
	direntsI int            // the read offset, an index into the files slice
}

// Ino implements the same method as documented on fsapi.File
func (d *openRootDir) Ino() (uint64, syscall.Errno) {
	return d.f.Ino()
}

// Stat implements the same method as documented on fsapi.File
func (d *openRootDir) Stat() (fsapi.Stat_t, syscall.Errno) {
	return d.f.Stat()
}

// Seek implements the same method as documented on fsapi.File
func (d *openRootDir) Seek(offset int64, whence int) (newOffset int64, errno syscall.Errno) {
	if offset != 0 || whence != io.SeekStart {
		errno = syscall.ENOSYS
		return
	}
	d.dirents = nil
	d.direntsI = 0
	return d.f.Seek(offset, whence)
}

// Readdir implements the same method as documented on fsapi.File
func (d *openRootDir) Readdir(count int) (dirents []fsapi.Dirent, errno syscall.Errno) {
	if d.dirents == nil {
		if errno = d.readdir(); errno != 0 {
			return
		}
	}

	// logic similar to go:embed
	n := len(d.dirents) - d.direntsI
	if n == 0 {
		return
	}
	if count > 0 && n > count {
		n = count
	}
	dirents = make([]fsapi.Dirent, n)
	for i := range dirents {
		dirents[i] = d.dirents[d.direntsI+i]
	}
	d.direntsI += n
	return
}

func (d *openRootDir) readdir() (errno syscall.Errno) {
	// readDir reads the directory fully into d.dirents, replacing any entries that
	// correspond to prefix matches or appending them to the end.
	if d.dirents, errno = d.f.Readdir(-1); errno != 0 {
		return
	}

	remaining := make(map[string]int, len(d.c.rootGuestPaths))
	for k, v := range d.c.rootGuestPaths {
		remaining[k] = v
	}

	for i := range d.dirents {
		e := d.dirents[i]
		if fsI, ok := remaining[e.Name]; ok {
			if d.dirents[i], errno = d.rootEntry(e.Name, fsI); errno != 0 {
				return
			}
			delete(remaining, e.Name)
		}
	}

	var di fsapi.Dirent
	for n, fsI := range remaining {
		if di, errno = d.rootEntry(n, fsI); errno != 0 {
			return
		}
		d.dirents = append(d.dirents, di)
	}
	return
}

// Sync implements the same method as documented on fsapi.File
func (d *openRootDir) Sync() syscall.Errno {
	return d.f.Sync()
}

// Datasync implements the same method as documented on fsapi.File
func (d *openRootDir) Datasync() syscall.Errno {
	return d.f.Datasync()
}

// Chmod implements the same method as documented on fsapi.File
func (d *openRootDir) Chmod(fs.FileMode) syscall.Errno {
	return syscall.ENOSYS
}

// Chown implements the same method as documented on fsapi.File
func (d *openRootDir) Chown(int, int) syscall.Errno {
	return syscall.ENOSYS
}

// Utimens implements the same method as documented on fsapi.File
func (d *openRootDir) Utimens(*[2]syscall.Timespec) syscall.Errno {
	return syscall.ENOSYS
}

// Close implements fs.File
func (d *openRootDir) Close() syscall.Errno {
	return d.f.Close()
}

func (d *openRootDir) rootEntry(name string, fsI int) (fsapi.Dirent, syscall.Errno) {
	if st, errno := d.c.fs[fsI].Stat("."); errno != 0 {
		return fsapi.Dirent{}, errno
	} else {
		return fsapi.Dirent{Name: name, Ino: st.Ino, Type: st.Mode.Type()}, 0
	}
}

// Lstat implements the same method as documented on api.FS
func (c *CompositeFS) Lstat(path string) (fsapi.Stat_t, syscall.Errno) {
	matchIndex, relativePath := c.chooseFS(path)
	return c.fs[matchIndex].Lstat(relativePath)
}

// Stat implements the same method as documented on api.FS
func (c *CompositeFS) Stat(path string) (fsapi.Stat_t, syscall.Errno) {
	matchIndex, relativePath := c.chooseFS(path)
	return c.fs[matchIndex].Stat(relativePath)
}

// Mkdir implements the same method as documented on api.FS
func (c *CompositeFS) Mkdir(path string, perm fs.FileMode) syscall.Errno {
	matchIndex, relativePath := c.chooseFS(path)
	return c.fs[matchIndex].Mkdir(relativePath, perm)
}

// Chmod implements the same method as documented on api.FS
func (c *CompositeFS) Chmod(path string, perm fs.FileMode) syscall.Errno {
	matchIndex, relativePath := c.chooseFS(path)
	return c.fs[matchIndex].Chmod(relativePath, perm)
}

// Chown implements the same method as documented on api.FS
func (c *CompositeFS) Chown(path string, uid, gid int) syscall.Errno {
	matchIndex, relativePath := c.chooseFS(path)
	return c.fs[matchIndex].Chown(relativePath, uid, gid)
}

// Lchown implements the same method as documented on api.FS
func (c *CompositeFS) Lchown(path string, uid, gid int) syscall.Errno {
	matchIndex, relativePath := c.chooseFS(path)
	return c.fs[matchIndex].Lchown(relativePath, uid, gid)
}

// Rename implements the same method as documented on api.FS
func (c *CompositeFS) Rename(from, to string) syscall.Errno {
	fromFS, fromPath := c.chooseFS(from)
	toFS, toPath := c.chooseFS(to)
	if fromFS != toFS {
		return syscall.ENOSYS // not yet anyway
	}
	return c.fs[fromFS].Rename(fromPath, toPath)
}

// Readlink implements the same method as documented on api.FS
func (c *CompositeFS) Readlink(path string) (string, syscall.Errno) {
	matchIndex, relativePath := c.chooseFS(path)
	return c.fs[matchIndex].Readlink(relativePath)
}

// Link implements the same method as documented on api.FS
func (c *CompositeFS) Link(oldName, newName string) syscall.Errno {
	fromFS, oldNamePath := c.chooseFS(oldName)
	toFS, newNamePath := c.chooseFS(newName)
	if fromFS != toFS {
		return syscall.ENOSYS // not yet anyway
	}
	return c.fs[fromFS].Link(oldNamePath, newNamePath)
}

// Utimens implements the same method as documented on api.FS
func (c *CompositeFS) Utimens(path string, times *[2]syscall.Timespec, symlinkFollow bool) syscall.Errno {
	matchIndex, relativePath := c.chooseFS(path)
	return c.fs[matchIndex].Utimens(relativePath, times, symlinkFollow)
}

// Symlink implements the same method as documented on api.FS
func (c *CompositeFS) Symlink(oldName, link string) (err syscall.Errno) {
	fromFS, oldNamePath := c.chooseFS(oldName)
	toFS, linkPath := c.chooseFS(link)
	if fromFS != toFS {
		return syscall.ENOSYS // not yet anyway
	}
	return c.fs[fromFS].Symlink(oldNamePath, linkPath)
}

// Truncate implements the same method as documented on api.FS
func (c *CompositeFS) Truncate(path string, size int64) syscall.Errno {
	matchIndex, relativePath := c.chooseFS(path)
	return c.fs[matchIndex].Truncate(relativePath, size)
}

// Rmdir implements the same method as documented on api.FS
func (c *CompositeFS) Rmdir(path string) syscall.Errno {
	matchIndex, relativePath := c.chooseFS(path)
	return c.fs[matchIndex].Rmdir(relativePath)
}

// Unlink implements the same method as documented on api.FS
func (c *CompositeFS) Unlink(path string) syscall.Errno {
	matchIndex, relativePath := c.chooseFS(path)
	return c.fs[matchIndex].Unlink(relativePath)
}

// chooseFS chooses the best fs and the relative path to use for the input.
func (c *CompositeFS) chooseFS(path string) (matchIndex int, relativePath string) {
	matchIndex = -1
	matchPrefixLen := 0
	pathI, pathLen := stripPrefixesAndTrailingSlash(path)

	// Last is the highest precedence, so we iterate backwards. The last longest
	// match wins. e.g. the pre-open "tmp" wins vs "" regardless of order.
	for i := len(c.fs) - 1; i >= 0; i-- {
		prefix := c.cleanedGuestPaths[i]
		if eq, match := hasPathPrefix(path, pathI, pathLen, prefix); eq {
			// When the input equals the prefix, there cannot be a longer match
			// later. The relative path is the fsapi.FS root, so return empty
			// string.
			matchIndex = i
			relativePath = ""
			return
		} else if match {
			// Check to see if this is a longer match
			prefixLen := len(prefix)
			if prefixLen > matchPrefixLen || matchIndex == -1 {
				matchIndex = i
				matchPrefixLen = prefixLen
			}
		} // Otherwise, keep looking for a match
	}

	// Now, we know the path != prefix, but it matched an existing fs, because
	// setup ensures there's always a root filesystem.

	// If this was a root path match the cleaned path is the relative one to
	// pass to the underlying filesystem.
	if matchPrefixLen == 0 {
		// Avoid re-slicing when the input is already clean
		if pathI == 0 && len(path) == pathLen {
			relativePath = path
		} else {
			relativePath = path[pathI:pathLen]
		}
		return
	}

	// Otherwise, it is non-root match: the relative path is past "$prefix/"
	pathI += matchPrefixLen + 1 // e.g. prefix=foo, path=foo/bar -> bar
	relativePath = path[pathI:pathLen]
	return
}

// hasPathPrefix compares an input path against a prefix, both cleaned by
// stripPrefixesAndTrailingSlash. This returns a pair of eq, match to allow an
// early short circuit on match.
//
// Note: This is case-sensitive because POSIX paths are compared case
// sensitively.
func hasPathPrefix(path string, pathI, pathLen int, prefix string) (eq, match bool) {
	matchLen := pathLen - pathI
	if prefix == "" {
		return matchLen == 0, true // e.g. prefix=, path=foo
	}

	prefixLen := len(prefix)
	// reset pathLen temporarily to represent the length to match as opposed to
	// the length of the string (that may contain leading slashes).
	if matchLen == prefixLen {
		if pathContainsPrefix(path, pathI, prefixLen, prefix) {
			return true, true // e.g. prefix=bar, path=bar
		}
		return false, false
	} else if matchLen < prefixLen {
		return false, false // e.g. prefix=fooo, path=foo
	}

	if path[pathI+prefixLen] != '/' {
		return false, false // e.g. prefix=foo, path=fooo
	}

	// Not equal, but maybe a match. e.g. prefix=foo, path=foo/bar
	return false, pathContainsPrefix(path, pathI, prefixLen, prefix)
}

// pathContainsPrefix is faster than strings.HasPrefix even if we didn't cache
// the index,len. See benchmarks.
func pathContainsPrefix(path string, pathI, prefixLen int, prefix string) bool {
	for i := 0; i < prefixLen; i++ {
		if path[pathI] != prefix[i] {
			return false // e.g. prefix=bar, path=foo or foo/bar
		}
		pathI++
	}
	return true // e.g. prefix=foo, path=foo or foo/bar
}

func StripPrefixesAndTrailingSlash(path string) string {
	pathI, pathLen := stripPrefixesAndTrailingSlash(path)
	return path[pathI:pathLen]
}

// stripPrefixesAndTrailingSlash skips any leading "./" or "/" such that the
// result index begins with another string. A result of "." coerces to the
// empty string "" because the current directory is handled by the guest.
//
// Results are the offset/len pair which is an optimization to avoid re-slicing
// overhead, as this function is called for every path operation.
//
// Note: Relative paths should be handled by the guest, as that's what knows
// what the current directory is. However, paths that escape the current
// directory e.g. "../.." have been found in `tinygo test` and this
// implementation takes care to avoid it.
func stripPrefixesAndTrailingSlash(path string) (pathI, pathLen int) {
	// strip trailing slashes
	pathLen = len(path)
	for ; pathLen > 0 && path[pathLen-1] == '/'; pathLen-- {
	}

	pathI = 0
loop:
	for pathI < pathLen {
		switch path[pathI] {
		case '/':
			pathI++
		case '.':
			nextI := pathI + 1
			if nextI < pathLen && path[nextI] == '/' {
				pathI = nextI + 1
			} else if nextI == pathLen {
				pathI = nextI
			} else {
				break loop
			}
		default:
			break loop
		}
	}
	return
}

type fakeRootFS struct {
	fsapi.UnimplementedFS
}

// OpenFile implements the same method as documented on api.FS
func (fakeRootFS) OpenFile(path string, flag int, perm fs.FileMode) (fsapi.File, syscall.Errno) {
	switch path {
	case ".", "/", "":
		return fakeRootDir{}, 0
	}
	return nil, syscall.ENOENT
}

type fakeRootDir struct {
	fsapi.DirFile
}

// Ino implements the same method as documented on fsapi.File
func (fakeRootDir) Ino() (uint64, syscall.Errno) {
	return 0, 0
}

// Stat implements the same method as documented on fsapi.File
func (fakeRootDir) Stat() (fsapi.Stat_t, syscall.Errno) {
	return fsapi.Stat_t{Mode: fs.ModeDir, Nlink: 1}, 0
}

// Readdir implements the same method as documented on fsapi.File
func (fakeRootDir) Readdir(int) (dirents []fsapi.Dirent, errno syscall.Errno) {
	return // empty
}

// Sync implements the same method as documented on fsapi.File
func (fakeRootDir) Sync() syscall.Errno {
	return 0
}

// Datasync implements the same method as documented on fsapi.File
func (fakeRootDir) Datasync() syscall.Errno {
	return 0
}

// Chmod implements the same method as documented on fsapi.File
func (fakeRootDir) Chmod(fs.FileMode) syscall.Errno {
	return syscall.ENOSYS
}

// Chown implements the same method as documented on fsapi.File
func (fakeRootDir) Chown(int, int) syscall.Errno {
	return syscall.ENOSYS
}

// Utimens implements the same method as documented on fsapi.File
func (fakeRootDir) Utimens(*[2]syscall.Timespec) syscall.Errno {
	return syscall.ENOSYS
}

// Close implements the same method as documented on fsapi.File
func (fakeRootDir) Close() syscall.Errno {
	return 0
}
