package util

import (
	"context"

	"github.com/tetratelabs/wazero/experimental"

	"github.com/ncruces/go-sqlite3/internal/alloc"
)

type ConnKey struct{}

type moduleKey struct{}
type moduleState struct {
	sysError error
	mmapState
	handleState
}

func NewContext(ctx context.Context) context.Context {
	state := new(moduleState)
	ctx = experimental.WithMemoryAllocator(ctx, experimental.MemoryAllocatorFunc(alloc.NewMemory))
	ctx = experimental.WithCloseNotifier(ctx, state)
	ctx = context.WithValue(ctx, moduleKey{}, state)
	return ctx
}

func GetSystemError(ctx context.Context) error {
	// Test needed to simplify testing.
	s, ok := ctx.Value(moduleKey{}).(*moduleState)
	if ok {
		return s.sysError
	}
	return nil
}

func SetSystemError(ctx context.Context, err error) {
	// Test needed to simplify testing.
	s, ok := ctx.Value(moduleKey{}).(*moduleState)
	if ok {
		s.sysError = err
	}
}
