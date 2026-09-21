// Package sqlite3 wraps the C SQLite API.
package sqlite3

import (
	"context"
	"math/bits"

	sqlite3_wasm "github.com/ncruces/go-sqlite3-wasm/v6"
	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
	"github.com/ncruces/go-sqlite3/vfs"
)

type configKey struct{}

// WithMaxMemory returns a derived context that configures
// each SQLite connection not to use more than max amount of memory.
func WithMaxMemory(ctx context.Context, max int64) context.Context {
	if max < 0 || max > 65536*65536 {
		panic(errutil.OOMErr)
	}
	return context.WithValue(ctx, configKey{}, max/65536)
}

type env struct{ *sqlite3_wrap.Wrapper }

func createWrapper(ctx context.Context) (*sqlite3_wrap.Wrapper, error) {
	mem := &sqlite3_wrap.Memory{Max: 4096} // 256MB
	if bits.UintSize < 64 {
		mem.Max = 512 // 32MB
	}
	if cfg, ok := ctx.Value(configKey{}).(int64); ok {
		mem.Max = max(cfg, int64(len(mem.Buf))/65536)
	}
	mem.Grow(5 /*320KB*/, mem.Max)

	env := env{&sqlite3_wrap.Wrapper{Memory: mem}}
	env.Module = sqlite3_wasm.New(env)
	env.X_initialize()
	return env.Wrapper, nil
}

func (e env) Xgo_vfs_find(zVfsName int32) int32 {
	if vfs.Find(e.ReadString(ptr_t(zVfsName), _MAX_NAME)) != nil {
		return 1
	}
	return 0
}
