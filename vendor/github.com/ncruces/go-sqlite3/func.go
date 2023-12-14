package sqlite3

import (
	"context"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero/api"
)

// AnyCollationNeeded registers a fake collating function
// for any unknown collating sequence.
// The fake collating function works like BINARY.
//
// This can be used to load schemas that contain
// one or more unknown collating sequences.
func (c *Conn) AnyCollationNeeded() {
	c.call("sqlite3_anycollseq_init", uint64(c.handle), 0, 0)
}

// CreateCollation defines a new collating sequence.
//
// https://sqlite.org/c3ref/create_collation.html
func (c *Conn) CreateCollation(name string, fn func(a, b []byte) int) error {
	defer c.arena.mark()()
	namePtr := c.arena.string(name)
	funcPtr := util.AddHandle(c.ctx, fn)
	r := c.call("sqlite3_create_collation_go",
		uint64(c.handle), uint64(namePtr), uint64(funcPtr))
	return c.error(r)
}

// CreateFunction defines a new scalar SQL function.
//
// https://sqlite.org/c3ref/create_function.html
func (c *Conn) CreateFunction(name string, nArg int, flag FunctionFlag, fn ScalarFunction) error {
	defer c.arena.mark()()
	namePtr := c.arena.string(name)
	funcPtr := util.AddHandle(c.ctx, fn)
	r := c.call("sqlite3_create_function_go",
		uint64(c.handle), uint64(namePtr), uint64(nArg),
		uint64(flag), uint64(funcPtr))
	return c.error(r)
}

// ScalarFunction is the type of a scalar SQL function.
type ScalarFunction func(ctx Context, arg ...Value)

// CreateWindowFunction defines a new aggregate or aggregate window SQL function.
// If fn returns a [WindowFunction], then an aggregate window function is created.
// If fn returns an [io.Closer], it will be called to free resources.
//
// https://sqlite.org/c3ref/create_function.html
func (c *Conn) CreateWindowFunction(name string, nArg int, flag FunctionFlag, fn func() AggregateFunction) error {
	defer c.arena.mark()()
	call := "sqlite3_create_aggregate_function_go"
	namePtr := c.arena.string(name)
	funcPtr := util.AddHandle(c.ctx, fn)
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
	// The function arguments, if any, corresponding to the row being added are passed to Step.
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
	Inverse(ctx Context, arg ...Value)
}

func destroyCallback(ctx context.Context, mod api.Module, pApp uint32) {
	util.DelHandle(ctx, pApp)
}

func compareCallback(ctx context.Context, mod api.Module, pApp, nKey1, pKey1, nKey2, pKey2 uint32) uint32 {
	fn := util.GetHandle(ctx, pApp).(func(a, b []byte) int)
	return uint32(fn(util.View(mod, pKey1, uint64(nKey1)), util.View(mod, pKey2, uint64(nKey2))))
}

func funcCallback(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {
	db := ctx.Value(connKey{}).(*Conn)
	fn := userDataHandle(db, pCtx).(ScalarFunction)
	fn(Context{db, pCtx}, callbackArgs(db, nArg, pArg)...)
}

func stepCallback(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {
	db := ctx.Value(connKey{}).(*Conn)
	fn := aggregateCtxHandle(db, pCtx, nil)
	fn.Step(Context{db, pCtx}, callbackArgs(db, nArg, pArg)...)
}

func finalCallback(ctx context.Context, mod api.Module, pCtx uint32) {
	var handle uint32
	db := ctx.Value(connKey{}).(*Conn)
	fn := aggregateCtxHandle(db, pCtx, &handle)
	fn.Value(Context{db, pCtx})
	if err := util.DelHandle(ctx, handle); err != nil {
		Context{db, pCtx}.ResultError(err)
	}
}

func valueCallback(ctx context.Context, mod api.Module, pCtx uint32) {
	db := ctx.Value(connKey{}).(*Conn)
	fn := aggregateCtxHandle(db, pCtx, nil)
	fn.Value(Context{db, pCtx})
}

func inverseCallback(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {
	db := ctx.Value(connKey{}).(*Conn)
	fn := aggregateCtxHandle(db, pCtx, nil).(WindowFunction)
	fn.Inverse(Context{db, pCtx}, callbackArgs(db, nArg, pArg)...)
}

func userDataHandle(db *Conn, pCtx uint32) any {
	pApp := uint32(db.call("sqlite3_user_data", uint64(pCtx)))
	return util.GetHandle(db.ctx, pApp)
}

func aggregateCtxHandle(db *Conn, pCtx uint32, close *uint32) AggregateFunction {
	// On close, we're getting rid of the aggregate.
	// Don't allocate space to store it.
	var size uint64
	if close == nil {
		size = ptrlen
	}
	ptr := uint32(db.call("sqlite3_aggregate_context", uint64(pCtx), size))

	// If we already have an aggregate, return it.
	if ptr != 0 {
		if handle := util.ReadUint32(db.mod, ptr); handle != 0 {
			fn := util.GetHandle(db.ctx, handle).(AggregateFunction)
			if close != nil {
				*close = handle
			}
			return fn
		}
	}

	// Create a new aggregate, and store it if needed.
	fn := userDataHandle(db, pCtx).(func() AggregateFunction)()
	if ptr != 0 {
		util.WriteUint32(db.mod, ptr, util.AddHandle(db.ctx, fn))
	}
	return fn
}

func callbackArgs(db *Conn, nArg, pArg uint32) []Value {
	args := make([]Value, nArg)
	for i := range args {
		args[i] = Value{
			sqlite: db.sqlite,
			handle: util.ReadUint32(db.mod, pArg+ptrlen*uint32(i)),
		}
	}
	return args
}
