//go:build sqlite3_dotlk

package vfs

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"sync"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

type vfsShmParent struct {
	shared [][_WALINDEX_PGSZ]byte
	refs   int // +checklocks:vfsShmListMtx

	lock [_SHM_NLOCK]int16 // +checklocks:Mutex
	sync.Mutex
}

var (
	// +checklocks:vfsShmListMtx
	vfsShmList    = map[string]*vfsShmParent{}
	vfsShmListMtx sync.Mutex
)

type vfsShm struct {
	*vfsShmParent
	mod    api.Module
	alloc  api.Function
	free   api.Function
	path   string
	shadow [][_WALINDEX_PGSZ]byte
	ptrs   []uint32
	stack  [1]uint64
	lock   [_SHM_NLOCK]bool
}

func (s *vfsShm) Close() error {
	if s.vfsShmParent == nil {
		return nil
	}

	vfsShmListMtx.Lock()
	defer vfsShmListMtx.Unlock()

	// Unlock everything.
	s.shmLock(0, _SHM_NLOCK, _SHM_UNLOCK)

	// Decrease reference count.
	if s.vfsShmParent.refs > 0 {
		s.vfsShmParent.refs--
		s.vfsShmParent = nil
		return nil
	}

	err := os.Remove(s.path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return _IOERR_UNLOCK
	}
	delete(vfsShmList, s.path)
	s.vfsShmParent = nil
	return nil
}

func (s *vfsShm) shmOpen() _ErrorCode {
	if s.vfsShmParent != nil {
		return _OK
	}

	vfsShmListMtx.Lock()
	defer vfsShmListMtx.Unlock()

	// Find a shared buffer, increase the reference count.
	if g, ok := vfsShmList[s.path]; ok {
		s.vfsShmParent = g
		g.refs++
		return _OK
	}

	// Create a directory on disk to ensure only this process
	// uses this path to register a shared memory.
	err := os.Mkdir(s.path, 0777)
	if errors.Is(err, fs.ErrExist) {
		return _BUSY
	}
	if err != nil {
		return _IOERR_LOCK
	}

	// Add the new shared buffer.
	s.vfsShmParent = &vfsShmParent{}
	vfsShmList[s.path] = s.vfsShmParent
	return _OK
}

func (s *vfsShm) shmMap(ctx context.Context, mod api.Module, id, size int32, extend bool) (uint32, _ErrorCode) {
	if size != _WALINDEX_PGSZ {
		return 0, _IOERR_SHMMAP
	}
	if s.mod == nil {
		s.mod = mod
		s.free = mod.ExportedFunction("sqlite3_free")
		s.alloc = mod.ExportedFunction("sqlite3_malloc64")
	}
	if rc := s.shmOpen(); rc != _OK {
		return 0, rc
	}

	s.Lock()
	defer s.Unlock()
	defer s.shmAcquire(nil)

	// Extend shared memory.
	if int(id) >= len(s.shared) {
		if !extend {
			return 0, _OK
		}
		s.shared = append(s.shared, make([][_WALINDEX_PGSZ]byte, int(id)-len(s.shared)+1)...)
	}

	// Allocate shadow memory.
	if int(id) >= len(s.shadow) {
		s.shadow = append(s.shadow, make([][_WALINDEX_PGSZ]byte, int(id)-len(s.shadow)+1)...)
	}

	// Allocate local memory.
	for int(id) >= len(s.ptrs) {
		s.stack[0] = uint64(size)
		if err := s.alloc.CallWithStack(ctx, s.stack[:]); err != nil {
			panic(err)
		}
		if s.stack[0] == 0 {
			panic(util.OOMErr)
		}
		clear(util.View(s.mod, uint32(s.stack[0]), _WALINDEX_PGSZ))
		s.ptrs = append(s.ptrs, uint32(s.stack[0]))
	}

	s.shadow[0][4] = 1
	return s.ptrs[id], _OK
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) (rc _ErrorCode) {
	s.Lock()
	defer s.Unlock()

	switch {
	case flags&_SHM_LOCK != 0:
		defer s.shmAcquire(&rc)
	case flags&_SHM_EXCLUSIVE != 0:
		s.shmRelease()
	}

	return s.shmMemLock(offset, n, flags)
}

func (s *vfsShm) shmUnmap(delete bool) {
	if s.vfsShmParent == nil {
		return
	}
	defer s.Close()

	s.Lock()
	s.shmRelease()
	defer s.Unlock()

	for _, p := range s.ptrs {
		s.stack[0] = uint64(p)
		if err := s.free.CallWithStack(context.Background(), s.stack[:]); err != nil {
			panic(err)
		}
	}
	s.ptrs = nil
	s.shadow = nil
}
