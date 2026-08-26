//go:build linux && !android && arm && arm.7

package memmod

import (
	"fmt"

	"golang.org/x/sys/unix"
)

const linuxARMCacheflush = 0x0f0002

func flushLinuxInstructionCache(start, end uintptr) error {
	if start >= end {
		return nil
	}
	_, _, errno := unix.Syscall(linuxARMCacheflush, start, end, 0)
	if errno != 0 {
		return fmt.Errorf("cacheflush executable memory: %w", errno)
	}
	return nil
}
