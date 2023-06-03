package wasi_snapshot_preview1

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"math"
	"path"
	"reflect"
	"strings"
	"syscall"
	"unsafe"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/platform"
	"github.com/tetratelabs/wazero/internal/sys"
	"github.com/tetratelabs/wazero/internal/sysfs"
	"github.com/tetratelabs/wazero/internal/wasip1"
	"github.com/tetratelabs/wazero/internal/wasm"
)

// The following interfaces are used until we finalize our own FD-scoped file.
type (
	// syncFile is implemented by os.File in file_posix.go
	syncFile interface{ Sync() error }
	// truncateFile is implemented by os.File in file_posix.go
	truncateFile interface{ Truncate(size int64) error }
)

// fdAdvise is the WASI function named FdAdviseName which provides file
// advisory information on a file descriptor.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_advisefd-fd-offset-filesize-len-filesize-advice-advice---errno
var fdAdvise = newHostFunc(
	wasip1.FdAdviseName, fdAdviseFn,
	[]wasm.ValueType{i32, i64, i64, i32},
	"fd", "offset", "len", "advice",
)

func fdAdviseFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fd := uint32(params[0])
	_ = params[1]
	_ = params[2]
	advice := byte(params[3])
	fsc := mod.(*wasm.CallContext).Sys.FS()

	_, ok := fsc.LookupFile(fd)
	if !ok {
		return syscall.EBADF
	}

	switch advice {
	case wasip1.FdAdviceNormal,
		wasip1.FdAdviceSequential,
		wasip1.FdAdviceRandom,
		wasip1.FdAdviceWillNeed,
		wasip1.FdAdviceDontNeed,
		wasip1.FdAdviceNoReuse:
	default:
		return syscall.EINVAL
	}

	// FdAdvice corresponds to posix_fadvise, but it can only be supported on linux.
	// However, the purpose of the call is just to do best-effort optimization on OS kernels,
	// so just making this noop rather than returning NoSup error makes sense and doesn't affect
	// the semantics of Wasm applications.
	// TODO: invoke posix_fadvise on linux, and partially on darwin.
	// - https://gitlab.com/cznic/fileutil/-/blob/v1.1.2/fileutil_linux.go#L87-95
	// - https://github.com/bytecodealliance/system-interface/blob/62b97f9776b86235f318c3a6e308395a1187439b/src/fs/file_io_ext.rs#L430-L442
	return 0
}

// fdAllocate is the WASI function named FdAllocateName which forces the
// allocation of space in a file.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_allocatefd-fd-offset-filesize-len-filesize---errno
var fdAllocate = newHostFunc(
	wasip1.FdAllocateName, fdAllocateFn,
	[]wasm.ValueType{i32, i64, i64},
	"fd", "offset", "len",
)

func fdAllocateFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fd := uint32(params[0])
	offset := params[1]
	length := params[2]

	fsc := mod.(*wasm.CallContext).Sys.FS()
	f, ok := fsc.LookupFile(fd)
	if !ok {
		return syscall.EBADF
	}

	tail := int64(offset + length)
	if tail < 0 {
		return syscall.EINVAL
	}

	st, err := f.Stat()
	if err != nil {
		return platform.UnwrapOSError(err)
	}

	if st.Size >= tail {
		// We already have enough space.
		return 0
	}

	osf, ok := f.File.(truncateFile)
	if !ok {
		return syscall.EBADF
	}

	return platform.UnwrapOSError(osf.Truncate(tail))
}

// fdClose is the WASI function named FdCloseName which closes a file
// descriptor.
//
// # Parameters
//
//   - fd: file descriptor to close
//
// Result (Errno)
//
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: the fd was not open.
//   - syscall.ENOTSUP: the fs was a pre-open
//
// Note: This is similar to `close` in POSIX.
// See https://github.com/WebAssembly/WASI/blob/main/phases/snapshot/docs.md#fd_close
// and https://linux.die.net/man/3/close
var fdClose = newHostFunc(wasip1.FdCloseName, fdCloseFn, []api.ValueType{i32}, "fd")

func fdCloseFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()
	fd := uint32(params[0])

	return fsc.CloseFile(fd)
}

// fdDatasync is the WASI function named FdDatasyncName which synchronizes
// the data of a file to disk.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_datasyncfd-fd---errno
var fdDatasync = newHostFunc(wasip1.FdDatasyncName, fdDatasyncFn, []api.ValueType{i32}, "fd")

func fdDatasyncFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()
	fd := uint32(params[0])

	// Check to see if the file descriptor is available
	if f, ok := fsc.LookupFile(fd); !ok {
		return syscall.EBADF
	} else {
		return sysfs.FileDatasync(f.File)
	}
}

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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid
//   - syscall.EFAULT: `resultFdstat` points to an offset out of memory
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
var fdFdstatGet = newHostFunc(wasip1.FdFdstatGetName, fdFdstatGetFn, []api.ValueType{i32, i32}, "fd", "result.stat")

// fdFdstatGetFn cannot currently use proxyResultParams because fdstat is larger
// than api.ValueTypeI64 (i64 == 8 bytes, but fdstat is 24).
func fdFdstatGetFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd, resultFdstat := uint32(params[0]), uint32(params[1])

	// Ensure we can write the fdstat
	buf, ok := mod.Memory().Read(resultFdstat, 24)
	if !ok {
		return syscall.EFAULT
	}

	var fdflags uint16
	var st fs.FileInfo
	var err error
	if f, ok := fsc.LookupFile(fd); !ok {
		return syscall.EBADF
	} else if st, err = f.File.Stat(); err != nil {
		return platform.UnwrapOSError(err)
	} else if _, ok := f.File.(io.Writer); ok {
		// TODO: maybe cache flags to open instead
		fdflags = wasip1.FD_APPEND
	}

	filetype := getWasiFiletype(st.Mode())
	writeFdstat(buf, filetype, fdflags)

	return 0
}

var blockFdstat = []byte{
	wasip1.FILETYPE_BLOCK_DEVICE, 0, // filetype
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
var fdFdstatSetFlags = newHostFunc(wasip1.FdFdstatSetFlagsName, fdFdstatSetFlagsFn, []wasm.ValueType{i32, i32}, "fd", "flags")

func fdFdstatSetFlagsFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fd, wasiFlag := uint32(params[0]), uint16(params[1])
	fsc := mod.(*wasm.CallContext).Sys.FS()

	// We can only support APPEND flag.
	if wasip1.FD_DSYNC&wasiFlag != 0 || wasip1.FD_NONBLOCK&wasiFlag != 0 || wasip1.FD_RSYNC&wasiFlag != 0 || wasip1.FD_SYNC&wasiFlag != 0 {
		return syscall.EINVAL
	}

	var flag int
	if wasip1.FD_APPEND&wasiFlag != 0 {
		flag = syscall.O_APPEND
	}

	return fsc.ChangeOpenFlag(fd, flag)
}

// fdFdstatSetRights will not be implemented as rights were removed from WASI.
//
// See https://github.com/bytecodealliance/wasmtime/pull/4666
var fdFdstatSetRights = stubFunction(
	wasip1.FdFdstatSetRightsName,
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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid
//   - syscall.EIO: could not stat `fd` on filesystem
//   - syscall.EFAULT: `resultFilestat` points to an offset out of memory
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
var fdFilestatGet = newHostFunc(wasip1.FdFilestatGetName, fdFilestatGetFn, []api.ValueType{i32, i32}, "fd", "result.filestat")

// fdFilestatGetFn cannot currently use proxyResultParams because filestat is
// larger than api.ValueTypeI64 (i64 == 8 bytes, but filestat is 64).
func fdFilestatGetFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	return fdFilestatGetFunc(mod, uint32(params[0]), uint32(params[1]))
}

func fdFilestatGetFunc(mod api.Module, fd, resultBuf uint32) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	// Ensure we can write the filestat
	buf, ok := mod.Memory().Read(resultBuf, 64)
	if !ok {
		return syscall.EFAULT
	}

	f, ok := fsc.LookupFile(fd)
	if !ok {
		return syscall.EBADF
	}

	st, err := f.Stat()
	if err != nil {
		return platform.UnwrapOSError(err)
	}

	return writeFilestat(buf, &st)
}

func getWasiFiletype(fm fs.FileMode) uint8 {
	switch {
	case fm.IsRegular():
		return wasip1.FILETYPE_REGULAR_FILE
	case fm.IsDir():
		return wasip1.FILETYPE_DIRECTORY
	case fm&fs.ModeSymlink != 0:
		return wasip1.FILETYPE_SYMBOLIC_LINK
	case fm&fs.ModeDevice != 0:
		// Unlike ModeDevice and ModeCharDevice, FILETYPE_CHARACTER_DEVICE and
		// FILETYPE_BLOCK_DEVICE are set mutually exclusively.
		if fm&fs.ModeCharDevice != 0 {
			return wasip1.FILETYPE_CHARACTER_DEVICE
		}
		return wasip1.FILETYPE_BLOCK_DEVICE
	default: // unknown
		return wasip1.FILETYPE_UNKNOWN
	}
}

func writeFilestat(buf []byte, st *platform.Stat_t) (errno syscall.Errno) {
	le.PutUint64(buf, st.Dev)
	le.PutUint64(buf[8:], st.Ino)
	le.PutUint64(buf[16:], uint64(getWasiFiletype(st.Mode)))
	le.PutUint64(buf[24:], st.Nlink)
	le.PutUint64(buf[32:], uint64(st.Size))
	le.PutUint64(buf[40:], uint64(st.Atim))
	le.PutUint64(buf[48:], uint64(st.Mtim))
	le.PutUint64(buf[56:], uint64(st.Ctim))
	return
}

// fdFilestatSetSize is the WASI function named FdFilestatSetSizeName which
// adjusts the size of an open file.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_filestat_set_sizefd-fd-size-filesize---errno
var fdFilestatSetSize = newHostFunc(wasip1.FdFilestatSetSizeName, fdFilestatSetSizeFn, []wasm.ValueType{i32, i64}, "fd", "size")

func fdFilestatSetSizeFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fd := uint32(params[0])
	size := uint32(params[1])

	fsc := mod.(*wasm.CallContext).Sys.FS()

	// Check to see if the file descriptor is available
	if f, ok := fsc.LookupFile(fd); !ok {
		return syscall.EBADF
	} else if truncateFile, ok := f.File.(truncateFile); !ok {
		return syscall.EBADF // possibly a fake file
	} else if err := truncateFile.Truncate(int64(size)); err != nil {
		return platform.UnwrapOSError(err)
	}
	return 0
}

// fdFilestatSetTimes is the WASI function named functionFdFilestatSetTimes
// which adjusts the times of an open file.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_filestat_set_timesfd-fd-atim-timestamp-mtim-timestamp-fst_flags-fstflags---errno
var fdFilestatSetTimes = newHostFunc(
	wasip1.FdFilestatSetTimesName, fdFilestatSetTimesFn,
	[]wasm.ValueType{i32, i64, i64, i32},
	"fd", "atim", "mtim", "fst_flags",
)

func fdFilestatSetTimesFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fd := uint32(params[0])
	atim := int64(params[1])
	mtim := int64(params[2])
	fstFlags := uint16(params[3])

	sys := mod.(*wasm.CallContext).Sys
	fsc := sys.FS()

	f, ok := fsc.LookupFile(fd)
	if !ok {
		return syscall.EBADF
	}

	times, errno := toTimes(atim, mtim, fstFlags)
	if errno != 0 {
		return errno
	}

	// Try to update the file timestamps by file-descriptor.
	errno = platform.UtimensFile(f.File, &times)

	// Fall back to path based, despite it being less precise.
	switch errno {
	case syscall.EPERM, syscall.ENOSYS:
		errno = f.FS.Utimens(f.Name, &times, true)
	}

	return errno
}

func toTimes(atim, mtime int64, fstFlags uint16) (times [2]syscall.Timespec, errno syscall.Errno) {
	// times[0] == atim, times[1] == mtim

	// coerce atim into a timespec
	if set, now := fstFlags&wasip1.FstflagsAtim != 0, fstFlags&wasip1.FstflagsAtimNow != 0; set && now {
		errno = syscall.EINVAL
		return
	} else if set {
		times[0] = syscall.NsecToTimespec(atim)
	} else if now {
		times[0].Nsec = platform.UTIME_NOW
	} else {
		times[0].Nsec = platform.UTIME_OMIT
	}

	// coerce mtim into a timespec
	if set, now := fstFlags&wasip1.FstflagsMtim != 0, fstFlags&wasip1.FstflagsMtimNow != 0; set && now {
		errno = syscall.EINVAL
		return
	} else if set {
		times[1] = syscall.NsecToTimespec(mtime)
	} else if now {
		times[1].Nsec = platform.UTIME_NOW
	} else {
		times[1].Nsec = platform.UTIME_OMIT
	}
	return
}

// fdPread is the WASI function named FdPreadName which reads from a file
// descriptor, without using and updating the file descriptor's offset.
//
// Except for handling offset, this implementation is identical to fdRead.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_preadfd-fd-iovs-iovec_array-offset-filesize---errno-size
var fdPread = newHostFunc(
	wasip1.FdPreadName, fdPreadFn,
	[]api.ValueType{i32, i32, i32, i64, i32},
	"fd", "iovs", "iovs_len", "offset", "result.nread",
)

func fdPreadFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid or the `fd` is not a pre-opened directory
//   - syscall.EFAULT: `resultPrestat` points to an offset out of memory
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
var fdPrestatGet = newHostFunc(wasip1.FdPrestatGetName, fdPrestatGetFn, []api.ValueType{i32, i32}, "fd", "result.prestat")

func fdPrestatGetFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()
	fd, resultPrestat := uint32(params[0]), uint32(params[1])

	name, errno := preopenPath(fsc, fd)
	if errno != 0 {
		return errno
	}

	// Upper 32-bits are zero because...
	// * Zero-value 8-bit tag, and 3-byte zero-value padding
	prestat := uint64(len(name) << 32)
	if !mod.Memory().WriteUint64Le(resultPrestat, prestat) {
		return syscall.EFAULT
	}
	return 0
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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid
//   - syscall.EFAULT: `path` points to an offset out of memory
//   - syscall.ENAMETOOLONG: `pathLen` is longer than the actual length of the result
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
	wasip1.FdPrestatDirNameName, fdPrestatDirNameFn,
	[]api.ValueType{i32, i32, i32},
	"fd", "result.path", "result.path_len",
)

func fdPrestatDirNameFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()
	fd, path, pathLen := uint32(params[0]), uint32(params[1]), uint32(params[2])

	name, errno := preopenPath(fsc, fd)
	if errno != 0 {
		return errno
	}

	// Some runtimes may have another semantics. See /RATIONALE.md
	if uint32(len(name)) < pathLen {
		return syscall.ENAMETOOLONG
	}

	if !mod.Memory().Write(path, []byte(name)[:pathLen]) {
		return syscall.EFAULT
	}
	return 0
}

// fdPwrite is the WASI function named FdPwriteName which writes to a file
// descriptor, without using and updating the file descriptor's offset.
//
// Except for handling offset, this implementation is identical to fdWrite.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_pwritefd-fd-iovs-ciovec_array-offset-filesize---errno-size
var fdPwrite = newHostFunc(
	wasip1.FdPwriteName, fdPwriteFn,
	[]api.ValueType{i32, i32, i32, i64, i32},
	"fd", "iovs", "iovs_len", "offset", "result.nwritten",
)

func fdPwriteFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	return fdWriteOrPwrite(mod, params, true)
}

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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid
//   - syscall.EFAULT: `iovs` or `resultNread` point to an offset out of memory
//   - syscall.EIO: a file system error
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
	wasip1.FdReadName, fdReadFn,
	[]api.ValueType{i32, i32, i32, i32},
	"fd", "iovs", "iovs_len", "result.nread",
)

func fdReadFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	return fdReadOrPread(mod, params, false)
}

func fdReadOrPread(mod api.Module, params []uint64, isPread bool) syscall.Errno {
	mem := mod.Memory()
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd := uint32(params[0])

	r, ok := fsc.LookupFile(fd)
	if !ok {
		return syscall.EBADF
	}

	var reader io.Reader = r.File

	iovs := uint32(params[1])
	iovsCount := uint32(params[2])

	var resultNread uint32
	if isPread {
		offset := int64(params[3])
		reader = sysfs.ReaderAtOffset(r.File, offset)
		resultNread = uint32(params[4])
	} else {
		resultNread = uint32(params[3])
	}

	var nread uint32
	iovsStop := iovsCount << 3 // iovsCount * 8
	iovsBuf, ok := mem.Read(iovs, iovsStop)
	if !ok {
		return syscall.EFAULT
	}

	for iovsPos := uint32(0); iovsPos < iovsStop; iovsPos += 8 {
		offset := le.Uint32(iovsBuf[iovsPos:])
		l := le.Uint32(iovsBuf[iovsPos+4:])

		b, ok := mem.Read(offset, l)
		if !ok {
			return syscall.EFAULT
		}

		n, err := reader.Read(b)
		nread += uint32(n)

		shouldContinue, errno := fdRead_shouldContinueRead(uint32(n), l, err)
		if errno != 0 {
			return errno
		} else if !shouldContinue {
			break
		}
	}
	if !mem.WriteUint32Le(resultNread, nread) {
		return syscall.EFAULT
	} else {
		return 0
	}
}

// fdRead_shouldContinueRead decides whether to continue reading the next iovec
// based on the amount read (n/l) and a possible error returned from io.Reader.
//
// Note: When there are both bytes read (n) and an error, this continues.
// See /RATIONALE.md "Why ignore the error returned by io.Reader when n > 1?"
func fdRead_shouldContinueRead(n, l uint32, err error) (bool, syscall.Errno) {
	if errors.Is(err, io.EOF) {
		return false, 0 // EOF isn't an error, and we shouldn't continue.
	} else if err != nil && n == 0 {
		return false, syscall.EIO
	} else if err != nil {
		return false, 0 // Allow the caller to process n bytes.
	}
	// Continue reading, unless there's a partial read or nothing to read.
	return n == l && n != 0, 0
}

// fdReaddir is the WASI function named FdReaddirName which reads directory
// entries from a directory.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_readdirfd-fd-buf-pointeru8-buf_len-size-cookie-dircookie---errno-size
var fdReaddir = newHostFunc(
	wasip1.FdReaddirName, fdReaddirFn,
	[]wasm.ValueType{i32, i32, i32, i64, i32},
	"fd", "buf", "buf_len", "cookie", "result.bufused",
)

func fdReaddirFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
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
	if bufLen < wasip1.DirentSize {
		return syscall.EINVAL
	}

	// Validate the FD is a directory
	rd, dir, errno := openedDir(fsc, fd)
	if errno != 0 {
		return errno
	}

	if cookie == 0 && dir.CountRead > 0 {
		// This means that there was a previous call to the dir, but cookie is reset.
		// This happens when the program calls rewinddir, for example:
		// https://github.com/WebAssembly/wasi-libc/blob/659ff414560721b1660a19685110e484a081c3d4/libc-bottom-half/cloudlibc/src/libc/dirent/rewinddir.c#L10-L12
		//
		// Since we cannot unwind fs.ReadDirFile results, we re-open while keeping the same file descriptor.
		f, errno := fsc.ReOpenDir(fd)
		if errno != 0 {
			return errno
		}
		rd, dir = f.File, f.ReadDir
	}

	// First, determine the maximum directory entries that can be encoded as
	// dirents. The total size is DirentSize(24) + nameSize, for each file.
	// Since a zero-length file name is invalid, the minimum size entry is
	// 25 (DirentSize + 1 character).
	maxDirEntries := int(bufLen/wasip1.DirentSize + 1)

	// While unlikely maxDirEntries will fit into bufLen, add one more just in
	// case, as we need to know if we hit the end of the directory or not to
	// write the correct bufused (e.g. == bufLen unless EOF).
	//	>> If less than the size of the read buffer, the end of the
	//	>> directory has been reached.
	maxDirEntries += 1

	// The host keeps state for any unread entries from the prior call because
	// we cannot seek to a previous directory position. Collect these entries.
	dirents, errno := lastDirents(dir, cookie)
	if errno != 0 {
		return errno
	}

	// Add entries for dot and dot-dot as wasi-testsuite requires them.
	if cookie == 0 && dirents == nil {
		if f, ok := fsc.LookupFile(fd); !ok {
			return syscall.EBADF
		} else if dirents, errno = dotDirents(f); errno != 0 {
			return errno
		}
		dir.Dirents = dirents
		dir.CountRead = 2 // . and ..
	}

	// Check if we have maxDirEntries, and read more from the FS as needed.
	if entryCount := len(dirents); entryCount < maxDirEntries {
		// Note: platform.Readdir does not return io.EOF as it is
		// inconsistently returned (e.g. darwin does, but linux doesn't).
		l, errno := platform.Readdir(rd, maxDirEntries-entryCount)
		if errno != 0 {
			return errno
		}

		// Zero length read is possible on an empty or exhausted directory.
		if len(l) > 0 {
			dir.CountRead += uint64(len(l))
			dirents = append(dirents, l...)
			// Replace the cache with up to maxDirEntries, starting at cookie.
			dir.Dirents = dirents
		}
	}

	// Determine how many dirents we can write, excluding a potentially
	// truncated entry.
	bufused, direntCount, writeTruncatedEntry := maxDirents(dirents, bufLen)

	// Now, write entries to the underlying buffer.
	if bufused > 0 {

		// d_next is the index of the next file in the list, so it should
		// always be one higher than the requested cookie.
		d_next := uint64(cookie + 1)
		// ^^ yes this can overflow to negative, which means our implementation
		// doesn't support writing greater than max int64 entries.

		buf, ok := mem.Read(buf, bufused)
		if !ok {
			return syscall.EFAULT
		}

		writeDirents(dirents, direntCount, writeTruncatedEntry, buf, d_next)
	}

	if !mem.WriteUint32Le(resultBufused, bufused) {
		return syscall.EFAULT
	}
	return 0
}

// dotDirents returns "." and "..", where "." because wasi-testsuite does inode
// validation.
func dotDirents(f *sys.FileEntry) ([]*platform.Dirent, syscall.Errno) {
	dotIno, ft, err := f.CachedStat()
	if err != nil {
		return nil, platform.UnwrapOSError(err)
	} else if ft.Type() != fs.ModeDir {
		return nil, syscall.ENOTDIR
	}
	dotDotIno := uint64(0)
	if !f.IsPreopen && f.Name != "." {
		if st, errno := f.FS.Stat(path.Dir(f.Name)); errno != 0 {
			return nil, errno
		} else {
			dotDotIno = st.Ino
		}
	}
	return []*platform.Dirent{
		{Name: ".", Ino: dotIno, Type: fs.ModeDir},
		{Name: "..", Ino: dotDotIno, Type: fs.ModeDir},
	}, 0
}

const largestDirent = int64(math.MaxUint32 - wasip1.DirentSize)

// lastDirents is broken out from fdReaddirFn for testability.
func lastDirents(dir *sys.ReadDir, cookie int64) (dirents []*platform.Dirent, errno syscall.Errno) {
	if cookie < 0 {
		errno = syscall.EINVAL // invalid as we will never send a negative cookie.
		return
	}

	entryCount := int64(len(dir.Dirents))
	if entryCount == 0 { // there was no prior call
		if cookie != 0 {
			errno = syscall.EINVAL // invalid as we haven't sent that cookie
		}
		return
	}

	// Get the first absolute position in our window of results
	firstPos := int64(dir.CountRead) - entryCount
	cookiePos := cookie - firstPos

	switch {
	case cookiePos < 0: // cookie is asking for results outside our window.
		errno = syscall.ENOSYS // we can't implement directory seeking backwards.
	case cookiePos > entryCount:
		errno = syscall.EINVAL // invalid as we read that far, yet.
	case cookiePos > 0: // truncate so to avoid large lists.
		dirents = dir.Dirents[cookiePos:]
	default:
		dirents = dir.Dirents
	}
	if len(dirents) == 0 {
		dirents = nil
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
func maxDirents(entries []*platform.Dirent, bufLen uint32) (bufused, direntCount uint32, writeTruncatedEntry bool) {
	lenRemaining := bufLen
	for _, e := range entries {
		if lenRemaining < wasip1.DirentSize {
			// We don't have enough space in bufLen for another struct,
			// entry. A caller who wants more will retry.

			// bufused == bufLen means more entries exist, which is the case
			// when the dirent is larger than bytes remaining.
			bufused = bufLen
			break
		}

		// use int64 to guard against huge filenames
		nameLen := int64(len(e.Name))
		var entryLen uint32

		// Check to see if DirentSize + nameLen overflows, or if it would be
		// larger than possible to encode.
		if el := int64(wasip1.DirentSize) + nameLen; el < 0 || el > largestDirent {
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
	dirents []*platform.Dirent,
	direntCount uint32,
	writeTruncatedEntry bool,
	buf []byte,
	d_next uint64,
) {
	pos, i := uint32(0), uint32(0)
	for ; i < direntCount; i++ {
		e := dirents[i]
		nameLen := uint32(len(e.Name))

		writeDirent(buf[pos:], d_next, e.Ino, nameLen, e.Type)
		pos += wasip1.DirentSize

		copy(buf[pos:], e.Name)
		pos += nameLen
		d_next++
	}

	if !writeTruncatedEntry {
		return
	}

	// Write a dirent without its name
	dirent := make([]byte, wasip1.DirentSize)
	e := dirents[i]
	writeDirent(dirent, d_next, e.Ino, uint32(len(e.Name)), e.Type)

	// Potentially truncate it
	copy(buf[pos:], dirent)
}

// writeDirent writes DirentSize bytes
func writeDirent(buf []byte, dNext uint64, ino uint64, dNamlen uint32, dType fs.FileMode) {
	le.PutUint64(buf, dNext)        // d_next
	le.PutUint64(buf[8:], ino)      // d_ino
	le.PutUint32(buf[16:], dNamlen) // d_namlen
	filetype := getWasiFiletype(dType)
	le.PutUint32(buf[20:], uint32(filetype)) //  d_type
}

// openedDir returns the directory and 0 if the fd points to a readable directory.
func openedDir(fsc *sys.FSContext, fd uint32) (fs.File, *sys.ReadDir, syscall.Errno) {
	if f, ok := fsc.LookupFile(fd); !ok {
		return nil, nil, syscall.EBADF
	} else if _, ft, err := f.CachedStat(); err != nil {
		return nil, nil, platform.UnwrapOSError(err)
	} else if ft.Type() != fs.ModeDir {
		// fd_readdir docs don't indicate whether to return syscall.ENOTDIR or
		// syscall.EBADF. It has been noticed that rust will crash on syscall.ENOTDIR,
		// and POSIX C ref seems to not return this, so we don't either.
		//
		// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#fd_readdir
		// and https://en.wikibooks.org/wiki/C_Programming/POSIX_Reference/dirent.h
		return nil, nil, syscall.EBADF
	} else {
		if f.ReadDir == nil {
			f.ReadDir = &sys.ReadDir{}
		}
		return f.File, f.ReadDir, 0
	}
}

// fdRenumber is the WASI function named FdRenumberName which atomically
// replaces a file descriptor by renumbering another file descriptor.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_renumberfd-fd-to-fd---errno
var fdRenumber = newHostFunc(wasip1.FdRenumberName, fdRenumberFn, []wasm.ValueType{i32, i32}, "fd", "to")

func fdRenumberFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	from := uint32(params[0])
	to := uint32(params[1])

	if errno := fsc.Renumber(from, to); errno != 0 {
		return errno
	}
	return 0
}

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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid
//   - syscall.EFAULT: `resultNewoffset` points to an offset out of memory
//   - syscall.EINVAL: `whence` is an invalid value
//   - syscall.EIO: a file system error
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
	wasip1.FdSeekName, fdSeekFn,
	[]api.ValueType{i32, i64, i32, i32},
	"fd", "offset", "whence", "result.newoffset",
)

func fdSeekFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()
	fd := uint32(params[0])
	offset := params[1]
	whence := uint32(params[2])
	resultNewoffset := uint32(params[3])

	var seeker io.Seeker
	// Check to see if the file descriptor is available
	if f, ok := fsc.LookupFile(fd); !ok {
		return syscall.EBADF
		// fs.FS doesn't declare io.Seeker, but implementations such as os.File implement it.
	} else if _, ft, err := f.CachedStat(); err != nil {
		return platform.UnwrapOSError(err)
	} else if ft.Type() == fs.ModeDir {
		return syscall.EBADF
	} else if seeker, ok = f.File.(io.Seeker); !ok {
		return syscall.EBADF
	}

	if whence > io.SeekEnd /* exceeds the largest valid whence */ {
		return syscall.EINVAL
	}

	newOffset, err := seeker.Seek(int64(offset), int(whence))
	if err != nil {
		return platform.UnwrapOSError(err)
	}

	if !mod.Memory().WriteUint64Le(resultNewoffset, uint64(newOffset)) {
		return syscall.EFAULT
	}
	return 0
}

// fdSync is the WASI function named FdSyncName which synchronizes the data
// and metadata of a file to disk.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_syncfd-fd---errno
var fdSync = newHostFunc(wasip1.FdSyncName, fdSyncFn, []api.ValueType{i32}, "fd")

func fdSyncFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()
	fd := uint32(params[0])

	// Check to see if the file descriptor is available
	if f, ok := fsc.LookupFile(fd); !ok {
		return syscall.EBADF
	} else if syncFile, ok := f.File.(syncFile); !ok {
		return syscall.EBADF // possibly a fake file
	} else if err := syncFile.Sync(); err != nil {
		return platform.UnwrapOSError(err)
	}
	return 0
}

// fdTell is the WASI function named FdTellName which returns the current
// offset of a file descriptor.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-fd_tellfd-fd---errno-filesize
var fdTell = newHostFunc(wasip1.FdTellName, fdTellFn, []api.ValueType{i32, i32}, "fd", "result.offset")

func fdTellFn(ctx context.Context, mod api.Module, params []uint64) syscall.Errno {
	fd := params[0]
	offset := uint64(0)
	whence := uint64(io.SeekCurrent)
	resultNewoffset := params[1]

	fdSeekParams := []uint64{fd, offset, whence, resultNewoffset}
	return fdSeekFn(ctx, mod, fdSeekParams)
}

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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid
//   - syscall.EFAULT: `iovs` or `resultNwritten` point to an offset out of memory
//   - syscall.EIO: a file system error
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
	wasip1.FdWriteName, fdWriteFn,
	[]api.ValueType{i32, i32, i32, i32},
	"fd", "iovs", "iovs_len", "result.nwritten",
)

func fdWriteFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	return fdWriteOrPwrite(mod, params, false)
}

func fdWriteOrPwrite(mod api.Module, params []uint64, isPwrite bool) syscall.Errno {
	mem := mod.Memory()
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd := uint32(params[0])
	iovs := uint32(params[1])
	iovsCount := uint32(params[2])

	var resultNwritten uint32
	var writer io.Writer
	if f, ok := fsc.LookupFile(fd); !ok {
		return syscall.EBADF
	} else if isPwrite {
		offset := int64(params[3])
		writer = sysfs.WriterAtOffset(f.File, offset)
		resultNwritten = uint32(params[4])
	} else if writer, ok = f.File.(io.Writer); !ok {
		return syscall.EBADF
	} else {
		resultNwritten = uint32(params[3])
	}

	var err error
	var nwritten uint32
	iovsStop := iovsCount << 3 // iovsCount * 8
	iovsBuf, ok := mem.Read(iovs, iovsStop)
	if !ok {
		return syscall.EFAULT
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
				return syscall.EFAULT
			}
			n, err = writer.Write(b)
			if err != nil {
				return platform.UnwrapOSError(err)
			}
		}
		nwritten += uint32(n)
	}

	if !mod.Memory().WriteUint32Le(resultNwritten, nwritten) {
		return syscall.EFAULT
	}
	return 0
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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid
//   - syscall.ENOENT: `path` does not exist.
//   - syscall.ENOTDIR: `path` is a file
//
// # Notes
//   - This is similar to mkdirat in POSIX.
//     See https://linux.die.net/man/2/mkdirat
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_create_directoryfd-fd-path-string---errno
var pathCreateDirectory = newHostFunc(
	wasip1.PathCreateDirectoryName, pathCreateDirectoryFn,
	[]wasm.ValueType{i32, i32, i32},
	"fd", "path", "path_len",
)

func pathCreateDirectoryFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd := uint32(params[0])
	path := uint32(params[1])
	pathLen := uint32(params[2])

	preopen, pathName, errno := atPath(fsc, mod.Memory(), fd, path, pathLen)
	if errno != 0 {
		return errno
	}

	if errno = preopen.Mkdir(pathName, 0o700); errno != 0 {
		return errno
	}

	return 0
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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid
//   - syscall.ENOTDIR: `fd` points to a file not a directory
//   - syscall.EIO: could not stat `fd` on filesystem
//   - syscall.EINVAL: the path contained "../"
//   - syscall.ENAMETOOLONG: `path` + `path_len` is out of memory
//   - syscall.EFAULT: `resultFilestat` points to an offset out of memory
//   - syscall.ENOENT: could not find the path
//
// The rest of this implementation matches that of fdFilestatGet, so is not
// repeated here.
//
// Note: This is similar to `fstatat` in POSIX.
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_filestat_getfd-fd-flags-lookupflags-path-string---errno-filestat
// and https://linux.die.net/man/2/fstatat
var pathFilestatGet = newHostFunc(
	wasip1.PathFilestatGetName, pathFilestatGetFn,
	[]api.ValueType{i32, i32, i32, i32, i32},
	"fd", "flags", "path", "path_len", "result.filestat",
)

func pathFilestatGetFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd := uint32(params[0])
	flags := uint16(params[1])
	path := uint32(params[2])
	pathLen := uint32(params[3])

	preopen, pathName, errno := atPath(fsc, mod.Memory(), fd, path, pathLen)
	if errno != 0 {
		return errno
	}

	// Stat the file without allocating a file descriptor.
	//
	// Note: `preopen` is a `sysfs.FS` interface, so passing the address of `st`
	// causes the value to escape to the heap because the compiler doesn't know
	// whether the pointer will be retained by the method.
	//
	// This could be optimized by modifying Stat/Lstat to return the `Stat_t`
	// value instead of passing a pointer as output parameter.
	var st platform.Stat_t

	if (flags & wasip1.LOOKUP_SYMLINK_FOLLOW) == 0 {
		st, errno = preopen.Lstat(pathName)
	} else {
		st, errno = preopen.Stat(pathName)
	}
	if errno != 0 {
		return errno
	}

	// Write the stat result to memory
	resultBuf := uint32(params[4])
	buf, ok := mod.Memory().Read(resultBuf, 64)
	if !ok {
		return syscall.EFAULT
	}

	return writeFilestat(buf, &st)
}

// pathFilestatSetTimes is the WASI function named PathFilestatSetTimesName
// which adjusts the timestamps of a file or directory.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_filestat_set_timesfd-fd-flags-lookupflags-path-string-atim-timestamp-mtim-timestamp-fst_flags-fstflags---errno
var pathFilestatSetTimes = newHostFunc(
	wasip1.PathFilestatSetTimesName, pathFilestatSetTimesFn,
	[]wasm.ValueType{i32, i32, i32, i32, i64, i64, i32},
	"fd", "flags", "path", "path_len", "atim", "mtim", "fst_flags",
)

func pathFilestatSetTimesFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fd := uint32(params[0])
	flags := uint16(params[1])
	path := uint32(params[2])
	pathLen := uint32(params[3])
	atim := int64(params[4])
	mtim := int64(params[5])
	fstFlags := uint16(params[6])

	sys := mod.(*wasm.CallContext).Sys
	fsc := sys.FS()

	times, errno := toTimes(atim, mtim, fstFlags)
	if errno != 0 {
		return errno
	}

	preopen, pathName, errno := atPath(fsc, mod.Memory(), fd, path, pathLen)
	if errno != 0 {
		return errno
	}

	symlinkFollow := flags&wasip1.LOOKUP_SYMLINK_FOLLOW != 0
	return preopen.Utimens(pathName, &times, symlinkFollow)
}

// pathLink is the WASI function named PathLinkName which adjusts the
// timestamps of a file or directory.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#path_link
var pathLink = newHostFunc(
	wasip1.PathLinkName, pathLinkFn,
	[]wasm.ValueType{i32, i32, i32, i32, i32, i32, i32},
	"old_fd", "old_flags", "old_path", "old_path_len", "new_fd", "new_path", "new_path_len",
)

func pathLinkFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	mem := mod.Memory()
	fsc := mod.(*wasm.CallContext).Sys.FS()

	oldFd := uint32(params[0])
	// TODO: use old_flags?
	_ = uint32(params[1])
	oldPath := uint32(params[2])
	oldPathLen := uint32(params[3])

	oldFS, oldName, errno := atPath(fsc, mem, oldFd, oldPath, oldPathLen)
	if errno != 0 {
		return errno
	}

	newFD := uint32(params[4])
	newPath := uint32(params[5])
	newPathLen := uint32(params[6])

	newFS, newName, errno := atPath(fsc, mem, newFD, newPath, newPathLen)
	if errno != 0 {
		return errno
	}

	if oldFS != newFS { // TODO: handle link across filesystems
		return syscall.ENOSYS
	}

	return oldFS.Link(oldName, newName)
}

// pathOpen is the WASI function named PathOpenName which opens a file or
// directory. This returns syscall.EBADF if the fd is invalid.
//
// # Parameters
//
//   - fd: file descriptor of a directory that `path` is relative to
//   - dirflags: flags to indicate how to resolve `path`
//   - path: offset in api.Memory to read the path string from
//   - pathLen: length of `path`
//   - oFlags: open flags to indicate the method by which to open the file
//   - fsRightsBase: interpret RIGHT_FD_WRITE to set O_RDWR
//   - fsRightsInheriting: ignored as rights were removed from WASI.
//     created file descriptor for `path`
//   - fdFlags: file descriptor flags
//   - resultOpenedFd: offset in api.Memory to write the newly created file
//     descriptor to.
//   - The result FD value is guaranteed to be less than 2**31
//
// Result (Errno)
//
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid
//   - syscall.EFAULT: `resultOpenedFd` points to an offset out of memory
//   - syscall.ENOENT: `path` does not exist.
//   - syscall.EEXIST: `path` exists, while `oFlags` requires that it must not.
//   - syscall.ENOTDIR: `path` is not a directory, while `oFlags` requires it.
//   - syscall.EIO: a file system error
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
	wasip1.PathOpenName, pathOpenFn,
	[]api.ValueType{i32, i32, i32, i32, i32, i64, i64, i32, i32},
	"fd", "dirflags", "path", "path_len", "oflags", "fs_rights_base", "fs_rights_inheriting", "fdflags", "result.opened_fd",
)

func pathOpenFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	preopenFD := uint32(params[0])

	// TODO: dirflags is a lookupflags, and it only has one bit: symlink_follow
	// https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#lookupflags
	dirflags := uint16(params[1])

	path := uint32(params[2])
	pathLen := uint32(params[3])

	oflags := uint16(params[4])

	rights := uint32(params[5])
	// inherited rights aren't used
	_ = params[6]

	fdflags := uint16(params[7])
	resultOpenedFd := uint32(params[8])

	preopen, pathName, errno := atPath(fsc, mod.Memory(), preopenFD, path, pathLen)
	if errno != 0 {
		return errno
	}

	fileOpenFlags := openFlags(dirflags, oflags, fdflags, rights)
	isDir := fileOpenFlags&platform.O_DIRECTORY != 0

	if isDir && oflags&wasip1.O_CREAT != 0 {
		return syscall.EINVAL // use pathCreateDirectory!
	}

	newFD, errno := fsc.OpenFile(preopen, pathName, fileOpenFlags, 0o600)
	if errno != 0 {
		return errno
	}

	// Check any flags that require the file to evaluate.
	if isDir {
		if f, ok := fsc.LookupFile(newFD); !ok {
			return syscall.EBADF // unexpected
		} else if _, ft, err := f.CachedStat(); err != nil {
			_ = fsc.CloseFile(newFD)
			return platform.UnwrapOSError(err)
		} else if ft.Type() != fs.ModeDir {
			_ = fsc.CloseFile(newFD)
			return syscall.ENOTDIR
		}
	}

	if !mod.Memory().WriteUint32Le(resultOpenedFd, newFD) {
		_ = fsc.CloseFile(newFD)
		return syscall.EFAULT
	}
	return 0
}

// atPath returns the pre-open specific path after verifying it is a directory.
//
// # Notes
//
// Languages including Zig and Rust use only pre-opens for the FD because
// wasi-libc `__wasilibc_find_relpath` will only return a preopen. That said,
// our wasi.c example shows other languages act differently and can use a non
// pre-opened file descriptor.
//
// We don't handle `AT_FDCWD`, as that's resolved in the compiler. There's no
// working directory function in WASI, so most assume CWD is "/". Notably, Zig
// has different behavior which assumes it is whatever the first pre-open name
// is.
//
// See https://github.com/WebAssembly/wasi-libc/blob/659ff414560721b1660a19685110e484a081c3d4/libc-bottom-half/sources/at_fdcwd.c
// See https://linux.die.net/man/2/openat
func atPath(fsc *sys.FSContext, mem api.Memory, fd, p, pathLen uint32) (sysfs.FS, string, syscall.Errno) {
	b, ok := mem.Read(p, pathLen)
	if !ok {
		return nil, "", syscall.EFAULT
	}
	pathName := string(b)

	// interesting_paths wants us to break on trailing slash if the input ends
	// up a file, not a directory!
	hasTrailingSlash := strings.HasSuffix(pathName, "/")

	// interesting_paths includes paths that include relative links but end up
	// not escaping
	pathName = path.Clean(pathName)

	// interesting_paths wants to break on root paths or anything that escapes.
	// This part is the same as fs.FS.Open()
	if !fs.ValidPath(pathName) {
		return nil, "", syscall.EPERM
	}

	// add the trailing slash back
	if hasTrailingSlash {
		pathName = pathName + "/"
	}

	if f, ok := fsc.LookupFile(fd); !ok {
		return nil, "", syscall.EBADF // closed
	} else if _, ft, err := f.CachedStat(); err != nil {
		return nil, "", platform.UnwrapOSError(err)
	} else if ft.Type() != fs.ModeDir {
		return nil, "", syscall.ENOTDIR
	} else if f.IsPreopen { // don't append the pre-open name
		return f.FS, pathName, 0
	} else {
		// Join via concat to avoid name conflict on path.Join
		return f.FS, f.Name + "/" + pathName, 0
	}
}

func preopenPath(fsc *sys.FSContext, fd uint32) (string, syscall.Errno) {
	if f, ok := fsc.LookupFile(fd); !ok {
		return "", syscall.EBADF // closed
	} else if !f.IsPreopen {
		return "", syscall.EBADF
	} else {
		return f.Name, 0
	}
}

func openFlags(dirflags, oflags, fdflags uint16, rights uint32) (openFlags int) {
	if dirflags&wasip1.LOOKUP_SYMLINK_FOLLOW == 0 {
		openFlags |= platform.O_NOFOLLOW
	}
	if oflags&wasip1.O_DIRECTORY != 0 {
		openFlags |= platform.O_DIRECTORY
		return // Early return for directories as the rest of flags doesn't make sense for it.
	} else if oflags&wasip1.O_EXCL != 0 {
		openFlags |= syscall.O_EXCL
	}
	if oflags&wasip1.O_TRUNC != 0 {
		openFlags |= syscall.O_RDWR | syscall.O_TRUNC
	}
	if oflags&wasip1.O_CREAT != 0 {
		openFlags |= syscall.O_RDWR | syscall.O_CREAT
	}
	if fdflags&wasip1.FD_APPEND != 0 {
		openFlags |= syscall.O_RDWR | syscall.O_APPEND
	}
	// Since rights were discontinued in wasi, we only interpret RIGHT_FD_WRITE
	// because it is the only way to know that we need to set write permissions
	// on a file if the application did not pass any of O_CREATE, O_APPEND, nor
	// O_TRUNC.
	if rights&wasip1.RIGHT_FD_WRITE != 0 {
		openFlags |= syscall.O_RDWR
	}
	if openFlags == 0 {
		openFlags = syscall.O_RDONLY
	}
	return
}

// pathReadlink is the WASI function named PathReadlinkName that reads the
// contents of a symbolic link.
//
// See: https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_readlinkfd-fd-path-string-buf-pointeru8-buf_len-size---errno-size
var pathReadlink = newHostFunc(
	wasip1.PathReadlinkName, pathReadlinkFn,
	[]wasm.ValueType{i32, i32, i32, i32, i32, i32},
	"fd", "path", "path_len", "buf", "buf_len", "result.bufused",
)

func pathReadlinkFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd := uint32(params[0])
	path := uint32(params[1])
	pathLen := uint32(params[2])
	buf := uint32(params[3])
	bufLen := uint32(params[4])
	resultBufused := uint32(params[5])

	if pathLen == 0 || bufLen == 0 {
		return syscall.EINVAL
	}

	mem := mod.Memory()
	preopen, p, errno := atPath(fsc, mem, fd, path, pathLen)
	if errno != 0 {
		return errno
	}

	dst, errno := preopen.Readlink(p)
	if errno != 0 {
		return errno
	}

	if ok := mem.WriteString(buf, dst); !ok {
		return syscall.EFAULT
	}

	if !mem.WriteUint32Le(resultBufused, uint32(len(dst))) {
		return syscall.EFAULT
	}
	return 0
}

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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid
//   - syscall.ENOENT: `path` does not exist.
//   - syscall.ENOTEMPTY: `path` is not empty
//   - syscall.ENOTDIR: `path` is a file
//
// # Notes
//   - This is similar to unlinkat with AT_REMOVEDIR in POSIX.
//     See https://linux.die.net/man/2/unlinkat
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_remove_directoryfd-fd-path-string---errno
var pathRemoveDirectory = newHostFunc(
	wasip1.PathRemoveDirectoryName, pathRemoveDirectoryFn,
	[]wasm.ValueType{i32, i32, i32},
	"fd", "path", "path_len",
)

func pathRemoveDirectoryFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd := uint32(params[0])
	path := uint32(params[1])
	pathLen := uint32(params[2])

	preopen, pathName, errno := atPath(fsc, mod.Memory(), fd, path, pathLen)
	if errno != 0 {
		return errno
	}

	return preopen.Rmdir(pathName)
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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` or `new_fd` are invalid
//   - syscall.ENOENT: `old_path` does not exist.
//   - syscall.ENOTDIR: `old` is a directory and `new` exists, but is a file.
//   - syscall.EISDIR: `old` is a file and `new` exists, but is a directory.
//
// # Notes
//   - This is similar to unlinkat in POSIX.
//     See https://linux.die.net/man/2/renameat
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_renamefd-fd-old_path-string-new_fd-fd-new_path-string---errno
var pathRename = newHostFunc(
	wasip1.PathRenameName, pathRenameFn,
	[]wasm.ValueType{i32, i32, i32, i32, i32, i32},
	"fd", "old_path", "old_path_len", "new_fd", "new_path", "new_path_len",
)

func pathRenameFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd := uint32(params[0])
	oldPath := uint32(params[1])
	oldPathLen := uint32(params[2])

	newFD := uint32(params[3])
	newPath := uint32(params[4])
	newPathLen := uint32(params[5])

	oldFS, oldPathName, errno := atPath(fsc, mod.Memory(), fd, oldPath, oldPathLen)
	if errno != 0 {
		return errno
	}

	newFS, newPathName, errno := atPath(fsc, mod.Memory(), newFD, newPath, newPathLen)
	if errno != 0 {
		return errno
	}

	if oldFS != newFS { // TODO: handle renames across filesystems
		return syscall.ENOSYS
	}

	return oldFS.Rename(oldPathName, newPathName)
}

// pathSymlink is the WASI function named PathSymlinkName which creates a
// symbolic link.
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#path_symlink
var pathSymlink = newHostFunc(
	wasip1.PathSymlinkName, pathSymlinkFn,
	[]wasm.ValueType{i32, i32, i32, i32, i32},
	"old_path", "old_path_len", "fd", "new_path", "new_path_len",
)

func pathSymlinkFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	oldPath := uint32(params[0])
	oldPathLen := uint32(params[1])
	fd := uint32(params[2])
	newPath := uint32(params[3])
	newPathLen := uint32(params[4])

	mem := mod.Memory()

	dir, ok := fsc.LookupFile(fd)
	if !ok {
		return syscall.EBADF // closed
	} else if _, ft, err := dir.CachedStat(); err != nil {
		return platform.UnwrapOSError(err)
	} else if ft.Type() != fs.ModeDir {
		return syscall.ENOTDIR
	}

	if oldPathLen == 0 || newPathLen == 0 {
		return syscall.EINVAL
	}

	oldPathBuf, ok := mem.Read(oldPath, oldPathLen)
	if !ok {
		return syscall.EFAULT
	}

	newPathBuf, ok := mem.Read(newPath, newPathLen)
	if !ok {
		return syscall.EFAULT
	}

	return dir.FS.Symlink(
		// Do not join old path since it's only resolved when dereference the link created here.
		// And the dereference result depends on the opening directory's file descriptor at that point.
		bufToStr(oldPathBuf, int(oldPathLen)),
		path.Join(dir.Name, bufToStr(newPathBuf, int(newPathLen))),
	)
}

// bufToStr converts the given byte slice as string unsafely.
func bufToStr(buf []byte, l int) string {
	return *(*string)(unsafe.Pointer(&reflect.SliceHeader{ //nolint
		Data: uintptr(unsafe.Pointer(&buf[0])),
		Len:  l,
		Cap:  l,
	}))
}

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
// The return value is 0 except the following error conditions:
//   - syscall.EBADF: `fd` is invalid
//   - syscall.ENOENT: `path` does not exist.
//   - syscall.EISDIR: `path` is a directory
//
// # Notes
//   - This is similar to unlinkat without AT_REMOVEDIR in POSIX.
//     See https://linux.die.net/man/2/unlinkat
//
// See https://github.com/WebAssembly/WASI/blob/snapshot-01/phases/snapshot/docs.md#-path_unlink_filefd-fd-path-string---errno
var pathUnlinkFile = newHostFunc(
	wasip1.PathUnlinkFileName, pathUnlinkFileFn,
	[]wasm.ValueType{i32, i32, i32},
	"fd", "path", "path_len",
)

func pathUnlinkFileFn(_ context.Context, mod api.Module, params []uint64) syscall.Errno {
	fsc := mod.(*wasm.CallContext).Sys.FS()

	fd := uint32(params[0])
	path := uint32(params[1])
	pathLen := uint32(params[2])

	preopen, pathName, errno := atPath(fsc, mod.Memory(), fd, path, pathLen)
	if errno != 0 {
		return errno
	}

	return preopen.Unlink(pathName)
}
