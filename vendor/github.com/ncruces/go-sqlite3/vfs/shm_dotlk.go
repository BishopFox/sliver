//go:build sqlite3_dotlk

package vfs

import (
	"errors"
	"io/fs"
	"sync"

	"github.com/ncruces/go-sqlite3/internal/dotlk"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
)

type vfsShmParent struct {
	shared [][_WALINDEX_PGSZ]byte
	refs   int // +checklocks:vfsShmListMtx

	lock [_SHM_NLOCK]int8 // +checklocks:Mutex
	sync.Mutex
}

var (
	// +checklocks:vfsShmListMtx
	vfsShmList    = map[string]*vfsShmParent{}
	vfsShmListMtx sync.Mutex
)

type vfsShm struct {
	*vfsShmParent
	wrp    *sqlite3_wrap.Wrapper
	path   string
	shadow [][_WALINDEX_PGSZ]byte
	ptrs   []ptr_t
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

	if err := dotlk.Unlock(s.path); err != nil {
		return sysError{err, _IOERR_UNLOCK}
	}
	delete(vfsShmList, s.path)
	s.vfsShmParent = nil
	return nil
}

func (s *vfsShm) shmOpen() error {
	if s.vfsShmParent != nil {
		return nil
	}

	vfsShmListMtx.Lock()
	defer vfsShmListMtx.Unlock()

	// Find a shared buffer, increase the reference count.
	if g, ok := vfsShmList[s.path]; ok {
		s.vfsShmParent = g
		g.refs++
		return nil
	}

	// Dead man's switch.
	err := dotlk.LockShm(s.path)
	if errors.Is(err, fs.ErrExist) {
		return _BUSY
	}
	if err != nil {
		return sysError{err, _IOERR_LOCK}
	}

	// Add the new shared buffer.
	s.vfsShmParent = &vfsShmParent{}
	vfsShmList[s.path] = s.vfsShmParent
	return nil
}

func (s *vfsShm) shmMap(wrp *sqlite3_wrap.Wrapper, id, size int32, extend bool) (ptr_t, error) {
	if size != _WALINDEX_PGSZ {
		return 0, _IOERR_SHMMAP
	}
	if s.wrp == nil {
		s.wrp = wrp
	}
	if err := s.shmOpen(); err != nil {
		return 0, err
	}

	s.Lock()
	defer s.Unlock()
	defer s.shmAcquire(nil)

	// Extend shared memory.
	if int(id) >= len(s.shared) {
		if !extend {
			return 0, nil
		}
		s.shared = append(s.shared, make([][_WALINDEX_PGSZ]byte, int(id)-len(s.shared)+1)...)
	}

	// Allocate shadow memory.
	if int(id) >= len(s.shadow) {
		s.shadow = append(s.shadow, make([][_WALINDEX_PGSZ]byte, int(id)-len(s.shadow)+1)...)
	}

	// Allocate local memory.
	for int(id) >= len(s.ptrs) {
		ptr := wrp.Xsqlite3_malloc64(int64(size))
		if ptr == 0 {
			return 0, _IOERR_NOMEM
		}
		clear(wrp.Bytes(ptr_t(ptr), _WALINDEX_PGSZ))
		s.ptrs = append(s.ptrs, ptr_t(ptr))
	}

	s.shadow[0][4] = 1
	return s.ptrs[id], nil
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) (err error) {
	if s.vfsShmParent == nil {
		return _IOERR_SHMLOCK
	}

	s.Lock()
	defer s.Unlock()

	switch {
	case flags&_SHM_LOCK != 0:
		defer s.shmAcquire(&err)
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
		s.wrp.Xsqlite3_free(int32(p))
	}
	s.ptrs = nil
	s.shadow = nil
}
