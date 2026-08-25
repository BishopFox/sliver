//go:build linux && !android && ppc64le && cgo

package memmod

/*
#include <stdint.h>

static void reflektor_clear_cache(uintptr_t start, uintptr_t end) {
	const uintptr_t block_size = 32;
	start &= ~(block_size - 1);
	end = (end + block_size - 1) & ~(block_size - 1);
	uintptr_t cursor;
	for (cursor = start; cursor < end; cursor += block_size) {
		__asm__ __volatile__("dcbf 0,%0" : : "r"(cursor) : "memory");
	}
	__asm__ __volatile__("sync" : : : "memory");
	for (cursor = start; cursor < end; cursor += block_size) {
		__asm__ __volatile__("icbi 0,%0" : : "r"(cursor) : "memory");
	}
	__asm__ __volatile__("sync\n\tisync" : : : "memory");
}
*/
import "C"

func flushLinuxInstructionCache(start, end uintptr) error {
	if start < end {
		C.reflektor_clear_cache(C.uintptr_t(start), C.uintptr_t(end))
	}
	return nil
}
