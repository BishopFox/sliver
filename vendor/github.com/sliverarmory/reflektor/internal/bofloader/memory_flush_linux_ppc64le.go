//go:build linux && !android && ppc64le

package bofloader

const ppc64CacheBlockSize uintptr = 32

// flushPPC64InstructionCache establishes instruction/data cache coherency for
// whole cache blocks in the aligned range [start, end). PowerPC has no Linux
// cache-flush syscall, so this is implemented with userspace cache-management
// instructions.
//
//go:noescape
func flushPPC64InstructionCache(start, end uintptr)

func ppc64CacheBlockRange(start, end uintptr) (uintptr, uintptr) {
	start &^= ppc64CacheBlockSize - 1
	if remainder := end & (ppc64CacheBlockSize - 1); remainder != 0 {
		end += ppc64CacheBlockSize - remainder
	}
	return start, end
}

func (region *memoryRegion) flushInstructionCache(offset, length int) error {
	if _, err := region.rangeBytes(offset, length); err != nil || length == 0 {
		return err
	}
	start := region.base() + uintptr(offset)
	start, end := ppc64CacheBlockRange(start, start+uintptr(length))
	flushPPC64InstructionCache(start, end)
	return nil
}
