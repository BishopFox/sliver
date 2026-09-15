//go:build linux && !android && ppc64le && cgo

package linuxmem

import (
	"errors"
	"fmt"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	ppc64CacheFlushOnce   sync.Once
	ppc64CacheFlushAddr   uintptr
	ppc64CacheFlushHandle uintptr
	ppc64CacheFlushErr    error
)

func flushMappedInstructionCache(mapping []byte) error {
	if len(mapping) == 0 {
		return nil
	}
	ppc64CacheFlushOnce.Do(func() {
		ppc64CacheFlushAddr, ppc64CacheFlushErr = purego.Dlsym(purego.RTLD_DEFAULT, "__clear_cache")
		if ppc64CacheFlushErr == nil && ppc64CacheFlushAddr != 0 {
			return
		}
		ppc64CacheFlushHandle, ppc64CacheFlushErr = purego.Dlopen("libgcc_s.so.1", purego.RTLD_NOW|purego.RTLD_LOCAL)
		if ppc64CacheFlushErr != nil {
			return
		}
		ppc64CacheFlushAddr, ppc64CacheFlushErr = purego.Dlsym(ppc64CacheFlushHandle, "__clear_cache")
	})
	if ppc64CacheFlushErr != nil {
		return fmt.Errorf("resolve __clear_cache: %w", ppc64CacheFlushErr)
	}
	if ppc64CacheFlushAddr == 0 {
		return errors.New("resolve __clear_cache: symbol address is zero")
	}
	start := uintptr(unsafe.Pointer(unsafe.SliceData(mapping)))
	purego.SyscallN(ppc64CacheFlushAddr, start, start+uintptr(len(mapping)))
	return nil
}
