package sqlite3

import (
	"errors"
	"reflect"
)

// CreateModule registers a new virtual table module name.
// If create is nil, the virtual table is eponymous.
//
// https://sqlite.org/c3ref/create_module.html
//
// Deprecated: use the generic method.
func CreateModule[T VTab](db *Conn, name string, create, connect VTabConstructor[T]) error {
	var flags int

	const (
		VTAB_CREATOR     = 0x001
		VTAB_DESTROYER   = 0x002
		VTAB_UPDATER     = 0x004
		VTAB_RENAMER     = 0x008
		VTAB_OVERLOADER  = 0x010
		VTAB_CHECKER     = 0x020
		VTAB_TXN         = 0x040
		VTAB_SAVEPOINTER = 0x080
		VTAB_SHADOWTABS  = 0x100
	)

	if create != nil {
		flags |= VTAB_CREATOR
	}

	vtab := reflect.TypeFor[T]()
	if implements[VTabDestroyer](vtab) {
		flags |= VTAB_DESTROYER
	}
	if implements[VTabUpdater](vtab) {
		flags |= VTAB_UPDATER
	}
	if implements[VTabRenamer](vtab) {
		flags |= VTAB_RENAMER
	}
	if implements[VTabOverloader](vtab) {
		flags |= VTAB_OVERLOADER
	}
	if implements[VTabChecker](vtab) {
		flags |= VTAB_CHECKER
	}
	if implements[VTabTxn](vtab) {
		flags |= VTAB_TXN
	}
	if implements[VTabSavepointer](vtab) {
		flags |= VTAB_SAVEPOINTER
	}
	if implements[VTabShadowTabler](vtab) {
		flags |= VTAB_SHADOWTABS
	}

	var modulePtr ptr_t
	defer db.arena.Mark()()
	namePtr := db.arena.String(name)
	if connect != nil {
		modulePtr = db.wrp.AddHandle(vtabModule{
			vtabConstructor(create),
			vtabConstructor(connect)})
	}
	rc := res_t(db.wrp.Xsqlite3_create_module_go(int32(db.handle),
		int32(namePtr), int32(flags), int32(modulePtr)))
	return db.error(rc)
}

func implements[I any](typ reflect.Type) bool {
	return typ.Implements(reflect.TypeFor[I]())
}

// DeclareVTab declares the schema of a virtual table.
//
// https://sqlite.org/c3ref/declare_vtab.html
func (c *Conn) DeclareVTab(sql string) error {
	if c.interrupt.Err() != nil {
		return INTERRUPT
	}
	defer c.arena.Mark()()
	textPtr := c.arena.String(sql)
	rc := res_t(c.wrp.Xsqlite3_declare_vtab(int32(c.handle), int32(textPtr)))
	return c.error(rc)
}

// VTabConflictMode is a virtual table conflict resolution mode.
//
// https://sqlite.org/c3ref/c_fail.html
type VTabConflictMode uint8

const (
	VTAB_ROLLBACK VTabConflictMode = 1
	VTAB_IGNORE   VTabConflictMode = 2
	VTAB_FAIL     VTabConflictMode = 3
	VTAB_ABORT    VTabConflictMode = 4
	VTAB_REPLACE  VTabConflictMode = 5
)

// VTabOnConflict determines the virtual table conflict policy.
//
// https://sqlite.org/c3ref/vtab_on_conflict.html
func (c *Conn) VTabOnConflict() VTabConflictMode {
	return VTabConflictMode(c.wrp.Xsqlite3_vtab_on_conflict(int32(c.handle)))
}

// VTabConfigOption is a virtual table configuration option.
//
// https://sqlite.org/c3ref/c_vtab_constraint_support.html
type VTabConfigOption uint8

const (
	VTAB_CONSTRAINT_SUPPORT VTabConfigOption = 1
	VTAB_INNOCUOUS          VTabConfigOption = 2
	VTAB_DIRECTONLY         VTabConfigOption = 3
	VTAB_USES_ALL_SCHEMAS   VTabConfigOption = 4
)

// VTabConfig configures various facets of the virtual table interface.
//
// https://sqlite.org/c3ref/vtab_config.html
func (c *Conn) VTabConfig(op VTabConfigOption, args ...any) error {
	var i int32
	if op == VTAB_CONSTRAINT_SUPPORT && len(args) > 0 {
		if b, ok := args[0].(bool); ok && b {
			i = 1
		}
	}
	rc := res_t(c.wrp.Xsqlite3_vtab_config_go(int32(c.handle), int32(op), i))
	return c.error(rc)
}

// VTabConstructor is a virtual table constructor function.
type VTabConstructor[T VTab] func(db *Conn, module, schema, table string, arg ...string) (T, error)

// A VTab describes a particular instance of the virtual table.
// A VTab may optionally implement [io.Closer] to free resources.
//
// https://sqlite.org/c3ref/vtab.html
type VTab interface {
	// https://sqlite.org/vtab.html#xbestindex
	BestIndex(*IndexInfo) error
	// https://sqlite.org/vtab.html#xopen
	Open() (VTabCursor, error)
}

// A VTabDestroyer allows a virtual table to drop persistent state.
type VTabDestroyer interface {
	VTab
	// https://sqlite.org/vtab.html#sqlite3_module.xDestroy
	Destroy() error
}

// A VTabUpdater allows a virtual table to be updated.
// Implementations must not retain arg.
type VTabUpdater interface {
	VTab
	// https://sqlite.org/vtab.html#xupdate
	Update(arg ...Value) (rowid int64, err error)
}

// A VTabRenamer allows a virtual table to be renamed.
type VTabRenamer interface {
	VTab
	// https://sqlite.org/vtab.html#xrename
	Rename(new string) error
}

// A VTabOverloader allows a virtual table to overload SQL functions.
type VTabOverloader interface {
	VTab
	// https://sqlite.org/vtab.html#xfindfunction
	FindFunction(arg int, name string) (ScalarFunction, IndexConstraintOp)
}

// A VTabShadowTabler allows a virtual table to protect the content
// of shadow tables from being corrupted by hostile SQL.
//
// Implementing this interface signals that a virtual table named
// "mumble" reserves all table names starting with "mumble_".
type VTabShadowTabler interface {
	VTab
	// https://sqlite.org/vtab.html#the_xshadowname_method
	ShadowTables()
}

// A VTabChecker allows a virtual table to report errors
// to the PRAGMA integrity_check and PRAGMA quick_check commands.
//
// Integrity should return an error if it finds problems in the content of the virtual table,
// but should avoid returning a (wrapped) [Error], [ErrorCode] or [ExtendedErrorCode],
// as those indicate the Integrity method itself encountered problems
// while trying to evaluate the virtual table content.
type VTabChecker interface {
	VTab
	// https://sqlite.org/vtab.html#xintegrity
	Integrity(schema, table string, flags int) error
}

// A VTabTxn allows a virtual table to implement
// transactions with two-phase commit.
//
// Anything that is required as part of a commit that may fail
// should be performed in the Sync() callback.
// Current versions of SQLite ignore any errors
// returned by Commit() and Rollback().
type VTabTxn interface {
	VTab
	// https://sqlite.org/vtab.html#xBegin
	Begin() error
	// https://sqlite.org/vtab.html#xsync
	Sync() error
	// https://sqlite.org/vtab.html#xcommit
	Commit() error
	// https://sqlite.org/vtab.html#xrollback
	Rollback() error
}

// A VTabSavepointer allows a virtual table to implement
// nested transactions.
//
// https://sqlite.org/vtab.html#xsavepoint
type VTabSavepointer interface {
	VTabTxn
	Savepoint(id int) error
	Release(id int) error
	RollbackTo(id int) error
}

// A VTabCursor describes cursors that point
// into the virtual table and are used
// to loop through the virtual table.
// A VTabCursor may optionally implement
// [io.Closer] to free resources.
// Implementations of Filter must not retain arg.
//
// https://sqlite.org/c3ref/vtab_cursor.html
type VTabCursor interface {
	// https://sqlite.org/vtab.html#xfilter
	Filter(idxNum int, idxStr string, arg ...Value) error
	// https://sqlite.org/vtab.html#xnext
	Next() error
	// https://sqlite.org/vtab.html#xeof
	EOF() bool
	// https://sqlite.org/vtab.html#xcolumn
	Column(ctx Context, n int) error
	// https://sqlite.org/vtab.html#xrowid
	RowID() (int64, error)
}

// An IndexInfo describes virtual table indexing information.
//
// https://sqlite.org/c3ref/index_info.html
type IndexInfo struct {
	// Inputs
	Constraint  []IndexConstraint
	OrderBy     []IndexOrderBy
	ColumnsUsed uint64
	// Outputs
	ConstraintUsage []IndexConstraintUsage
	IdxNum          int
	IdxStr          string
	IdxFlags        IndexScanFlag
	OrderByConsumed bool
	EstimatedCost   float64
	EstimatedRows   int64
	// Internal
	c      *Conn
	handle ptr_t
}

// An IndexConstraint describes virtual table indexing constraint information.
//
// https://sqlite.org/c3ref/index_info.html
type IndexConstraint struct {
	Column int
	Op     IndexConstraintOp
	Usable bool
}

// An IndexOrderBy describes virtual table indexing order by information.
//
// https://sqlite.org/c3ref/index_info.html
type IndexOrderBy struct {
	Column int
	Desc   bool
}

// An IndexConstraintUsage describes how virtual table indexing constraints will be used.
//
// https://sqlite.org/c3ref/index_info.html
type IndexConstraintUsage struct {
	ArgvIndex int
	Omit      bool
}

// RHSValue returns the value of the right-hand operand of a constraint
// if the right-hand operand is known.
//
// https://sqlite.org/c3ref/vtab_rhs_value.html
func (idx *IndexInfo) RHSValue(column int) (Value, error) {
	return idx.c.columnValue(idx.c.wrp.Xsqlite3_vtab_rhs_value, idx.handle, column)
}

// Collation returns the name of the collation for a virtual table constraint.
//
// https://sqlite.org/c3ref/vtab_collation.html
func (idx *IndexInfo) Collation(column int) string {
	ptr := ptr_t(idx.c.wrp.Xsqlite3_vtab_collation(int32(idx.handle),
		int32(column)))
	return idx.c.wrp.ReadString(ptr, _MAX_NAME)
}

// Distinct determines if a virtual table query is DISTINCT.
//
// https://sqlite.org/c3ref/vtab_distinct.html
func (idx *IndexInfo) Distinct() int {
	return int(idx.c.wrp.Xsqlite3_vtab_distinct(int32(idx.handle)))
}

// In identifies and handles IN constraints.
//
// https://sqlite.org/c3ref/vtab_in.html
func (idx *IndexInfo) In(column, handle int) bool {
	b := idx.c.wrp.Xsqlite3_vtab_in(int32(idx.handle),
		int32(column), int32(handle))
	return b != 0
}

func (idx *IndexInfo) load() {
	// https://sqlite.org/c3ref/index_info.html
	mem := idx.c.wrp.Memory
	ptr := idx.handle

	nConstraint := int32(mem.Read32(ptr + 0))
	idx.Constraint = make([]IndexConstraint, nConstraint)
	idx.ConstraintUsage = make([]IndexConstraintUsage, nConstraint)
	idx.OrderBy = make([]IndexOrderBy, int32(mem.Read32(ptr+8)))

	constraintPtr := ptr_t(mem.Read32(ptr + 4))
	constraint := idx.Constraint
	for i := range idx.Constraint {
		constraint[i] = IndexConstraint{
			Column: int(int32(mem.Read32(constraintPtr + 0))),
			Op:     IndexConstraintOp(mem.Read(constraintPtr + 4)),
			Usable: mem.Read(constraintPtr+5) != 0,
		}
		constraintPtr += 12
	}

	orderByPtr := ptr_t(mem.Read32(ptr + 12))
	orderBy := idx.OrderBy
	for i := range orderBy {
		orderBy[i] = IndexOrderBy{
			Column: int(int32(mem.Read32(orderByPtr + 0))),
			Desc:   mem.Read(orderByPtr+4) != 0,
		}
		orderByPtr += 8
	}

	idx.EstimatedCost = mem.ReadFloat64(ptr + 40)
	idx.EstimatedRows = int64(mem.Read64(ptr + 48))
	idx.ColumnsUsed = mem.Read64(ptr + 64)
}

func (idx *IndexInfo) save() {
	// https://sqlite.org/c3ref/index_info.html
	mem := idx.c.wrp.Memory
	ptr := idx.handle

	usagePtr := ptr_t(mem.Read32(ptr + 16))
	for _, usage := range idx.ConstraintUsage {
		mem.Write32(usagePtr+0, uint32(usage.ArgvIndex))
		if usage.Omit {
			mem.Write(usagePtr+4, 1)
		}
		usagePtr += 8
	}

	mem.Write32(ptr+20, uint32(idx.IdxNum))
	if idx.IdxStr != "" {
		mem.Write32(ptr+24, uint32(idx.c.wrp.NewString(idx.IdxStr)))
		mem.WriteBool(ptr+28, true) // needToFreeIdxStr
	}
	if idx.OrderByConsumed {
		mem.WriteBool(ptr+32, true)
	}
	mem.WriteFloat64(ptr+40, idx.EstimatedCost)
	mem.Write64(ptr+48, uint64(idx.EstimatedRows))
	mem.Write32(ptr+56, uint32(idx.IdxFlags))
}

// IndexConstraintOp is a virtual table constraint operator code.
//
// https://sqlite.org/c3ref/c_index_constraint_eq.html
type IndexConstraintOp uint8

const (
	INDEX_CONSTRAINT_EQ        IndexConstraintOp = 2
	INDEX_CONSTRAINT_GT        IndexConstraintOp = 4
	INDEX_CONSTRAINT_LE        IndexConstraintOp = 8
	INDEX_CONSTRAINT_LT        IndexConstraintOp = 16
	INDEX_CONSTRAINT_GE        IndexConstraintOp = 32
	INDEX_CONSTRAINT_MATCH     IndexConstraintOp = 64
	INDEX_CONSTRAINT_LIKE      IndexConstraintOp = 65
	INDEX_CONSTRAINT_GLOB      IndexConstraintOp = 66
	INDEX_CONSTRAINT_REGEXP    IndexConstraintOp = 67
	INDEX_CONSTRAINT_NE        IndexConstraintOp = 68
	INDEX_CONSTRAINT_ISNOT     IndexConstraintOp = 69
	INDEX_CONSTRAINT_ISNOTNULL IndexConstraintOp = 70
	INDEX_CONSTRAINT_ISNULL    IndexConstraintOp = 71
	INDEX_CONSTRAINT_IS        IndexConstraintOp = 72
	INDEX_CONSTRAINT_LIMIT     IndexConstraintOp = 73
	INDEX_CONSTRAINT_OFFSET    IndexConstraintOp = 74
	INDEX_CONSTRAINT_FUNCTION  IndexConstraintOp = 150
)

// IndexScanFlag is a virtual table scan flag.
//
// https://sqlite.org/c3ref/c_index_scan_unique.html
type IndexScanFlag uint32

const (
	INDEX_SCAN_UNIQUE IndexScanFlag = 0x00000001
	INDEX_SCAN_HEX    IndexScanFlag = 0x00000002
)

type vtabModule = [2]VTabConstructor[VTab]

func vtabConstructor[T VTab](c VTabConstructor[T]) VTabConstructor[VTab] {
	return func(db *Conn, module, schema, table string, arg ...string) (VTab, error) {
		return c(db, module, schema, table, arg...)
	}
}

func (e env) vtabConstruct(fn VTabConstructor[VTab], nArg, pArg, ppVTab, pzErr int32) int32 {
	arg := make([]string, nArg)
	for i := range nArg {
		ptr := ptr_t(e.Memory.Read32(ptr_t(pArg + i*ptrlen)))
		arg[i] = e.ReadString(ptr, _MAX_SQL_LENGTH)
	}

	tab, err := fn(e.DB.(*Conn), arg[0], arg[1], arg[2], arg[3:]...)
	if err == nil {
		e.vtabPutHandle(ppVTab, tab)
	}
	return e.vtabError(pzErr, _PTR_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_create(pMod, nArg, pArg, ppVTab, pzErr int32) int32 {
	module := e.vtabGetHandle(pMod).(vtabModule)
	return e.vtabConstruct(module[0], nArg, pArg, ppVTab, pzErr)
}

func (e env) Xgo_vtab_connect(pMod, nArg, pArg, ppVTab, pzErr int32) int32 {
	module := e.vtabGetHandle(pMod).(vtabModule)
	return e.vtabConstruct(module[1], nArg, pArg, ppVTab, pzErr)
}

func (e env) Xgo_vtab_disconnect(pVTab int32) int32 {
	err := e.vtabDelHandle(pVTab)
	return e.vtabError(0, _PTR_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_destroy(pVTab int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTabDestroyer)
	err := errors.Join(vtab.Destroy(), e.vtabDelHandle(pVTab))
	return e.vtabError(0, _PTR_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_best_index(pVTab, pIdxInfo int32) int32 {
	var info IndexInfo
	info.handle = ptr_t(pIdxInfo)
	info.c = e.DB.(*Conn)
	info.load()

	vtab := e.vtabGetHandle(pVTab).(VTab)
	err := vtab.BestIndex(&info)

	info.save()
	return e.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_update(pVTab, nArg, pArg, pRowID int32) int32 {
	db := e.DB.(*Conn)
	args := callbackArgs(db, nArg, ptr_t(pArg))
	defer returnArgs(args)

	vtab := e.vtabGetHandle(pVTab).(VTabUpdater)
	rowID, err := vtab.Update(*args...)
	if err == nil {
		e.Memory.Write64(ptr_t(pRowID), uint64(rowID))
	}

	return e.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_rename(pVTab, zNew int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTabRenamer)
	err := vtab.Rename(e.ReadString(ptr_t(zNew), _MAX_NAME))
	return e.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_find_function(pVTab, nArg, zName, pxFunc int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTabOverloader)
	f, op := vtab.FindFunction(int(nArg), e.ReadString(ptr_t(zName), _MAX_NAME))
	if op != 0 {
		var wrapper ptr_t
		wrapper = e.AddHandle(func(c Context, arg ...Value) {
			defer e.DelHandle(wrapper)
			f(c, arg...)
		})
		e.Memory.Write32(ptr_t(pxFunc), uint32(wrapper))
	}
	return int32(op)
}

func (e env) Xgo_vtab_integrity(pVTab, zSchema, zTabName, mFlags, pzErr int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTabChecker)
	schema := e.ReadString(ptr_t(zSchema), _MAX_NAME)
	table := e.ReadString(ptr_t(zTabName), _MAX_NAME)
	err := vtab.Integrity(schema, table, int(uint32(mFlags)))
	// xIntegrity should return OK - even if it finds problems in the content of the virtual table.
	// https://sqlite.org/vtab.html#xintegrity
	return e.vtabError(pzErr, _PTR_ERROR, err, _OK)
}

func (e env) Xgo_vtab_begin(pVTab int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTabTxn)
	err := vtab.Begin()
	return e.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_sync(pVTab int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTabTxn)
	err := vtab.Sync()
	return e.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_commit(pVTab int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTabTxn)
	err := vtab.Commit()
	return e.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_rollback(pVTab int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTabTxn)
	err := vtab.Rollback()
	return e.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_savepoint(pVTab, id int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTabSavepointer)
	err := vtab.Savepoint(int(id))
	return e.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_release(pVTab, id int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTabSavepointer)
	err := vtab.Release(int(id))
	return e.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (e env) Xgo_vtab_rollback_to(pVTab, id int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTabSavepointer)
	err := vtab.RollbackTo(int(id))
	return e.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (e env) Xgo_cur_open(pVTab, ppCur int32) int32 {
	vtab := e.vtabGetHandle(pVTab).(VTab)

	cursor, err := vtab.Open()
	if err == nil {
		e.vtabPutHandle(ppCur, cursor)
	}

	return e.vtabError(pVTab, _VTAB_ERROR, err, ERROR)
}

func (e env) Xgo_cur_close(pCur int32) int32 {
	err := e.vtabDelHandle(pCur)
	return e.vtabError(0, _PTR_ERROR, err, ERROR)
}

func (e env) Xgo_cur_filter(pCur, idxNum, idxStr, nArg, pArg int32) int32 {
	db := e.DB.(*Conn)
	args := callbackArgs(db, nArg, ptr_t(pArg))
	defer returnArgs(args)

	var idxName string
	if idxStr != 0 {
		idxName = e.ReadString(ptr_t(idxStr), _MAX_LENGTH)
	}

	cursor := e.vtabGetHandle(pCur).(VTabCursor)
	err := cursor.Filter(int(idxNum), idxName, *args...)
	return e.vtabError(pCur, _CURSOR_ERROR, err, ERROR)
}

func (e env) Xgo_cur_eof(pCur int32) int32 {
	cursor := e.vtabGetHandle(pCur).(VTabCursor)
	if cursor.EOF() {
		return 1
	}
	return 0
}

func (e env) Xgo_cur_next(pCur int32) int32 {
	cursor := e.vtabGetHandle(pCur).(VTabCursor)
	err := cursor.Next()
	return e.vtabError(pCur, _CURSOR_ERROR, err, ERROR)
}

func (e env) Xgo_cur_column(pCur, pCtx, n int32) int32 {
	cursor := e.vtabGetHandle(pCur).(VTabCursor)
	db := e.DB.(*Conn)
	err := cursor.Column(Context{db, ptr_t(pCtx)}, int(n))
	return e.vtabError(pCur, _CURSOR_ERROR, err, ERROR)
}

func (e env) Xgo_cur_rowid(pCur, pRowID int32) int32 {
	cursor := e.vtabGetHandle(pCur).(VTabCursor)

	rowID, err := cursor.RowID()
	if err == nil {
		e.Memory.Write64(ptr_t(pRowID), uint64(rowID))
	}

	return e.vtabError(pCur, _CURSOR_ERROR, err, ERROR)
}

const (
	_PTR_ERROR = iota
	_VTAB_ERROR
	_CURSOR_ERROR
)

func (e env) vtabError(ptr int32, kind uint32, err error, def ErrorCode) int32 {
	const zErrMsgOffset = 8
	msg, code := errorCode(err, def)
	if ptr != 0 && msg != "" {
		switch kind {
		case _VTAB_ERROR:
			ptr = ptr + zErrMsgOffset // zErrMsg
		case _CURSOR_ERROR:
			ptr = int32(e.Memory.Read32(ptr_t(ptr))) + zErrMsgOffset // pVTab->zErrMsg
		}
		db := e.DB.(*Conn)
		if ptr := ptr_t(e.Memory.Read32(ptr_t(ptr))); ptr != 0 {
			db.wrp.Free(ptr)
		}
		e.Memory.Write32(ptr_t(ptr), uint32(db.wrp.NewString(msg)))
	}
	return int32(code)
}

func (e env) vtabGetHandle(ptr int32) any {
	const handleOffset = 4
	handle := ptr_t(e.Memory.Read32(ptr_t(ptr) - handleOffset))
	return e.GetHandle(handle)
}

func (e env) vtabDelHandle(ptr int32) error {
	const handleOffset = 4
	handle := ptr_t(e.Memory.Read32(ptr_t(ptr) - handleOffset))
	return e.DelHandle(handle)
}

func (e env) vtabPutHandle(pptr int32, val any) {
	const handleOffset = 4
	handle := e.AddHandle(val)
	ptr := ptr_t(e.Memory.Read32(ptr_t(pptr)))
	e.Memory.Write32(ptr-handleOffset, uint32(handle))
}
