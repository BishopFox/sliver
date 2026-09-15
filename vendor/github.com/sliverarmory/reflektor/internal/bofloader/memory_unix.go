//go:build (darwin && !ios && (amd64 || arm64)) || (freebsd && (amd64 || arm64)) || (linux && !android && (386 || amd64 || (arm && arm.7) || arm64 || ppc64le || riscv64))

package bofloader

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

type memoryRegion struct {
	data []byte
}

func allocateMemory(size int) (*memoryRegion, error) {
	if size <= 0 || size > maxImageSize {
		return nil, fmt.Errorf("invalid allocation size %d", size)
	}
	data, err := unix.Mmap(-1, 0, size, unix.PROT_READ|unix.PROT_WRITE, unix.MAP_PRIVATE|unix.MAP_ANON)
	if err != nil {
		return nil, err
	}
	if len(data) != size {
		_ = unix.Munmap(data)
		return nil, fmt.Errorf("mmap returned %d bytes, want %d", len(data), size)
	}
	return &memoryRegion{data: data}, nil
}

func (region *memoryRegion) base() uintptr {
	if region == nil || len(region.data) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(unsafe.SliceData(region.data)))
}

func (region *memoryRegion) protect(offset, length int, requested protection) error {
	data, err := region.rangeBytes(offset, length)
	if err != nil || length == 0 {
		return err
	}
	if requested&protWrite != 0 && requested&protExec != 0 {
		return errors.New("writable and executable memory protection is forbidden")
	}
	flags := 0
	if requested&protRead != 0 {
		flags |= unix.PROT_READ
	}
	if requested&protWrite != 0 {
		flags |= unix.PROT_WRITE
	}
	if requested&protExec != 0 {
		flags |= unix.PROT_EXEC
	}
	return unix.Mprotect(data, flags)
}

func (region *memoryRegion) close() error {
	if region == nil || len(region.data) == 0 {
		return nil
	}
	data := region.data
	if err := unix.Munmap(data); err != nil {
		return err
	}
	region.data = nil
	return nil
}

func (region *memoryRegion) rangeBytes(offset, length int) ([]byte, error) {
	if region == nil || len(region.data) == 0 {
		return nil, errors.New("memory region is closed")
	}
	if offset < 0 || length < 0 || offset > len(region.data) || length > len(region.data)-offset {
		return nil, fmt.Errorf("memory range offset=%d length=%d exceeds %d-byte region", offset, length, len(region.data))
	}
	if length != 0 {
		pageSize := memoryPageSize()
		if offset%pageSize != 0 || length%pageSize != 0 {
			return nil, fmt.Errorf("memory range offset=%d length=%d is not page aligned to %d", offset, length, pageSize)
		}
	}
	return region.data[offset : offset+length], nil
}

func memoryPageSize() int {
	return os.Getpagesize()
}
