//go:build linux && !android && riscv64

package bofloader

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func (region *memoryRegion) flushInstructionCache(offset, length int) error {
	if _, err := region.rangeBytes(offset, length); err != nil || length == 0 {
		return err
	}
	start := region.base() + uintptr(offset)
	// Zero flags requests a process-wide flush. A local-hart flush is not safe
	// because the Go scheduler may resume BOF execution on another hart.
	_, _, errno := unix.Syscall(unix.SYS_RISCV_FLUSH_ICACHE, start, start+uintptr(length), 0)
	if errno != 0 {
		return fmt.Errorf("flush executable instruction cache: %w", errno)
	}
	return nil
}
