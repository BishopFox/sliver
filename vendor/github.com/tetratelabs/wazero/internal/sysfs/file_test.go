package sysfs

import (
	"embed"
	"io"
	"io/fs"
	"os"
	"path"
	"runtime"
	"testing"
	gofstest "testing/fstest"

	experimentalsys "github.com/tetratelabs/wazero/experimental/sys"
	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/internal/testing/require"
	"github.com/tetratelabs/wazero/sys"
)

//go:embed file_test.go
var embedFS embed.FS

var (
	//go:embed testdata
	testdata   embed.FS
	wazeroFile = "wazero.txt"
	emptyFile  = "empty.txt"
)

func TestStdioFileSetNonblock(t *testing.T) {
	// Test using os.Pipe as it is known to support non-blocking reads.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()

	rF, err := NewStdioFile(true, r)
	require.NoError(t, err)

	errno := rF.SetNonblock(true)
	require.EqualErrno(t, 0, errno)
	require.True(t, rF.IsNonblock())

	errno = rF.SetNonblock(false)
	require.EqualErrno(t, 0, errno)
	require.False(t, rF.IsNonblock())
}

func TestRegularFileSetNonblock(t *testing.T) {
	// Test using os.Pipe as it is known to support non-blocking reads.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()

	rF := newOsFile("", experimentalsys.O_RDONLY, 0, r)

	errno := rF.SetNonblock(true)
	require.EqualErrno(t, 0, errno)
	require.True(t, rF.IsNonblock())

	// Read from the file without ever writing to it should not block.
	buf := make([]byte, 8)
	_, e := rF.Read(buf)
	require.EqualErrno(t, experimentalsys.EAGAIN, e)

	errno = rF.SetNonblock(false)
	require.EqualErrno(t, 0, errno)
	require.False(t, rF.IsNonblock())
}

func TestReadFdNonblock(t *testing.T) {
	// Test using os.Pipe as it is known to support non-blocking reads.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()

	fd := r.Fd()
	errno := setNonblock(fd, true)
	require.EqualErrno(t, 0, errno)

	// Read from the file without ever writing to it should not block.
	buf := make([]byte, 8)
	_, errno = readFd(fd, buf)
	require.EqualErrno(t, experimentalsys.EAGAIN, errno)
}

func TestWriteFdNonblock(t *testing.T) {
	// Test using os.Pipe as it is known to support non-blocking reads.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()

	fd := w.Fd()
	errno := setNonblock(fd, true)

	require.EqualErrno(t, 0, errno)

	// Create a buffer (the content is not relevant)
	buf := make([]byte, 1024)
	// Write to the file until the pipe buffer gets filled up.
	numWrites := 100
	for i := 0; i < numWrites; i++ {
		_, e := writeFd(fd, buf)
		if e != 0 {
			if runtime.GOOS == "windows" {
				// This is currently not supported on Windows.
				require.EqualErrno(t, experimentalsys.ENOSYS, e)
			} else {
				require.EqualErrno(t, experimentalsys.EAGAIN, e)
			}
			return
		}
	}
	t.Fatal("writeFd should return EAGAIN at some point")
}

func TestFileSetAppend(t *testing.T) {
	tmpDir := t.TempDir()

	fPath := path.Join(tmpDir, "file")
	require.NoError(t, os.WriteFile(fPath, []byte("0123456789"), 0o600))

	// Open without APPEND.
	f, errno := OpenOSFile(fPath, experimentalsys.O_RDWR, 0o600)
	require.EqualErrno(t, 0, errno)
	require.False(t, f.IsAppend())

	// Set the APPEND flag.
	require.EqualErrno(t, 0, f.SetAppend(true))
	require.True(t, f.IsAppend())

	requireFileContent := func(exp string) {
		buf, err := os.ReadFile(fPath)
		require.NoError(t, err)
		require.Equal(t, exp, string(buf))
	}

	// with O_APPEND flag, the data is appended to buffer.
	_, errno = f.Write([]byte("wazero"))
	require.EqualErrno(t, 0, errno)
	requireFileContent("0123456789wazero")

	// Remove the APPEND flag.
	require.EqualErrno(t, 0, f.SetAppend(false))
	require.False(t, f.IsAppend())

	_, errno = f.Seek(0, 0)
	require.EqualErrno(t, 0, errno)

	// without O_APPEND flag, the data writes at offset zero
	_, errno = f.Write([]byte("wazero"))
	require.EqualErrno(t, 0, errno)
	requireFileContent("wazero6789wazero")
}

func TestStdioFile_SetAppend(t *testing.T) {
	// SetAppend should not affect Stdio.
	file, err := NewStdioFile(false, os.Stdout)
	require.NoError(t, err)
	errno := file.SetAppend(true)
	require.EqualErrno(t, 0, errno)
	_, errno = file.Write([]byte{})
	require.EqualErrno(t, 0, errno)
}

func TestFileIno(t *testing.T) {
	tmpDir := t.TempDir()
	dirFS, embedFS, mapFS := dirEmbedMapFS(t, tmpDir)

	// get the expected inode
	st, errno := stat(tmpDir)
	require.EqualErrno(t, 0, errno)

	tests := []struct {
		name        string
		fs          fs.FS
		expectedIno sys.Inode
	}{
		{name: "os.DirFS", fs: dirFS, expectedIno: st.Ino},
		{name: "embed.api.FS", fs: embedFS},
		{name: "fstest.MapFS", fs: mapFS},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			d, errno := OpenFSFile(tc.fs, ".", experimentalsys.O_RDONLY, 0)
			require.EqualErrno(t, 0, errno)
			defer d.Close()

			ino, errno := d.Ino()
			require.EqualErrno(t, 0, errno)
			// Results are inconsistent, so don't validate the opposite.
			require.Equal(t, tc.expectedIno, ino)
		})
	}

	t.Run("OS", func(t *testing.T) {
		d, errno := OpenOSFile(tmpDir, experimentalsys.O_RDONLY, 0)
		require.EqualErrno(t, 0, errno)
		defer d.Close()

		ino, errno := d.Ino()
		require.EqualErrno(t, 0, errno)
		// Results are inconsistent, so don't validate the opposite.
		require.Equal(t, st.Ino, ino)
	})
}

func TestFileIsDir(t *testing.T) {
	dirFS, embedFS, mapFS := dirEmbedMapFS(t, t.TempDir())

	tests := []struct {
		name string
		fs   fs.FS
	}{
		{name: "os.DirFS", fs: dirFS},
		{name: "embed.api.FS", fs: embedFS},
		{name: "fstest.MapFS", fs: mapFS},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Run("file", func(t *testing.T) {
				f, errno := OpenFSFile(tc.fs, wazeroFile, experimentalsys.O_RDONLY, 0)
				require.EqualErrno(t, 0, errno)
				defer f.Close()

				isDir, errno := f.IsDir()
				require.EqualErrno(t, 0, errno)
				require.False(t, isDir)
			})

			t.Run("dir", func(t *testing.T) {
				d, errno := OpenFSFile(tc.fs, ".", experimentalsys.O_RDONLY, 0)
				require.EqualErrno(t, 0, errno)
				defer d.Close()

				isDir, errno := d.IsDir()
				require.EqualErrno(t, 0, errno)
				require.True(t, isDir)
			})
		})
	}

	t.Run("OS dir", func(t *testing.T) {
		d, errno := OpenOSFile(t.TempDir(), experimentalsys.O_RDONLY, 0)
		require.EqualErrno(t, 0, errno)
		defer d.Close()

		isDir, errno := d.IsDir()
		require.EqualErrno(t, 0, errno)
		require.True(t, isDir)
	})
}

func TestFileReadAndPread(t *testing.T) {
	dirFS, embedFS, mapFS := dirEmbedMapFS(t, t.TempDir())

	tests := []struct {
		name string
		fs   fs.FS
	}{
		{name: "os.DirFS", fs: dirFS},
		{name: "embed.api.FS", fs: embedFS},
		{name: "fstest.MapFS", fs: mapFS},
	}

	buf := make([]byte, 3)

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			f, errno := OpenFSFile(tc.fs, wazeroFile, experimentalsys.O_RDONLY, 0)
			require.EqualErrno(t, 0, errno)
			defer f.Close()

			// The file should be readable (base case)
			requireRead(t, f, buf)
			require.Equal(t, "waz", string(buf))
			buf = buf[:]

			// We should be able to pread from zero also
			requirePread(t, f, buf, 0)
			require.Equal(t, "waz", string(buf))
			buf = buf[:]

			// If the offset didn't change, read should expect the next three chars.
			requireRead(t, f, buf)
			require.Equal(t, "ero", string(buf))
			buf = buf[:]

			// We should also be able pread from any offset
			requirePread(t, f, buf, 2)
			require.Equal(t, "zer", string(buf))
		})
	}
}

func TestFilePoll_POLLIN(t *testing.T) {
	pflag := fsapi.POLLIN

	// Test using os.Pipe as it is known to support poll.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()

	rF, err := NewStdioFile(true, r)
	require.NoError(t, err)
	buf := make([]byte, 10)
	timeout := int32(0) // return immediately

	// When there's nothing in the pipe, it isn't ready.
	ready, errno := rF.Poll(pflag, timeout)
	require.EqualErrno(t, 0, errno)
	require.False(t, ready)

	// Write to the pipe to make the data available
	expected := []byte("wazero")
	_, err = w.Write([]byte("wazero"))
	require.NoError(t, err)

	// We should now be able to poll ready
	ready, errno = rF.Poll(pflag, timeout)
	require.EqualErrno(t, 0, errno)
	require.True(t, ready)

	// We should now be able to read from the pipe
	n, errno := rF.Read(buf)
	require.EqualErrno(t, 0, errno)
	require.Equal(t, len(expected), n)
	require.Equal(t, expected, buf[:len(expected)])
}

func TestFilePoll_POLLOUT(t *testing.T) {
	pflag := fsapi.POLLOUT

	// Test using os.Pipe as it is known to support poll.
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()

	wF, err := NewStdioFile(false, w)
	require.NoError(t, err)
	timeout := int32(0) // return immediately

	// We don't yet implement write blocking.
	ready, errno := wF.Poll(pflag, timeout)
	require.EqualErrno(t, experimentalsys.ENOTSUP, errno)
	require.False(t, ready)
}

func requireRead(t *testing.T, f experimentalsys.File, buf []byte) {
	n, errno := f.Read(buf)
	require.EqualErrno(t, 0, errno)
	require.Equal(t, len(buf), n)
}

func requirePread(t *testing.T, f experimentalsys.File, buf []byte, off int64) {
	n, errno := f.Pread(buf, off)
	require.EqualErrno(t, 0, errno)
	require.Equal(t, len(buf), n)
}

func TestFileRead_empty(t *testing.T) {
	dirFS, embedFS, mapFS := dirEmbedMapFS(t, t.TempDir())

	tests := []struct {
		name string
		fs   fs.FS
	}{
		{name: "os.DirFS", fs: dirFS},
		{name: "embed.api.FS", fs: embedFS},
		{name: "fstest.MapFS", fs: mapFS},
	}

	buf := make([]byte, 3)

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			f, errno := OpenFSFile(tc.fs, emptyFile, experimentalsys.O_RDONLY, 0)
			require.EqualErrno(t, 0, errno)
			defer f.Close()

			t.Run("Read", func(t *testing.T) {
				// We should be able to read an empty file
				n, errno := f.Read(buf)
				require.EqualErrno(t, 0, errno)
				require.Zero(t, n)
			})

			t.Run("Pread", func(t *testing.T) {
				n, errno := f.Pread(buf, 0)
				require.EqualErrno(t, 0, errno)
				require.Zero(t, n)
			})
		})
	}
}

type maskFS struct {
	fs.FS
}

func (m *maskFS) Open(name string) (fs.File, error) {
	f, err := m.FS.Open(name)
	return struct{ fs.File }{f}, err
}

func TestFilePread_Unsupported(t *testing.T) {
	embedFS, err := fs.Sub(testdata, "testdata")
	require.NoError(t, err)

	f, errno := OpenFSFile(&maskFS{embedFS}, emptyFile, experimentalsys.O_RDONLY, 0)
	require.EqualErrno(t, 0, errno)
	defer f.Close()

	buf := make([]byte, 3)
	_, errno = f.Pread(buf, 0)
	require.EqualErrno(t, experimentalsys.ENOSYS, errno)
}

func TestFileRead_Errors(t *testing.T) {
	// Create the file
	path := path.Join(t.TempDir(), emptyFile)

	// Open the file write-only
	flag := experimentalsys.O_WRONLY | experimentalsys.O_CREAT
	f := requireOpenFile(t, path, flag, 0o600)
	defer f.Close()
	buf := make([]byte, 5)

	tests := []struct {
		name string
		fn   func(experimentalsys.File) experimentalsys.Errno
	}{
		{name: "Read", fn: func(f experimentalsys.File) experimentalsys.Errno {
			_, errno := f.Read(buf)
			return errno
		}},
		{name: "Pread", fn: func(f experimentalsys.File) experimentalsys.Errno {
			_, errno := f.Pread(buf, 0)
			return errno
		}},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Run("EBADF when not open for reading", func(t *testing.T) {
				// The descriptor exists, but not open for reading
				errno := tc.fn(f)
				require.EqualErrno(t, experimentalsys.EBADF, errno)
			})
			testEISDIR(t, tc.fn)
		})
	}
}

func TestFileSeek(t *testing.T) {
	tmpDir := t.TempDir()
	dirFS, embedFS, mapFS := dirEmbedMapFS(t, tmpDir)

	tests := []struct {
		name     string
		openFile func(string) (experimentalsys.File, experimentalsys.Errno)
	}{
		{name: "fsFile os.DirFS", openFile: func(name string) (experimentalsys.File, experimentalsys.Errno) {
			return OpenFSFile(dirFS, name, experimentalsys.O_RDONLY, 0)
		}},
		{name: "fsFile embed.api.FS", openFile: func(name string) (experimentalsys.File, experimentalsys.Errno) {
			return OpenFSFile(embedFS, name, experimentalsys.O_RDONLY, 0)
		}},
		{name: "fsFile fstest.MapFS", openFile: func(name string) (experimentalsys.File, experimentalsys.Errno) {
			return OpenFSFile(mapFS, name, experimentalsys.O_RDONLY, 0)
		}},
		{name: "osFile", openFile: func(name string) (experimentalsys.File, experimentalsys.Errno) {
			return OpenOSFile(path.Join(tmpDir, name), experimentalsys.O_RDONLY, 0o666)
		}},
	}

	buf := make([]byte, 3)

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			f, errno := tc.openFile(wazeroFile)
			require.EqualErrno(t, 0, errno)
			defer f.Close()

			// Shouldn't be able to use an invalid whence
			_, errno = f.Seek(0, io.SeekEnd+1)
			require.EqualErrno(t, experimentalsys.EINVAL, errno)
			_, errno = f.Seek(0, -1)
			require.EqualErrno(t, experimentalsys.EINVAL, errno)

			// Shouldn't be able to seek before the file starts.
			_, errno = f.Seek(-1, io.SeekStart)
			require.EqualErrno(t, experimentalsys.EINVAL, errno)

			requireRead(t, f, buf) // read 3 bytes

			// Seek to the start
			newOffset, errno := f.Seek(0, io.SeekStart)
			require.EqualErrno(t, 0, errno)

			// verify we can re-read from the beginning now.
			require.Zero(t, newOffset)
			requireRead(t, f, buf) // read 3 bytes again
			require.Equal(t, "waz", string(buf))
			buf = buf[:]

			// Seek to the start with zero allows you to read it back.
			newOffset, errno = f.Seek(0, io.SeekCurrent)
			require.EqualErrno(t, 0, errno)
			require.Equal(t, int64(3), newOffset)

			// Seek to the last two bytes
			newOffset, errno = f.Seek(-2, io.SeekEnd)
			require.EqualErrno(t, 0, errno)

			// verify we can read the last two bytes
			require.Equal(t, int64(5), newOffset)
			n, errno := f.Read(buf)
			require.EqualErrno(t, 0, errno)
			require.Equal(t, 2, n)
			require.Equal(t, "o\n", string(buf[:2]))

			t.Run("directory seek to zero", func(t *testing.T) {
				dotF, errno := tc.openFile(".")
				require.EqualErrno(t, 0, errno)
				defer dotF.Close()

				dirents, errno := dotF.Readdir(-1)
				require.EqualErrno(t, 0, errno)
				direntCount := len(dirents)
				require.False(t, direntCount == 0)

				// rewind via seek to zero
				newOffset, errno := dotF.Seek(0, io.SeekStart)
				require.EqualErrno(t, 0, errno)
				require.Zero(t, newOffset)

				// redundantly seek to zero again
				newOffset, errno = dotF.Seek(0, io.SeekStart)
				require.EqualErrno(t, 0, errno)
				require.Zero(t, newOffset)

				// We should be able to read again
				dirents, errno = dotF.Readdir(-1)
				require.EqualErrno(t, 0, errno)
				require.Equal(t, direntCount, len(dirents))
			})

			seekToZero := func(f experimentalsys.File) experimentalsys.Errno {
				_, errno := f.Seek(0, io.SeekStart)
				return errno
			}
			testEBADFIfFileClosed(t, seekToZero)
		})
	}
}

func requireSeek(t *testing.T, f experimentalsys.File, off int64, whence int) int64 {
	n, errno := f.Seek(off, whence)
	require.EqualErrno(t, 0, errno)
	return n
}

func TestFileSeek_empty(t *testing.T) {
	dirFS, embedFS, mapFS := dirEmbedMapFS(t, t.TempDir())

	tests := []struct {
		name string
		fs   fs.FS
	}{
		{name: "os.DirFS", fs: dirFS},
		{name: "embed.api.FS", fs: embedFS},
		{name: "fstest.MapFS", fs: mapFS},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			f, errno := OpenFSFile(tc.fs, emptyFile, experimentalsys.O_RDONLY, 0)
			require.EqualErrno(t, 0, errno)
			defer f.Close()

			t.Run("Start", func(t *testing.T) {
				require.Zero(t, requireSeek(t, f, 0, io.SeekStart))
			})

			t.Run("Current", func(t *testing.T) {
				require.Zero(t, requireSeek(t, f, 0, io.SeekCurrent))
			})

			t.Run("End", func(t *testing.T) {
				require.Zero(t, requireSeek(t, f, 0, io.SeekEnd))
			})
		})
	}
}

func TestFileSeek_Unsupported(t *testing.T) {
	embedFS, err := fs.Sub(testdata, "testdata")
	require.NoError(t, err)

	f, errno := OpenFSFile(&maskFS{embedFS}, emptyFile, experimentalsys.O_RDONLY, 0)
	require.EqualErrno(t, 0, errno)
	defer f.Close()

	_, errno = f.Seek(0, io.SeekCurrent)
	require.EqualErrno(t, experimentalsys.ENOSYS, errno)
}

func TestFileWriteAndPwrite(t *testing.T) {
	// sys.FS doesn't support writes, and there is no other built-in
	// implementation except os.File.
	path := path.Join(t.TempDir(), wazeroFile)
	f := requireOpenFile(t, path, experimentalsys.O_RDWR|experimentalsys.O_CREAT, 0o600)
	defer f.Close()

	text := "wazero"
	buf := make([]byte, 3)
	copy(buf, text[:3])

	// The file should be writeable
	requireWrite(t, f, buf)

	// We should be able to pwrite at gap
	requirePwrite(t, f, buf, 6)

	copy(buf, text[3:])

	// If the offset didn't change, the next chars will write after the
	// first
	requireWrite(t, f, buf)

	// We should be able to pwrite the same bytes as above
	requirePwrite(t, f, buf, 9)

	// We should also be able to pwrite past the above.
	requirePwrite(t, f, buf, 12)

	b, err := os.ReadFile(path)
	require.NoError(t, err)

	// We expect to have written the text two and a half times:
	//  1. Write: (file offset 0) "waz"
	//  2. Pwrite: offset 6 "waz"
	//  3. Write: (file offset 3) "ero"
	//  4. Pwrite: offset 9 "ero"
	//  4. Pwrite: offset 12 "ero"
	require.Equal(t, "wazerowazeroero", string(b))
}

func requireWrite(t *testing.T, f experimentalsys.File, buf []byte) {
	n, errno := f.Write(buf)
	require.EqualErrno(t, 0, errno)
	require.Equal(t, len(buf), n)
}

func requirePwrite(t *testing.T, f experimentalsys.File, buf []byte, off int64) {
	n, errno := f.Pwrite(buf, off)
	require.EqualErrno(t, 0, errno)
	require.Equal(t, len(buf), n)
}

func TestFileWrite_empty(t *testing.T) {
	// sys.FS doesn't support writes, and there is no other built-in
	// implementation except os.File.
	path := path.Join(t.TempDir(), emptyFile)
	f := requireOpenFile(t, path, experimentalsys.O_RDWR|experimentalsys.O_CREAT, 0o600)
	defer f.Close()

	tests := []struct {
		name string
		fn   func(experimentalsys.File, []byte) (int, experimentalsys.Errno)
	}{
		{name: "Write", fn: func(f experimentalsys.File, buf []byte) (int, experimentalsys.Errno) {
			return f.Write(buf)
		}},
		{name: "Pwrite from zero", fn: func(f experimentalsys.File, buf []byte) (int, experimentalsys.Errno) {
			return f.Pwrite(buf, 0)
		}},
		{name: "Pwrite from 3", fn: func(f experimentalsys.File, buf []byte) (int, experimentalsys.Errno) {
			return f.Pwrite(buf, 3)
		}},
	}

	var emptyBuf []byte

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			n, errno := tc.fn(f, emptyBuf)
			require.EqualErrno(t, 0, errno)
			require.Zero(t, n)

			// The file should be empty
			b, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Zero(t, len(b))
		})
	}
}

func TestFileWrite_Unsupported(t *testing.T) {
	embedFS, err := fs.Sub(testdata, "testdata")
	require.NoError(t, err)

	// Use sys.O_RDWR so that it fails due to type not flags
	f, errno := OpenFSFile(&maskFS{embedFS}, wazeroFile, experimentalsys.O_RDWR, 0)
	require.EqualErrno(t, 0, errno)
	defer f.Close()

	tests := []struct {
		name string
		fn   func(experimentalsys.File, []byte) (int, experimentalsys.Errno)
	}{
		{name: "Write", fn: func(f experimentalsys.File, buf []byte) (int, experimentalsys.Errno) {
			return f.Write(buf)
		}},
		{name: "Pwrite", fn: func(f experimentalsys.File, buf []byte) (int, experimentalsys.Errno) {
			return f.Pwrite(buf, 0)
		}},
	}

	buf := []byte("wazero")

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			_, errno := tc.fn(f, buf)
			require.EqualErrno(t, experimentalsys.ENOSYS, errno)
		})
	}
}

func TestFileWrite_Errors(t *testing.T) {
	// Create the file
	path := path.Join(t.TempDir(), emptyFile)
	of, err := os.Create(path)
	require.NoError(t, err)
	require.NoError(t, of.Close())

	// Open the file read-only
	flag := experimentalsys.O_RDONLY
	f := requireOpenFile(t, path, flag, 0o600)
	defer f.Close()
	buf := []byte("wazero")

	tests := []struct {
		name string
		fn   func(experimentalsys.File) experimentalsys.Errno
	}{
		{name: "Write", fn: func(f experimentalsys.File) experimentalsys.Errno {
			_, errno := f.Write(buf)
			return errno
		}},
		{name: "Pwrite", fn: func(f experimentalsys.File) experimentalsys.Errno {
			_, errno := f.Pwrite(buf, 0)
			return errno
		}},
	}

	for _, tc := range tests {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Run("EBADF when not open for writing", func(t *testing.T) {
				// The descriptor exists, but not open for writing
				errno := tc.fn(f)
				require.EqualErrno(t, experimentalsys.EBADF, errno)
			})
			testEISDIR(t, tc.fn)
		})
	}
}

func TestFileSync_NoError(t *testing.T) {
	testSync_NoError(t, experimentalsys.File.Sync)
}

func TestFileDatasync_NoError(t *testing.T) {
	testSync_NoError(t, experimentalsys.File.Datasync)
}

func testSync_NoError(t *testing.T, sync func(experimentalsys.File) experimentalsys.Errno) {
	roPath := "file_test.go"
	ro, errno := OpenFSFile(embedFS, roPath, experimentalsys.O_RDONLY, 0)
	require.EqualErrno(t, 0, errno)
	defer ro.Close()

	rwPath := path.Join(t.TempDir(), "datasync")
	rw, errno := OpenOSFile(rwPath, experimentalsys.O_CREAT|experimentalsys.O_RDWR, 0o600)
	require.EqualErrno(t, 0, errno)
	defer rw.Close()

	tests := []struct {
		name string
		f    experimentalsys.File
	}{
		{name: "UnimplementedFile", f: experimentalsys.UnimplementedFile{}},
		{name: "File of read-only FS.File", f: ro},
		{name: "File of os.File", f: rw},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name, func(t *testing.T) {
			require.EqualErrno(t, 0, sync(tc.f))
		})
	}
}

func TestFileSync(t *testing.T) {
	testSync(t, experimentalsys.File.Sync)
}

func TestFileDatasync(t *testing.T) {
	testSync(t, experimentalsys.File.Datasync)
}

// testSync doesn't guarantee sync works because the operating system may
// sync anyway. There is no test in Go for syscall.Fdatasync, but closest is
// similar to below. Effectively, this only tests that things don't error.
func testSync(t *testing.T, sync func(experimentalsys.File) experimentalsys.Errno) {
	// Even though it is invalid, try to sync a directory
	dPath := t.TempDir()
	d := requireOpenFile(t, dPath, experimentalsys.O_RDONLY, 0)
	defer d.Close()

	errno := sync(d)
	require.EqualErrno(t, 0, errno)

	fPath := path.Join(dPath, t.Name())

	f := requireOpenFile(t, fPath, experimentalsys.O_RDWR|experimentalsys.O_CREAT, 0o600)
	defer f.Close()

	expected := "hello world!"

	// Write the expected data
	_, errno = f.Write([]byte(expected))
	require.EqualErrno(t, 0, errno)

	// Sync the data.
	errno = sync(f)
	require.EqualErrno(t, 0, errno)

	// Rewind while the file is still open.
	_, errno = f.Seek(0, io.SeekStart)
	require.EqualErrno(t, 0, errno)

	// Read data from the file
	buf := make([]byte, 50)
	n, errno := f.Read(buf)
	require.EqualErrno(t, 0, errno)

	// It may be the case that sync worked.
	require.Equal(t, expected, string(buf[:n]))

	// Windows allows you to sync a closed file
	if runtime.GOOS != "windows" {
		testEBADFIfFileClosed(t, sync)
		testEBADFIfDirClosed(t, sync)
	}
}

func TestFileTruncate(t *testing.T) {
	content := []byte("123456")

	tests := []struct {
		name            string
		size            int64
		expectedContent []byte
		expectedErr     error
	}{
		{
			name:            "one less",
			size:            5,
			expectedContent: []byte("12345"),
		},
		{
			name:            "same",
			size:            6,
			expectedContent: content,
		},
		{
			name:            "zero",
			size:            0,
			expectedContent: []byte(""),
		},
		{
			name:            "larger",
			size:            106,
			expectedContent: append(content, make([]byte, 100)...),
		},
	}

	for _, tt := range tests {
		tc := tt
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			fPath := path.Join(tmpDir, tc.name)
			f := openForWrite(t, fPath, content)
			defer f.Close()

			errno := f.Truncate(tc.size)
			require.EqualErrno(t, 0, errno)

			actual, err := os.ReadFile(fPath)
			require.NoError(t, err)
			require.Equal(t, tc.expectedContent, actual)
		})
	}

	truncateToZero := func(f experimentalsys.File) experimentalsys.Errno {
		return f.Truncate(0)
	}

	if runtime.GOOS != "windows" {
		// TODO: os.Truncate on windows passes even when closed
		testEBADFIfFileClosed(t, truncateToZero)
	}

	testEISDIR(t, truncateToZero)

	t.Run("negative", func(t *testing.T) {
		tmpDir := t.TempDir()

		f := openForWrite(t, path.Join(tmpDir, "truncate"), content)
		defer f.Close()

		errno := f.Truncate(-1)
		require.EqualErrno(t, experimentalsys.EINVAL, errno)
	})
}

func TestFileUtimens(t *testing.T) {
	switch runtime.GOOS {
	case "linux", "darwin": // supported
	case "freebsd": // TODO: support freebsd w/o CGO
	case "windows":
	default: // expect ENOSYS and callers need to fall back to Utimens
		t.Skip("unsupported GOOS", runtime.GOOS)
	}

	testUtimens(t, true)

	testEBADFIfFileClosed(t, func(f experimentalsys.File) experimentalsys.Errno {
		return f.Utimens(experimentalsys.UTIME_OMIT, experimentalsys.UTIME_OMIT)
	})
	testEBADFIfDirClosed(t, func(d experimentalsys.File) experimentalsys.Errno {
		return d.Utimens(experimentalsys.UTIME_OMIT, experimentalsys.UTIME_OMIT)
	})
}

func TestNewStdioFile(t *testing.T) {
	// simulate regular file attached to stdin
	f, err := os.CreateTemp(t.TempDir(), "somefile")
	require.NoError(t, err)
	defer f.Close()

	stdin, err := NewStdioFile(true, os.Stdin)
	require.NoError(t, err)
	stdinStat, err := os.Stdin.Stat()
	require.NoError(t, err)

	stdinFile, err := NewStdioFile(true, f)
	require.NoError(t, err)

	stdout, err := NewStdioFile(false, os.Stdout)
	require.NoError(t, err)
	stdoutStat, err := os.Stdout.Stat()
	require.NoError(t, err)

	stdoutFile, err := NewStdioFile(false, f)
	require.NoError(t, err)

	tests := []struct {
		name string
		f    experimentalsys.File
		// Depending on how the tests run, os.Stdin won't necessarily be a char
		// device. We compare against an os.File, to account for this.
		expectedType fs.FileMode
	}{
		{
			name:         "stdin",
			f:            stdin,
			expectedType: stdinStat.Mode().Type(),
		},
		{
			name:         "stdin file",
			f:            stdinFile,
			expectedType: 0, // normal file
		},
		{
			name:         "stdout",
			f:            stdout,
			expectedType: stdoutStat.Mode().Type(),
		},
		{
			name:         "stdout file",
			f:            stdoutFile,
			expectedType: 0, // normal file
		},
	}

	for _, tt := range tests {
		tc := tt

		t.Run(tc.name+" Stat", func(t *testing.T) {
			st, errno := tc.f.Stat()
			require.EqualErrno(t, 0, errno)
			require.Equal(t, tc.expectedType, st.Mode&fs.ModeType)
			require.Equal(t, uint64(1), st.Nlink)

			// Fake times are needed to pass wasi-testsuite.
			// See https://github.com/WebAssembly/wasi-testsuite/blob/af57727/tests/rust/src/bin/fd_filestat_get.rs#L1-L19
			require.Zero(t, st.Ctim)
			require.Zero(t, st.Mtim)
			require.Zero(t, st.Atim)
		})
	}
}

func testEBADFIfDirClosed(t *testing.T, fn func(experimentalsys.File) experimentalsys.Errno) bool {
	return t.Run("EBADF if dir closed", func(t *testing.T) {
		d := requireOpenFile(t, t.TempDir(), experimentalsys.O_RDONLY, 0o755)

		// close the directory underneath
		require.EqualErrno(t, 0, d.Close())

		require.EqualErrno(t, experimentalsys.EBADF, fn(d))
	})
}

func testEBADFIfFileClosed(t *testing.T, fn func(experimentalsys.File) experimentalsys.Errno) bool {
	return t.Run("EBADF if file closed", func(t *testing.T) {
		tmpDir := t.TempDir()

		f := openForWrite(t, path.Join(tmpDir, "EBADF"), []byte{1, 2, 3, 4})

		// close the file underneath
		require.EqualErrno(t, 0, f.Close())

		require.EqualErrno(t, experimentalsys.EBADF, fn(f))
	})
}

func testEISDIR(t *testing.T, fn func(experimentalsys.File) experimentalsys.Errno) bool {
	return t.Run("EISDIR if directory", func(t *testing.T) {
		f := requireOpenFile(t, os.TempDir(), experimentalsys.O_RDONLY|experimentalsys.O_DIRECTORY, 0o666)
		defer f.Close()

		require.EqualErrno(t, experimentalsys.EISDIR, fn(f))
	})
}

func openForWrite(t *testing.T, path string, content []byte) experimentalsys.File {
	require.NoError(t, os.WriteFile(path, content, 0o0666))
	f := requireOpenFile(t, path, experimentalsys.O_RDWR, 0o666)
	_, errno := f.Write(content)
	require.EqualErrno(t, 0, errno)
	return f
}

func requireOpenFile(t *testing.T, path string, flag experimentalsys.Oflag, perm fs.FileMode) experimentalsys.File {
	f, errno := OpenOSFile(path, flag, perm)
	require.EqualErrno(t, 0, errno)
	return f
}

func dirEmbedMapFS(t *testing.T, tmpDir string) (fs.FS, fs.FS, fs.FS) {
	embedFS, err := fs.Sub(testdata, "testdata")
	require.NoError(t, err)

	f, err := embedFS.Open(wazeroFile)
	require.NoError(t, err)
	defer f.Close()

	bytes, err := io.ReadAll(f)
	require.NoError(t, err)

	mapFS := gofstest.MapFS{
		emptyFile:  &gofstest.MapFile{},
		wazeroFile: &gofstest.MapFile{Data: bytes},
	}

	// Write a file as can't open "testdata" in scratch tests because they
	// can't read the original filesystem.
	require.NoError(t, os.WriteFile(path.Join(tmpDir, emptyFile), nil, 0o600))
	require.NoError(t, os.WriteFile(path.Join(tmpDir, wazeroFile), bytes, 0o600))
	dirFS := os.DirFS(tmpDir)
	return dirFS, embedFS, mapFS
}
