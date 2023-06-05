// Package sqlite3 wraps the C SQLite API.
package sqlite3

import (
	"context"
	"io"
	"math"
	"os"
	"sync"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// Configure SQLite WASM.
//
// Importing package embed initializes these
// with an appropriate build of SQLite:
//
//	import _ "github.com/ncruces/go-sqlite3/embed"
var (
	Binary []byte // WASM binary to load.
	Path   string // Path to load the binary from.
)

var sqlite3 struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	err      error
	once     sync.Once
}

func instantiateModule() (*module, error) {
	ctx := context.Background()

	sqlite3.once.Do(compileModule)
	if sqlite3.err != nil {
		return nil, sqlite3.err
	}

	cfg := wazero.NewModuleConfig()

	mod, err := sqlite3.runtime.InstantiateModule(ctx, sqlite3.compiled, cfg)
	if err != nil {
		return nil, err
	}
	return newModule(mod)
}

func compileModule() {
	ctx := context.Background()
	sqlite3.runtime = wazero.NewRuntime(ctx)

	env := vfs.ExportHostFunctions(sqlite3.runtime.NewHostModuleBuilder("env"))
	_, sqlite3.err = env.Instantiate(ctx)
	if sqlite3.err != nil {
		return
	}

	bin := Binary
	if bin == nil && Path != "" {
		bin, sqlite3.err = os.ReadFile(Path)
		if sqlite3.err != nil {
			return
		}
	}
	if bin == nil {
		sqlite3.err = util.BinaryErr
		return
	}

	sqlite3.compiled, sqlite3.err = sqlite3.runtime.CompileModule(ctx, bin)
}

type module struct {
	ctx context.Context
	mod api.Module
	vfs io.Closer
	api sqliteAPI
	arg [8]uint64
}

func newModule(mod api.Module) (m *module, err error) {
	m = new(module)
	m.mod = mod
	m.ctx, m.vfs = vfs.NewContext(context.Background())

	getFun := func(name string) api.Function {
		f := mod.ExportedFunction(name)
		if f == nil {
			err = util.NoFuncErr + util.ErrorString(name)
			return nil
		}
		return f
	}

	getVal := func(name string) uint32 {
		g := mod.ExportedGlobal(name)
		if g == nil {
			err = util.NoGlobalErr + util.ErrorString(name)
			return 0
		}
		return util.ReadUint32(mod, uint32(g.Get()))
	}

	m.api = sqliteAPI{
		free:            getFun("free"),
		malloc:          getFun("malloc"),
		destructor:      getVal("malloc_destructor"),
		errcode:         getFun("sqlite3_errcode"),
		errstr:          getFun("sqlite3_errstr"),
		errmsg:          getFun("sqlite3_errmsg"),
		erroff:          getFun("sqlite3_error_offset"),
		open:            getFun("sqlite3_open_v2"),
		close:           getFun("sqlite3_close"),
		closeZombie:     getFun("sqlite3_close_v2"),
		prepare:         getFun("sqlite3_prepare_v3"),
		finalize:        getFun("sqlite3_finalize"),
		reset:           getFun("sqlite3_reset"),
		step:            getFun("sqlite3_step"),
		exec:            getFun("sqlite3_exec"),
		clearBindings:   getFun("sqlite3_clear_bindings"),
		bindCount:       getFun("sqlite3_bind_parameter_count"),
		bindIndex:       getFun("sqlite3_bind_parameter_index"),
		bindName:        getFun("sqlite3_bind_parameter_name"),
		bindNull:        getFun("sqlite3_bind_null"),
		bindInteger:     getFun("sqlite3_bind_int64"),
		bindFloat:       getFun("sqlite3_bind_double"),
		bindText:        getFun("sqlite3_bind_text64"),
		bindBlob:        getFun("sqlite3_bind_blob64"),
		bindZeroBlob:    getFun("sqlite3_bind_zeroblob64"),
		columnCount:     getFun("sqlite3_column_count"),
		columnName:      getFun("sqlite3_column_name"),
		columnType:      getFun("sqlite3_column_type"),
		columnInteger:   getFun("sqlite3_column_int64"),
		columnFloat:     getFun("sqlite3_column_double"),
		columnText:      getFun("sqlite3_column_text"),
		columnBlob:      getFun("sqlite3_column_blob"),
		columnBytes:     getFun("sqlite3_column_bytes"),
		blobOpen:        getFun("sqlite3_blob_open"),
		blobClose:       getFun("sqlite3_blob_close"),
		blobReopen:      getFun("sqlite3_blob_reopen"),
		blobBytes:       getFun("sqlite3_blob_bytes"),
		blobRead:        getFun("sqlite3_blob_read"),
		blobWrite:       getFun("sqlite3_blob_write"),
		backupInit:      getFun("sqlite3_backup_init"),
		backupStep:      getFun("sqlite3_backup_step"),
		backupFinish:    getFun("sqlite3_backup_finish"),
		backupRemaining: getFun("sqlite3_backup_remaining"),
		backupPageCount: getFun("sqlite3_backup_pagecount"),
		changes:         getFun("sqlite3_changes64"),
		lastRowid:       getFun("sqlite3_last_insert_rowid"),
		autocommit:      getFun("sqlite3_get_autocommit"),
	}
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (m *module) close() error {
	err := m.mod.Close(m.ctx)
	m.vfs.Close()
	return err
}

func (m *module) error(rc uint64, handle uint32, sql ...string) error {
	if rc == _OK {
		return nil
	}

	err := Error{code: rc}

	if err.Code() == NOMEM || err.ExtendedCode() == IOERR_NOMEM {
		panic(util.OOMErr)
	}

	if r := m.call(m.api.errstr, rc); r != 0 {
		err.str = util.ReadString(m.mod, uint32(r), _MAX_STRING)
	}

	if r := m.call(m.api.errmsg, uint64(handle)); r != 0 {
		err.msg = util.ReadString(m.mod, uint32(r), _MAX_STRING)
	}

	if sql != nil {
		if r := m.call(m.api.erroff, uint64(handle)); r != math.MaxUint32 {
			err.sql = sql[0][r:]
		}
	}

	switch err.msg {
	case err.str, "not an error":
		err.msg = ""
	}
	return &err
}

func (m *module) call(fn api.Function, params ...uint64) uint64 {
	copy(m.arg[:], params)
	err := fn.CallWithStack(m.ctx, m.arg[:])
	if err != nil {
		// The module closed or panicked; release resources.
		m.vfs.Close()
		panic(err)
	}
	return m.arg[0]
}

func (m *module) free(ptr uint32) {
	if ptr == 0 {
		return
	}
	m.call(m.api.free, uint64(ptr))
}

func (m *module) new(size uint64) uint32 {
	if size > _MAX_ALLOCATION_SIZE {
		panic(util.OOMErr)
	}
	ptr := uint32(m.call(m.api.malloc, size))
	if ptr == 0 && size != 0 {
		panic(util.OOMErr)
	}
	return ptr
}

func (m *module) newBytes(b []byte) uint32 {
	if b == nil {
		return 0
	}
	ptr := m.new(uint64(len(b)))
	util.WriteBytes(m.mod, ptr, b)
	return ptr
}

func (m *module) newString(s string) uint32 {
	ptr := m.new(uint64(len(s) + 1))
	util.WriteString(m.mod, ptr, s)
	return ptr
}

func (m *module) newArena(size uint64) arena {
	return arena{
		m:    m,
		base: m.new(size),
		size: uint32(size),
	}
}

type arena struct {
	m    *module
	ptrs []uint32
	base uint32
	next uint32
	size uint32
}

func (a *arena) free() {
	if a.m == nil {
		return
	}
	a.reset()
	a.m.free(a.base)
	a.m = nil
}

func (a *arena) reset() {
	for _, ptr := range a.ptrs {
		a.m.free(ptr)
	}
	a.ptrs = nil
	a.next = 0
}

func (a *arena) new(size uint64) uint32 {
	if size <= uint64(a.size-a.next) {
		ptr := a.base + a.next
		a.next += uint32(size)
		return ptr
	}
	ptr := a.m.new(size)
	a.ptrs = append(a.ptrs, ptr)
	return ptr
}

func (a *arena) bytes(b []byte) uint32 {
	if b == nil {
		return 0
	}
	ptr := a.new(uint64(len(b)))
	util.WriteBytes(a.m.mod, ptr, b)
	return ptr
}

func (a *arena) string(s string) uint32 {
	ptr := a.new(uint64(len(s) + 1))
	util.WriteString(a.m.mod, ptr, s)
	return ptr
}

type sqliteAPI struct {
	free            api.Function
	malloc          api.Function
	errcode         api.Function
	errstr          api.Function
	errmsg          api.Function
	erroff          api.Function
	open            api.Function
	close           api.Function
	closeZombie     api.Function
	prepare         api.Function
	finalize        api.Function
	reset           api.Function
	step            api.Function
	exec            api.Function
	clearBindings   api.Function
	bindNull        api.Function
	bindCount       api.Function
	bindIndex       api.Function
	bindName        api.Function
	bindInteger     api.Function
	bindFloat       api.Function
	bindText        api.Function
	bindBlob        api.Function
	bindZeroBlob    api.Function
	columnCount     api.Function
	columnName      api.Function
	columnType      api.Function
	columnInteger   api.Function
	columnFloat     api.Function
	columnText      api.Function
	columnBlob      api.Function
	columnBytes     api.Function
	blobOpen        api.Function
	blobClose       api.Function
	blobReopen      api.Function
	blobBytes       api.Function
	blobRead        api.Function
	blobWrite       api.Function
	backupInit      api.Function
	backupStep      api.Function
	backupFinish    api.Function
	backupRemaining api.Function
	backupPageCount api.Function
	changes         api.Function
	lastRowid       api.Function
	autocommit      api.Function
	destructor      uint32
}
