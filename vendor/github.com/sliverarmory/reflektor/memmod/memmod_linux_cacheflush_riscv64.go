//go:build linux && !android && riscv64

package memmod

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func flushLinuxInstructionCache(start, end uintptr) error {
	if start >= end {
		return nil
	}
	// A process-wide flush is required because the Go scheduler may resume the
	// export call on a different hart from the one that populated the mapping.
	_, _, errno := unix.Syscall(unix.SYS_RISCV_FLUSH_ICACHE, start, end, 0)
	if errno != 0 {
		return fmt.Errorf("flush executable instruction cache: %w", errno)
	}
	return nil
}
