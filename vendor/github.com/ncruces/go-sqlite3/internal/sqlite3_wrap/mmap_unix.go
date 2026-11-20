//go:build unix

package sqlite3_wrap

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

func (w *Wrapper) MapRegion(f *os.File, offset int64, size int32, readOnly bool) (*MappedRegion, error) {
	pageSize := int64(unix.Getpagesize())
	align := offset & (pageSize - 1)
	size += int32(align + pageSize - 1)
	size &^= int32(pageSize - 1)

	r := w.newRegion(size)
	if r == nil {
		return nil, nil
	}
	if err := r.mmap(f, offset-align, readOnly); err != nil {
		return nil, err
	}
	r.Ptr = r.base + Ptr_t(align)
	return r, nil
}

func (w *Wrapper) newRegion(size int32) *MappedRegion {
	// Find unused region.
	for _, r := range w.regions {
		if !r.used && r.size == size {
			return r
		}
	}

	// Allocate page aligned memmory.
	ptr := Ptr_t(w.Xmemalign(int32(unix.Getpagesize()), size))
	if ptr == 0 {
		return nil
	}

	// Save the newly allocated region.
	ret := &MappedRegion{
		base: ptr,
		size: size,
		addr: unsafe.Pointer(&w.Buf[ptr]),
	}
	w.regions = append(w.regions, ret)
	return ret
}

type MappedRegion struct {
	addr unsafe.Pointer
	base Ptr_t
	Ptr  Ptr_t
	size int32
	used bool
}

func (r *MappedRegion) Unmap() error {
	// We can't munmap the region, otherwise it could be remapped by the runtime.
	// Instead, map anonymous (zeroed) pages readonly.
	// If successful, the region can be reused for a subsequent mmap.
	_, err := unix.MmapPtr(-1, 0, r.addr, uintptr(r.size),
		unix.PROT_READ, unix.MAP_PRIVATE|unix.MAP_FIXED|unix.MAP_ANON)
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
