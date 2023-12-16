// Package sqlite3 wraps the C SQLite API.
package sqlite3

import (
	"context"
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

var instance struct {
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	err      error
	once     sync.Once
}

func compileSQLite() {
	ctx := context.Background()
	instance.runtime = wazero.NewRuntime(ctx)

	env := instance.runtime.NewHostModuleBuilder("env")
	env = vfs.ExportHostFunctions(env)
	env = exportHostFunctions(env)
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
		instance.err = util.BinaryErr
		return
	}

	instance.compiled, instance.err = instance.runtime.CompileModule(ctx, bin)
}

type sqlite struct {
	ctx   context.Context
	mod   api.Module
	api   sqliteAPI
	stack [8]uint64
}

type sqliteKey struct{}

func instantiateSQLite() (sqlt *sqlite, err error) {
	instance.once.Do(compileSQLite)
	if instance.err != nil {
		return nil, instance.err
	}

	sqlt = new(sqlite)
	sqlt.ctx = util.NewContext(context.Background())
	sqlt.ctx = context.WithValue(sqlt.ctx, sqliteKey{}, sqlt)

	sqlt.mod, err = instance.runtime.InstantiateModule(sqlt.ctx,
		instance.compiled, wazero.NewModuleConfig())
	if err != nil {
		return nil, err
	}

	getFun := func(name string) api.Function {
		f := sqlt.mod.ExportedFunction(name)
		if f == nil {
			err = util.NoFuncErr + util.ErrorString(name)
			return nil
		}
		return f
	}

	getVal := func(name string) uint32 {
		g := sqlt.mod.ExportedGlobal(name)
		if g == nil {
			err = util.NoGlobalErr + util.ErrorString(name)
			return 0
		}
		return util.ReadUint32(sqlt.mod, uint32(g.Get()))
	}

	sqlt.api = sqliteAPI{
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
		anyCollation:    getFun("sqlite3_anycollseq_init"),
		createCollation: getFun("sqlite3_create_collation_go"),
		createFunction:  getFun("sqlite3_create_function_go"),
		createAggregate: getFun("sqlite3_create_aggregate_function_go"),
		createWindow:    getFun("sqlite3_create_window_function_go"),
		aggregateCtx:    getFun("sqlite3_aggregate_context"),
		userData:        getFun("sqlite3_user_data"),
		setAuxData:      getFun("sqlite3_set_auxdata_go"),
		getAuxData:      getFun("sqlite3_get_auxdata"),
		valueType:       getFun("sqlite3_value_type"),
		valueInteger:    getFun("sqlite3_value_int64"),
		valueFloat:      getFun("sqlite3_value_double"),
		valueText:       getFun("sqlite3_value_text"),
		valueBlob:       getFun("sqlite3_value_blob"),
		valueBytes:      getFun("sqlite3_value_bytes"),
		resultNull:      getFun("sqlite3_result_null"),
		resultInteger:   getFun("sqlite3_result_int64"),
		resultFloat:     getFun("sqlite3_result_double"),
		resultText:      getFun("sqlite3_result_text64"),
		resultBlob:      getFun("sqlite3_result_blob64"),
		resultZeroBlob:  getFun("sqlite3_result_zeroblob64"),
		resultError:     getFun("sqlite3_result_error"),
		resultErrorCode: getFun("sqlite3_result_error_code"),
		resultErrorMem:  getFun("sqlite3_result_error_nomem"),
		resultErrorBig:  getFun("sqlite3_result_error_toobig"),
	}
	if err != nil {
		return nil, err
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

	if r := sqlt.call(sqlt.api.errstr, rc); r != 0 {
		err.str = util.ReadString(sqlt.mod, uint32(r), _MAX_STRING)
	}

	if r := sqlt.call(sqlt.api.errmsg, uint64(handle)); r != 0 {
		err.msg = util.ReadString(sqlt.mod, uint32(r), _MAX_STRING)
	}

	if sql != nil {
		if r := sqlt.call(sqlt.api.erroff, uint64(handle)); r != math.MaxUint32 {
			err.sql = sql[0][r:]
		}
	}

	switch err.msg {
	case err.str, "not an error":
		err.msg = ""
	}
	return &err
}

func (sqlt *sqlite) call(fn api.Function, params ...uint64) uint64 {
	copy(sqlt.stack[:], params)
	err := fn.CallWithStack(sqlt.ctx, sqlt.stack[:])
	if err != nil {
		panic(err)
	}
	return sqlt.stack[0]
}

func (sqlt *sqlite) free(ptr uint32) {
	if ptr == 0 {
		return
	}
	sqlt.call(sqlt.api.free, uint64(ptr))
}

func (sqlt *sqlite) new(size uint64) uint32 {
	if size > _MAX_ALLOCATION_SIZE {
		panic(util.OOMErr)
	}
	ptr := uint32(sqlt.call(sqlt.api.malloc, size))
	if ptr == 0 && size != 0 {
		panic(util.OOMErr)
	}
	return ptr
}

func (sqlt *sqlite) newBytes(b []byte) uint32 {
	if b == nil {
		return 0
	}
	ptr := sqlt.new(uint64(len(b)))
	util.WriteBytes(sqlt.mod, ptr, b)
	return ptr
}

func (sqlt *sqlite) newString(s string) uint32 {
	ptr := sqlt.new(uint64(len(s) + 1))
	util.WriteString(sqlt.mod, ptr, s)
	return ptr
}

func (sqlt *sqlite) newArena(size uint64) arena {
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
	a.reset()
	a.sqlt.free(a.base)
	a.sqlt = nil
}

func (a *arena) reset() {
	for _, ptr := range a.ptrs {
		a.sqlt.free(ptr)
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
	ptr := a.sqlt.new(size)
	a.ptrs = append(a.ptrs, ptr)
	return ptr
}

func (a *arena) bytes(b []byte) uint32 {
	if b == nil {
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
	bindCount       api.Function
	bindIndex       api.Function
	bindName        api.Function
	bindNull        api.Function
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
	anyCollation    api.Function
	createCollation api.Function
	createFunction  api.Function
	createAggregate api.Function
	createWindow    api.Function
	aggregateCtx    api.Function
	userData        api.Function
	setAuxData      api.Function
	getAuxData      api.Function
	valueType       api.Function
	valueInteger    api.Function
	valueFloat      api.Function
	valueText       api.Function
	valueBlob       api.Function
	valueBytes      api.Function
	resultNull      api.Function
	resultInteger   api.Function
	resultFloat     api.Function
	resultText      api.Function
	resultBlob      api.Function
	resultZeroBlob  api.Function
	resultError     api.Function
	resultErrorCode api.Function
	resultErrorMem  api.Function
	resultErrorBig  api.Function
	destructor      uint32
}
