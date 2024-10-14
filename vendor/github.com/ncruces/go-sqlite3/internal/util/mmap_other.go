//go:build !unix || !(amd64 || arm64 || riscv64 || ppc64le) || sqlite3_noshm || sqlite3_nosys

package util

import (
	"context"

	"github.com/ncruces/go-sqlite3/internal/alloc"
	"github.com/tetratelabs/wazero/experimental"
)

type mmapState struct{}

func withAllocator(ctx context.Context) context.Context {
	return experimental.WithMemoryAllocator(ctx,
		experimental.MemoryAllocatorFunc(func(cap, max uint64) experimental.LinearMemory {
			if cap == max {
				return alloc.Virtual(cap, max)
			}
			return alloc.Slice(cap, max)
		}))
}
