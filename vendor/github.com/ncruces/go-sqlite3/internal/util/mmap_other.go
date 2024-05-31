//go:build !(darwin || linux) || !(amd64 || arm64 || riscv64) || sqlite3_noshm || sqlite3_nosys

package util

import (
	"context"

	"github.com/tetratelabs/wazero/experimental"
)

type mmapState struct{}

func withAllocator(ctx context.Context) context.Context {
	return experimental.WithMemoryAllocator(ctx,
		experimental.MemoryAllocatorFunc(func(cap, max uint64) experimental.LinearMemory {
			if cap == max {
				return virtualAlloc(cap, max)
			}
			return sliceAlloc(cap, max)
		}))
}
