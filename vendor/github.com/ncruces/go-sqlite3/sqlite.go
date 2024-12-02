// Package sqlite3 wraps the C SQLite API.
package sqlite3

import (
	"context"
	"math"
	"math/bits"
	"os"
	"sync"
	"unsafe"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/experimental"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs"
)

// Configure SQLite Wasm.
//
// Importing package embed initializes [Binary]
// with an appropriate build of SQLite:
//
//	import _ "github.com/ncruces/go-sqlite3/embed"
var (
	Binary []byte // Wasm binary to load.
	Path   string // Path to load the binary from.

	RuntimeConfig wazero.RuntimeConfig
)

// Initialize decodes and compiles the SQLite Wasm binary.
// This is called implicitly when the first connection is openned,
// but is potentially slow, so you may want to call it at a more convenient time.
func Initialize() error {
	instance.once.Do(compileSQLite)
	return instance.err
}

var instance struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	err      error
	once     sync.Once
}

func compileSQLite() {
	ctx := context.Background()
	cfg := RuntimeConfig
	if cfg == nil {
		cfg = wazero.NewRuntimeConfig()
		if bits.UintSize >= 64 {
			cfg = cfg.WithMemoryLimitPages(4096) // 256MB
		} else {
			cfg = cfg.WithMemoryLimitPages(512) // 32MB
		}
	}
	cfg = cfg.WithCoreFeatures(api.CoreFeaturesV2 | experimental.CoreFeaturesThreads)

	instance.runtime = wazero.NewRuntimeWithConfig(ctx, cfg)

	env := instance.runtime.NewHostModuleBuilder("env")
	env = vfs.ExportHostFunctions(env)
	env = exportCallbacks(env)
	_, instance.err = env.Instantiate(ctx)
	if instance.err != nil {
		return
	}

	bin := Binary
	if bin == nil && Path != "" {
		bin, instance.err = os.ReadFile(Path)
		if instance.err != nil {
			return
		}
	}
	if bin == nil {
		instance.err = util.NoBinaryErr
		return
	}

	instance.compiled, instance.err = instance.runtime.CompileModule(ctx, bin)
}

type sqlite struct {
	ctx   context.Context
	mod   api.Module
	funcs struct {
		fn   [32]api.Function
		id   [32]*byte
		mask uint32
	}
	stack [9]uint64
}

func instantiateSQLite() (sqlt *sqlite, err error) {
	if err := Initialize(); err != nil {
		return nil, err
	}

	sqlt = new(sqlite)
	sqlt.ctx = util.NewContext(context.Background())

	sqlt.mod, err = instance.runtime.InstantiateModule(sqlt.ctx,
		instance.compiled, wazero.NewModuleConfig().WithName(""))
	if err != nil {
		return nil, err
	}
	if sqlt.getfn("sqlite3_progress_handler_go") == nil {
		return nil, util.BadBinaryErr
	}
	return sqlt, nil
}

func (sqlt *sqlite) close() error {
	return sqlt.mod.Close(sqlt.ctx)
}

func (sqlt *sqlite) error(rc uint64, handle uint32, sql ...string) error {
	if rc == _OK {
		return nil
	}

	err := Error{code: rc}

	if err.Code() == NOMEM || err.ExtendedCode() == IOERR_NOMEM {
		panic(util.OOMErr)
	}

	if r := sqlt.call("sqlite3_errstr", rc); r != 0 {
		err.str = util.ReadString(sqlt.mod, uint32(r), _MAX_NAME)
	}

	if handle != 0 {
		if r := sqlt.call("sqlite3_errmsg", uint64(handle)); r != 0 {
			err.msg = util.ReadString(sqlt.mod, uint32(r), _MAX_LENGTH)
		}

		if len(sql) != 0 {
			if r := sqlt.call("sqlite3_error_offset", uint64(handle)); r != math.MaxUint32 {
				err.sql = sql[0][r:]
			}
		}
	}

	switch err.msg {
	case err.str, "not an error":
		err.msg = ""
	}
	return &err
}

func (sqlt *sqlite) getfn(name string) api.Function {
	c := &sqlt.funcs
	p := unsafe.StringData(name)
	for i := range c.id {
		if c.id[i] == p {
			c.id[i] = nil
			c.mask &^= uint32(1) << i
			return c.fn[i]
		}
	}
	return sqlt.mod.ExportedFunction(name)
}

func (sqlt *sqlite) putfn(name string, fn api.Function) {
	c := &sqlt.funcs
	p := unsafe.StringData(name)
	i := bits.TrailingZeros32(^c.mask)
	if i < 32 {
		c.id[i] = p
		c.fn[i] = fn
		c.mask |= uint32(1) << i
	} else {
		c.id[0] = p
		c.fn[0] = fn
		c.mask = uint32(1)
	}
}

func (sqlt *sqlite) call(name string, params ...uint64) uint64 {
	copy(sqlt.stack[:], params)
	fn := sqlt.getfn(name)
	err := fn.CallWithStack(sqlt.ctx, sqlt.stack[:])
	if err != nil {
		panic(err)
	}
	sqlt.putfn(name, fn)
	return sqlt.stack[0]
}

func (sqlt *sqlite) free(ptr uint32) {
	if ptr == 0 {
		return
	}
	sqlt.call("sqlite3_free", uint64(ptr))
}

func (sqlt *sqlite) new(size uint64) uint32 {
	ptr := uint32(sqlt.call("sqlite3_malloc64", size))
	if ptr == 0 && size != 0 {
		panic(util.OOMErr)
	}
	return ptr
}

func (sqlt *sqlite) realloc(ptr uint32, size uint64) uint32 {
	ptr = uint32(sqlt.call("sqlite3_realloc64", uint64(ptr), size))
	if ptr == 0 && size != 0 {
		panic(util.OOMErr)
	}
	return ptr
}

func (sqlt *sqlite) newBytes(b []byte) uint32 {
	if (*[0]byte)(b) == nil {
		return 0
	}
	size := len(b)
	if size == 0 {
		size = 1
	}
	ptr := sqlt.new(uint64(size))
	util.WriteBytes(sqlt.mod, ptr, b)
	return ptr
}

func (sqlt *sqlite) newString(s string) uint32 {
	ptr := sqlt.new(uint64(len(s) + 1))
	util.WriteString(sqlt.mod, ptr, s)
	return ptr
}

func (sqlt *sqlite) newArena(size uint64) arena {
	// Ensure the arena's size is a multiple of 8.
	size = (size + 7) &^ 7
	return arena{
		sqlt: sqlt,
		size: uint32(size),
		base: sqlt.new(size),
	}
}

type arena struct {
	sqlt *sqlite
	ptrs []uint32
	base uint32
	next uint32
	size uint32
}

func (a *arena) free() {
	if a.sqlt == nil {
		return
	}
	for _, ptr := range a.ptrs {
		a.sqlt.free(ptr)
	}
	a.sqlt.free(a.base)
	a.sqlt = nil
}

func (a *arena) mark() (reset func()) {
	ptrs := len(a.ptrs)
	next := a.next
	return func() {
		for _, ptr := range a.ptrs[ptrs:] {
			a.sqlt.free(ptr)
		}
		a.ptrs = a.ptrs[:ptrs]
		a.next = next
	}
}

func (a *arena) new(size uint64) uint32 {
	// Align the next address, to 4 or 8 bytes.
	if size&7 != 0 {
		a.next = (a.next + 3) &^ 3
	} else {
		a.next = (a.next + 7) &^ 7
	}
	if size <= uint64(a.size-a.next) {
		ptr := a.base + a.next
		a.next += uint32(size)
		return ptr
	}
	ptr := a.sqlt.new(size)
	a.ptrs = append(a.ptrs, ptr)
	return ptr
}

func (a *arena) bytes(b []byte) uint32 {
	if (*[0]byte)(b) == nil {
		return 0
	}
	ptr := a.new(uint64(len(b)))
	util.WriteBytes(a.sqlt.mod, ptr, b)
	return ptr
}

func (a *arena) string(s string) uint32 {
	ptr := a.new(uint64(len(s) + 1))
	util.WriteString(a.sqlt.mod, ptr, s)
	return ptr
}

func exportCallbacks(env wazero.HostModuleBuilder) wazero.HostModuleBuilder {
	util.ExportFuncII(env, "go_progress_handler", progressCallback)
	util.ExportFuncIII(env, "go_busy_timeout", timeoutCallback)
	util.ExportFuncIII(env, "go_busy_handler", busyCallback)
	util.ExportFuncII(env, "go_commit_hook", commitCallback)
	util.ExportFuncVI(env, "go_rollback_hook", rollbackCallback)
	util.ExportFuncVIIIIJ(env, "go_update_hook", updateCallback)
	util.ExportFuncIIIII(env, "go_wal_hook", walCallback)
	util.ExportFuncIIIII(env, "go_trace", traceCallback)
	util.ExportFuncIIIIII(env, "go_autovacuum_pages", autoVacuumCallback)
	util.ExportFuncIIIIIII(env, "go_authorizer", authorizerCallback)
	util.ExportFuncVIII(env, "go_log", logCallback)
	util.ExportFuncVI(env, "go_destroy", destroyCallback)
	util.ExportFuncVIIII(env, "go_func", funcCallback)
	util.ExportFuncVIIIII(env, "go_step", stepCallback)
	util.ExportFuncVIII(env, "go_final", finalCallback)
	util.ExportFuncVII(env, "go_value", valueCallback)
	util.ExportFuncVIIII(env, "go_inverse", inverseCallback)
	util.ExportFuncVIIII(env, "go_collation_needed", collationCallback)
	util.ExportFuncIIIIII(env, "go_compare", compareCallback)
	util.ExportFuncIIIIII(env, "go_vtab_create", vtabModuleCallback(xCreate))
	util.ExportFuncIIIIII(env, "go_vtab_connect", vtabModuleCallback(xConnect))
	util.ExportFuncII(env, "go_vtab_disconnect", vtabDisconnectCallback)
	util.ExportFuncII(env, "go_vtab_destroy", vtabDestroyCallback)
	util.ExportFuncIII(env, "go_vtab_best_index", vtabBestIndexCallback)
	util.ExportFuncIIIII(env, "go_vtab_update", vtabUpdateCallback)
	util.ExportFuncIII(env, "go_vtab_rename", vtabRenameCallback)
	util.ExportFuncIIIII(env, "go_vtab_find_function", vtabFindFuncCallback)
	util.ExportFuncII(env, "go_vtab_begin", vtabBeginCallback)
	util.ExportFuncII(env, "go_vtab_sync", vtabSyncCallback)
	util.ExportFuncII(env, "go_vtab_commit", vtabCommitCallback)
	util.ExportFuncII(env, "go_vtab_rollback", vtabRollbackCallback)
	util.ExportFuncIII(env, "go_vtab_savepoint", vtabSavepointCallback)
	util.ExportFuncIII(env, "go_vtab_release", vtabReleaseCallback)
	util.ExportFuncIII(env, "go_vtab_rollback_to", vtabRollbackToCallback)
	util.ExportFuncIIIIII(env, "go_vtab_integrity", vtabIntegrityCallback)
	util.ExportFuncIII(env, "go_cur_open", cursorOpenCallback)
	util.ExportFuncII(env, "go_cur_close", cursorCloseCallback)
	util.ExportFuncIIIIII(env, "go_cur_filter", cursorFilterCallback)
	util.ExportFuncII(env, "go_cur_next", cursorNextCallback)
	util.ExportFuncII(env, "go_cur_eof", cursorEOFCallback)
	util.ExportFuncIIII(env, "go_cur_column", cursorColumnCallback)
	util.ExportFuncIII(env, "go_cur_rowid", cursorRowIDCallback)
	return env
}
