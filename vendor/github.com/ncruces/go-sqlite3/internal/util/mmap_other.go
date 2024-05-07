//go:build !(darwin || linux) || !(amd64 || arm64 || riscv64) || sqlite3_flock || sqlite3_noshm || sqlite3_nosys

package util

import "context"

type mmapState struct{}

func withMmappedAllocator(ctx context.Context) context.Context {
	return ctx
}
