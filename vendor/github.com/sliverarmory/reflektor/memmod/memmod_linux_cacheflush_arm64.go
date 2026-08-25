//go:build arm64 && ((linux && !android) || freebsd)

package memmod

import (
	"errors"
	"fmt"
	"runtime"
	"sync"

	"github.com/ebitengine/purego"
)

var (
	linuxARM64CacheFlushOnce   sync.Once
	linuxARM64CacheFlushAddr   uintptr
	linuxARM64CacheFlushHandle uintptr
	linuxARM64CacheFlushErr    error
)

func flushLinuxInstructionCache(start, end uintptr) error {
	if start >= end {
		return nil
	}
	linuxARM64CacheFlushOnce.Do(func() {
		linuxARM64CacheFlushAddr, linuxARM64CacheFlushErr = purego.Dlsym(purego.RTLD_DEFAULT, "__clear_cache")
		if linuxARM64CacheFlushErr == nil && linuxARM64CacheFlushAddr != 0 {
			return
		}
		if runtime.GOOS == "freebsd" {
			for _, library := range []string{"libgcc_s.so.1", "libc.so.7"} {
				linuxARM64CacheFlushHandle, linuxARM64CacheFlushErr = purego.Dlopen(library, purego.RTLD_NOW|purego.RTLD_LOCAL)
				if linuxARM64CacheFlushErr != nil {
					continue
				}
				linuxARM64CacheFlushAddr, linuxARM64CacheFlushErr = purego.Dlsym(linuxARM64CacheFlushHandle, "__clear_cache")
				if linuxARM64CacheFlushErr == nil && linuxARM64CacheFlushAddr != 0 {
					return
				}
				_ = purego.Dlclose(linuxARM64CacheFlushHandle)
				linuxARM64CacheFlushHandle = 0
			}
			return
		}
		linuxARM64CacheFlushHandle, linuxARM64CacheFlushErr = purego.Dlopen("libgcc_s.so.1", purego.RTLD_NOW|purego.RTLD_LOCAL)
		if linuxARM64CacheFlushErr != nil {
			return
		}
		linuxARM64CacheFlushAddr, linuxARM64CacheFlushErr = purego.Dlsym(linuxARM64CacheFlushHandle, "__clear_cache")
	})
	if linuxARM64CacheFlushErr != nil {
		return fmt.Errorf("resolve __clear_cache: %w", linuxARM64CacheFlushErr)
	}
	if linuxARM64CacheFlushAddr == 0 {
		return errors.New("resolve __clear_cache: symbol address is zero")
	}
	purego.SyscallN(linuxARM64CacheFlushAddr, start, end)
	return nil
}
