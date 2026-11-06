package sqlite3_wrap

import (
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

func (w *Wrapper) MapRegion(f *os.File, offset int64, size int32, readOnly bool) (*MappedRegion, error) {
	align := offset & (allocationGranularity - 1)
	size += int32(align + allocationGranularity - 1)
	size &^= int32(allocationGranularity - 1)

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
		if !r.file && r.size == size {
			return r
		}
	}

	// Reserve page aligned memmory.
	ptr := Ptr_t(w.reserve(int64(size)))
	if ptr == 0 {
		return nil
	}

	// Save the newly reserved region.
	ret := &MappedRegion{
		base: ptr,
		size: size,
		addr: uintptr(unsafe.Pointer(&w.Buf[ptr])),
	}
	w.regions = append(w.regions, ret)
	return ret
}

type MappedRegion struct {
	addr uintptr
	base Ptr_t
	Ptr  Ptr_t
	size int32
	file bool
	zero bool
}

func (r *MappedRegion) Close() error {
	if !r.file && !r.zero {
		return nil
	}

	// Convert the file view back to a placeholder.
	err := unmapViewOfFile2(r.addr, _MEM_PRESERVE_PLACEHOLDER)
	if err != nil {
		return err
	}
	r.file = false
	r.zero = false
	return nil
}

func (r *MappedRegion) Unmap() error {
	err := r.Close()
	if err != nil {
		return err
	}

	err = mmapZero(r.addr, r.size)
	r.zero = err == nil
	return err
}

func (r *MappedRegion) mmap(f *os.File, offset int64, readOnly bool) error {
	err := r.Close()
	if err != nil {
		return err
	}

	prot := uint32(windows.PAGE_READWRITE)
	if readOnly {
		prot = windows.PAGE_READONLY
	}
	err = mmapFile(f, offset, r.addr, r.size, prot)
	r.file = err == nil
	return err
}

func mmapFile(f *os.File, offset int64, addr uintptr, size int32, prot uint32) error {
	maxSize := offset + int64(size)

	h, err := windows.CreateFileMapping(
		windows.Handle(f.Fd()), nil, prot,
		uint32(maxSize>>32), uint32(maxSize), nil)
	if h == 0 {
		return err
	}
	defer windows.CloseHandle(h)

	_, err = mapViewOfFile3(h, addr, uint64(offset), uintptr(size),
		_MEM_REPLACE_PLACEHOLDER, prot)
	return err
}

func mmapZero(addr uintptr, size int32) error {
	maxSize := uint64(size)
	h, err := windows.CreateFileMapping(
		^windows.Handle(0), nil, windows.PAGE_READONLY,
		uint32(maxSize>>32), uint32(maxSize), nil)
	if h == 0 {
		return err
	}
	defer windows.CloseHandle(h)

	_, err = mapViewOfFile3(h, addr, 0, uintptr(size),
		_MEM_REPLACE_PLACEHOLDER, windows.PAGE_READONLY)
	return err
}
