//go:build windows && (386 || amd64 || arm64)

package bofloader

import (
	"errors"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var flushInstructionCacheProc = windows.NewLazySystemDLL("kernel32.dll").NewProc("FlushInstructionCache")

type memoryRegion struct {
	address uintptr
	data    []byte
}

func allocateMemory(size int) (*memoryRegion, error) {
	if size <= 0 || size > maxImageSize {
		return nil, fmt.Errorf("invalid allocation size %d", size)
	}
	address, err := windows.VirtualAlloc(0, uintptr(size), windows.MEM_COMMIT|windows.MEM_RESERVE, windows.PAGE_READWRITE)
	if err != nil {
		return nil, err
	}
	if address == 0 {
		return nil, errors.New("VirtualAlloc returned a nil address")
	}
	return &memoryRegion{address: address, data: unsafe.Slice((*byte)(unsafe.Pointer(address)), size)}, nil
}

func (region *memoryRegion) base() uintptr {
	if region == nil {
		return 0
	}
	return region.address
}

func (region *memoryRegion) protect(offset, length int, requested protection) error {
	if err := region.validateRange(offset, length); err != nil || length == 0 {
		return err
	}
	if requested&protWrite != 0 && requested&protExec != 0 {
		return errors.New("writable and executable memory protection is forbidden")
	}
	var nativeProtection uint32
	switch requested {
	case 0:
		nativeProtection = windows.PAGE_NOACCESS
	case protRead:
		nativeProtection = windows.PAGE_READONLY
	case protWrite, protRead | protWrite:
		nativeProtection = windows.PAGE_READWRITE
	case protExec:
		nativeProtection = windows.PAGE_EXECUTE
	case protRead | protExec:
		nativeProtection = windows.PAGE_EXECUTE_READ
	default:
		return fmt.Errorf("unsupported memory protection %#x", requested)
	}
	var previous uint32
	return windows.VirtualProtect(region.address+uintptr(offset), uintptr(length), nativeProtection, &previous)
}

func (region *memoryRegion) flushInstructionCache(offset, length int) error {
	if err := region.validateRange(offset, length); err != nil || length == 0 {
		return err
	}
	process, err := windows.GetCurrentProcess()
	if err != nil {
		return err
	}
	result, _, callErr := flushInstructionCacheProc.Call(uintptr(process), region.address+uintptr(offset), uintptr(length))
	if result == 0 {
		return callErr
	}
	return nil
}

func (region *memoryRegion) close() error {
	if region == nil || region.address == 0 {
		return nil
	}
	address := region.address
	if err := windows.VirtualFree(address, 0, windows.MEM_RELEASE); err != nil {
		return err
	}
	region.address = 0
	region.data = nil
	return nil
}

func (region *memoryRegion) validateRange(offset, length int) error {
	if region == nil || region.address == 0 || len(region.data) == 0 {
		return errors.New("memory region is closed")
	}
	if offset < 0 || length < 0 || offset > len(region.data) || length > len(region.data)-offset {
		return fmt.Errorf("memory range offset=%d length=%d exceeds %d-byte region", offset, length, len(region.data))
	}
	if length != 0 {
		pageSize := memoryPageSize()
		if offset%pageSize != 0 || length%pageSize != 0 {
			return fmt.Errorf("memory range offset=%d length=%d is not page aligned to %d", offset, length, pageSize)
		}
	}
	return nil
}

func memoryPageSize() int {
	return os.Getpagesize()
}
