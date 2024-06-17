//go:build (freebsd || openbsd || netbsd || dragonfly || illumos || sqlite3_flock) && (amd64 || arm64 || riscv64) && !(sqlite3_noshm || sqlite3_nosys)

package vfs

import (
	"context"
	"io"
	"os"
	"sync"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
	"golang.org/x/sys/unix"
)

// SupportsSharedMemory is false on platforms that do not support shared memory.
// To use [WAL without shared-memory], you need to set [EXCLUSIVE locking mode].
//
// [WAL without shared-memory]: https://sqlite.org/wal.html#noshm
// [EXCLUSIVE locking mode]: https://sqlite.org/pragma.html#pragma_locking_mode
const SupportsSharedMemory = true

const _SHM_NLOCK = 8

func (f *vfsFile) SharedMemory() SharedMemory { return f.shm }

// NewSharedMemory returns a shared-memory WAL-index
// backed by a file with the given path.
// It will return nil if shared-memory is not supported,
// or not appropriate for the given flags.
// Only [OPEN_MAIN_DB] databases may need a WAL-index.
// You must ensure all concurrent accesses to a database
// use shared-memory instances created with the same path.
func NewSharedMemory(path string, flags OpenFlag) SharedMemory {
	if flags&OPEN_MAIN_DB == 0 || flags&(OPEN_DELETEONCLOSE|OPEN_MEMORY) != 0 {
		return nil
	}
	return &vfsShm{
		path:     path,
		readOnly: flags&OPEN_READONLY != 0,
	}
}

type vfsShmFile struct {
	*os.File
	info os.FileInfo

	// +checklocks:vfsShmFilesMtx
	refs int

	// +checklocks:lockMtx
	lock    [_SHM_NLOCK]int16
	lockMtx sync.Mutex
}

var (
	// +checklocks:vfsShmFilesMtx
	vfsShmFiles    []*vfsShmFile
	vfsShmFilesMtx sync.Mutex
)

type vfsShm struct {
	*vfsShmFile
	path     string
	lock     [_SHM_NLOCK]bool
	regions  []*util.MappedRegion
	readOnly bool
}

func (s *vfsShm) Close() error {
	if s.vfsShmFile == nil {
		return nil
	}

	// Unlock everything.
	s.shmLock(0, _SHM_NLOCK, _SHM_UNLOCK)

	vfsShmFilesMtx.Lock()
	defer vfsShmFilesMtx.Unlock()

	// Decrease reference count.
	if s.vfsShmFile.refs > 1 {
		s.vfsShmFile.refs--
		s.vfsShmFile = nil
		return nil
	}
	for i, g := range vfsShmFiles {
		if g == s.vfsShmFile {
			vfsShmFiles[i] = nil
			break
		}
	}

	err := s.File.Close()
	s.vfsShmFile = nil
	return err
}

func (s *vfsShm) shmOpen() (rc _ErrorCode) {
	if s.vfsShmFile != nil {
		return _OK
	}

	// Always open file read-write, as it will be shared.
	f, err := os.OpenFile(s.path,
		unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW, 0666)
	if err != nil {
		return _CANTOPEN
	}
	// Closes file if it's not nil.
	defer func() { f.Close() }()

	fi, err := f.Stat()
	if err != nil {
		return _IOERR_FSTAT
	}

	vfsShmFilesMtx.Lock()
	defer vfsShmFilesMtx.Unlock()

	// Find a shared file, increase the reference count.
	for _, g := range vfsShmFiles {
		if g != nil && os.SameFile(fi, g.info) {
			g.refs++
			s.vfsShmFile = g
			return _OK
		}
	}

	// Lock and truncate the file, if not readonly.
	if s.readOnly {
		rc = _READONLY_CANTINIT
	} else {
		if rc := osWriteLock(f, 0, 0, 0); rc != _OK {
			return rc
		}
		if err := f.Truncate(0); err != nil {
			return _IOERR_SHMOPEN
		}
	}

	// Add the new shared file.
	s.vfsShmFile = &vfsShmFile{
		File: f,
		info: fi,
		refs: 1,
	}
	f = nil // Don't close the file.
	for i, g := range vfsShmFiles {
		if g == nil {
			vfsShmFiles[i] = s.vfsShmFile
			return rc
		}
	}
	vfsShmFiles = append(vfsShmFiles, s.vfsShmFile)
	return rc
}

func (s *vfsShm) shmMap(ctx context.Context, mod api.Module, id, size int32, extend bool) (uint32, _ErrorCode) {
	// Ensure size is a multiple of the OS page size.
	if int(size)&(unix.Getpagesize()-1) != 0 {
		return 0, _IOERR_SHMMAP
	}

	if rc := s.shmOpen(); rc != _OK {
		return 0, rc
	}

	// Check if file is big enough.
	o, err := s.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, _IOERR_SHMSIZE
	}
	if n := (int64(id) + 1) * int64(size); n > o {
		if !extend {
			return 0, _OK
		}
		err := osAllocate(s.File, n)
		if err != nil {
			return 0, _IOERR_SHMSIZE
		}
	}

	var prot int
	if s.readOnly {
		prot = unix.PROT_READ
	} else {
		prot = unix.PROT_READ | unix.PROT_WRITE
	}
	r, err := util.MapRegion(ctx, mod, s.File, int64(id)*int64(size), size, prot)
	if err != nil {
		return 0, _IOERR_SHMMAP
	}
	s.regions = append(s.regions, r)
	if s.readOnly {
		return r.Ptr, _READONLY
	}
	return r.Ptr, _OK
}

func (s *vfsShm) shmLock(offset, n int32, flags _ShmFlag) _ErrorCode {
	s.lockMtx.Lock()
	defer s.lockMtx.Unlock()

	switch {
	case flags&_SHM_UNLOCK != 0:
		for i := offset; i < offset+n; i++ {
			if s.lock[i] {
				if s.vfsShmFile.lock[i] <= 0 {
					s.vfsShmFile.lock[i] = 0
				} else {
					s.vfsShmFile.lock[i]--
				}
			}
		}
	case flags&_SHM_SHARED != 0:
		for i := offset; i < offset+n; i++ {
			if s.vfsShmFile.lock[i] < 0 {
				return _BUSY
			}
		}
		for i := offset; i < offset+n; i++ {
			s.vfsShmFile.lock[i]++
			s.lock[i] = true
		}
	case flags&_SHM_EXCLUSIVE != 0:
		for i := offset; i < offset+n; i++ {
			if s.vfsShmFile.lock[i] != 0 {
				return _BUSY
			}
		}
		for i := offset; i < offset+n; i++ {
			s.vfsShmFile.lock[i] = -1
			s.lock[i] = true
		}
	}

	return _OK
}

func (s *vfsShm) shmUnmap(delete bool) {
	if s.vfsShmFile == nil {
		return
	}

	// Unmap regions.
	for _, r := range s.regions {
		r.Unmap()
	}
	clear(s.regions)
	s.regions = s.regions[:0]

	// Close the file.
	if delete {
		os.Remove(s.path)
	}
	s.Close()
	s.vfsShmFile = nil
}
