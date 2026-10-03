package vfs

import (
	"io"
	"reflect"
	"strings"
	_ "unsafe"

	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
	"github.com/ncruces/go-sqlite3/internal/util"
)

//go:linkname vfsFullPathname
func vfsFullPathname(wrp *sqlite3_wrap.Wrapper, pVfs, zRelative ptr_t, nFull int32, zFull ptr_t) _ErrorCode {
	vfs := vfsGet(wrp, pVfs)
	path := wrp.ReadString(zRelative, _MAX_PATHNAME)

	path, err := vfs.FullPathname(path)

	if len(path) >= int(nFull) {
		return _CANTOPEN_FULLPATH
	}
	wrp.WriteString(zFull, path)

	return vfsErrorCode(wrp, err, _CANTOPEN_FULLPATH)
}

//go:linkname vfsDelete
func vfsDelete(wrp *sqlite3_wrap.Wrapper, pVfs, zPath ptr_t, syncDir int32) _ErrorCode {
	vfs := vfsGet(wrp, pVfs)
	path := wrp.ReadString(zPath, _MAX_PATHNAME)

	err := vfs.Delete(path, syncDir != 0)
	return vfsErrorCode(wrp, err, _IOERR_DELETE)
}

//go:linkname vfsAccess
func vfsAccess(wrp *sqlite3_wrap.Wrapper, pVfs, zPath ptr_t, flags AccessFlag, pResOut ptr_t) _ErrorCode {
	vfs := vfsGet(wrp, pVfs)
	path := wrp.ReadString(zPath, _MAX_PATHNAME)

	ok, err := vfs.Access(path, flags)
	wrp.WriteBool(pResOut, ok)
	return vfsErrorCode(wrp, err, _IOERR_ACCESS)
}

//go:linkname vfsOpen
func vfsOpen(wrp *sqlite3_wrap.Wrapper, pVfs, zPath, pFile ptr_t, flags OpenFlag, pOutFlags, pOutVFS ptr_t) _ErrorCode {
	vfs := vfsGet(wrp, pVfs)
	name := GetFilename(wrp, zPath, flags)

	var file File
	var err error
	if ffs, ok := vfs.(VFSFilename); ok {
		file, flags, err = ffs.OpenFilename(name, flags)
	} else {
		file, flags, err = vfs.Open(name.String(), flags)
	}
	if err != nil {
		return vfsErrorCode(wrp, err, _CANTOPEN)
	}

	if file, ok := file.(FilePowersafeOverwrite); ok {
		if b, ok := util.ParseBool(name.URIParameter("psow")); ok {
			file.SetPowersafeOverwrite(b)
		}
	}
	if pOutFlags != 0 {
		wrp.Write32(pOutFlags, uint32(flags))
	}
	var outVFS uint32
	if file, ok := file.(FileSharedMemory); ok && file.SharedMemory() != nil {
		outVFS |= 1
	}
	if file, ok := file.(FileMemoryMapper); ok && file.MemoryMapper() != nil {
		outVFS |= 2
	}
	wrp.Write32(pOutVFS, outVFS)
	file = cksmWrapFile(file, flags)
	vfsFileRegister(wrp, pFile, file)
	return _OK
}

//go:linkname vfsClose
func vfsClose(wrp *sqlite3_wrap.Wrapper, pFile ptr_t) _ErrorCode {
	err := vfsFileClose(wrp, pFile)
	return vfsErrorCode(wrp, err, _IOERR_CLOSE)
}

//go:linkname vfsRead
func vfsRead(wrp *sqlite3_wrap.Wrapper, pFile, zBuf ptr_t, iAmt int32, iOfst int64) _ErrorCode {
	file := vfsFileGet(wrp, pFile).(File)
	buf := wrp.Bytes(zBuf, int64(iAmt))

	n, err := file.ReadAt(buf, iOfst)
	if n == int(iAmt) {
		return _OK
	}
	if err != io.EOF {
		return vfsErrorCode(wrp, err, _IOERR_READ)
	}
	clear(buf[n:])
	return _IOERR_SHORT_READ
}

//go:linkname vfsWrite
func vfsWrite(wrp *sqlite3_wrap.Wrapper, pFile, zBuf ptr_t, iAmt int32, iOfst int64) _ErrorCode {
	file := vfsFileGet(wrp, pFile).(File)
	buf := wrp.Bytes(zBuf, int64(iAmt))

	_, err := file.WriteAt(buf, iOfst)
	return vfsErrorCode(wrp, err, _IOERR_WRITE)
}

//go:linkname vfsTruncate
func vfsTruncate(wrp *sqlite3_wrap.Wrapper, pFile ptr_t, nByte int64) _ErrorCode {
	file := vfsFileGet(wrp, pFile).(File)
	err := file.Truncate(nByte)
	return vfsErrorCode(wrp, err, _IOERR_TRUNCATE)
}

//go:linkname vfsSync
func vfsSync(wrp *sqlite3_wrap.Wrapper, pFile ptr_t, flags SyncFlag) _ErrorCode {
	file := vfsFileGet(wrp, pFile).(File)
	err := file.Sync(flags)
	return vfsErrorCode(wrp, err, _IOERR_FSYNC)
}

//go:linkname vfsFileSize
func vfsFileSize(wrp *sqlite3_wrap.Wrapper, pFile, pSize ptr_t) _ErrorCode {
	file := vfsFileGet(wrp, pFile).(File)
	size, err := file.Size()
	wrp.Write64(pSize, uint64(size))
	return vfsErrorCode(wrp, err, _IOERR_SEEK)
}

//go:linkname vfsLock
func vfsLock(wrp *sqlite3_wrap.Wrapper, pFile ptr_t, eLock LockLevel) _ErrorCode {
	file := vfsFileGet(wrp, pFile).(File)
	err := file.Lock(eLock)
	return vfsErrorCode(wrp, err, _IOERR_LOCK)
}

//go:linkname vfsUnlock
func vfsUnlock(wrp *sqlite3_wrap.Wrapper, pFile ptr_t, eLock LockLevel) _ErrorCode {
	file := vfsFileGet(wrp, pFile).(File)
	err := file.Unlock(LockLevel(eLock))
	return vfsErrorCode(wrp, err, _IOERR_UNLOCK)
}

//go:linkname vfsCheckReservedLock
func vfsCheckReservedLock(wrp *sqlite3_wrap.Wrapper, pFile, pResOut ptr_t) _ErrorCode {
	file := vfsFileGet(wrp, pFile).(File)
	locked, err := file.CheckReservedLock()
	wrp.WriteBool(pResOut, locked)
	return vfsErrorCode(wrp, err, _IOERR_CHECKRESERVEDLOCK)
}

//go:linkname vfsFileControl
func vfsFileControl(wrp *sqlite3_wrap.Wrapper, pFile ptr_t, op _FcntlOpcode, pArg ptr_t) _ErrorCode {
	file := vfsFileGet(wrp, pFile).(File)
	if file, ok := file.(fileControl); ok {
		return file.fileControl(wrp, op, pArg)
	}
	return vfsFileControlImpl(wrp, file, op, pArg)
}

func vfsFileControlImpl(wrp *sqlite3_wrap.Wrapper, file File, op _FcntlOpcode, pArg ptr_t) _ErrorCode {
	mem := wrp.Memory
	switch op {
	case _FCNTL_LOCKSTATE:
		if file, ok := file.(FileLockState); ok {
			if lk := file.LockState(); lk <= LOCK_EXCLUSIVE {
				mem.Write32(pArg, uint32(lk))
				return _OK
			}
		}

	case _FCNTL_PERSIST_WAL:
		if file, ok := file.(FilePersistWAL); ok {
			if i := int32(mem.Read32(pArg)); i < 0 {
				mem.WriteBool(pArg, file.PersistWAL())
			} else {
				file.SetPersistWAL(i != 0)
			}
			return _OK
		}

	case _FCNTL_POWERSAFE_OVERWRITE:
		if file, ok := file.(FilePowersafeOverwrite); ok {
			if i := int32(mem.Read32(pArg)); i < 0 {
				mem.WriteBool(pArg, file.PowersafeOverwrite())
			} else {
				file.SetPowersafeOverwrite(i != 0)
			}
			return _OK
		}

	case _FCNTL_CHUNK_SIZE:
		if file, ok := file.(FileChunkSize); ok {
			size := int32(mem.Read32(pArg))
			file.ChunkSize(int(size))
			return _OK
		}

	case _FCNTL_SIZE_HINT:
		if file, ok := file.(FileSizeHint); ok {
			size := int64(mem.Read64(pArg))
			err := file.SizeHint(size)
			return vfsErrorCode(wrp, err, _IOERR_TRUNCATE)
		}

	case _FCNTL_HAS_MOVED:
		if file, ok := file.(FileHasMoved); ok {
			moved, err := file.HasMoved()
			mem.WriteBool(pArg, moved)
			return vfsErrorCode(wrp, err, _IOERR_FSTAT)
		}

	case _FCNTL_OVERWRITE:
		if file, ok := file.(FileOverwrite); ok {
			err := file.Overwrite()
			return vfsErrorCode(wrp, err, _IOERR)
		}

	case _FCNTL_SYNC:
		if file, ok := file.(FileSync); ok {
			var name string
			if pArg != 0 {
				name = mem.ReadString(pArg, _MAX_PATHNAME)
			}
			err := file.SyncSuper(name)
			return vfsErrorCode(wrp, err, _IOERR)
		}

	case _FCNTL_COMMIT_PHASETWO:
		if file, ok := file.(FileCommitPhaseTwo); ok {
			err := file.CommitPhaseTwo()
			return vfsErrorCode(wrp, err, _IOERR)
		}

	case _FCNTL_BEGIN_ATOMIC_WRITE:
		if file, ok := file.(FileBatchAtomicWrite); ok {
			err := file.BeginAtomicWrite()
			return vfsErrorCode(wrp, err, _IOERR_BEGIN_ATOMIC)
		}
	case _FCNTL_COMMIT_ATOMIC_WRITE:
		if file, ok := file.(FileBatchAtomicWrite); ok {
			err := file.CommitAtomicWrite()
			return vfsErrorCode(wrp, err, _IOERR_COMMIT_ATOMIC)
		}
	case _FCNTL_ROLLBACK_ATOMIC_WRITE:
		if file, ok := file.(FileBatchAtomicWrite); ok {
			err := file.RollbackAtomicWrite()
			return vfsErrorCode(wrp, err, _IOERR_ROLLBACK_ATOMIC)
		}

	case _FCNTL_CKPT_START:
		if file, ok := file.(FileCheckpoint); ok {
			file.CheckpointStart()
			return _OK
		}
	case _FCNTL_CKPT_DONE:
		if file, ok := file.(FileCheckpoint); ok {
			file.CheckpointDone()
			return _OK
		}

	case _FCNTL_PRAGMA:
		if file, ok := file.(FilePragma); ok {
			var value string
			ptr := ptr_t(mem.Read32(pArg + 1*ptrlen))
			name := mem.ReadString(ptr, _MAX_SQL_LENGTH)
			if ptr := ptr_t(mem.Read32(pArg + 2*ptrlen)); ptr != 0 {
				value = mem.ReadString(ptr, _MAX_SQL_LENGTH)
			}

			out, err := file.Pragma(strings.ToLower(name), value)

			ret := vfsErrorCode(wrp, err, _ERROR)
			if ret == _ERROR {
				out = err.Error()
			}
			if out != "" {
				ptr := ptr_t(wrp.Xsqlite3_malloc64(int64(len(out)) + 1))
				mem.Write32(pArg, uint32(ptr))
				mem.WriteString(ptr, out)
			}
			return ret
		}

	case _FCNTL_BUSYHANDLER:
		if file, ok := file.(FileBusyHandler); ok {
			arg := int64(mem.Read64(pArg))
			file.BusyHandler(func() bool {
				return wrp.Xsqlite3_invoke_busy_handler_go(arg) != 0
			})
			return _OK
		}

	case _FCNTL_LOCK_TIMEOUT:
		if file, ok := file.(FileSharedMemory); ok {
			if shm, ok := file.SharedMemory().(blockingSharedMemory); ok {
				shm.shmEnableBlocking(mem.ReadBool(pArg))
				return _OK
			}
		}

	case _FCNTL_MMAP_SIZE:
		if file, ok := file.(FileMemoryMapper); ok {
			if mmap := file.MemoryMapper(); mmap != nil {
				mmap.mmapSize(wrp, pArg)
				return _OK
			}
		}

	case _FCNTL_PDB:
		if file, ok := file.(filePDB); ok {
			file.SetDB(wrp.DB)
			return _OK
		}

	case _FCNTL_NULL_IO:
		file.Close()
		return _OK
	}

	return _NOTFOUND
}

//go:linkname vfsSectorSize
func vfsSectorSize(wrp *sqlite3_wrap.Wrapper, pFile ptr_t) int32 {
	file := vfsFileGet(wrp, pFile).(File)
	return int32(file.SectorSize())
}

//go:linkname vfsDeviceCharacteristics
func vfsDeviceCharacteristics(wrp *sqlite3_wrap.Wrapper, pFile ptr_t) DeviceCharacteristic {
	file := vfsFileGet(wrp, pFile).(File)
	return file.DeviceCharacteristics()
}

//go:linkname vfsShmBarrier
func vfsShmBarrier(wrp *sqlite3_wrap.Wrapper, pFile ptr_t) {
	shm := vfsFileGet(wrp, pFile).(FileSharedMemory).SharedMemory()
	shm.shmBarrier()
}

//go:linkname vfsShmMap
func vfsShmMap(wrp *sqlite3_wrap.Wrapper, pFile ptr_t, iRegion, szRegion, bExtend int32, pp ptr_t) _ErrorCode {
	shm := vfsFileGet(wrp, pFile).(FileSharedMemory).SharedMemory()
	p, err := shm.shmMap(wrp, iRegion, szRegion, bExtend != 0)
	wrp.Write32(pp, uint32(p))
	return vfsErrorCode(wrp, err, _IOERR_SHMMAP)
}

//go:linkname vfsShmLock
func vfsShmLock(wrp *sqlite3_wrap.Wrapper, pFile ptr_t, offset, n int32, flags _ShmFlag) _ErrorCode {
	shm := vfsFileGet(wrp, pFile).(FileSharedMemory).SharedMemory()
	err := shm.shmLock(offset, n, flags)
	return vfsErrorCode(wrp, err, _IOERR_SHMLOCK)
}

//go:linkname vfsShmUnmap
func vfsShmUnmap(wrp *sqlite3_wrap.Wrapper, pFile ptr_t, bDelete int32) _ErrorCode {
	shm := vfsFileGet(wrp, pFile).(FileSharedMemory).SharedMemory()
	shm.shmUnmap(bDelete != 0)
	return _OK
}

//go:linkname vfsFetch
func vfsFetch(wrp *sqlite3_wrap.Wrapper, pFile ptr_t, iOfst int64, iAmt int32, pp ptr_t) _ErrorCode {
	mem := vfsFileGet(wrp, pFile).(FileMemoryMapper).MemoryMapper()
	err := mem.fetch(wrp, iOfst, iAmt, pp)
	return vfsErrorCode(wrp, err, _IOERR_MMAP)
}

//go:linkname vfsUnfetch
func vfsUnfetch(wrp *sqlite3_wrap.Wrapper, pFile ptr_t, _ int64, p ptr_t) _ErrorCode {
	if p == 0 {
		mmap := vfsFileGet(wrp, pFile).(FileMemoryMapper).MemoryMapper()
		mmap.Close()
	}
	return _OK
}

func vfsGet(wrp *sqlite3_wrap.Wrapper, pVfs ptr_t) VFS {
	var name string
	if pVfs != 0 {
		const zNameOffset = 16
		ptr := ptr_t(wrp.Read32(pVfs + zNameOffset))
		name = wrp.ReadString(ptr, _MAX_NAME)
	}
	if vfs := Find(name); vfs != nil {
		return vfs
	}
	panic(errutil.NoVFSErr + errutil.ErrorString(name))
}

func vfsFileRegister(wrp *sqlite3_wrap.Wrapper, pFile ptr_t, file File) {
	const fileHandleOffset = 4
	id := wrp.AddHandle(file)
	wrp.Write32(pFile+fileHandleOffset, uint32(id))
}

func vfsFileGet(wrp *sqlite3_wrap.Wrapper, pFile ptr_t) any {
	const fileHandleOffset = 4
	id := ptr_t(wrp.Read32(pFile + fileHandleOffset))
	return wrp.GetHandle(id)
}

func vfsFileClose(wrp *sqlite3_wrap.Wrapper, pFile ptr_t) error {
	const fileHandleOffset = 4
	id := ptr_t(wrp.Read32(pFile + fileHandleOffset))
	return wrp.DelHandle(id)
}

func vfsErrorCode(wrp *sqlite3_wrap.Wrapper, err error, code _ErrorCode) _ErrorCode {
	var sys error

	switch err := err.(type) {
	case nil:
		code = _OK
	case _ErrorCode:
		code = err
	case sysError:
		code = err.code
		sys = err.error
	default:
		switch v := reflect.ValueOf(err); v.Kind() {
		case reflect.Uint8, reflect.Uint16:
			code = _ErrorCode(v.Uint())
		default:
			sys = err
		}
	}

	wrp.SysError = sys
	return code
}

// SystemError tags an error with a given
// sqlite3.ErrorCode or sqlite3.ExtendedErrorCode.
func SystemError[T interface{ ~uint8 | ~uint16 }](err error, code T) error {
	if err == nil {
		return nil
	}
	return sysError{error: err, code: _ErrorCode(code)}
}

type sysError struct {
	error
	code _ErrorCode
}
