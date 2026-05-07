package util

import (
	"os"
	"reflect"
	"unsafe"

	"golang.org/x/sys/windows"
)

type MappedRegion struct {
	windows.Handle
	Data []byte
	addr uintptr
}

func MapRegion(f *os.File, offset int64, size int32) (*MappedRegion, error) {
	maxSize := offset + int64(size)
	h, err := windows.CreateFileMapping(
		windows.Handle(f.Fd()), nil, windows.PAGE_READWRITE,
		uint32(maxSize>>32), uint32(maxSize), nil)
	if h == 0 {
		return nil, err
	}

	const allocationGranularity = 64 * 1024
	align := offset % allocationGranularity
	offset -= align

	a, err := windows.MapViewOfFile(h, windows.FILE_MAP_WRITE,
		uint32(offset>>32), uint32(offset), uintptr(size)+uintptr(align))
	if a == 0 {
		windows.CloseHandle(h)
		return nil, err
	}

	ret := &MappedRegion{Handle: h, addr: a}
	// SliceHeader, although deprecated, avoids a go vet warning.
	sh := (*reflect.SliceHeader)(unsafe.Pointer(&ret.Data))
	sh.Data = a + uintptr(align)
	sh.Len = int(size)
	sh.Cap = int(size)
	return ret, nil
}

func (r *MappedRegion) Unmap() error {
	if r.Data == nil {
		return nil
	}
	err := windows.UnmapViewOfFile(r.addr)
	if err != nil {
		return err
	}
	r.Data = nil
	return windows.CloseHandle(r.Handle)
}
