//go:build unix && !sqlite3_nosys

package util

import (
	"context"
	"os"
	"unsafe"

	"github.com/tetratelabs/wazero/api"
	"golang.org/x/sys/unix"
)

type mmapState struct {
	regions []*MappedRegion
}

func (s *mmapState) new(ctx context.Context, mod api.Module, size int32) *MappedRegion {
	// Find unused region.
	for _, r := range s.regions {
		if !r.used && r.size == size {
			return r
		}
	}

	// Allocate page aligned memmory.
	alloc := mod.ExportedFunction("aligned_alloc")
	stack := [...]uint64{
		uint64(unix.Getpagesize()),
		uint64(size),
	}
	if err := alloc.CallWithStack(ctx, stack[:]); err != nil {
		panic(err)
	}
	if stack[0] == 0 {
		panic(OOMErr)
	}

	// Save the newly allocated region.
	ptr := uint32(stack[0])
	buf := View(mod, ptr, uint64(size))
	addr := unsafe.Pointer(&buf[0])
	s.regions = append(s.regions, &MappedRegion{
		Ptr:  ptr,
		addr: addr,
		size: size,
	})
	return s.regions[len(s.regions)-1]
}

type MappedRegion struct {
	addr unsafe.Pointer
	Ptr  uint32
	size int32
	used bool
}

func MapRegion(ctx context.Context, mod api.Module, f *os.File, offset int64, size int32, readOnly bool) (*MappedRegion, error) {
	s := ctx.Value(moduleKey{}).(*moduleState)
	r := s.new(ctx, mod, size)
	err := r.mmap(f, offset, readOnly)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (r *MappedRegion) Unmap() error {
	// We can't munmap the region, otherwise it could be remaped.
	// Instead, convert it to a protected, private, anonymous mapping.
	// If successful, it can be reused for a subsequent mmap.
	_, err := unix.MmapPtr(-1, 0, r.addr, uintptr(r.size),
		unix.PROT_NONE, unix.MAP_PRIVATE|unix.MAP_FIXED|unix.MAP_ANON)
	r.used = err != nil
	return err
}

func (r *MappedRegion) mmap(f *os.File, offset int64, readOnly bool) error {
	prot := unix.PROT_READ
	if !readOnly {
		prot |= unix.PROT_WRITE
	}
	_, err := unix.MmapPtr(int(f.Fd()), offset, r.addr, uintptr(r.size),
		prot, unix.MAP_SHARED|unix.MAP_FIXED)
	r.used = err == nil
	return err
}
