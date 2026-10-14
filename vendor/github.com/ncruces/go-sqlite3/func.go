package sqlite3

import (
	"bytes"
	"io"
	"iter"
	"sync"
	"sync/atomic"

	"github.com/ncruces/go-sqlite3/internal/errutil"
)

// CollationNeeded registers a callback to be invoked
// whenever an unknown collation sequence is required.
//
// https://sqlite.org/c3ref/collation_needed.html
func (c *Conn) CollationNeeded(cb func(db *Conn, name string)) error {
	var enable int32
	if cb != nil {
		enable = 1
	}
	rc := res_t(c.wrp.Xsqlite3_collation_needed_go(int32(c.handle), enable))
	if err := c.error(rc); err != nil {
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
func (c *Conn) AnyCollationNeeded() error {
	return c.CollationNeeded(func(db *Conn, name string) {
		db.CreateCollation(name, bytes.Compare)
	})
}

// CreateCollation defines a new collating sequence.
//
// https://sqlite.org/c3ref/create_collation.html
func (c *Conn) CreateCollation(name string, fn CollatingFunction) error {
	var funcPtr ptr_t
	defer c.arena.Mark()()
	namePtr := c.arena.String(name)
	if fn != nil {
		funcPtr = c.wrp.AddHandle(fn)
	}
	rc := res_t(c.wrp.Xsqlite3_create_collation_go(
		int32(c.handle), int32(namePtr), int32(funcPtr)))
	return c.error(rc)
}

// CollatingFunction is the type of a collation callback.
// Implementations must not retain a or b.
type CollatingFunction func(a, b []byte) int

// CreateFunction defines a new scalar SQL function.
//
// https://sqlite.org/c3ref/create_function.html
func (c *Conn) CreateFunction(name string, nArg int, flag FunctionFlag, fn ScalarFunction) error {
	var funcPtr ptr_t
	defer c.arena.Mark()()
	namePtr := c.arena.String(name)
	if fn != nil {
		funcPtr = c.wrp.AddHandle(fn)
	}
	rc := res_t(c.wrp.Xsqlite3_create_function_go(
		int32(c.handle), int32(namePtr), int32(nArg),
		int32(flag), int32(funcPtr)))
	return c.error(rc)
}

// ScalarFunction is the type of a scalar SQL function.
// Implementations must not retain arg.
type ScalarFunction func(ctx Context, arg ...Value)

// CreateAggregateFunction defines a new aggregate SQL function.
//
// https://sqlite.org/c3ref/create_function.html
func (c *Conn) CreateAggregateFunction(name string, nArg int, flag FunctionFlag, fn AggregateSeqFunction) error {
	var funcPtr ptr_t
	defer c.arena.Mark()()
	namePtr := c.arena.String(name)
	if fn != nil {
		funcPtr = c.wrp.AddHandle(AggregateConstructor(func() AggregateFunction {
			a := aggregateFunc{fn: fn}
			coro := func(yieldCoro func(struct{}) bool) {
				seq := func(yieldSeq func([]Value) bool) {
					for yieldSeq(a.arg) {
						if !yieldCoro(struct{}{}) {
							break
						}
					}
				}
				fn(&a.ctx, seq)
			}
			a.next, a.stop = iter.Pull(coro)
			return &a
		}))
	}
	rc := res_t(c.wrp.Xsqlite3_create_aggregate_function_go(
		int32(c.handle), int32(namePtr), int32(nArg),
		int32(flag), int32(funcPtr)))
	return c.error(rc)
}

// AggregateSeqFunction is the type of an aggregate SQL function.
// Implementations must not retain the slices yielded by seq.
type AggregateSeqFunction func(ctx *Context, seq iter.Seq[[]Value])

// CreateWindowFunction defines a new aggregate or aggregate window SQL function.
// If fn returns a [WindowFunction], an aggregate window function is created.
// If fn returns an [io.Closer], it will be called to free resources.
//
// https://sqlite.org/c3ref/create_function.html
func (c *Conn) CreateWindowFunction(name string, nArg int, flag FunctionFlag, fn AggregateConstructor) error {
	var funcPtr ptr_t
	defer c.arena.Mark()()
	namePtr := c.arena.String(name)
	if fn != nil {
		funcPtr = c.wrp.AddHandle(fn)
	}
	rc := res_t(c.wrp.Xsqlite3_create_window_function_go(
		int32(c.handle), int32(namePtr), int32(nArg),
		int32(flag), int32(funcPtr)))
	return c.error(rc)
}

// AggregateConstructor is a an [AggregateFunction] constructor.
type AggregateConstructor func() AggregateFunction

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
	defer c.arena.Mark()()
	namePtr := c.arena.String(name)
	rc := res_t(c.wrp.Xsqlite3_overload_function(
		int32(c.handle), int32(namePtr), int32(nArg)))
	return c.error(rc)
}

func (e env) Xgo_collation_needed(pArg, pDB, eTextRep, zName int32) {
	if c, ok := e.DB.(*Conn); ok && c.handle == ptr_t(pDB) && c.collation != nil {
		name := e.ReadString(ptr_t(zName), _MAX_NAME)
		c.collation(c, name)
	}
}

func (e env) Xgo_compare(pApp, nKey1, pKey1, nKey2, pKey2 int32) int32 {
	fn := e.GetHandle(ptr_t(pApp)).(CollatingFunction)
	return int32(fn(
		e.Bytes(ptr_t(pKey1), int64(nKey1)),
		e.Bytes(ptr_t(pKey2), int64(nKey2))))
}

func (e env) Xgo_func(pCtx, pApp, nArg, pArg int32) {
	db := e.DB.(*Conn)
	args := callbackArgs(db, nArg, ptr_t(pArg))
	defer returnArgs(args)
	fn := db.wrp.GetHandle(ptr_t(pApp)).(ScalarFunction)
	fn(Context{db, ptr_t(pCtx)}, *args...)
}

func (e env) Xgo_step(pCtx, pAgg, pApp, nArg, pArg int32) {
	db := e.DB.(*Conn)
	args := callbackArgs(db, nArg, ptr_t(pArg))
	defer returnArgs(args)
	fn, _ := callbackAggregate(db, ptr_t(pAgg), ptr_t(pApp))
	fn.Step(Context{db, ptr_t(pCtx)}, *args...)
}

func (e env) Xgo_value(pCtx, pAgg, pApp, final int32) {
	db := e.DB.(*Conn)
	fn, handle := callbackAggregate(db, ptr_t(pAgg), ptr_t(pApp))
	fn.Value(Context{db, ptr_t(pCtx)})

	// Cleanup.
	if final != 0 {
		var err error
		if handle != 0 {
			err = e.DelHandle(handle)
		} else if c, ok := fn.(io.Closer); ok {
			err = c.Close()
		}
		if err != nil {
			Context{db, ptr_t(pCtx)}.ResultError(err)
			return // notest
		}
	}
}

func (e env) Xgo_inverse(pCtx, pAgg, nArg, pArg int32) {
	db := e.DB.(*Conn)
	args := callbackArgs(db, nArg, ptr_t(pArg))
	defer returnArgs(args)
	fn, ok := db.wrp.GetHandle(ptr_t(pAgg)).(WindowFunction)
	if !ok {
		Context{db, ptr_t(pCtx)}.ResultError(
			errutil.ErrorString("may not be used as a window function"))
		return // notest
	}
	fn.Inverse(Context{db, ptr_t(pCtx)}, *args...)
}

func callbackAggregate(db *Conn, pAgg, pApp ptr_t) (AggregateFunction, ptr_t) {
	if pApp == 0 {
		handle := ptr_t(db.wrp.Read32(pAgg))
		return db.wrp.GetHandle(handle).(AggregateFunction), handle
	}

	// We need to create the aggregate.
	fn := db.wrp.GetHandle(pApp).(AggregateConstructor)()
	if pAgg != 0 {
		handle := db.wrp.AddHandle(fn)
		db.wrp.Write32(pAgg, uint32(handle))
		return fn, handle
	}
	return fn, 0
}

var (
	valueArgsPool sync.Pool
	valueArgsLen  atomic.Int32
)

func callbackArgs(db *Conn, nArg int32, pArg ptr_t) *[]Value {
	arg, ok := valueArgsPool.Get().(*[]Value)
	if !ok || cap(*arg) < int(nArg) {
		max := valueArgsLen.Or(nArg) | nArg
		lst := make([]Value, max)
		arg = &lst
	}
	lst := (*arg)[:nArg]
	for i := range lst {
		lst[i] = Value{
			c:      db,
			handle: ptr_t(db.wrp.Read32(pArg + ptr_t(i)*ptrlen)),
		}
	}
	*arg = lst
	return arg
}

func returnArgs(p *[]Value) {
	valueArgsPool.Put(p)
}

type aggregateFunc struct {
	next func() (struct{}, bool)
	stop func()
	fn   AggregateSeqFunction
	ctx  Context
	arg  []Value
	step bool
}

func (a *aggregateFunc) Step(ctx Context, arg ...Value) {
	a.step = true
	a.ctx = ctx
	a.arg = append(a.arg[:0], arg...)
	if _, more := a.next(); !more {
		a.stop()
	}
}

func (a *aggregateFunc) Value(ctx Context) {
	a.ctx = ctx
	a.stop()
	if !a.step {
		a.fn(&a.ctx, noValuesSeq)
	}
}

func (a *aggregateFunc) Close() error {
	a.stop()
	return nil
}

func noValuesSeq(func([]Value) bool) {}
