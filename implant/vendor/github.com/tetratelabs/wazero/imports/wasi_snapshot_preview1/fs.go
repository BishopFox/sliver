package wasi_snapshot_preview1

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"math"
	"os"
	"path"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/platform"
	"github.com/tetratelabs/wazero/internal/sys"
	. "github.com/tetratelabs/wazero/internal/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// fdAdvise is the WASI function named FdAdviseName which provides file
// advisory information on a file descriptor.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_advisefd-fd-offset-filesize-len-filesize-advice-advice---errno
var fdAdvise = stubFunction(
	FdAdviseName,
	[]wasm.ValueType{i32, i64, i64, i32},
	"fd", "offset", "len", "advice",
)

// fdAllocate is the WASI function named FdAllocateName which forces the
// allocation of space in a file.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_allocatefd-fd-offset-filesize-len-filesize---errno
var fdAllocate = stubFunction(
	FdAllocateName,
	[]wasm.ValueType{i32, i64, i64},
	"fd", "offset", "len",
)

// fdClose is the WASI function named FdCloseName which closes a file
// descriptor.
//
// # Parameters
//
//   - fd: file descriptor to close
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: the fd was not open.
//
// Note: This is similar to `close` in POSIX.
// See https://github.com/WebAssembly/WASI/blob/main/phases/snapshot/docs.md#fd_close
// and https://linux.die.net/man/3/close
var fdClose = newHostFunc(FdCloseName, fdCloseFn, []api.ValueType{i32}, "fd")

func fdCloseFn(_ context.Context, mod api.Module, params []uint64) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()
	fd := uint32(params[0])

	if ok := fsc.CloseFile(fd); !ok {
		return ErrnoBadf
	}
	return ErrnoSuccess
}

// fdDatasync is the WASI function named FdDatasyncName which synchronizes
// the data of a file to disk.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_datasyncfd-fd---errno
var fdDatasync = stubFunction(FdDatasyncName, []api.ValueType{i32}, "fd")

// fdFdstatGet is the WASI function named FdFdstatGetName which returns the
// attributes of a file descriptor.
//
// # Parameters
//
//   - fd: file descriptor to get the fdstat attributes data
//   - resultFdstat: offset to write the result fdstat data
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid
//   - ErrnoFault: `resultFdstat` points to an offset out of memory
//
// fdstat byte layout is 24-byte size, with the following fields:
//   - fs_filetype 1 byte: the file type
//   - fs_flags 2 bytes: the file descriptor flag
//   - 5 pad bytes
//   - fs_right_base 8 bytes: ignored as rights were removed from WASI.
//   - fs_right_inheriting 8 bytes: ignored as rights were removed from WASI.
//
// For example, with a file corresponding with `fd` was a directory (=3) opened
// with `fd_read` right (=1) and no fs_flags (=0), parameter resultFdstat=1,
// this function writes the below to api.Memory:
//
//	                uint16le   padding            uint64le                uint64le
//	       uint8 --+  +--+  +-----------+  +--------------------+  +--------------------+
//	               |  |  |  |           |  |                    |  |                    |
//	     []byte{?, 3, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0, 0, 0}
//	resultFdstat --^  ^-- fs_flags         ^-- fs_right_base       ^-- fs_right_inheriting
//	               |
//	               +-- fs_filetype
//
// Note: fdFdstatGet returns similar flags to `fsync(fd, F_GETFL)` in POSIX, as
// well as additional fields.
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#fdstat
// and https://linux.die.net/man/3/fsync
var fdFdstatGet = newHostFunc(FdFdstatGetName, fdFdstatGetFn, []api.ValueType{i32, i32}, "fd", "result.stat")

// fdFdstatGetFn cannot currently use proxyResultParams because fdstat is larger
// than api.ValueTypeI64 (i64 == 8 bytes, but fdstat is 24).
func fdFdstatGetFn(_ context.Context, mod api.Module, params []uint64) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd, resultFdstat := uint32(params[0]), uint32(params[1])

	// Ensure we can write the fdstat
	buf, ok := mod.Memory().Read(resultFdstat, 24)
	if !ok {
		return ErrnoFault
	}

	stat, err := fsc.StatFile(fd)
	if err != nil {
		return ToErrno(err)
	}

	filetype := getWasiFiletype(stat.Mode())
	var fdflags uint16

	// Determine if it is writeable
	if w := fsc.FdWriter(fd); w != nil {
		// TODO: maybe cache flags to open instead
		fdflags = FD_APPEND
	}

	writeFdstat(buf, filetype, fdflags)

	return ErrnoSuccess
}

var blockFdstat = []byte{
	FILETYPE_BLOCK_DEVICE, 0, // filetype
	0, 0, 0, 0, 0, 0, // fdflags
	0, 0, 0, 0, 0, 0, 0, 0, // fs_rights_base
	0, 0, 0, 0, 0, 0, 0, 0, // fs_rights_inheriting
}

func writeFdstat(buf []byte, filetype uint8, fdflags uint16) {
	// memory is re-used, so ensure the result is defaulted.
	copy(buf, blockFdstat)
	buf[0] = filetype
	buf[2] = byte(fdflags)
}

// fdFdstatSetFlags is the WASI function named FdFdstatSetFlagsName which
// adjusts the flags associated with a file descriptor.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_fdstat_set_flagsfd-fd-flags-fdflags---errnoand is stubbed for GrainLang per #271
var fdFdstatSetFlags = stubFunction(FdFdstatSetFlagsName, []wasm.ValueType{i32, i32}, "fd", "flags")

// fdFdstatSetRights will not be implemented as rights were removed from WASI.
//
// See https://github.com/bytecodealliance/wasmtime/pull/4666
var fdFdstatSetRights = stubFunction(
	FdFdstatSetRightsName,
	[]wasm.ValueType{i32, i64, i64},
	"fd", "fs_rights_base", "fs_rights_inheriting",
)

// fdFilestatGet is the WASI function named FdFilestatGetName which returns
// the stat attributes of an open file.
//
// # Parameters
//
//   - fd: file descriptor to get the filestat attributes data for
//   - resultFilestat: offset to write the result filestat data
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid
//   - ErrnoIo: could not stat `fd` on filesystem
//   - ErrnoFault: `resultFilestat` points to an offset out of memory
//
// filestat byte layout is 64-byte size, with the following fields:
//   - dev 8 bytes: the device ID of device containing the file
//   - ino 8 bytes: the file serial number
//   - filetype 1 byte: the type of the file
//   - 7 pad bytes
//   - nlink 8 bytes: number of hard links to the file
//   - size 8 bytes: for regular files, the file size in bytes. For symbolic links, the length in bytes of the pathname contained in the symbolic link
//   - atim 8 bytes: ast data access timestamp
//   - mtim 8 bytes: last data modification timestamp
//   - ctim 8 bytes: ast file status change timestamp
//
// For example, with a regular file this function writes the below to api.Memory:
//
//	                                                             uint8 --+
//		                         uint64le                uint64le        |        padding               uint64le                uint64le                         uint64le                               uint64le                             uint64le
//		                 +--------------------+  +--------------------+  |  +-----------------+  +--------------------+  +-----------------------+  +----------------------------------+  +----------------------------------+  +----------------------------------+
//		                 |                    |  |                    |  |  |                 |  |                    |  |                       |  |                                  |  |                                  |  |                                  |
//		          []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 117, 80, 0, 0, 0, 0, 0, 0, 160, 153, 212, 128, 110, 221, 35, 23, 160, 153, 212, 128, 110, 221, 35, 23, 160, 153, 212, 128, 110, 221, 35, 23}
//		resultFilestat   ^-- dev                 ^-- ino                 ^                       ^-- nlink               ^-- size                   ^-- atim                              ^-- mtim                              ^-- ctim
//		                                                                 |
//		                                                                 +-- filetype
//
// The following properties of filestat are not implemented:
//   - dev: not supported by Golang FS
//   - ino: not supported by Golang FS
//   - nlink: not supported by Golang FS, we use 1
//   - atime: not supported by Golang FS, we use mtim for this
//   - ctim: not supported by Golang FS, we use mtim for this
//
// Note: This is similar to `fstat` in POSIX.
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_filestat_getfd-fd---errno-filestat
// and https://linux.die.net/man/3/fstat
var fdFilestatGet = newHostFunc(FdFilestatGetName, fdFilestatGetFn, []api.ValueType{i32, i32}, "fd", "result.filestat")

// fdFilestatGetFn cannot currently use proxyResultParams because filestat is
// larger than api.ValueTypeI64 (i64 == 8 bytes, but filestat is 64).
func fdFilestatGetFn(_ context.Context, mod api.Module, params []uint64) Errno {
	return fdFilestatGetFunc(mod, uint32(params[0]), uint32(params[1]))
}

func fdFilestatGetFunc(mod api.Module, fd, resultBuf uint32) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	// Ensure we can write the filestat
	buf, ok := mod.Memory().Read(resultBuf, 64)
	if !ok {
		return ErrnoFault
	}

	stat, err := fsc.StatFile(fd)
	if err != nil {
		return ToErrno(err)
	}

	writeFilestat(buf, stat)

	return ErrnoSuccess
}

func getWasiFiletype(fileMode fs.FileMode) uint8 {
	wasiFileType := FILETYPE_UNKNOWN
	if fileMode&fs.ModeDevice != 0 {
		wasiFileType = FILETYPE_BLOCK_DEVICE
	} else if fileMode&fs.ModeCharDevice != 0 {
		wasiFileType = FILETYPE_CHARACTER_DEVICE
	} else if fileMode&fs.ModeDir != 0 {
		wasiFileType = FILETYPE_DIRECTORY
	} else if fileMode&fs.ModeType == 0 {
		wasiFileType = FILETYPE_REGULAR_FILE
	} else if fileMode&fs.ModeSymlink != 0 {
		wasiFileType = FILETYPE_SYMBOLIC_LINK
	}
	return wasiFileType
}

var blockFilestat = []byte{
	0, 0, 0, 0, 0, 0, 0, 0, // device
	0, 0, 0, 0, 0, 0, 0, 0, // inode
	FILETYPE_BLOCK_DEVICE, 0, 0, 0, 0, 0, 0, 0, // filetype
	1, 0, 0, 0, 0, 0, 0, 0, // nlink
	0, 0, 0, 0, 0, 0, 0, 0, // filesize
	0, 0, 0, 0, 0, 0, 0, 0, // atim
	0, 0, 0, 0, 0, 0, 0, 0, // mtim
	0, 0, 0, 0, 0, 0, 0, 0, // ctim
}

func writeFilestat(buf []byte, stat fs.FileInfo) {
	filetype := getWasiFiletype(stat.Mode())
	filesize := uint64(stat.Size())
	atimeNsec, mtimeNsec, ctimeNsec := platform.StatTimes(stat)

	// memory is re-used, so ensure the result is defaulted.
	copy(buf, blockFilestat[:32])
	buf[16] = filetype
	le.PutUint64(buf[32:], filesize)          // filesize
	le.PutUint64(buf[40:], uint64(atimeNsec)) // atim
	le.PutUint64(buf[48:], uint64(mtimeNsec)) // mtim
	le.PutUint64(buf[56:], uint64(ctimeNsec)) // ctim
}

// fdFilestatSetSize is the WASI function named FdFilestatSetSizeName which
// adjusts the size of an open file.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_filestat_set_sizefd-fd-size-filesize---errno
var fdFilestatSetSize = stubFunction(FdFilestatSetSizeName, []wasm.ValueType{i32, i64}, "fd", "size")

// fdFilestatSetTimes is the WASI function named functionFdFilestatSetTimes
// which adjusts the times of an open file.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_filestat_set_timesfd-fd-atim-timestamp-mtim-timestamp-fst_flags-fstflags---errno
var fdFilestatSetTimes = stubFunction(
	FdFilestatSetTimesName,
	[]wasm.ValueType{i32, i64, i64, i32},
	"fd", "atim", "mtim", "fst_flags",
)

// fdPread is the WASI function named FdPreadName which reads from a file
// descriptor, without using and updating the file descriptor's offset.
//
// Except for handling offset, this implementation is identical to fdRead.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_preadfd-fd-iovs-iovec_array-offset-filesize---errno-size
var fdPread = newHostFunc(
	FdPreadName, fdPreadFn,
	[]api.ValueType{i32, i32, i32, i64, i32},
	"fd", "iovs", "iovs_len", "offset", "result.nread",
)

func fdPreadFn(_ context.Context, mod api.Module, params []uint64) Errno {
	return fdReadOrPread(mod, params, true)
}

// fdPrestatGet is the WASI function named FdPrestatGetName which returns
// the prestat data of a file descriptor.
//
// # Parameters
//
//   - fd: file descriptor to get the prestat
//   - resultPrestat: offset to write the result prestat data
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid or the `fd` is not a pre-opened directory
//   - ErrnoFault: `resultPrestat` points to an offset out of memory
//
// prestat byte layout is 8 bytes, beginning with an 8-bit tag and 3 pad bytes.
// The only valid tag is `prestat_dir`, which is tag zero. This simplifies the
// byte layout to 4 empty bytes followed by the uint32le encoded path length.
//
// For example, the directory name corresponding with `fd` was "/tmp" and
// parameter resultPrestat=1, this function writes the below to api.Memory:
//
//	                   padding   uint32le
//	        uint8 --+  +-----+  +--------+
//	                |  |     |  |        |
//	      []byte{?, 0, 0, 0, 0, 4, 0, 0, 0, ?}
//	resultPrestat --^           ^
//	          tag --+           |
//	                            +-- size in bytes of the string "/tmp"
//
// See fdPrestatDirName and
// https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#prestat
var fdPrestatGet = newHostFunc(FdPrestatGetName, fdPrestatGetFn, []api.ValueType{i32, i32}, "fd", "result.prestat")

func fdPrestatGetFn(_ context.Context, mod api.Module, params []uint64) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()
	fd, resultPrestat := uint32(params[0]), uint32(params[1])

	// Currently, we only pre-open the root file descriptor.
	if fd != sys.FdRoot {
		return ErrnoBadf
	}

	entry, ok := fsc.OpenedFile(fd)
	if !ok {
		return ErrnoBadf
	}

	// Upper 32-bits are zero because...
	// * Zero-value 8-bit tag, and 3-byte zero-value padding
	prestat := uint64(len(entry.Name) << 32)
	if !mod.Memory().WriteUint64Le(resultPrestat, prestat) {
		return ErrnoFault
	}
	return ErrnoSuccess
}

// fdPrestatDirName is the WASI function named FdPrestatDirNameName which
// returns the path of the pre-opened directory of a file descriptor.
//
// # Parameters
//
//   - fd: file descriptor to get the path of the pre-opened directory
//   - path: offset in api.Memory to write the result path
//   - pathLen: count of bytes to write to `path`
//   - This should match the uint32le fdPrestatGet writes to offset
//     `resultPrestat`+4
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid
//   - ErrnoFault: `path` points to an offset out of memory
//   - ErrnoNametoolong: `pathLen` is longer than the actual length of the result
//
// For example, the directory name corresponding with `fd` was "/tmp" and
// # Parameters path=1 pathLen=4 (correct), this function will write the below to
// api.Memory:
//
//	               pathLen
//	           +--------------+
//	           |              |
//	[]byte{?, '/', 't', 'm', 'p', ?}
//	    path --^
//
// See fdPrestatGet
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#fd_prestat_dir_name
var fdPrestatDirName = newHostFunc(
	FdPrestatDirNameName, fdPrestatDirNameFn,
	[]api.ValueType{i32, i32, i32},
	"fd", "result.path", "result.path_len",
)

func fdPrestatDirNameFn(_ context.Context, mod api.Module, params []uint64) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()
	fd, path, pathLen := uint32(params[0]), uint32(params[1]), uint32(params[2])

	// Currently, we only pre-open the root file descriptor.
	if fd != sys.FdRoot {
		return ErrnoBadf
	}

	f, ok := fsc.OpenedFile(fd)
	if !ok {
		return ErrnoBadf
	}

	// Some runtimes may have another semantics. See /RATIONALE.md
	if uint32(len(f.Name)) < pathLen {
		return ErrnoNametoolong
	}

	if !mod.Memory().Write(path, []byte(f.Name)[:pathLen]) {
		return ErrnoFault
	}
	return ErrnoSuccess
}

// fdPwrite is the WASI function named FdPwriteName which writes to a file
// descriptor, without using and updating the file descriptor's offset.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_pwritefd-fd-iovs-ciovec_array-offset-filesize---errno-size
var fdPwrite = stubFunction(
	FdPwriteName,
	[]wasm.ValueType{i32, i32, i32, i64, i32},
	"fd", "iovs", "iovs_len", "offset", "result.nwritten",
)

// fdRead is the WASI function named FdReadName which reads from a file
// descriptor.
//
// # Parameters
//
//   - fd: an opened file descriptor to read data from
//   - iovs: offset in api.Memory to read offset, size pairs representing where
//     to write file data
//   - Both offset and length are encoded as uint32le
//   - iovsCount: count of memory offset, size pairs to read sequentially
//     starting at iovs
//   - resultNread: offset in api.Memory to write the number of bytes read
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid
//   - ErrnoFault: `iovs` or `resultNread` point to an offset out of memory
//   - ErrnoIo: a file system error
//
// For example, this function needs to first read `iovs` to determine where
// to write contents. If parameters iovs=1 iovsCount=2, this function reads two
// offset/length pairs from api.Memory:
//
//	                  iovs[0]                  iovs[1]
//	          +---------------------+   +--------------------+
//	          | uint32le    uint32le|   |uint32le    uint32le|
//	          +---------+  +--------+   +--------+  +--------+
//	          |         |  |        |   |        |  |        |
//	[]byte{?, 18, 0, 0, 0, 4, 0, 0, 0, 23, 0, 0, 0, 2, 0, 0, 0, ?... }
//	   iovs --^            ^            ^           ^
//	          |            |            |           |
//	 offset --+   length --+   offset --+  length --+
//
// If the contents of the `fd` parameter was "wazero" (6 bytes) and parameter
// resultNread=26, this function writes the below to api.Memory:
//
//	                    iovs[0].length        iovs[1].length
//	                   +--------------+       +----+       uint32le
//	                   |              |       |    |      +--------+
//	[]byte{ 0..16, ?, 'w', 'a', 'z', 'e', ?, 'r', 'o', ?, 6, 0, 0, 0 }
//	  iovs[0].offset --^                      ^           ^
//	                         iovs[1].offset --+           |
//	                                        resultNread --+
//
// Note: This is similar to `readv` in POSIX. https://linux.die.net/man/3/readv
//
// See fdWrite
// and https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#fd_read
var fdRead = newHostFunc(
	FdReadName, fdReadFn,
	[]api.ValueType{i32, i32, i32, i32},
	"fd", "iovs", "iovs_len", "result.nread",
)

func fdReadFn(_ context.Context, mod api.Module, params []uint64) Errno {
	return fdReadOrPread(mod, params, false)
}

func fdReadOrPread(mod api.Module, params []uint64, isPread bool) Errno {
	mem := mod.Memory()
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd := uint32(params[0])
	iovs := uint32(params[1])
	iovsCount := uint32(params[2])

	var offset int64
	var resultNread uint32
	if isPread {
		offset = int64(params[3])
		resultNread = uint32(params[4])
	} else {
		resultNread = uint32(params[3])
	}

	r := fsc.FdReader(fd)
	if r == nil {
		return ErrnoBadf
	}

	read := r.Read
	if isPread {
		if ra, ok := r.(io.ReaderAt); ok {
			// ReadAt is the Go equivalent to pread.
			read = func(p []byte) (int, error) {
				n, err := ra.ReadAt(p, offset)
				offset += int64(n)
				return n, err
			}
		} else if s, ok := r.(io.Seeker); ok {
			// Unfortunately, it is often not supported.
			// See /RATIONALE.md "fd_pread: io.Seeker fallback when io.ReaderAt is not supported"
			initialOffset, err := s.Seek(0, io.SeekCurrent)
			if err != nil {
				return ErrnoInval
			}
			defer func() { _, _ = s.Seek(initialOffset, io.SeekStart) }()
			if offset != initialOffset {
				_, err := s.Seek(offset, io.SeekStart)
				if err != nil {
					return ErrnoInval
				}
			}
		} else {
			return ErrnoInval
		}
	}

	var nread uint32
	iovsStop := iovsCount << 3 // iovsCount * 8
	iovsBuf, ok := mem.Read(iovs, iovsStop)
	if !ok {
		return ErrnoFault
	}

	for iovsPos := uint32(0); iovsPos < iovsStop; iovsPos += 8 {
		offset := le.Uint32(iovsBuf[iovsPos:])
		l := le.Uint32(iovsBuf[iovsPos+4:])

		b, ok := mem.Read(offset, l)
		if !ok {
			return ErrnoFault
		}

		n, err := read(b)
		nread += uint32(n)

		shouldContinue, errno := fdRead_shouldContinueRead(uint32(n), l, err)
		if errno != ErrnoSuccess {
			return errno
		} else if !shouldContinue {
			break
		}
	}
	if !mem.WriteUint32Le(resultNread, nread) {
		return ErrnoFault
	} else {
		return ErrnoSuccess
	}
}

// fdRead_shouldContinueRead decides whether to continue reading the next iovec
// based on the amount read (n/l) and a possible error returned from io.Reader.
//
// Note: When there are both bytes read (n) and an error, this continues.
// See /RATIONALE.md "Why ignore the error returned by io.Reader when n > 1?"
func fdRead_shouldContinueRead(n, l uint32, err error) (bool, Errno) {
	if errors.Is(err, io.EOF) {
		return false, ErrnoSuccess // EOF isn't an error, and we shouldn't continue.
	} else if err != nil && n == 0 {
		return false, ErrnoIo
	} else if err != nil {
		return false, ErrnoSuccess // Allow the caller to process n bytes.
	}
	// Continue reading, unless there's a partial read or nothing to read.
	return n == l && n != 0, ErrnoSuccess
}

// fdReaddir is the WASI function named FdReaddirName which reads directory
// entries from a directory.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_readdirfd-fd-buf-pointeru8-buf_len-size-cookie-dircookie---errno-size
var fdReaddir = newHostFunc(
	FdReaddirName, fdReaddirFn,
	[]wasm.ValueType{i32, i32, i32, i64, i32},
	"fd", "buf", "buf_len", "cookie", "result.bufused",
)

func fdReaddirFn(_ context.Context, mod api.Module, params []uint64) Errno {
	mem := mod.Memory()
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd := uint32(params[0])
	buf := uint32(params[1])
	bufLen := uint32(params[2])
	// We control the value of the cookie, and it should never be negative.
	// However, we coerce it to signed to ensure the caller doesn't manipulate
	// it in such a way that becomes negative.
	cookie := int64(params[3])
	resultBufused := uint32(params[4])

	// The bufLen must be enough to write a dirent. Otherwise, the caller can't
	// read what the next cookie is.
	if bufLen < DirentSize {
		return ErrnoInval
	}

	// Validate the FD is a directory
	rd, dir, errno := openedDir(fsc, fd)
	if errno != ErrnoSuccess {
		return errno
	}

	// expect a cookie only if we are continuing a read.
	if cookie == 0 && dir.CountRead > 0 {
		return ErrnoInval // cookie is minimally one.
	}

	// First, determine the maximum directory entries that can be encoded as
	// dirents. The total size is DirentSize(24) + nameSize, for each file.
	// Since a zero-length file name is invalid, the minimum size entry is
	// 25 (DirentSize + 1 character).
	maxDirEntries := int(bufLen/DirentSize + 1)

	// While unlikely maxDirEntries will fit into bufLen, add one more just in
	// case, as we need to know if we hit the end of the directory or not to
	// write the correct bufused (e.g. == bufLen unless EOF).
	//	>> If less than the size of the read buffer, the end of the
	//	>> directory has been reached.
	maxDirEntries += 1

	// The host keeps state for any unread entries from the prior call because
	// we cannot seek to a previous directory position. Collect these entries.
	entries, errno := lastDirEntries(dir, cookie)
	if errno != ErrnoSuccess {
		return errno
	}

	// Check if we have maxDirEntries, and read more from the FS as needed.
	if entryCount := len(entries); entryCount < maxDirEntries {
		if l, err := rd.ReadDir(maxDirEntries - entryCount); err != io.EOF {
			if err != nil {
				return ErrnoIo
			}
			dir.CountRead += uint64(len(l))
			entries = append(entries, l...)
			// Replace the cache with up to maxDirEntries, starting at cookie.
			dir.Entries = entries
		}
	}

	// Determine how many dirents we can write, excluding a potentially
	// truncated entry.
	bufused, direntCount, writeTruncatedEntry := maxDirents(entries, bufLen)

	// Now, write entries to the underlying buffer.
	if bufused > 0 {

		// d_next is the index of the next file in the list, so it should
		// always be one higher than the requested cookie.
		d_next := uint64(cookie + 1)
		// ^^ yes this can overflow to negative, which means our implementation
		// doesn't support writing greater than max int64 entries.

		dirents, ok := mem.Read(buf, bufused)
		if !ok {
			return ErrnoFault
		}

		writeDirents(entries, direntCount, writeTruncatedEntry, dirents, d_next)
	}

	if !mem.WriteUint32Le(resultBufused, bufused) {
		return ErrnoFault
	}
	return ErrnoSuccess
}

const largestDirent = int64(math.MaxUint32 - DirentSize)

// lastDirEntries is broken out from fdReaddirFn for testability.
func lastDirEntries(dir *sys.ReadDir, cookie int64) (entries []fs.DirEntry, errno Errno) {
	if cookie < 0 {
		errno = ErrnoInval // invalid as we will never send a negative cookie.
		return
	}

	entryCount := int64(len(dir.Entries))
	if entryCount == 0 { // there was no prior call
		if cookie != 0 {
			errno = ErrnoInval // invalid as we haven't sent that cookie
		}
		return
	}

	// Get the first absolute position in our window of results
	firstPos := int64(dir.CountRead) - entryCount
	cookiePos := cookie - firstPos

	switch {
	case cookiePos < 0: // cookie is asking for results outside our window.
		errno = ErrnoNosys // we can't implement directory seeking backwards.
	case cookiePos == 0: // cookie is asking for the next page.
	case cookiePos > entryCount:
		errno = ErrnoInval // invalid as we read that far, yet.
	case cookiePos > 0: // truncate so to avoid large lists.
		entries = dir.Entries[cookiePos:]
	default:
		entries = dir.Entries
	}
	if len(entries) == 0 {
		entries = nil
	}
	return
}

// maxDirents returns the maximum count and total entries that can fit in
// maxLen bytes.
//
// truncatedEntryLen is the amount of bytes past bufLen needed to write the
// next entry. We have to return bufused == bufLen unless the directory is
// exhausted.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#fd_readdir
// See https://github.com/WebAssembly/wasi-libc/blob/659ff414560721b1660a19685110e484a081c3d4/libc-bottom-half/cloudlibc/src/libc/dirent/readdir.c#L44
func maxDirents(entries []fs.DirEntry, bufLen uint32) (bufused, direntCount uint32, writeTruncatedEntry bool) {
	lenRemaining := bufLen
	for _, e := range entries {
		if lenRemaining < DirentSize {
			// We don't have enough space in bufLen for another struct,
			// entry. A caller who wants more will retry.

			// bufused == bufLen means more entries exist, which is the case
			// when the dirent is larger than bytes remaining.
			bufused = bufLen
			break
		}

		// use int64 to guard against huge filenames
		nameLen := int64(len(e.Name()))
		var entryLen uint32

		// Check to see if DirentSize + nameLen overflows, or if it would be
		// larger than possible to encode.
		if el := int64(DirentSize) + nameLen; el < 0 || el > largestDirent {
			// panic, as testing is difficult. ex we would have to extract a
			// function to get size of a string or allocate a 2^32 size one!
			panic("invalid filename: too large")
		} else { // we know this can fit into a uint32
			entryLen = uint32(el)
		}

		if entryLen > lenRemaining {
			// We haven't room to write the entry, and docs say to write the
			// header. This helps especially when there is an entry with a very
			// long filename. Ex if bufLen is 4096 and the filename is 4096,
			// we need to write DirentSize(24) + 4096 bytes to write the entry.
			// In this case, we only write up to DirentSize(24) to allow the
			// caller to resize.

			// bufused == bufLen means more entries exist, which is the case
			// when the next entry is larger than bytes remaining.
			bufused = bufLen

			// We do have enough space to write the header, this value will be
			// passed on to writeDirents to only write the header for this entry.
			writeTruncatedEntry = true
			break
		}

		// This won't go negative because we checked entryLen <= lenRemaining.
		lenRemaining -= entryLen
		bufused += entryLen
		direntCount++
	}
	return
}

// writeDirents writes the directory entries to the buffer, which is pre-sized
// based on maxDirents.	truncatedEntryLen means write one past entryCount,
// without its name. See maxDirents for why
func writeDirents(
	entries []fs.DirEntry,
	entryCount uint32,
	writeTruncatedEntry bool,
	dirents []byte,
	d_next uint64,
) {
	pos, i := uint32(0), uint32(0)
	for ; i < entryCount; i++ {
		e := entries[i]
		nameLen := uint32(len(e.Name()))

		writeDirent(dirents[pos:], d_next, nameLen, e.IsDir())
		pos += DirentSize

		copy(dirents[pos:], e.Name())
		pos += nameLen
		d_next++
	}

	if !writeTruncatedEntry {
		return
	}

	// Write a dirent without its name
	dirent := make([]byte, DirentSize)
	e := entries[i]
	writeDirent(dirent, d_next, uint32(len(e.Name())), e.IsDir())

	// Potentially truncate it
	copy(dirents[pos:], dirent)
}

// writeDirent writes DirentSize bytes
func writeDirent(buf []byte, dNext uint64, dNamlen uint32, dType bool) {
	le.PutUint64(buf, dNext)        // d_next
	le.PutUint64(buf[8:], 0)        // no d_ino
	le.PutUint32(buf[16:], dNamlen) // d_namlen

	filetype := FILETYPE_REGULAR_FILE
	if dType {
		filetype = FILETYPE_DIRECTORY
	}
	le.PutUint32(buf[20:], uint32(filetype)) //  d_type
}

// openedDir returns the directory and ErrnoSuccess if the fd points to a readable directory.
func openedDir(fsc *sys.FSContext, fd uint32) (fs.ReadDirFile, *sys.ReadDir, Errno) {
	if f, ok := fsc.OpenedFile(fd); !ok {
		return nil, nil, ErrnoBadf
	} else if d, ok := f.File.(fs.ReadDirFile); !ok {
		// fd_readdir docs don't indicate whether to return ErrnoNotdir or
		// ErrnoBadf. It has been noticed that rust will crash on ErrnoNotdir,
		// and POSIX C ref seems to not return this, so we don't either.
		//
		// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#fd_readdir
		// and https://en.wikibooks.org/wiki/C_Programming/POSIX_Reference/dirent.h
		return nil, nil, ErrnoBadf
	} else {
		if f.ReadDir == nil {
			f.ReadDir = &sys.ReadDir{}
		}
		return d, f.ReadDir, ErrnoSuccess
	}
}

// fdRenumber is the WASI function named FdRenumberName which atomically
// replaces a file descriptor by renumbering another file descriptor.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_renumberfd-fd-to-fd---errno
var fdRenumber = stubFunction(FdRenumberName, []wasm.ValueType{i32, i32}, "fd", "to")

// fdSeek is the WASI function named FdSeekName which moves the offset of a
// file descriptor.
//
// # Parameters
//
//   - fd: file descriptor to move the offset of
//   - offset: signed int64, which is encoded as uint64, input argument to
//     `whence`, which results in a new offset
//   - whence: operator that creates the new offset, given `offset` bytes
//   - If io.SeekStart, new offset == `offset`.
//   - If io.SeekCurrent, new offset == existing offset + `offset`.
//   - If io.SeekEnd, new offset == file size of `fd` + `offset`.
//   - resultNewoffset: offset in api.Memory to write the new offset to,
//     relative to start of the file
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid
//   - ErrnoFault: `resultNewoffset` points to an offset out of memory
//   - ErrnoInval: `whence` is an invalid value
//   - ErrnoIo: a file system error
//
// For example, if fd 3 is a file with offset 0, and parameters fd=3, offset=4,
// whence=0 (=io.SeekStart), resultNewOffset=1, this function writes the below
// to api.Memory:
//
//	                         uint64le
//	                  +--------------------+
//	                  |                    |
//	        []byte{?, 4, 0, 0, 0, 0, 0, 0, 0, ? }
//	resultNewoffset --^
//
// Note: This is similar to `lseek` in POSIX. https://linux.die.net/man/3/lseek
//
// See io.Seeker
// and https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#fd_seek
var fdSeek = newHostFunc(
	FdSeekName, fdSeekFn,
	[]api.ValueType{i32, i64, i32, i32},
	"fd", "offset", "whence", "result.newoffset",
)

func fdSeekFn(_ context.Context, mod api.Module, params []uint64) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()
	fd := uint32(params[0])
	offset := params[1]
	whence := uint32(params[2])
	resultNewoffset := uint32(params[3])

	if fd == sys.FdRoot {
		return ErrnoBadf // cannot seek a directory
	}

	var seeker io.Seeker
	// Check to see if the file descriptor is available
	if f, ok := fsc.OpenedFile(fd); !ok {
		return ErrnoBadf
		// fs.FS doesn't declare io.Seeker, but implementations such as os.File implement it.
	} else if seeker, ok = f.File.(io.Seeker); !ok {
		return ErrnoBadf
	}

	if whence > io.SeekEnd /* exceeds the largest valid whence */ {
		return ErrnoInval
	}

	newOffset, err := seeker.Seek(int64(offset), int(whence))
	if err != nil {
		return ErrnoIo
	}

	if !mod.Memory().WriteUint64Le(resultNewoffset, uint64(newOffset)) {
		return ErrnoFault
	}
	return ErrnoSuccess
}

// fdSync is the WASI function named FdSyncName which synchronizes the data
// and metadata of a file to disk.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_syncfd-fd---errno
var fdSync = stubFunction(FdSyncName, []api.ValueType{i32}, "fd")

// fdTell is the WASI function named FdTellName which returns the current
// offset of a file descriptor.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_tellfd-fd---errno-filesize
var fdTell = stubFunction(FdTellName, []wasm.ValueType{i32, i32}, "fd", "result.offset")

// fdWrite is the WASI function named FdWriteName which writes to a file
// descriptor.
//
// # Parameters
//
//   - fd: an opened file descriptor to write data to
//   - iovs: offset in api.Memory to read offset, size pairs representing the
//     data to write to `fd`
//   - Both offset and length are encoded as uint32le.
//   - iovsCount: count of memory offset, size pairs to read sequentially
//     starting at iovs
//   - resultNwritten: offset in api.Memory to write the number of bytes
//     written
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid
//   - ErrnoFault: `iovs` or `resultNwritten` point to an offset out of memory
//   - ErrnoIo: a file system error
//
// For example, this function needs to first read `iovs` to determine what to
// write to `fd`. If parameters iovs=1 iovsCount=2, this function reads two
// offset/length pairs from api.Memory:
//
//	                  iovs[0]                  iovs[1]
//	          +---------------------+   +--------------------+
//	          | uint32le    uint32le|   |uint32le    uint32le|
//	          +---------+  +--------+   +--------+  +--------+
//	          |         |  |        |   |        |  |        |
//	[]byte{?, 18, 0, 0, 0, 4, 0, 0, 0, 23, 0, 0, 0, 2, 0, 0, 0, ?... }
//	   iovs --^            ^            ^           ^
//	          |            |            |           |
//	 offset --+   length --+   offset --+  length --+
//
// This function reads those chunks api.Memory into the `fd` sequentially.
//
//	                    iovs[0].length        iovs[1].length
//	                   +--------------+       +----+
//	                   |              |       |    |
//	[]byte{ 0..16, ?, 'w', 'a', 'z', 'e', ?, 'r', 'o', ? }
//	  iovs[0].offset --^                      ^
//	                         iovs[1].offset --+
//
// Since "wazero" was written, if parameter resultNwritten=26, this function
// writes the below to api.Memory:
//
//	                   uint32le
//	                  +--------+
//	                  |        |
//	[]byte{ 0..24, ?, 6, 0, 0, 0', ? }
//	 resultNwritten --^
//
// Note: This is similar to `writev` in POSIX. https://linux.die.net/man/3/writev
//
// See fdRead
// https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#ciovec
// and https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#fd_write
var fdWrite = newHostFunc(
	FdWriteName, fdWriteFn,
	[]api.ValueType{i32, i32, i32, i32},
	"fd", "iovs", "iovs_len", "result.nwritten",
)

func fdWriteFn(_ context.Context, mod api.Module, params []uint64) Errno {
	mem := mod.Memory()
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd := uint32(params[0])
	iovs := uint32(params[1])
	iovsCount := uint32(params[2])
	resultNwritten := uint32(params[3])

	writer := fsc.FdWriter(fd)
	if writer == nil {
		return ErrnoBadf
	}

	var err error
	var nwritten uint32
	iovsStop := iovsCount << 3 // iovsCount * 8
	iovsBuf, ok := mem.Read(iovs, iovsStop)
	if !ok {
		return ErrnoFault
	}

	for iovsPos := uint32(0); iovsPos < iovsStop; iovsPos += 8 {
		offset := le.Uint32(iovsBuf[iovsPos:])
		l := le.Uint32(iovsBuf[iovsPos+4:])

		var n int
		if writer == io.Discard { // special-case default
			n = int(l)
		} else {
			b, ok := mem.Read(offset, l)
			if !ok {
				return ErrnoFault
			}
			n, err = writer.Write(b)
			if err != nil {
				return ErrnoIo
			}
		}
		nwritten += uint32(n)
	}

	if !mod.Memory().WriteUint32Le(resultNwritten, nwritten) {
		return ErrnoFault
	}
	return ErrnoSuccess
}

// pathCreateDirectory is the WASI function named PathCreateDirectoryName which
// creates a directory.
//
// # Parameters
//
//   - fd: file descriptor of a directory that `path` is relative to
//   - path: offset in api.Memory to read the path string from
//   - pathLen: length of `path`
//
// # Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid
//   - ErrnoNoent: `path` does not exist.
//   - ErrnoNotdir: `path` is a file
//
// # Notes
//   - This is similar to mkdirat in POSIX.
//     See https://linux.die.net/man/2/mkdirat
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_create_directoryfd-fd-path-string---errno
var pathCreateDirectory = newHostFunc(
	PathCreateDirectoryName, pathCreateDirectoryFn,
	[]wasm.ValueType{i32, i32, i32},
	"fd", "path", "path_len",
)

func pathCreateDirectoryFn(_ context.Context, mod api.Module, params []uint64) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	dirfd := uint32(params[0])
	path := uint32(params[1])
	pathLen := uint32(params[2])

	pathName, errno := atPath(fsc, mod.Memory(), dirfd, path, pathLen)
	if errno != ErrnoSuccess {
		return errno
	}

	if fd, err := fsc.Mkdir(pathName, 0o700); err != nil {
		return ToErrno(err)
	} else {
		_ = fsc.CloseFile(fd)
	}

	return ErrnoSuccess
}

// pathFilestatGet is the WASI function named PathFilestatGetName which
// returns the stat attributes of a file or directory.
//
// # Parameters
//
//   - fd: file descriptor of the folder to look in for the path
//   - flags: flags determining the method of how paths are resolved
//   - path: path under fd to get the filestat attributes data for
//   - path_len: length of the path that was given
//   - resultFilestat: offset to write the result filestat data
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid
//   - ErrnoNotdir: `fd` points to a file not a directory
//   - ErrnoIo: could not stat `fd` on filesystem
//   - ErrnoInval: the path contained "../"
//   - ErrnoNametoolong: `path` + `path_len` is out of memory
//   - ErrnoFault: `resultFilestat` points to an offset out of memory
//   - ErrnoNoent: could not find the path
//
// The rest of this implementation matches that of fdFilestatGet, so is not
// repeated here.
//
// Note: This is similar to `fstatat` in POSIX.
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_filestat_getfd-fd-flags-lookupflags-path-string---errno-filestat
// and https://linux.die.net/man/2/fstatat
var pathFilestatGet = newHostFunc(
	PathFilestatGetName, pathFilestatGetFn,
	[]api.ValueType{i32, i32, i32, i32, i32},
	"fd", "flags", "path", "path_len", "result.filestat",
)

func pathFilestatGetFn(_ context.Context, mod api.Module, params []uint64) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	dirfd := uint32(params[0])

	// TODO: flags is a lookupflags and it only has one bit: symlink_follow
	// https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#lookupflags
	_ /* flags */ = uint32(params[1])

	pathOffset := uint32(params[2])
	pathLen := uint32(params[3])

	resultBuf := uint32(params[4])

	// open_at isn't supported in fs.FS, so we check the path can't escape,
	// then join it with its parent
	b, ok := mod.Memory().Read(pathOffset, pathLen)
	if !ok {
		return ErrnoNametoolong
	}
	pathName := string(b)

	// Prepend the path if necessary.
	if dir, ok := fsc.OpenedFile(dirfd); !ok {
		return ErrnoBadf
	} else if _, ok := dir.File.(fs.ReadDirFile); !ok {
		return ErrnoNotdir // TODO: cache filetype instead of poking.
	} else {
		// TODO: consolidate "at" logic with path_open as same issues occur.
		pathName = path.Join(dir.Name, pathName)
	}

	// Stat the file without allocating a file descriptor
	stat, errnoResult := statFile(fsc, pathName)
	if errnoResult != ErrnoSuccess {
		return errnoResult
	}

	// Write the stat result to memory
	buf, ok := mod.Memory().Read(resultBuf, 64)
	if !ok {
		return ErrnoFault
	}
	writeFilestat(buf, stat)

	return ErrnoSuccess
}

// pathFilestatSetTimes is the WASI function named PathFilestatSetTimesName
// which adjusts the timestamps of a file or directory.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_filestat_set_timesfd-fd-flags-lookupflags-path-string-atim-timestamp-mtim-timestamp-fst_flags-fstflags---errno
var pathFilestatSetTimes = stubFunction(
	PathFilestatSetTimesName,
	[]wasm.ValueType{i32, i32, i32, i32, i64, i64, i32},
	"fd", "flags", "path", "path_len", "atim", "mtim", "fst_flags",
)

// pathLink is the WASI function named PathLinkName which adjusts the
// timestamps of a file or directory.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#path_link
var pathLink = stubFunction(
	PathLinkName,
	[]wasm.ValueType{i32, i32, i32, i32, i32, i32, i32},
	"old_fd", "old_flags", "old_path", "old_path_len", "new_fd", "new_path", "new_path_len",
)

// pathOpen is the WASI function named PathOpenName which opens a file or
// directory. This returns ErrnoBadf if the fd is invalid.
//
// # Parameters
//
//   - fd: file descriptor of a directory that `path` is relative to
//   - dirflags: flags to indicate how to resolve `path`
//   - path: offset in api.Memory to read the path string from
//   - pathLen: length of `path`
//   - oFlags: open flags to indicate the method by which to open the file
//   - fsRightsBase: ignored as rights were removed from WASI.
//   - fsRightsInheriting: ignored as rights were removed from WASI.
//     created file descriptor for `path`
//   - fdFlags: file descriptor flags
//   - resultOpenedFd: offset in api.Memory to write the newly created file
//     descriptor to.
//   - The result FD value is guaranteed to be less than 2**31
//
// Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid
//   - ErrnoFault: `resultOpenedFd` points to an offset out of memory
//   - ErrnoNoent: `path` does not exist.
//   - ErrnoExist: `path` exists, while `oFlags` requires that it must not.
//   - ErrnoNotdir: `path` is not a directory, while `oFlags` requires it.
//   - ErrnoIo: a file system error
//
// For example, this function needs to first read `path` to determine the file
// to open. If parameters `path` = 1, `pathLen` = 6, and the path is "wazero",
// pathOpen reads the path from api.Memory:
//
//	                pathLen
//	            +------------------------+
//	            |                        |
//	[]byte{ ?, 'w', 'a', 'z', 'e', 'r', 'o', ?... }
//	     path --^
//
// Then, if parameters resultOpenedFd = 8, and this function opened a new file
// descriptor 5 with the given flags, this function writes the below to
// api.Memory:
//
//	                  uint32le
//	                 +--------+
//	                 |        |
//	[]byte{ 0..6, ?, 5, 0, 0, 0, ?}
//	resultOpenedFd --^
//
// # Notes
//   - This is similar to `openat` in POSIX. https://linux.die.net/man/3/openat
//   - The returned file descriptor is not guaranteed to be the lowest-number
//
// See https://github.com/WebAssembly/WASI/blob/main/phases/snapshot/docs.md#path_open
var pathOpen = newHostFunc(
	PathOpenName, pathOpenFn,
	[]api.ValueType{i32, i32, i32, i32, i32, i64, i64, i32, i32},
	"fd", "dirflags", "path", "path_len", "oflags", "fs_rights_base", "fs_rights_inheriting", "fdflags", "result.opened_fd",
)

func pathOpenFn(_ context.Context, mod api.Module, params []uint64) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	preopenFD := uint32(params[0])

	// TODO: dirflags is a lookupflags, and it only has one bit: symlink_follow
	// https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#lookupflags
	_ /* dirflags */ = uint32(params[1])

	path := uint32(params[2])
	pathLen := uint32(params[3])

	oflags := uint16(params[4])

	// rights aren't used
	_, _ = params[5], params[6]

	fdflags := uint16(params[7])
	resultOpenedFd := uint32(params[8])

	pathName, errno := atPath(fsc, mod.Memory(), preopenFD, path, pathLen)
	if errno != ErrnoSuccess {
		return errno
	}

	fileOpenFlags, isDir := openFlags(oflags, fdflags)

	var newFD uint32
	var err error
	if isDir && oflags&O_CREAT != 0 {
		return ErrnoInval // use pathCreateDirectory!
	} else {
		newFD, err = fsc.OpenFile(pathName, fileOpenFlags, 0o600)
	}

	if err != nil {
		return ToErrno(err)
	}

	// Check any flags that require the file to evaluate.
	if isDir {
		if errno := failIfNotDirectory(fsc, newFD); errno != ErrnoSuccess {
			return errno
		}
	}

	if !mod.Memory().WriteUint32Le(resultOpenedFd, newFD) {
		_ = fsc.CloseFile(newFD)
		return ErrnoFault
	}
	return ErrnoSuccess
}

// Note: We don't handle AT_FDCWD, as that's resolved in the compiler.
// There's no working directory function in WASI, so CWD cannot be handled
// here in any way except assuming it is "/".
//
// See https://github.com/WebAssembly/wasi-libc/blob/659ff414560721b1660a19685110e484a081c3d4/libc-bottom-half/sources/at_fdcwd.c#L24-L26
// See https://linux.die.net/man/2/openat
func atPath(fsc *sys.FSContext, mem api.Memory, dirFd, path, pathLen uint32) (string, Errno) {
	if dirFd != sys.FdRoot { //nolint
		// TODO: Research if dirFd is always a pre-open. If so, it should
		// always be rootFd (3), until we support multiple pre-opens.
		//
		// Otherwise, the dirFd could be a file created dynamically, and mean
		// paths for Open may need to be built up. For example, if dirFd
		// represents "/tmp/foo" and path="bar", this should open
		// "/tmp/foo/bar" not "/bar".
	}

	if _, ok := fsc.OpenedFile(dirFd); !ok {
		return "", ErrnoBadf
	}

	b, ok := mem.Read(path, pathLen)
	if !ok {
		return "", ErrnoFault
	}
	return string(b), ErrnoSuccess
}

func openFlags(oflags, fdflags uint16) (openFlags int, isDir bool) {
	isDir = oflags&O_DIRECTORY != 0
	if oflags&O_TRUNC != 0 {
		openFlags = os.O_RDWR | os.O_TRUNC
	}
	if oflags&O_CREAT != 0 {
		openFlags = os.O_RDWR | os.O_CREATE
	}
	if fdflags&FD_APPEND != 0 {
		openFlags = os.O_RDWR | os.O_APPEND
	}
	if openFlags == 0 {
		openFlags = os.O_RDONLY
	}
	if isDir {
		return
	}
	if oflags&O_EXCL != 0 {
		openFlags |= os.O_EXCL
	}
	return
}

func failIfNotDirectory(fsc *sys.FSContext, fd uint32) Errno {
	// Lookup the previous file
	if f, ok := fsc.OpenedFile(fd); !ok {
		return ErrnoBadf
	} else if _, ok := f.File.(fs.ReadDirFile); !ok {
		_ = fsc.CloseFile(fd)
		return ErrnoNotdir
	}
	return ErrnoSuccess
}

// pathReadlink is the WASI function named PathReadlinkName that reads the
// contents of a symbolic link.
//
// See: https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_readlinkfd-fd-path-string-buf-pointeru8-buf_len-size---errno-size
var pathReadlink = stubFunction(
	PathReadlinkName,
	[]wasm.ValueType{i32, i32, i32, i32, i32, i32},
	"fd", "path", "path_len", "buf", "buf_len", "result.bufused",
)

// pathRemoveDirectory is the WASI function named PathRemoveDirectoryName which
// removes a directory.
//
// # Parameters
//
//   - fd: file descriptor of a directory that `path` is relative to
//   - path: offset in api.Memory to read the path string from
//   - pathLen: length of `path`
//
// # Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid
//   - ErrnoNoent: `path` does not exist.
//   - ErrnoNotempty: `path` is not empty
//   - ErrnoNotdir: `path` is a file
//
// # Notes
//   - This is similar to unlinkat with AT_REMOVEDIR in POSIX.
//     See https://linux.die.net/man/2/unlinkat
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_remove_directoryfd-fd-path-string---errno
var pathRemoveDirectory = newHostFunc(
	PathRemoveDirectoryName, pathRemoveDirectoryFn,
	[]wasm.ValueType{i32, i32, i32},
	"fd", "path", "path_len",
)

func pathRemoveDirectoryFn(_ context.Context, mod api.Module, params []uint64) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	dirfd := uint32(params[0])
	path := uint32(params[1])
	pathLen := uint32(params[2])

	pathName, errno := atPath(fsc, mod.Memory(), dirfd, path, pathLen)
	if errno != ErrnoSuccess {
		return errno
	}

	if err := fsc.Rmdir(pathName); err != nil {
		return ToErrno(err)
	}

	return ErrnoSuccess
}

// pathRename is the WASI function named PathRenameName which renames a file or
// directory.
//
// # Parameters
//
//   - fd: file descriptor of a directory that `old_path` is relative to
//   - old_path: offset in api.Memory to read the old path string from
//   - old_path_len: length of `old_path`
//   - new_fd: file descriptor of a directory that `new_path` is relative to
//   - new_path: offset in api.Memory to read the new path string from
//   - new_path_len: length of `new_path`
//
// # Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` or `new_fd` are invalid
//   - ErrnoNoent: `old_path` does not exist.
//   - ErrnoNotdir: `old` is a directory and `new` exists, but is a file.
//   - ErrnoIsdir: `old` is a file and `new` exists, but is a directory.
//
// # Notes
//   - This is similar to unlinkat in POSIX.
//     See https://linux.die.net/man/2/renameat
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_renamefd-fd-old_path-string-new_fd-fd-new_path-string---errno
var pathRename = newHostFunc(
	PathRenameName, pathRenameFn,
	[]wasm.ValueType{i32, i32, i32, i32, i32, i32},
	"fd", "old_path", "old_path_len", "new_fd", "new_path", "new_path_len",
)

func pathRenameFn(_ context.Context, mod api.Module, params []uint64) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	oldDirFd := uint32(params[0])
	oldPath := uint32(params[1])
	oldPathLen := uint32(params[2])

	newDirFd := uint32(params[3])
	newPath := uint32(params[4])
	newPathLen := uint32(params[5])

	oldPathName, errno := atPath(fsc, mod.Memory(), oldDirFd, oldPath, oldPathLen)
	if errno != ErrnoSuccess {
		return errno
	}

	newPathName, errno := atPath(fsc, mod.Memory(), newDirFd, newPath, newPathLen)
	if errno != ErrnoSuccess {
		return errno
	}

	if err := fsc.Rename(oldPathName, newPathName); err != nil {
		return ToErrno(err)
	}

	return ErrnoSuccess
}

// pathSymlink is the WASI function named PathSymlinkName which creates a
// symbolic link.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#path_symlink
var pathSymlink = stubFunction(
	PathSymlinkName,
	[]wasm.ValueType{i32, i32, i32, i32, i32},
	"old_path", "old_path_len", "fd", "new_path", "new_path_len",
)

// pathUnlinkFile is the WASI function named PathUnlinkFileName which unlinks a
// file.
//
// # Parameters
//
//   - fd: file descriptor of a directory that `path` is relative to
//   - path: offset in api.Memory to read the path string from
//   - pathLen: length of `path`
//
// # Result (Errno)
//
// The return value is ErrnoSuccess except the following error conditions:
//   - ErrnoBadf: `fd` is invalid
//   - ErrnoNoent: `path` does not exist.
//   - ErrnoIsdir: `path` is a directory
//
// # Notes
//   - This is similar to unlinkat without AT_REMOVEDIR in POSIX.
//     See https://linux.die.net/man/2/unlinkat
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_unlink_filefd-fd-path-string---errno
var pathUnlinkFile = newHostFunc(
	PathUnlinkFileName, pathUnlinkFileFn,
	[]wasm.ValueType{i32, i32, i32},
	"fd", "path", "path_len",
)

func pathUnlinkFileFn(_ context.Context, mod api.Module, params []uint64) Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	dirfd := uint32(params[0])
	path := uint32(params[1])
	pathLen := uint32(params[2])

	pathName, errno := atPath(fsc, mod.Memory(), dirfd, path, pathLen)
	if errno != ErrnoSuccess {
		return errno
	}

	if err := fsc.Unlink(pathName); err != nil {
		return ToErrno(err)
	}

	return ErrnoSuccess
}

// statFile attempts to stat the file at the given path. Errors coerce to WASI
// Errno.
func statFile(fsc *sys.FSContext, name string) (stat fs.FileInfo, errno Errno) {
	var err error
	stat, err = fsc.StatPath(name)
	if err != nil {
		errno = ToErrno(err)
	}
	return
}
