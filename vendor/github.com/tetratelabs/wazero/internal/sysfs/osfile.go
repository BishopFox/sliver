package sysfs

import (
	"io"
	"io/fs"
	"os"
	"runtime"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/sys"
)

func newOsFile(path string, flag experimentalsys.Oflag, perm fs.FileMode, f *os.File) fsapi.File {
	// Windows cannot read files written to a directory after it was opened.
	// This was noticed in #1087 in zig tests. Use a flag instead of a
	// different type.
	reopenDir := runtime.GOOS == "windows"
	return &osFile{path: path, flag: flag, perm: perm, reopenDir: reopenDir, file: f, fd: f.Fd()}
}

// osFile is a file opened with this package, and uses os.File or syscalls to
// implement api.File.
type osFile struct {
	path string
	flag experimentalsys.Oflag
	perm fs.FileMode
	file *os.File
	fd   uintptr

	// reopenDir is true if reopen should be called before Readdir. This flag
	// is deferred until Readdir to prevent redundant rewinds. This could
	// happen if Seek(0) was called twice, or if in Windows, Seek(0) was called
	// before Readdir.
	reopenDir bool

	// closed is true when closed was called. This ensures proper sys.EBADF
	closed bool

	// cachedStat includes fields that won't change while a file is open.
	cachedSt *cachedStat
}

// cachedStat returns the cacheable parts of sys.Stat_t or an error if they
// couldn't be retrieved.
func (f *osFile) cachedStat() (dev uint64, ino sys.Inode, isDir bool, errno experimentalsys.Errno) {
	if f.cachedSt == nil {
		if _, errno = f.Stat(); errno != 0 {
			return
		}
	}
	return f.cachedSt.dev, f.cachedSt.ino, f.cachedSt.isDir, 0
}

// Dev implements the same method as documented on sys.File
func (f *osFile) Dev() (uint64, experimentalsys.Errno) {
	dev, _, _, errno := f.cachedStat()
	return dev, errno
}

// Ino implements the same method as documented on sys.File
func (f *osFile) Ino() (sys.Inode, experimentalsys.Errno) {
	_, ino, _, errno := f.cachedStat()
	return ino, errno
}

// IsDir implements the same method as documented on sys.File
func (f *osFile) IsDir() (bool, experimentalsys.Errno) {
	_, _, isDir, errno := f.cachedStat()
	return isDir, errno
}

// IsAppend implements File.IsAppend
func (f *osFile) IsAppend() bool {
	return f.flag&experimentalsys.O_APPEND == experimentalsys.O_APPEND
}

// SetAppend implements the same method as documented on sys.File
func (f *osFile) SetAppend(enable bool) (errno experimentalsys.Errno) {
	if enable {
		f.flag |= experimentalsys.O_APPEND
	} else {
		f.flag &= ^experimentalsys.O_APPEND
	}

	// Clear any create flag, as we are re-opening, not re-creating.
	f.flag &= ^experimentalsys.O_CREAT

	// appendMode (bool) cannot be changed later, so we have to re-open the
	// file. https://github.com/golang/go/blob/go1.20/src/os/file_unix.go#L60
	return fileError(f, f.closed, f.reopen())
}

// compile-time check to ensure osFile.reopen implements reopenFile.
var _ reopenFile = (*fsFile)(nil).reopen

func (f *osFile) reopen() (errno experimentalsys.Errno) {
	// Clear any create flag, as we are re-opening, not re-creating.
	f.flag &= ^experimentalsys.O_CREAT

	_ = f.close()
	f.file, errno = OpenFile(f.path, f.flag, f.perm)
	return
}

// IsNonblock implements the same method as documented on fsapi.File
func (f *osFile) IsNonblock() bool {
	return isNonblock(f)
}

// SetNonblock implements the same method as documented on fsapi.File
func (f *osFile) SetNonblock(enable bool) (errno experimentalsys.Errno) {
	if enable {
		f.flag |= experimentalsys.O_NONBLOCK
	} else {
		f.flag &= ^experimentalsys.O_NONBLOCK
	}
	if errno = setNonblock(f.fd, enable); errno != 0 {
		return fileError(f, f.closed, errno)
	}
	return 0
}

// Stat implements the same method as documented on sys.File
func (f *osFile) Stat() (sys.Stat_t, experimentalsys.Errno) {
	if f.closed {
		return sys.Stat_t{}, experimentalsys.EBADF
	}

	st, errno := statFile(f.file)
	switch errno {
	case 0:
		f.cachedSt = &cachedStat{dev: st.Dev, ino: st.Ino, isDir: st.Mode&fs.ModeDir == fs.ModeDir}
	case experimentalsys.EIO:
		errno = experimentalsys.EBADF
	}
	return st, errno
}

// Read implements the same method as documented on sys.File
func (f *osFile) Read(buf []byte) (n int, errno experimentalsys.Errno) {
	if len(buf) == 0 {
		return 0, 0 // Short-circuit 0-len reads.
	}
	if nonBlockingFileReadSupported && f.IsNonblock() {
		n, errno = readFd(f.fd, buf)
	} else {
		n, errno = read(f.file, buf)
	}
	if errno != 0 {
		// Defer validation overhead until we've already had an error.
		errno = fileError(f, f.closed, errno)
	}
	return
}

// Pread implements the same method as documented on sys.File
func (f *osFile) Pread(buf []byte, off int64) (n int, errno experimentalsys.Errno) {
	if n, errno = pread(f.file, buf, off); errno != 0 {
		// Defer validation overhead until we've already had an error.
		errno = fileError(f, f.closed, errno)
	}
	return
}

// Seek implements the same method as documented on sys.File
func (f *osFile) Seek(offset int64, whence int) (newOffset int64, errno experimentalsys.Errno) {
	if newOffset, errno = seek(f.file, offset, whence); errno != 0 {
		// Defer validation overhead until we've already had an error.
		errno = fileError(f, f.closed, errno)

		// If the error was trying to rewind a directory, re-open it. Notably,
		// seeking to zero on a directory doesn't work on Windows with Go 1.19.
		if errno == experimentalsys.EISDIR && offset == 0 && whence == io.SeekStart {
			errno = 0
			f.reopenDir = true
		}
	}
	return
}

// Poll implements the same method as documented on fsapi.File
func (f *osFile) Poll(flag fsapi.Pflag, timeoutMillis int32) (ready bool, errno experimentalsys.Errno) {
	return poll(f.fd, flag, timeoutMillis)
}

// Readdir implements File.Readdir. Notably, this uses "Readdir", not
// "ReadDir", from os.File.
func (f *osFile) Readdir(n int) (dirents []experimentalsys.Dirent, errno experimentalsys.Errno) {
	if f.reopenDir { // re-open the directory if needed.
		f.reopenDir = false
		if errno = adjustReaddirErr(f, f.closed, f.reopen()); errno != 0 {
			return
		}
	}

	if dirents, errno = readdir(f.file, f.path, n); errno != 0 {
		errno = adjustReaddirErr(f, f.closed, errno)
	}
	return
}

// Write implements the same method as documented on sys.File
func (f *osFile) Write(buf []byte) (n int, errno experimentalsys.Errno) {
	if len(buf) == 0 {
		return 0, 0 // Short-circuit 0-len writes.
	}
	if nonBlockingFileWriteSupported && f.IsNonblock() {
		n, errno = writeFd(f.fd, buf)
	} else if n, errno = write(f.file, buf); errno != 0 {
		// Defer validation overhead until we've already had an error.
		errno = fileError(f, f.closed, errno)
	}
	return
}

// Pwrite implements the same method as documented on sys.File
func (f *osFile) Pwrite(buf []byte, off int64) (n int, errno experimentalsys.Errno) {
	if n, errno = pwrite(f.file, buf, off); errno != 0 {
		// Defer validation overhead until we've already had an error.
		errno = fileError(f, f.closed, errno)
	}
	return
}

// Truncate implements the same method as documented on sys.File
func (f *osFile) Truncate(size int64) (errno experimentalsys.Errno) {
	if errno = experimentalsys.UnwrapOSError(f.file.Truncate(size)); errno != 0 {
		// Defer validation overhead until we've already had an error.
		errno = fileError(f, f.closed, errno)
	}
	return
}

// Sync implements the same method as documented on sys.File
func (f *osFile) Sync() experimentalsys.Errno {
	return fsync(f.file)
}

// Datasync implements the same method as documented on sys.File
func (f *osFile) Datasync() experimentalsys.Errno {
	return datasync(f.file)
}

// Utimens implements the same method as documented on sys.File
func (f *osFile) Utimens(atim, mtim int64) experimentalsys.Errno {
	if f.closed {
		return experimentalsys.EBADF
	}

	err := futimens(f.fd, atim, mtim)
	return experimentalsys.UnwrapOSError(err)
}

// Close implements the same method as documented on sys.File
func (f *osFile) Close() experimentalsys.Errno {
	if f.closed {
		return 0
	}
	f.closed = true
	return f.close()
}

func (f *osFile) close() experimentalsys.Errno {
	return experimentalsys.UnwrapOSError(f.file.Close())
}
