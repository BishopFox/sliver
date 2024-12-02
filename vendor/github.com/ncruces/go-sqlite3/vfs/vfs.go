package vfs

import (
	"context"
	"crypto/rand"
	"io"
	"reflect"
	"strings"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/util/sql3util"
	"github.com/ncruces/julianday"
)

// ExportHostFunctions is an internal API users need not call directly.
//
// ExportHostFunctions registers the required VFS host functions
// with the provided env module.
func ExportHostFunctions(env wazero.HostModuleBuilder) wazero.HostModuleBuilder {
	util.ExportFuncII(env, "go_vfs_find", vfsFind)
	util.ExportFuncIIJ(env, "go_localtime", vfsLocaltime)
	util.ExportFuncIIII(env, "go_randomness", vfsRandomness)
	util.ExportFuncIII(env, "go_sleep", vfsSleep)
	util.ExportFuncIII(env, "go_current_time_64", vfsCurrentTime64)
	util.ExportFuncIIIII(env, "go_full_pathname", vfsFullPathname)
	util.ExportFuncIIII(env, "go_delete", vfsDelete)
	util.ExportFuncIIIII(env, "go_access", vfsAccess)
	util.ExportFuncIIIIIII(env, "go_open", vfsOpen)
	util.ExportFuncII(env, "go_close", vfsClose)
	util.ExportFuncIIIIJ(env, "go_read", vfsRead)
	util.ExportFuncIIIIJ(env, "go_write", vfsWrite)
	util.ExportFuncIIJ(env, "go_truncate", vfsTruncate)
	util.ExportFuncIII(env, "go_sync", vfsSync)
	util.ExportFuncIII(env, "go_file_size", vfsFileSize)
	util.ExportFuncIIII(env, "go_file_control", vfsFileControl)
	util.ExportFuncII(env, "go_sector_size", vfsSectorSize)
	util.ExportFuncII(env, "go_device_characteristics", vfsDeviceCharacteristics)
	util.ExportFuncIII(env, "go_lock", vfsLock)
	util.ExportFuncIII(env, "go_unlock", vfsUnlock)
	util.ExportFuncIII(env, "go_check_reserved_lock", vfsCheckReservedLock)
	util.ExportFuncIIIIII(env, "go_shm_map", vfsShmMap)
	util.ExportFuncIIIII(env, "go_shm_lock", vfsShmLock)
	util.ExportFuncIII(env, "go_shm_unmap", vfsShmUnmap)
	util.ExportFuncVI(env, "go_shm_barrier", vfsShmBarrier)
	return env
}

func vfsFind(ctx context.Context, mod api.Module, zVfsName uint32) uint32 {
	name := util.ReadString(mod, zVfsName, _MAX_NAME)
	if vfs := Find(name); vfs != nil && vfs != (vfsOS{}) {
		return 1
	}
	return 0
}

func vfsLocaltime(ctx context.Context, mod api.Module, pTm uint32, t int64) _ErrorCode {
	tm := time.Unix(t, 0)
	var isdst int
	if tm.IsDST() {
		isdst = 1
	}

	const size = 32 / 8
	// https://pubs.opengroup.org/onlinepubs/7908799/xsh/time.h.html
	util.WriteUint32(mod, pTm+0*size, uint32(tm.Second()))
	util.WriteUint32(mod, pTm+1*size, uint32(tm.Minute()))
	util.WriteUint32(mod, pTm+2*size, uint32(tm.Hour()))
	util.WriteUint32(mod, pTm+3*size, uint32(tm.Day()))
	util.WriteUint32(mod, pTm+4*size, uint32(tm.Month()-time.January))
	util.WriteUint32(mod, pTm+5*size, uint32(tm.Year()-1900))
	util.WriteUint32(mod, pTm+6*size, uint32(tm.Weekday()-time.Sunday))
	util.WriteUint32(mod, pTm+7*size, uint32(tm.YearDay()-1))
	util.WriteUint32(mod, pTm+8*size, uint32(isdst))
	return _OK
}

func vfsRandomness(ctx context.Context, mod api.Module, pVfs uint32, nByte int32, zByte uint32) uint32 {
	mem := util.View(mod, zByte, uint64(nByte))
	n, _ := rand.Reader.Read(mem)
	return uint32(n)
}

func vfsSleep(ctx context.Context, mod api.Module, pVfs uint32, nMicro int32) _ErrorCode {
	time.Sleep(time.Duration(nMicro) * time.Microsecond)
	return _OK
}

func vfsCurrentTime64(ctx context.Context, mod api.Module, pVfs, piNow uint32) _ErrorCode {
	day, nsec := julianday.Date(time.Now())
	msec := day*86_400_000 + nsec/1_000_000
	util.WriteUint64(mod, piNow, uint64(msec))
	return _OK
}

func vfsFullPathname(ctx context.Context, mod api.Module, pVfs, zRelative uint32, nFull int32, zFull uint32) _ErrorCode {
	vfs := vfsGet(mod, pVfs)
	path := util.ReadString(mod, zRelative, _MAX_PATHNAME)

	path, err := vfs.FullPathname(path)

	if len(path) >= int(nFull) {
		return _CANTOPEN_FULLPATH
	}
	util.WriteString(mod, zFull, path)

	return vfsErrorCode(err, _CANTOPEN_FULLPATH)
}

func vfsDelete(ctx context.Context, mod api.Module, pVfs, zPath, syncDir uint32) _ErrorCode {
	vfs := vfsGet(mod, pVfs)
	path := util.ReadString(mod, zPath, _MAX_PATHNAME)

	err := vfs.Delete(path, syncDir != 0)
	return vfsErrorCode(err, _IOERR_DELETE)
}

func vfsAccess(ctx context.Context, mod api.Module, pVfs, zPath uint32, flags AccessFlag, pResOut uint32) _ErrorCode {
	vfs := vfsGet(mod, pVfs)
	path := util.ReadString(mod, zPath, _MAX_PATHNAME)

	ok, err := vfs.Access(path, flags)
	var res uint32
	if ok {
		res = 1
	}

	util.WriteUint32(mod, pResOut, res)
	return vfsErrorCode(err, _IOERR_ACCESS)
}

func vfsOpen(ctx context.Context, mod api.Module, pVfs, zPath, pFile uint32, flags OpenFlag, pOutFlags, pOutVFS uint32) _ErrorCode {
	vfs := vfsGet(mod, pVfs)
	name := GetFilename(ctx, mod, zPath, flags)

	var file File
	var err error
	if ffs, ok := vfs.(VFSFilename); ok {
		file, flags, err = ffs.OpenFilename(name, flags)
	} else {
		file, flags, err = vfs.Open(name.String(), flags)
	}
	if err != nil {
		return vfsErrorCode(err, _CANTOPEN)
	}

	if file, ok := file.(FilePowersafeOverwrite); ok {
		if b, ok := sql3util.ParseBool(name.URIParameter("psow")); ok {
			file.SetPowersafeOverwrite(b)
		}
	}
	if file, ok := file.(FileSharedMemory); ok &&
		pOutVFS != 0 && file.SharedMemory() != nil {
		util.WriteUint32(mod, pOutVFS, 1)
	}
	if pOutFlags != 0 {
		util.WriteUint32(mod, pOutFlags, uint32(flags))
	}
	file = cksmWrapFile(name, flags, file)
	vfsFileRegister(ctx, mod, pFile, file)
	return _OK
}

func vfsClose(ctx context.Context, mod api.Module, pFile uint32) _ErrorCode {
	err := vfsFileClose(ctx, mod, pFile)
	return vfsErrorCode(err, _IOERR_CLOSE)
}

func vfsRead(ctx context.Context, mod api.Module, pFile, zBuf uint32, iAmt int32, iOfst int64) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile).(File)
	buf := util.View(mod, zBuf, uint64(iAmt))

	n, err := file.ReadAt(buf, iOfst)
	if n == int(iAmt) {
		return _OK
	}
	if err != io.EOF {
		return vfsErrorCode(err, _IOERR_READ)
	}
	clear(buf[n:])
	return _IOERR_SHORT_READ
}

func vfsWrite(ctx context.Context, mod api.Module, pFile, zBuf uint32, iAmt int32, iOfst int64) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile).(File)
	buf := util.View(mod, zBuf, uint64(iAmt))

	_, err := file.WriteAt(buf, iOfst)
	return vfsErrorCode(err, _IOERR_WRITE)
}

func vfsTruncate(ctx context.Context, mod api.Module, pFile uint32, nByte int64) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile).(File)
	err := file.Truncate(nByte)
	return vfsErrorCode(err, _IOERR_TRUNCATE)
}

func vfsSync(ctx context.Context, mod api.Module, pFile uint32, flags SyncFlag) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile).(File)
	err := file.Sync(flags)
	return vfsErrorCode(err, _IOERR_FSYNC)
}

func vfsFileSize(ctx context.Context, mod api.Module, pFile, pSize uint32) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile).(File)
	size, err := file.Size()
	util.WriteUint64(mod, pSize, uint64(size))
	return vfsErrorCode(err, _IOERR_SEEK)
}

func vfsLock(ctx context.Context, mod api.Module, pFile uint32, eLock LockLevel) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile).(File)
	err := file.Lock(eLock)
	return vfsErrorCode(err, _IOERR_LOCK)
}

func vfsUnlock(ctx context.Context, mod api.Module, pFile uint32, eLock LockLevel) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile).(File)
	err := file.Unlock(eLock)
	return vfsErrorCode(err, _IOERR_UNLOCK)
}

func vfsCheckReservedLock(ctx context.Context, mod api.Module, pFile, pResOut uint32) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile).(File)
	locked, err := file.CheckReservedLock()

	var res uint32
	if locked {
		res = 1
	}

	util.WriteUint32(mod, pResOut, res)
	return vfsErrorCode(err, _IOERR_CHECKRESERVEDLOCK)
}

func vfsFileControl(ctx context.Context, mod api.Module, pFile uint32, op _FcntlOpcode, pArg uint32) _ErrorCode {
	file := vfsFileGet(ctx, mod, pFile).(File)
	if file, ok := file.(fileControl); ok {
		return file.fileControl(ctx, mod, op, pArg)
	}
	return vfsFileControlImpl(ctx, mod, file, op, pArg)
}

func vfsFileControlImpl(ctx context.Context, mod api.Module, file File, op _FcntlOpcode, pArg uint32) _ErrorCode {
	switch op {
	case _FCNTL_LOCKSTATE:
		if file, ok := file.(FileLockState); ok {
			if lk := file.LockState(); lk <= LOCK_EXCLUSIVE {
				util.WriteUint32(mod, pArg, uint32(lk))
				return _OK
			}
		}

	case _FCNTL_PERSIST_WAL:
		if file, ok := file.(FilePersistentWAL); ok {
			if i := util.ReadUint32(mod, pArg); int32(i) >= 0 {
				file.SetPersistentWAL(i != 0)
			} else if file.PersistentWAL() {
				util.WriteUint32(mod, pArg, 1)
			} else {
				util.WriteUint32(mod, pArg, 0)
			}
			return _OK
		}

	case _FCNTL_POWERSAFE_OVERWRITE:
		if file, ok := file.(FilePowersafeOverwrite); ok {
			if i := util.ReadUint32(mod, pArg); int32(i) >= 0 {
				file.SetPowersafeOverwrite(i != 0)
			} else if file.PowersafeOverwrite() {
				util.WriteUint32(mod, pArg, 1)
			} else {
				util.WriteUint32(mod, pArg, 0)
			}
			return _OK
		}

	case _FCNTL_CHUNK_SIZE:
		if file, ok := file.(FileChunkSize); ok {
			size := util.ReadUint32(mod, pArg)
			file.ChunkSize(int(size))
			return _OK
		}

	case _FCNTL_SIZE_HINT:
		if file, ok := file.(FileSizeHint); ok {
			size := util.ReadUint64(mod, pArg)
			err := file.SizeHint(int64(size))
			return vfsErrorCode(err, _IOERR_TRUNCATE)
		}

	case _FCNTL_HAS_MOVED:
		if file, ok := file.(FileHasMoved); ok {
			moved, err := file.HasMoved()
			var res uint32
			if moved {
				res = 1
			}
			util.WriteUint32(mod, pArg, res)
			return vfsErrorCode(err, _IOERR_FSTAT)
		}

	case _FCNTL_OVERWRITE:
		if file, ok := file.(FileOverwrite); ok {
			err := file.Overwrite()
			return vfsErrorCode(err, _IOERR)
		}

	case _FCNTL_COMMIT_PHASETWO:
		if file, ok := file.(FileCommitPhaseTwo); ok {
			err := file.CommitPhaseTwo()
			return vfsErrorCode(err, _IOERR)
		}

	case _FCNTL_BEGIN_ATOMIC_WRITE:
		if file, ok := file.(FileBatchAtomicWrite); ok {
			err := file.BeginAtomicWrite()
			return vfsErrorCode(err, _IOERR_BEGIN_ATOMIC)
		}
	case _FCNTL_COMMIT_ATOMIC_WRITE:
		if file, ok := file.(FileBatchAtomicWrite); ok {
			err := file.CommitAtomicWrite()
			return vfsErrorCode(err, _IOERR_COMMIT_ATOMIC)
		}
	case _FCNTL_ROLLBACK_ATOMIC_WRITE:
		if file, ok := file.(FileBatchAtomicWrite); ok {
			err := file.RollbackAtomicWrite()
			return vfsErrorCode(err, _IOERR_ROLLBACK_ATOMIC)
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
			ptr := util.ReadUint32(mod, pArg+1*ptrlen)
			name := util.ReadString(mod, ptr, _MAX_SQL_LENGTH)
			var value string
			if ptr := util.ReadUint32(mod, pArg+2*ptrlen); ptr != 0 {
				value = util.ReadString(mod, ptr, _MAX_SQL_LENGTH)
			}

			out, err := file.Pragma(strings.ToLower(name), value)

			ret := vfsErrorCode(err, _ERROR)
			if ret == _ERROR {
				out = err.Error()
			}
			if out != "" {
				fn := mod.ExportedFunction("sqlite3_malloc64")
				stack := [...]uint64{uint64(len(out) + 1)}
				if err := fn.CallWithStack(ctx, stack[:]); err != nil {
					panic(err)
				}
				util.WriteUint32(mod, pArg, uint32(stack[0]))
				util.WriteString(mod, uint32(stack[0]), out)
			}
			return ret
		}

	case _FCNTL_LOCK_TIMEOUT:
		if file, ok := file.(FileSharedMemory); ok {
			if shm, ok := file.SharedMemory().(blockingSharedMemory); ok {
				shm.shmEnableBlocking(util.ReadUint32(mod, pArg) != 0)
				return _OK
			}
		}
	}

	// Consider also implementing these opcodes (in use by SQLite):
	//  _FCNTL_BUSYHANDLER
	//  _FCNTL_LAST_ERRNO
	//  _FCNTL_SYNC
	return _NOTFOUND
}

func vfsSectorSize(ctx context.Context, mod api.Module, pFile uint32) uint32 {
	file := vfsFileGet(ctx, mod, pFile).(File)
	return uint32(file.SectorSize())
}

func vfsDeviceCharacteristics(ctx context.Context, mod api.Module, pFile uint32) DeviceCharacteristic {
	file := vfsFileGet(ctx, mod, pFile).(File)
	return file.DeviceCharacteristics()
}

func vfsShmBarrier(ctx context.Context, mod api.Module, pFile uint32) {
	shm := vfsFileGet(ctx, mod, pFile).(FileSharedMemory).SharedMemory()
	shm.shmBarrier()
}

func vfsShmMap(ctx context.Context, mod api.Module, pFile uint32, iRegion, szRegion int32, bExtend, pp uint32) _ErrorCode {
	shm := vfsFileGet(ctx, mod, pFile).(FileSharedMemory).SharedMemory()
	p, rc := shm.shmMap(ctx, mod, iRegion, szRegion, bExtend != 0)
	util.WriteUint32(mod, pp, p)
	return rc
}

func vfsShmLock(ctx context.Context, mod api.Module, pFile uint32, offset, n int32, flags _ShmFlag) _ErrorCode {
	shm := vfsFileGet(ctx, mod, pFile).(FileSharedMemory).SharedMemory()
	return shm.shmLock(offset, n, flags)
}

func vfsShmUnmap(ctx context.Context, mod api.Module, pFile, bDelete uint32) _ErrorCode {
	shm := vfsFileGet(ctx, mod, pFile).(FileSharedMemory).SharedMemory()
	shm.shmUnmap(bDelete != 0)
	return _OK
}

func vfsGet(mod api.Module, pVfs uint32) VFS {
	var name string
	if pVfs != 0 {
		const zNameOffset = 16
		name = util.ReadString(mod, util.ReadUint32(mod, pVfs+zNameOffset), _MAX_NAME)
	}
	if vfs := Find(name); vfs != nil {
		return vfs
	}
	panic(util.NoVFSErr + util.ErrorString(name))
}

func vfsFileRegister(ctx context.Context, mod api.Module, pFile uint32, file File) {
	const fileHandleOffset = 4
	id := util.AddHandle(ctx, file)
	util.WriteUint32(mod, pFile+fileHandleOffset, id)
}

func vfsFileGet(ctx context.Context, mod api.Module, pFile uint32) any {
	const fileHandleOffset = 4
	id := util.ReadUint32(mod, pFile+fileHandleOffset)
	return util.GetHandle(ctx, id)
}

func vfsFileClose(ctx context.Context, mod api.Module, pFile uint32) error {
	const fileHandleOffset = 4
	id := util.ReadUint32(mod, pFile+fileHandleOffset)
	return util.DelHandle(ctx, id)
}

func vfsErrorCode(err error, def _ErrorCode) _ErrorCode {
	if err == nil {
		return _OK
	}
	switch v := reflect.ValueOf(err); v.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return _ErrorCode(v.Uint())
	}
	return def
}
