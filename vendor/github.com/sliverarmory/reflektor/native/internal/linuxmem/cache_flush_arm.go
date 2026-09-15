//go:build linux && !android && arm && arm.7

package linuxmem

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Linux reserves this ARM-private syscall number for cacheflush.
const linuxARMCacheflush = 0x0f0002

func flushMappedInstructionCache(mapping []byte) error {
	if len(mapping) == 0 {
		return nil
	}
	start := uintptr(unsafe.Pointer(unsafe.SliceData(mapping)))
	_, _, errno := unix.Syscall(linuxARMCacheflush, start, start+uintptr(len(mapping)), 0)
	if errno != 0 {
		return fmt.Errorf("cacheflush mapped ELF image: %w", errno)
	}
	return nil
}
