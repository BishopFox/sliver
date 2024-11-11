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

type vfsShmBuffer struct {
	shared [][_WALINDEX_PGSZ]byte
	refs   int // +checklocks:vfsShmBuffersMtx

	lock [_SHM_NLOCK]int16 // +checklocks:Mutex
	sync.Mutex
}

var (
	// +checklocks:vfsShmBuffersMtx
	vfsShmBuffers    = map[string]*vfsShmBuffer{}
	vfsShmBuffersMtx sync.Mutex
)

type vfsShm struct {
	*vfsShmBuffer
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
	if s.vfsShmBuffer == nil {
		return nil
	}

	vfsShmBuffersMtx.Lock()
	defer vfsShmBuffersMtx.Unlock()

	// Unlock everything.
	s.shmLock(0, _SHM_NLOCK, _SHM_UNLOCK)

	// Decrease reference count.
	if s.vfsShmBuffer.refs > 0 {
		s.vfsShmBuffer.refs--
		s.vfsShmBuffer = nil
		return nil
	}

	err := os.Remove(s.path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return _IOERR_UNLOCK
	}
	delete(vfsShmBuffers, s.path)
	s.vfsShmBuffer = nil
	return nil
}

func (s *vfsShm) shmOpen() _ErrorCode {
	if s.vfsShmBuffer != nil {
		return _OK
	}

	vfsShmBuffersMtx.Lock()
	defer vfsShmBuffersMtx.Unlock()

	// Find a shared buffer, increase the reference count.
	if g, ok := vfsShmBuffers[s.path]; ok {
		s.vfsShmBuffer = g
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
	s.vfsShmBuffer = &vfsShmBuffer{}
	vfsShmBuffers[s.path] = s.vfsShmBuffer
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
	defer s.shmAcquire()

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
		s.shadow[0][4] = 1 // force invalidation
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

	return s.ptrs[id], _OK
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) _ErrorCode {
	s.Lock()
	defer s.Unlock()

	switch {
	case flags&_SHM_LOCK != 0:
		defer s.shmAcquire()
	case flags&_SHM_EXCLUSIVE != 0:
		s.shmRelease()
	}

	switch {
	case flags&_SHM_UNLOCK != 0:
		for i := offset; i < offset+n; i++ {
			if s.lock[i] {
				if s.vfsShmBuffer.lock[i] == 0 {
					panic(util.AssertErr())
				}
				if s.vfsShmBuffer.lock[i] <= 0 {
					s.vfsShmBuffer.lock[i] = 0
				} else {
					s.vfsShmBuffer.lock[i]--
				}
				s.lock[i] = false
			}
		}
	case flags&_SHM_SHARED != 0:
		for i := offset; i < offset+n; i++ {
			if s.lock[i] {
				panic(util.AssertErr())
			}
			if s.vfsShmBuffer.lock[i]+1 <= 0 {
				return _BUSY
			}
		}
		for i := offset; i < offset+n; i++ {
			s.vfsShmBuffer.lock[i]++
			s.lock[i] = true
		}
	case flags&_SHM_EXCLUSIVE != 0:
		for i := offset; i < offset+n; i++ {
			if s.lock[i] {
				panic(util.AssertErr())
			}
			if s.vfsShmBuffer.lock[i] != 0 {
				return _BUSY
			}
		}
		for i := offset; i < offset+n; i++ {
			s.vfsShmBuffer.lock[i] = -1
			s.lock[i] = true
		}
	default:
		panic(util.AssertErr())
	}

	return _OK
}

func (s *vfsShm) shmUnmap(delete bool) {
	if s.vfsShmBuffer == nil {
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
