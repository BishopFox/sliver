//go:build arm64 && ((linux && !android) || freebsd)

package linuxmem

import (
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

var (
	linuxCacheFlushOnce   sync.Once
	linuxCacheFlushAddr   uintptr
	linuxCacheFlushHandle uintptr
	linuxCacheFlushErr    error
)

func flushMappedInstructionCache(mapping []byte) error {
	if len(mapping) == 0 {
		return nil
	}
	linuxCacheFlushOnce.Do(func() {
		linuxCacheFlushAddr, linuxCacheFlushErr = purego.Dlsym(purego.RTLD_DEFAULT, "__clear_cache")
		if linuxCacheFlushErr == nil && linuxCacheFlushAddr != 0 {
			return
		}
		if runtime.GOOS == "freebsd" {
			for _, library := range []string{"libgcc_s.so.1", "libc.so.7"} {
				linuxCacheFlushHandle, linuxCacheFlushErr = purego.Dlopen(library, purego.RTLD_NOW|purego.RTLD_LOCAL)
				if linuxCacheFlushErr != nil {
					continue
				}
				linuxCacheFlushAddr, linuxCacheFlushErr = purego.Dlsym(linuxCacheFlushHandle, "__clear_cache")
				if linuxCacheFlushErr == nil && linuxCacheFlushAddr != 0 {
					return
				}
				_ = purego.Dlclose(linuxCacheFlushHandle)
				linuxCacheFlushHandle = 0
			}
			return
		}
		linuxCacheFlushHandle, linuxCacheFlushErr = purego.Dlopen("libgcc_s.so.1", purego.RTLD_NOW|purego.RTLD_LOCAL)
		if linuxCacheFlushErr != nil {
			return
		}
		linuxCacheFlushAddr, linuxCacheFlushErr = purego.Dlsym(linuxCacheFlushHandle, "__clear_cache")
	})
	if linuxCacheFlushErr != nil {
		return fmt.Errorf("resolve __clear_cache: %w", linuxCacheFlushErr)
	}
	if linuxCacheFlushAddr == 0 {
		return errors.New("resolve __clear_cache: symbol address is zero")
	}
	start := uintptr(unsafe.Pointer(unsafe.SliceData(mapping)))
	purego.SyscallN(linuxCacheFlushAddr, start, start+uintptr(len(mapping)))
	return nil
}
