package sys

import (
	"io"
	"os"
	"syscall"
	"time"

	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/internal/platform"
	"github.com/tetratelabs/wazero/internal/sysfs"
)

// StdinFile is a fs.ModeDevice file for use implementing FdStdin.
// This is safer than reading from os.DevNull as it can never overrun
// operating system file descriptors.
type StdinFile struct {
	noopStdinFile
	io.Reader
}

// Read implements the same method as documented on internalapi.File
func (f *StdinFile) Read(buf []byte) (int, syscall.Errno) {
	n, err := f.Reader.Read(buf)
	return n, platform.UnwrapOSError(err)
}

type writerFile struct {
	noopStdoutFile

	w io.Writer
}

// Write implements the same method as documented on internalapi.File
func (f *writerFile) Write(buf []byte) (int, syscall.Errno) {
	n, err := f.w.Write(buf)
	return n, platform.UnwrapOSError(err)
}

// noopStdinFile is a fs.ModeDevice file for use implementing FdStdin. This is
// safer than reading from os.DevNull as it can never overrun operating system
// file descriptors.
type noopStdinFile struct {
	noopStdioFile
}

// AccessMode implements the same method as documented on internalapi.File
func (noopStdinFile) AccessMode() int {
	return syscall.O_RDONLY
}

// Read implements the same method as documented on internalapi.File
func (noopStdinFile) Read([]byte) (int, syscall.Errno) {
	return 0, 0 // Always EOF
}

// PollRead implements the same method as documented on internalapi.File
func (noopStdinFile) PollRead(*time.Duration) (ready bool, errno syscall.Errno) {
	return true, 0 // always ready to read nothing
}

// noopStdoutFile is a fs.ModeDevice file for use implementing FdStdout and
// FdStderr.
type noopStdoutFile struct {
	noopStdioFile
}

// AccessMode implements the same method as documented on internalapi.File
func (noopStdoutFile) AccessMode() int {
	return syscall.O_WRONLY
}

// Write implements the same method as documented on internalapi.File
func (noopStdoutFile) Write(buf []byte) (int, syscall.Errno) {
	return len(buf), 0 // same as io.Discard
}

type noopStdioFile struct {
	fsapi.UnimplementedFile
}

// Stat implements the same method as documented on internalapi.File
func (noopStdioFile) Stat() (fsapi.Stat_t, syscall.Errno) {
	return fsapi.Stat_t{Mode: modeDevice, Nlink: 1}, 0
}

// IsDir implements the same method as documented on internalapi.File
func (noopStdioFile) IsDir() (bool, syscall.Errno) {
	return false, 0
}

// Close implements the same method as documented on internalapi.File
func (noopStdioFile) Close() (errno syscall.Errno) { return }

func stdinFileEntry(r io.Reader) (*FileEntry, error) {
	if r == nil {
		return &FileEntry{Name: "stdin", IsPreopen: true, File: &noopStdinFile{}}, nil
	} else if f, ok := r.(*os.File); ok {
		if f, err := sysfs.NewStdioFile(true, f); err != nil {
			return nil, err
		} else {
			return &FileEntry{Name: "stdin", IsPreopen: true, File: f}, nil
		}
	} else {
		return &FileEntry{Name: "stdin", IsPreopen: true, File: &StdinFile{Reader: r}}, nil
	}
}

func stdioWriterFileEntry(name string, w io.Writer) (*FileEntry, error) {
	if w == nil {
		return &FileEntry{Name: name, IsPreopen: true, File: &noopStdoutFile{}}, nil
	} else if f, ok := w.(*os.File); ok {
		if f, err := sysfs.NewStdioFile(false, f); err != nil {
			return nil, err
		} else {
			return &FileEntry{Name: name, IsPreopen: true, File: f}, nil
		}
	} else {
		return &FileEntry{Name: name, IsPreopen: true, File: &writerFile{w: w}}, nil
	}
}
