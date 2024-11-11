package sqlite3

import (
	"context"
	"sync"

	"github.com/tetratelabs/wazero/api"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// CollationNeeded registers a callback to be invoked
// whenever an unknown collation sequence is required.
//
// https://sqlite.org/c3ref/collation_needed.html
func (c *Conn) CollationNeeded(cb func(db *Conn, name string)) error {
	var enable uint64
	if cb != nil {
		enable = 1
	}
	r := c.call("sqlite3_collation_needed_go", uint64(c.handle), enable)
	if err := c.error(r); err != nil {
		return err
	}
	c.collation = cb
	return nil
}

// AnyCollationNeeded uses [Conn.CollationNeeded] to register
// a fake collating function for any unknown collating sequence.
// The fake collating function works like BINARY.
//
// This can be used to load schemas that contain
// one or more unknown collating sequences.
func (c Conn) AnyCollationNeeded() error {
	r := c.call("sqlite3_anycollseq_init", uint64(c.handle), 0, 0)
	if err := c.error(r); err != nil {
		return err
	}
	c.collation = nil
	return nil
}

// CreateCollation defines a new collating sequence.
//
// https://sqlite.org/c3ref/create_collation.html
func (c *Conn) CreateCollation(name string, fn func(a, b []byte) int) error {
	var funcPtr uint32
	defer c.arena.mark()()
	namePtr := c.arena.string(name)
	if fn != nil {
		funcPtr = util.AddHandle(c.ctx, fn)
	}
	r := c.call("sqlite3_create_collation_go",
		uint64(c.handle), uint64(namePtr), uint64(funcPtr))
	return c.error(r)
}

// CreateFunction defines a new scalar SQL function.
//
// https://sqlite.org/c3ref/create_function.html
func (c *Conn) CreateFunction(name string, nArg int, flag FunctionFlag, fn ScalarFunction) error {
	var funcPtr uint32
	defer c.arena.mark()()
	namePtr := c.arena.string(name)
	if fn != nil {
		funcPtr = util.AddHandle(c.ctx, fn)
	}
	r := c.call("sqlite3_create_function_go",
		uint64(c.handle), uint64(namePtr), uint64(nArg),
		uint64(flag), uint64(funcPtr))
	return c.error(r)
}

// ScalarFunction is the type of a scalar SQL function.
// Implementations must not retain arg.
type ScalarFunction func(ctx Context, arg ...Value)

// CreateWindowFunction defines a new aggregate or aggregate window SQL function.
// If fn returns a [WindowFunction], then an aggregate window function is created.
// If fn returns an [io.Closer], it will be called to free resources.
//
// https://sqlite.org/c3ref/create_function.html
func (c *Conn) CreateWindowFunction(name string, nArg int, flag FunctionFlag, fn func() AggregateFunction) error {
	var funcPtr uint32
	defer c.arena.mark()()
	namePtr := c.arena.string(name)
	if fn != nil {
		funcPtr = util.AddHandle(c.ctx, fn)
	}
	call := "sqlite3_create_aggregate_function_go"
	if _, ok := fn().(WindowFunction); ok {
		call = "sqlite3_create_window_function_go"
	}
	r := c.call(call,
		uint64(c.handle), uint64(namePtr), uint64(nArg),
		uint64(flag), uint64(funcPtr))
	return c.error(r)
}

// AggregateFunction is the interface an aggregate function should implement.
//
// https://sqlite.org/appfunc.html
type AggregateFunction interface {
	// Step is invoked to add a row to the current window.
	// The function arguments, if any, corresponding to the row being added, are passed to Step.
	// Implementations must not retain arg.
	Step(ctx Context, arg ...Value)

	// Value is invoked to return the current (or final) value of the aggregate.
	Value(ctx Context)
}

// WindowFunction is the interface an aggregate window function should implement.
//
// https://sqlite.org/windowfunctions.html
type WindowFunction interface {
	AggregateFunction

	// Inverse is invoked to remove the oldest presently aggregated result of Step from the current window.
	// The function arguments, if any, are those passed to Step for the row being removed.
	// Implementations must not retain arg.
	Inverse(ctx Context, arg ...Value)
}

// OverloadFunction overloads a function for a virtual table.
//
// https://sqlite.org/c3ref/overload_function.html
func (c *Conn) OverloadFunction(name string, nArg int) error {
	defer c.arena.mark()()
	namePtr := c.arena.string(name)
	r := c.call("sqlite3_overload_function",
		uint64(c.handle), uint64(namePtr), uint64(nArg))
	return c.error(r)
}

func destroyCallback(ctx context.Context, mod api.Module, pApp uint32) {
	util.DelHandle(ctx, pApp)
}

func collationCallback(ctx context.Context, mod api.Module, pArg, pDB, eTextRep, zName uint32) {
	if c, ok := ctx.Value(connKey{}).(*Conn); ok && c.handle == pDB && c.collation != nil {
		name := util.ReadString(mod, zName, _MAX_NAME)
		c.collation(c, name)
	}
}

func compareCallback(ctx context.Context, mod api.Module, pApp, nKey1, pKey1, nKey2, pKey2 uint32) uint32 {
	fn := util.GetHandle(ctx, pApp).(func(a, b []byte) int)
	return uint32(fn(util.View(mod, pKey1, uint64(nKey1)), util.View(mod, pKey2, uint64(nKey2))))
}

func funcCallback(ctx context.Context, mod api.Module, pCtx, pApp, nArg, pArg uint32) {
	args := getFuncArgs()
	defer putFuncArgs(args)
	db := ctx.Value(connKey{}).(*Conn)
	fn := util.GetHandle(db.ctx, pApp).(ScalarFunction)
	callbackArgs(db, args[:nArg], pArg)
	fn(Context{db, pCtx}, args[:nArg]...)
}

func stepCallback(ctx context.Context, mod api.Module, pCtx, pAgg, pApp, nArg, pArg uint32) {
	args := getFuncArgs()
	defer putFuncArgs(args)
	db := ctx.Value(connKey{}).(*Conn)
	callbackArgs(db, args[:nArg], pArg)
	fn, _ := callbackAggregate(db, pAgg, pApp)
	fn.Step(Context{db, pCtx}, args[:nArg]...)
}

func finalCallback(ctx context.Context, mod api.Module, pCtx, pAgg, pApp uint32) {
	db := ctx.Value(connKey{}).(*Conn)
	fn, handle := callbackAggregate(db, pAgg, pApp)
	fn.Value(Context{db, pCtx})
	util.DelHandle(ctx, handle)
}

func valueCallback(ctx context.Context, mod api.Module, pCtx, pAgg uint32) {
	db := ctx.Value(connKey{}).(*Conn)
	fn := util.GetHandle(db.ctx, pAgg).(AggregateFunction)
	fn.Value(Context{db, pCtx})
}

func inverseCallback(ctx context.Context, mod api.Module, pCtx, pAgg, nArg, pArg uint32) {
	args := getFuncArgs()
	defer putFuncArgs(args)
	db := ctx.Value(connKey{}).(*Conn)
	callbackArgs(db, args[:nArg], pArg)
	fn := util.GetHandle(db.ctx, pAgg).(WindowFunction)
	fn.Inverse(Context{db, pCtx}, args[:nArg]...)
}

func callbackAggregate(db *Conn, pAgg, pApp uint32) (AggregateFunction, uint32) {
	if pApp == 0 {
		handle := util.ReadUint32(db.mod, pAgg)
		return util.GetHandle(db.ctx, handle).(AggregateFunction), handle
	}

	// We need to create the aggregate.
	fn := util.GetHandle(db.ctx, pApp).(func() AggregateFunction)()
	if pAgg != 0 {
		handle := util.AddHandle(db.ctx, fn)
		util.WriteUint32(db.mod, pAgg, handle)
		return fn, handle
	}
	return fn, 0
}

func callbackArgs(db *Conn, arg []Value, pArg uint32) {
	for i := range arg {
		arg[i] = Value{
			c:      db,
			handle: util.ReadUint32(db.mod, pArg+ptrlen*uint32(i)),
		}
	}
}

var funcArgsPool sync.Pool

func putFuncArgs(p *[_MAX_FUNCTION_ARG]Value) {
	funcArgsPool.Put(p)
}

func getFuncArgs() *[_MAX_FUNCTION_ARG]Value {
	if p := funcArgsPool.Get(); p == nil {
		return new([_MAX_FUNCTION_ARG]Value)
	} else {
		return p.(*[_MAX_FUNCTION_ARG]Value)
	}
}
