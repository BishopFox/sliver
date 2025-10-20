package util

import (
	"context"

	"github.com/tetratelabs/wazero/experimental"
)

type moduleKey struct{}
type moduleState struct {
	mmapState
	handleState
}

func NewContext(ctx context.Context) context.Context {
	state := new(moduleState)
	ctx = withMmappedAllocator(ctx)
	ctx = experimental.WithCloseNotifier(ctx, state)
	ctx = context.WithValue(ctx, moduleKey{}, state)
	return ctx
}
