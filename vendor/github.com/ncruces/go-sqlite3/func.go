package sqlite3

import (
	"context"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// AnyCollationNeeded registers a fake collating function
// for any unknown collating sequence.
// The fake collating function works like BINARY.
//
// This extension can be used to load schemas that contain
// one or more unknown collating sequences.
func (c *Conn) AnyCollationNeeded() {
	c.call(c.api.anyCollation, uint64(c.handle), 0, 0)
}

// CreateCollation defines a new collating sequence.
//
// https://www.sqlite.org/c3ref/create_collation.html
func (c *Conn) CreateCollation(name string, fn func(a, b []byte) int) error {
	namePtr := c.arena.string(name)
	funcPtr := util.AddHandle(c.ctx, fn)
	r := c.call(c.api.createCollation,
		uint64(c.handle), uint64(namePtr), uint64(funcPtr))
	if err := c.error(r); err != nil {
		util.DelHandle(c.ctx, funcPtr)
		return err
	}
	return nil
}

// CreateFunction defines a new scalar SQL function.
//
// https://www.sqlite.org/c3ref/create_function.html
func (c *Conn) CreateFunction(name string, nArg int, flag FunctionFlag, fn func(ctx Context, arg ...Value)) error {
	namePtr := c.arena.string(name)
	funcPtr := util.AddHandle(c.ctx, fn)
	r := c.call(c.api.createFunction,
		uint64(c.handle), uint64(namePtr), uint64(nArg),
		uint64(flag), uint64(funcPtr))
	return c.error(r)
}

// CreateWindowFunction defines a new aggregate or aggregate window SQL function.
// If fn returns a [WindowFunction], then an aggregate window function is created.
//
// https://www.sqlite.org/c3ref/create_function.html
func (c *Conn) CreateWindowFunction(name string, nArg int, flag FunctionFlag, fn func() AggregateFunction) error {
	call := c.api.createAggregate
	namePtr := c.arena.string(name)
	funcPtr := util.AddHandle(c.ctx, fn)
	if _, ok := fn().(WindowFunction); ok {
		call = c.api.createWindow
	}
	r := c.call(call,
		uint64(c.handle), uint64(namePtr), uint64(nArg),
		uint64(flag), uint64(funcPtr))
	return c.error(r)
}

// AggregateFunction is the interface an aggregate function should implement.
//
// https://www.sqlite.org/appfunc.html
type AggregateFunction interface {
	// Step is invoked to add a row to the current window.
	// The function arguments, if any, corresponding to the row being added are passed to Step.
	Step(ctx Context, arg ...Value)

	// Value is invoked to return the current value of the aggregate.
	Value(ctx Context)
}

// WindowFunction is the interface an aggregate window function should implement.
//
// https://www.sqlite.org/windowfunctions.html
type WindowFunction interface {
	AggregateFunction

	// Inverse is invoked to remove the oldest presently aggregated result of Step from the current window.
	// The function arguments, if any, are those passed to Step for the row being removed.
	Inverse(ctx Context, arg ...Value)
}

func exportHostFunctions(env wazero.HostModuleBuilder) wazero.HostModuleBuilder {
	util.ExportFuncVI(env, "go_destroy", callbackDestroy)
	util.ExportFuncIIIIII(env, "go_compare", callbackCompare)
	util.ExportFuncVIII(env, "go_func", callbackFunc)
	util.ExportFuncVIII(env, "go_step", callbackStep)
	util.ExportFuncVI(env, "go_final", callbackFinal)
	util.ExportFuncVI(env, "go_value", callbackValue)
	util.ExportFuncVIII(env, "go_inverse", callbackInverse)
	return env
}

func callbackDestroy(ctx context.Context, mod api.Module, pApp uint32) {
	util.DelHandle(ctx, pApp)
}

func callbackCompare(ctx context.Context, mod api.Module, pApp, nKey1, pKey1, nKey2, pKey2 uint32) uint32 {
	fn := util.GetHandle(ctx, pApp).(func(a, b []byte) int)
	return uint32(fn(util.View(mod, pKey1, uint64(nKey1)), util.View(mod, pKey2, uint64(nKey2))))
}

func callbackFunc(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {
	sqlite := ctx.Value(sqliteKey{}).(*sqlite)
	fn := callbackHandle(sqlite, pCtx).(func(ctx Context, arg ...Value))
	fn(Context{sqlite, pCtx}, callbackArgs(sqlite, nArg, pArg)...)
}

func callbackStep(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {
	sqlite := ctx.Value(sqliteKey{}).(*sqlite)
	fn := callbackAggregate(sqlite, pCtx, nil).(AggregateFunction)
	fn.Step(Context{sqlite, pCtx}, callbackArgs(sqlite, nArg, pArg)...)
}

func callbackFinal(ctx context.Context, mod api.Module, pCtx uint32) {
	var handle uint32
	sqlite := ctx.Value(sqliteKey{}).(*sqlite)
	fn := callbackAggregate(sqlite, pCtx, &handle).(AggregateFunction)
	fn.Value(Context{sqlite, pCtx})
	if err := util.DelHandle(ctx, handle); err != nil {
		Context{sqlite, pCtx}.ResultError(err)
	}
}

func callbackValue(ctx context.Context, mod api.Module, pCtx uint32) {
	sqlite := ctx.Value(sqliteKey{}).(*sqlite)
	fn := callbackAggregate(sqlite, pCtx, nil).(AggregateFunction)
	fn.Value(Context{sqlite, pCtx})
}

func callbackInverse(ctx context.Context, mod api.Module, pCtx, nArg, pArg uint32) {
	sqlite := ctx.Value(sqliteKey{}).(*sqlite)
	fn := callbackAggregate(sqlite, pCtx, nil).(WindowFunction)
	fn.Inverse(Context{sqlite, pCtx}, callbackArgs(sqlite, nArg, pArg)...)
}

func callbackHandle(sqlite *sqlite, pCtx uint32) any {
	pApp := uint32(sqlite.call(sqlite.api.userData, uint64(pCtx)))
	return util.GetHandle(sqlite.ctx, pApp)
}

func callbackAggregate(sqlite *sqlite, pCtx uint32, close *uint32) any {
	// On close, we're getting rid of the handle.
	// Don't allocate space to store it.
	var size uint64
	if close == nil {
		size = ptrlen
	}
	ptr := uint32(sqlite.call(sqlite.api.aggregateCtx, uint64(pCtx), size))

	// Try loading the handle, if we already have one, or want a new one.
	if ptr != 0 || size != 0 {
		if handle := util.ReadUint32(sqlite.mod, ptr); handle != 0 {
			fn := util.GetHandle(sqlite.ctx, handle)
			if close != nil {
				*close = handle
			}
			if fn != nil {
				return fn
			}
		}
	}

	// Create a new aggregate and store the handle.
	fn := callbackHandle(sqlite, pCtx).(func() AggregateFunction)()
	if ptr != 0 {
		util.WriteUint32(sqlite.mod, ptr, util.AddHandle(sqlite.ctx, fn))
	}
	return fn
}

func callbackArgs(sqlite *sqlite, nArg, pArg uint32) []Value {
	args := make([]Value, nArg)
	for i := range args {
		args[i] = Value{
			sqlite: sqlite,
			handle: util.ReadUint32(sqlite.mod, pArg+ptrlen*uint32(i)),
		}
	}
	return args
}
