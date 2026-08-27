//go:build linux && !android && ppc64le && !cgo

package memmod

const ppc64CacheBlockSize uintptr = 32

//go:noescape
func flushPPC64InstructionCache(start, end uintptr)

func flushLinuxInstructionCache(start, end uintptr) error {
	if start >= end {
		return nil
	}
	start &^= ppc64CacheBlockSize - 1
	if remainder := end & (ppc64CacheBlockSize - 1); remainder != 0 {
		end += ppc64CacheBlockSize - remainder
	}
	flushPPC64InstructionCache(start, end)
	return nil
}
