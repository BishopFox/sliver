package sys

import (
	"io"
	"io/fs"
	"net"
	"path"
	"syscall"

	"github.com/tetratelabs/wazero/internal/descriptor"
	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/internal/platform"
	socketapi "github.com/tetratelabs/wazero/internal/sock"
	"github.com/tetratelabs/wazero/internal/sysfs"
)

const (
	FdStdin int32 = iota
	FdStdout
	FdStderr
	// FdPreopen is the file descriptor of the first pre-opened directory.
	//
	// # Why file descriptor 3?
	//
	// While not specified, the most common WASI implementation, wasi-libc,
	// expects POSIX style file descriptor allocation, where the lowest
	// available number is used to open the next file. Since 1 and 2 are taken
	// by stdout and stderr, the next is 3.
	//   - https://github.com/WebAssembly/WASI/issues/122
	//   - https://pubs.opengroup.org/onlinepubs/9699919799/functions/V2_chap02.html#tag_15_14
	//   - https://github.com/WebAssembly/wasi-libc/blob/wasi-sdk-16/libc-bottom-half/sources/preopens.c#L215
	FdPreopen
)

const modeDevice = fs.ModeDevice | 0o640

// FileEntry maps a path to an open file in a file system.
type FileEntry struct {
	// Name is the name of the directory up to its pre-open, or the pre-open
	// name itself when IsPreopen.
	//
	// # Notes
	//
	//   - This can drift on rename.
	//   - This relates to the guest path, which is not the real file path
	//     except if the entire host filesystem was made available.
	Name string

	// IsPreopen is a directory that is lazily opened.
	IsPreopen bool

	// FS is the filesystem associated with the pre-open.
	FS fsapi.FS

	// File is always non-nil.
	File fsapi.File
}

const direntBufSize = 16

// Readdir is the status of a prior fs.ReadDirFile call.
type Readdir struct {
	// cursor is the current position in the buffer.
	cursor uint64

	// countRead is the total count of files read including Dirents.
	//
	// Notes:
	//
	// * countRead is the index of the next file in the list. This is
	//   also the value that Cookie returns, so it should always be
	//   higher or equal than the cookie given in Rewind.
	//
	// * this can overflow to negative, which means our implementation
	//   doesn't support writing greater than max int64 entries.
	//   countRead uint64
	countRead uint64

	// dirents is a fixed buffer of size direntBufSize. Notably,
	// directory listing are not rewindable, so we keep entries around in case
	// the caller mis-estimated their buffer and needs a few still cached.
	//
	// Note: This is wasi-specific and needs to be refactored.
	// In wasi preview1, dot and dot-dot entries are required to exist, but the
	// reverse is true for preview2. More importantly, preview2 holds separate
	// stateful dir-entry-streams per file.
	dirents []fsapi.Dirent

	// dirInit seeks and reset the provider for dirents to the beginning
	// and returns an initial batch (e.g. dot directories).
	dirInit func() ([]fsapi.Dirent, syscall.Errno)

	// dirReader fetches a new batch of direntBufSize elements.
	dirReader func(n uint64) ([]fsapi.Dirent, syscall.Errno)
}

// NewReaddir is a constructor for Readdir. It takes a dirInit
func NewReaddir(
	dirInit func() ([]fsapi.Dirent, syscall.Errno),
	dirReader func(n uint64) ([]fsapi.Dirent, syscall.Errno),
) (*Readdir, syscall.Errno) {
	d := &Readdir{dirReader: dirReader, dirInit: dirInit}
	return d, d.init()
}

// init resets the cursor and invokes the dirInit, dirReader
// methods to reset the internal state of the Readdir struct.
//
// Note: this is different from Reset, because it will not short-circuit
// when cursor is already 0, but it will force an unconditional reload.
func (d *Readdir) init() syscall.Errno {
	d.cursor = 0
	d.countRead = 0
	// Reset the buffer to the initial state.
	initialDirents, errno := d.dirInit()
	if errno != 0 {
		return errno
	}
	if len(initialDirents) > direntBufSize {
		return syscall.EINVAL
	}
	d.dirents = initialDirents
	// Fill the buffer with more data.
	count := direntBufSize - len(initialDirents)
	if count == 0 {
		// No need to fill up the buffer further.
		return 0
	}
	dirents, errno := d.dirReader(uint64(count))
	if errno != 0 {
		return errno
	}
	d.dirents = append(d.dirents, dirents...)
	return 0
}

// newReaddirFromFileEntry is a constructor for Readdir that takes a FileEntry to initialize.
func newReaddirFromFileEntry(f *FileEntry) (*Readdir, syscall.Errno) {
	// Generate the dotEntries only once and return it many times in the dirInit closure.
	dotEntries, errno := synthesizeDotEntries(f)
	if errno != 0 {
		return nil, errno
	}
	dirInit := func() ([]fsapi.Dirent, syscall.Errno) {
		// Ensure we always rewind to the beginning when we re-init.
		if _, errno := f.File.Seek(0, io.SeekStart); errno != 0 {
			return nil, errno
		}
		// Return the dotEntries that we have already generated outside the closure.
		return dotEntries, 0
	}
	dirReader := func(n uint64) ([]fsapi.Dirent, syscall.Errno) { return f.File.Readdir(int(n)) }
	return NewReaddir(dirInit, dirReader)
}

// synthesizeDotEntries generates a slice of the two elements "." and "..".
func synthesizeDotEntries(f *FileEntry) (result []fsapi.Dirent, errno syscall.Errno) {
	dotIno, errno := f.File.Ino()
	if errno != 0 {
		return nil, errno
	}
	result = append(result, fsapi.Dirent{Name: ".", Ino: dotIno, Type: fs.ModeDir})
	dotDotIno := uint64(0)
	if !f.IsPreopen && f.Name != "." {
		if st, errno := f.FS.Stat(path.Dir(f.Name)); errno != 0 {
			return nil, errno
		} else {
			dotDotIno = st.Ino
		}
	}
	result = append(result, fsapi.Dirent{Name: "..", Ino: dotDotIno, Type: fs.ModeDir})
	return result, 0
}

// Reset seeks the internal cursor to 0 and refills the buffer.
func (d *Readdir) Reset() syscall.Errno {
	if d.countRead == 0 {
		return 0
	}
	return d.init()
}

// Skip is equivalent to calling n times Advance.
func (d *Readdir) Skip(n uint64) {
	end := d.countRead + n
	var err syscall.Errno = 0
	for d.countRead < end && err == 0 {
		err = d.Advance()
	}
}

// Cookie returns a cookie representing the current state of the ReadDir struct.
//
// Note: this returns the countRead field, but it is an implementation detail.
func (d *Readdir) Cookie() uint64 {
	return d.countRead
}

// Rewind seeks the internal cursor to the state represented by the cookie.
// It returns a syscall.Errno if the cursor was reset and an I/O error occurred while trying to re-init.
func (d *Readdir) Rewind(cookie int64) syscall.Errno {
	unsignedCookie := uint64(cookie)
	switch {
	case cookie < 0 || unsignedCookie > d.countRead:
		// the cookie can neither be negative nor can it be larger than countRead.
		return syscall.EINVAL
	case cookie == 0 && d.countRead == 0:
		return 0
	case cookie == 0 && d.countRead != 0:
		// This means that there was a previous call to the dir, but cookie is reset.
		// This happens when the program calls rewinddir, for example:
		// https://github.com/WebAssembly/wasi-libc/blob/659ff414560721b1660a19685110e484a081c3d4/libc-bottom-half/cloudlibc/src/libc/dirent/rewinddir.c#L10-L12
		return d.Reset()
	case unsignedCookie < d.countRead:
		if cookie/direntBufSize != int64(d.countRead)/direntBufSize {
			// The cookie is not 0, but it points into a window before the current one.
			return syscall.ENOSYS
		}
		// We are allowed to rewind back to a previous offset within the current window.
		d.countRead = unsignedCookie
		d.cursor = d.countRead % direntBufSize
		return 0
	default:
		// The cookie is valid.
		return 0
	}
}

// Peek emits the current value.
// It returns syscall.ENOENT when there are no entries left in the directory.
func (d *Readdir) Peek() (*fsapi.Dirent, syscall.Errno) {
	switch {
	case d.cursor == uint64(len(d.dirents)):
		// We're past the buf size, fill it up again.
		dirents, errno := d.dirReader(direntBufSize)
		if errno != 0 {
			return nil, errno
		}
		d.dirents = append(d.dirents, dirents...)
		fallthrough
	default: // d.cursor < direntBufSize FIXME
		if d.cursor == uint64(len(d.dirents)) {
			return nil, syscall.ENOENT
		}
		dirent := &d.dirents[d.cursor]
		return dirent, 0
	}
}

// Advance advances the internal counters and indices to the next value.
// It also empties and refill the buffer with the next set of values when the internal cursor
// reaches the end of it.
func (d *Readdir) Advance() syscall.Errno {
	if d.cursor == uint64(len(d.dirents)) {
		return syscall.ENOENT
	}
	d.cursor++
	d.countRead++
	return 0
}

type FSContext struct {
	// rootFS is the root ("/") mount.
	rootFS fsapi.FS

	// openedFiles is a map of file descriptor numbers (>=FdPreopen) to open files
	// (or directories) and defaults to empty.
	// TODO: This is unguarded, so not goroutine-safe!
	openedFiles FileTable

	// readdirs is a map of numeric identifiers to Readdir structs
	// and defaults to empty.
	// TODO: This is unguarded, so not goroutine-safe!
	readdirs ReaddirTable
}

// FileTable is a specialization of the descriptor.Table type used to map file
// descriptors to file entries.
type FileTable = descriptor.Table[int32, *FileEntry]

// ReaddirTable is a specialization of the descriptor.Table type used to map file
// descriptors to Readdir structs.
type ReaddirTable = descriptor.Table[int32, *Readdir]

// RootFS returns the underlying filesystem. Any files that should be added to
// the table should be inserted via InsertFile.
func (c *FSContext) RootFS() fsapi.FS {
	return c.rootFS
}

// OpenFile opens the file into the table and returns its file descriptor.
// The result must be closed by CloseFile or Close.
func (c *FSContext) OpenFile(fs fsapi.FS, path string, flag int, perm fs.FileMode) (int32, syscall.Errno) {
	if f, errno := fs.OpenFile(path, flag, perm); errno != 0 {
		return 0, errno
	} else {
		fe := &FileEntry{FS: fs, File: f}
		if path == "/" || path == "." {
			fe.Name = ""
		} else {
			fe.Name = path
		}
		if newFD, ok := c.openedFiles.Insert(fe); !ok {
			return 0, syscall.EBADF
		} else {
			return newFD, 0
		}
	}
}

// SockAccept accepts a socketapi.TCPConn into the file table and returns
// its file descriptor.
func (c *FSContext) SockAccept(sockFD int32, nonblock bool) (int32, syscall.Errno) {
	var sock socketapi.TCPSock
	if e, ok := c.LookupFile(sockFD); !ok || !e.IsPreopen {
		return 0, syscall.EBADF // Not a preopen
	} else if sock, ok = e.File.(socketapi.TCPSock); !ok {
		return 0, syscall.EBADF // Not a sock
	}

	var conn socketapi.TCPConn
	var errno syscall.Errno
	if conn, errno = sock.Accept(); errno != 0 {
		return 0, errno
	} else if nonblock {
		if errno = conn.SetNonblock(true); errno != 0 {
			_ = conn.Close()
			return 0, errno
		}
	}

	fe := &FileEntry{File: conn}
	if newFD, ok := c.openedFiles.Insert(fe); !ok {
		return 0, syscall.EBADF
	} else {
		return newFD, 0
	}
}

// LookupFile returns a file if it is in the table.
func (c *FSContext) LookupFile(fd int32) (*FileEntry, bool) {
	return c.openedFiles.Lookup(fd)
}

// LookupReaddir returns a Readdir struct or creates an empty one if it was not present.
//
// Note: this currently assumes that idx == fd, where fd is the file descriptor of the directory.
// CloseFile will delete this idx from the internal store. In the future, idx may be independent
// of a file fd, and the idx may have to be disposed with an explicit CloseReaddir.
func (c *FSContext) LookupReaddir(idx int32, f *FileEntry) (*Readdir, syscall.Errno) {
	if item, _ := c.readdirs.Lookup(idx); item != nil {
		return item, 0
	} else {
		item, err := newReaddirFromFileEntry(f)
		if err != 0 {
			return nil, err
		}
		ok := c.readdirs.InsertAt(item, idx)
		if !ok {
			return nil, syscall.EINVAL
		}
		return item, 0
	}
}

// CloseReaddir delete the Readdir struct at the given index
//
// Note: Currently only necessary in tests. In the future, the idx will have to be disposed explicitly,
// unless we maintain a map fd -> []idx, and we let CloseFile close all the idx in []idx.
func (c *FSContext) CloseReaddir(idx int32) {
	c.readdirs.Delete(idx)
}

// Renumber assigns the file pointed by the descriptor `from` to `to`.
func (c *FSContext) Renumber(from, to int32) syscall.Errno {
	fromFile, ok := c.openedFiles.Lookup(from)
	if !ok || to < 0 {
		return syscall.EBADF
	} else if fromFile.IsPreopen {
		return syscall.ENOTSUP
	}

	// If toFile is already open, we close it to prevent windows lock issues.
	//
	// The doc is unclear and other implementations do nothing for already-opened To FDs.
	// https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_renumberfd-fd-to-fd---errno
	// https://github.com/bytecodealliance/wasmtime/blob/main/crates/wasi-common/src/snapshots/preview_1.rs#L531-L546
	if toFile, ok := c.openedFiles.Lookup(to); ok {
		if toFile.IsPreopen {
			return syscall.ENOTSUP
		}
		_ = toFile.File.Close()
	}

	c.openedFiles.Delete(from)
	if !c.openedFiles.InsertAt(fromFile, to) {
		return syscall.EBADF
	}
	return 0
}

// CloseFile returns any error closing the existing file.
func (c *FSContext) CloseFile(fd int32) syscall.Errno {
	f, ok := c.openedFiles.Lookup(fd)
	if !ok {
		return syscall.EBADF
	}
	c.openedFiles.Delete(fd)
	c.readdirs.Delete(fd)
	return platform.UnwrapOSError(f.File.Close())
}

// Close implements io.Closer
func (c *FSContext) Close() (err error) {
	// Close any files opened in this context
	c.openedFiles.Range(func(fd int32, entry *FileEntry) bool {
		if errno := entry.File.Close(); errno != 0 {
			err = errno // This means err returned == the last non-nil error.
		}
		return true
	})
	// A closed FSContext cannot be reused so clear the state instead of
	// using Reset.
	c.openedFiles = FileTable{}
	c.readdirs = ReaddirTable{}
	return
}

// NewFSContext creates a FSContext with stdio streams and an optional
// pre-opened filesystem.
//
// If `preopened` is not UnimplementedFS, it is inserted into
// the file descriptor table as FdPreopen.
func (c *Context) NewFSContext(
	stdin io.Reader,
	stdout, stderr io.Writer,
	rootFS fsapi.FS,
	tcpListeners []*net.TCPListener,
) (err error) {
	c.fsc.rootFS = rootFS
	inFile, err := stdinFileEntry(stdin)
	if err != nil {
		return err
	}
	c.fsc.openedFiles.Insert(inFile)
	outWriter, err := stdioWriterFileEntry("stdout", stdout)
	if err != nil {
		return err
	}
	c.fsc.openedFiles.Insert(outWriter)
	errWriter, err := stdioWriterFileEntry("stderr", stderr)
	if err != nil {
		return err
	}
	c.fsc.openedFiles.Insert(errWriter)

	if _, ok := rootFS.(fsapi.UnimplementedFS); ok {
		// don't add to the pre-opens
	} else if comp, ok := rootFS.(*sysfs.CompositeFS); ok {
		preopens := comp.FS()
		for i, p := range comp.GuestPaths() {
			c.fsc.openedFiles.Insert(&FileEntry{
				FS:        preopens[i],
				Name:      p,
				IsPreopen: true,
				File:      &lazyDir{fs: rootFS},
			})
		}
	} else {
		c.fsc.openedFiles.Insert(&FileEntry{
			FS:        rootFS,
			Name:      "/",
			IsPreopen: true,
			File:      &lazyDir{fs: rootFS},
		})
	}

	for _, tl := range tcpListeners {
		c.fsc.openedFiles.Insert(&FileEntry{IsPreopen: true, File: sysfs.NewTCPListenerFile(tl)})
	}
	return nil
}
