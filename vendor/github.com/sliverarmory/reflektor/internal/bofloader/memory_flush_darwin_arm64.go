//go:build darwin && !ios && arm64

package bofloader

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

var (
	darwinCacheFlushOnce sync.Once
	darwinCacheFlushAddr uintptr
	darwinCacheFlushErr  error
)

func (region *memoryRegion) flushInstructionCache(offset, length int) error {
	if _, err := region.rangeBytes(offset, length); err != nil || length == 0 {
		return err
	}
	darwinCacheFlushOnce.Do(func() {
		darwinCacheFlushAddr, darwinCacheFlushErr = purego.Dlsym(purego.RTLD_DEFAULT, "sys_icache_invalidate")
	})
	if darwinCacheFlushErr != nil || darwinCacheFlushAddr == 0 {
		if darwinCacheFlushErr == nil {
			darwinCacheFlushErr = errors.New("symbol address is zero")
		}
		return fmt.Errorf("resolve sys_icache_invalidate: %w", darwinCacheFlushErr)
	}
	purego.SyscallN(darwinCacheFlushAddr, region.base()+uintptr(offset), uintptr(length))
	return nil
}
