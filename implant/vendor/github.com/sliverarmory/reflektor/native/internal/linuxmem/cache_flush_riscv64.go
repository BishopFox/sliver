//go:build linux && !android && riscv64

package linuxmem

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

func flushMappedInstructionCache(mapping []byte) error {
	if len(mapping) == 0 {
		return nil
	}
	start := uintptr(unsafe.Pointer(unsafe.SliceData(mapping)))
	_, _, errno := unix.Syscall(unix.SYS_RISCV_FLUSH_ICACHE, start, start+uintptr(len(mapping)), 0)
	if errno != 0 {
		return fmt.Errorf("flush mapped ELF instruction cache: %w", errno)
	}
	return nil
}
