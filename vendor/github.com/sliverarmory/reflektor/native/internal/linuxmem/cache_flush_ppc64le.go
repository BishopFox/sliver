//go:build linux && !android && ppc64le && !cgo

package linuxmem

import "unsafe"

const ppc64CacheBlockSize uintptr = 32

//go:noescape
func flushPPC64InstructionCache(start, end uintptr)

func ppc64CacheBlockRange(start, end uintptr) (uintptr, uintptr) {
	start &^= ppc64CacheBlockSize - 1
	if remainder := end & (ppc64CacheBlockSize - 1); remainder != 0 {
		end += ppc64CacheBlockSize - remainder
	}
	return start, end
}

func flushMappedInstructionCache(mapping []byte) error {
	if len(mapping) == 0 {
		return nil
	}
	start := uintptr(unsafe.Pointer(unsafe.SliceData(mapping)))
	start, end := ppc64CacheBlockRange(start, start+uintptr(len(mapping)))
	flushPPC64InstructionCache(start, end)
	return nil
}
