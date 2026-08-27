//go:build (freebsd || (linux && !android)) && arm64

package bofloader

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ebitengine/purego"
)

var (
	elfCacheFlushOnce   sync.Once
	elfCacheFlushAddr   uintptr
	elfCacheFlushHandle uintptr
	elfCacheFlushErr    error
)

func (region *memoryRegion) flushInstructionCache(offset, length int) error {
	if _, err := region.rangeBytes(offset, length); err != nil || length == 0 {
		return err
	}
	elfCacheFlushOnce.Do(func() {
		elfCacheFlushAddr, elfCacheFlushErr = purego.Dlsym(purego.RTLD_DEFAULT, "__clear_cache")
		if elfCacheFlushErr == nil && elfCacheFlushAddr != 0 {
			return
		}
		for _, library := range []string{"libgcc_s.so.1", "libc.so.7"} {
			elfCacheFlushHandle, elfCacheFlushErr = purego.Dlopen(library, purego.RTLD_NOW|purego.RTLD_LOCAL)
			if elfCacheFlushErr != nil {
				continue
			}
			elfCacheFlushAddr, elfCacheFlushErr = purego.Dlsym(elfCacheFlushHandle, "__clear_cache")
			if elfCacheFlushErr == nil && elfCacheFlushAddr != 0 {
				return
			}
			_ = purego.Dlclose(elfCacheFlushHandle)
			elfCacheFlushHandle = 0
		}
	})
	if elfCacheFlushErr != nil {
		return fmt.Errorf("resolve __clear_cache: %w", elfCacheFlushErr)
	}
	if elfCacheFlushAddr == 0 {
		return errors.New("resolve __clear_cache: symbol address is zero")
	}
	start := region.base() + uintptr(offset)
	purego.SyscallN(elfCacheFlushAddr, start, start+uintptr(length))
	return nil
}
