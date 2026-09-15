//go:build linux && !android && arm && arm.7

package bofloader

import (
	"fmt"

	"golang.org/x/sys/unix"
)

// Linux reserves this ARM-private syscall number for cacheflush. Unlike a
// libgcc __clear_cache lookup, it is available without CGO or an extra DSO.
const linuxARMCacheflush = 0x0f0002

func (region *memoryRegion) flushInstructionCache(offset, length int) error {
	if _, err := region.rangeBytes(offset, length); err != nil || length == 0 {
		return err
	}
	start := region.base() + uintptr(offset)
	_, _, errno := unix.Syscall(linuxARMCacheflush, start, start+uintptr(length), 0)
	if errno != 0 {
		return fmt.Errorf("cacheflush executable memory: %w", errno)
	}
	return nil
}
