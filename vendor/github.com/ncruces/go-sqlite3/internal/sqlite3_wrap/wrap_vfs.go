package sqlite3_wrap

import (
	"crypto/rand"
	"time"
	_ "unsafe"

	"github.com/ncruces/julianday"
)

func (w *Wrapper) Xgo_randomness(pVfs, nByte, zByte int32) int32 {
	mem := w.Bytes(Ptr_t(zByte), int64(nByte))
	n, _ := rand.Reader.Read(mem)
	return int32(n)
}

func (w *Wrapper) Xgo_sleep(pVfs, nMicro int32) int32 {
	time.Sleep(time.Duration(nMicro) * time.Microsecond)
	return OK
}

func (w *Wrapper) Xgo_current_time_64(pVfs, nMicro int32) int32 {
	day, nsec := julianday.Date(time.Now())
	msec := day*86_400_000 + nsec/1_000_000
	w.Write64(Ptr_t(nMicro), uint64(msec))
	return OK
}

//go:linkname vfsFullPathname github.com/ncruces/go-sqlite3/vfs.vfsFullPathname
func vfsFullPathname(_ *Wrapper, v0, v1, v2, v3 int32) int32

func (w *Wrapper) Xgo_full_pathname(v0, v1, v2, v3 int32) int32 {
	return vfsFullPathname(w, v0, v1, v2, v3)
}

//go:linkname vfsDelete github.com/ncruces/go-sqlite3/vfs.vfsDelete
func vfsDelete(_ *Wrapper, v0, v1, v2 int32) int32

func (w *Wrapper) Xgo_delete(v0, v1, v2 int32) int32 {
	return vfsDelete(w, v0, v1, v2)
}

//go:linkname vfsAccess github.com/ncruces/go-sqlite3/vfs.vfsAccess
func vfsAccess(_ *Wrapper, v0, v1, v2, v3 int32) int32

func (w *Wrapper) Xgo_access(v0, v1, v2, v3 int32) int32 {
	return vfsAccess(w, v0, v1, v2, v3)
}

//go:linkname vfsOpen github.com/ncruces/go-sqlite3/vfs.vfsOpen
func vfsOpen(_ *Wrapper, v0, v1, v2, v3, v4, v5 int32) int32

func (w *Wrapper) Xgo_open(v0, v1, v2, v3, v4, v5 int32) int32 {
	return vfsOpen(w, v0, v1, v2, v3, v4, v5)
}

//go:linkname vfsClose github.com/ncruces/go-sqlite3/vfs.vfsClose
func vfsClose(_ *Wrapper, v0 int32) int32

func (w *Wrapper) Xgo_close(v0 int32) int32 {
	return vfsClose(w, v0)
}

//go:linkname vfsRead github.com/ncruces/go-sqlite3/vfs.vfsRead
func vfsRead(_ *Wrapper, v0, v1, v2 int32, v3 int64) int32

func (w *Wrapper) Xgo_read(v0, v1, v2 int32, v3 int64) int32 {
	return vfsRead(w, v0, v1, v2, v3)
}

//go:linkname vfsWrite github.com/ncruces/go-sqlite3/vfs.vfsWrite
func vfsWrite(_ *Wrapper, v0, v1, v2 int32, v3 int64) int32

func (w *Wrapper) Xgo_write(v0, v1, v2 int32, v3 int64) int32 {
	return vfsWrite(w, v0, v1, v2, v3)
}

//go:linkname vfsTruncate github.com/ncruces/go-sqlite3/vfs.vfsTruncate
func vfsTruncate(_ *Wrapper, v0 int32, v1 int64) int32

func (w *Wrapper) Xgo_truncate(v0 int32, v1 int64) int32 {
	return vfsTruncate(w, v0, v1)
}

//go:linkname vfsSync github.com/ncruces/go-sqlite3/vfs.vfsSync
func vfsSync(_ *Wrapper, v0, v1 int32) int32

func (w *Wrapper) Xgo_sync(v0, v1 int32) int32 {
	return vfsSync(w, v0, v1)
}

//go:linkname vfsFileSize github.com/ncruces/go-sqlite3/vfs.vfsFileSize
func vfsFileSize(_ *Wrapper, v0, v1 int32) int32

func (w *Wrapper) Xgo_file_size(v0, v1 int32) int32 {
	return vfsFileSize(w, v0, v1)
}

//go:linkname vfsLock github.com/ncruces/go-sqlite3/vfs.vfsLock
func vfsLock(_ *Wrapper, v0, v1 int32) int32

func (w *Wrapper) Xgo_lock(v0, v1 int32) int32 {
	return vfsLock(w, v0, v1)
}

//go:linkname vfsUnlock github.com/ncruces/go-sqlite3/vfs.vfsUnlock
func vfsUnlock(_ *Wrapper, v0, v1 int32) int32

func (w *Wrapper) Xgo_unlock(v0, v1 int32) int32 {
	return vfsUnlock(w, v0, v1)
}

//go:linkname vfsCheckReservedLock github.com/ncruces/go-sqlite3/vfs.vfsCheckReservedLock
func vfsCheckReservedLock(_ *Wrapper, v0, v1 int32) int32

func (w *Wrapper) Xgo_check_reserved_lock(v0, v1 int32) int32 {
	return vfsCheckReservedLock(w, v0, v1)
}

//go:linkname vfsFileControl github.com/ncruces/go-sqlite3/vfs.vfsFileControl
func vfsFileControl(_ *Wrapper, v0, v1, v2 int32) int32

func (w *Wrapper) Xgo_file_control(v0, v1, v2 int32) int32 {
	return vfsFileControl(w, v0, v1, v2)
}

//go:linkname vfsSectorSize github.com/ncruces/go-sqlite3/vfs.vfsSectorSize
func vfsSectorSize(_ *Wrapper, v0 int32) int32

func (w *Wrapper) Xgo_sector_size(v0 int32) int32 {
	return vfsSectorSize(w, v0)
}

//go:linkname vfsDeviceCharacteristics github.com/ncruces/go-sqlite3/vfs.vfsDeviceCharacteristics
func vfsDeviceCharacteristics(_ *Wrapper, v0 int32) int32

func (w *Wrapper) Xgo_device_characteristics(v0 int32) int32 {
	return vfsDeviceCharacteristics(w, v0)
}

//go:linkname vfsShmBarrier github.com/ncruces/go-sqlite3/vfs.vfsShmBarrier
func vfsShmBarrier(_ *Wrapper, v0 int32)

func (w *Wrapper) Xgo_shm_barrier(v0 int32) {
	vfsShmBarrier(w, v0)
}

//go:linkname vfsShmMap github.com/ncruces/go-sqlite3/vfs.vfsShmMap
func vfsShmMap(_ *Wrapper, v0, v1, v2, v3, v4 int32) int32

func (w *Wrapper) Xgo_shm_map(v0, v1, v2, v3, v4 int32) int32 {
	return vfsShmMap(w, v0, v1, v2, v3, v4)
}

//go:linkname vfsShmLock github.com/ncruces/go-sqlite3/vfs.vfsShmLock
func vfsShmLock(_ *Wrapper, v0, v1, v2, v3 int32) int32

func (w *Wrapper) Xgo_shm_lock(v0, v1, v2, v3 int32) int32 {
	return vfsShmLock(w, v0, v1, v2, v3)
}

//go:linkname vfsShmUnmap github.com/ncruces/go-sqlite3/vfs.vfsShmUnmap
func vfsShmUnmap(_ *Wrapper, v0, v1 int32) int32

func (w *Wrapper) Xgo_shm_unmap(v0, v1 int32) int32 {
	return vfsShmUnmap(w, v0, v1)
}

//go:linkname vfsFetch github.com/ncruces/go-sqlite3/vfs.vfsFetch
func vfsFetch(_ *Wrapper, v0 int32, v1 int64, v2, v3 int32) int32

func (w *Wrapper) Xgo_fetch(v0 int32, v1 int64, v2, v3 int32) int32 {
	return vfsFetch(w, v0, v1, v2, v3)
}

//go:linkname vfsUnfetch github.com/ncruces/go-sqlite3/vfs.vfsUnfetch
func vfsUnfetch(_ *Wrapper, v0 int32, v1 int64, v2 int32) int32

func (w *Wrapper) Xgo_unfetch(v0 int32, v1 int64, v2 int32) int32 {
	return vfsUnfetch(w, v0, v1, v2)
}
