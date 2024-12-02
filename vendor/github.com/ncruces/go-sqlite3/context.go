package sqlite3

import (
	"encoding/json"
	"errors"
	"math"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Context is the context in which an SQL function executes.
// An SQLite [Context] is in no way related to a Go [context.Context].
//
// https://sqlite.org/c3ref/context.html
type Context struct {
	c      *Conn
	handle uint32
}

// Conn returns the database connection of the
// [Conn.CreateFunction] or [Conn.CreateWindowFunction]
// routines that originally registered the application defined function.
//
// https://sqlite.org/c3ref/context_db_handle.html
func (ctx Context) Conn() *Conn {
	return ctx.c
}

// SetAuxData saves metadata for argument n of the function.
//
// https://sqlite.org/c3ref/get_auxdata.html
func (ctx Context) SetAuxData(n int, data any) {
	ptr := util.AddHandle(ctx.c.ctx, data)
	ctx.c.call("sqlite3_set_auxdata_go", uint64(ctx.handle), uint64(n), uint64(ptr))
}

// GetAuxData returns metadata for argument n of the function.
//
// https://sqlite.org/c3ref/get_auxdata.html
func (ctx Context) GetAuxData(n int) any {
	ptr := uint32(ctx.c.call("sqlite3_get_auxdata", uint64(ctx.handle), uint64(n)))
	return util.GetHandle(ctx.c.ctx, ptr)
}

// ResultBool sets the result of the function to a bool.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are stored as integers 0 (false) and 1 (true).
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultBool(value bool) {
	var i int64
	if value {
		i = 1
	}
	ctx.ResultInt64(i)
}

// ResultInt sets the result of the function to an int.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultInt(value int) {
	ctx.ResultInt64(int64(value))
}

// ResultInt64 sets the result of the function to an int64.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultInt64(value int64) {
	ctx.c.call("sqlite3_result_int64",
		uint64(ctx.handle), uint64(value))
}

// ResultFloat sets the result of the function to a float64.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultFloat(value float64) {
	ctx.c.call("sqlite3_result_double",
		uint64(ctx.handle), math.Float64bits(value))
}

// ResultText sets the result of the function to a string.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultText(value string) {
	ptr := ctx.c.newString(value)
	ctx.c.call("sqlite3_result_text_go",
		uint64(ctx.handle), uint64(ptr), uint64(len(value)))
}

// ResultRawText sets the text result of the function to a []byte.
// Returning a nil slice is the same as calling [Context.ResultNull].
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultRawText(value []byte) {
	ptr := ctx.c.newBytes(value)
	ctx.c.call("sqlite3_result_text_go",
		uint64(ctx.handle), uint64(ptr), uint64(len(value)))
}

// ResultBlob sets the result of the function to a []byte.
// Returning a nil slice is the same as calling [Context.ResultNull].
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultBlob(value []byte) {
	ptr := ctx.c.newBytes(value)
	ctx.c.call("sqlite3_result_blob_go",
		uint64(ctx.handle), uint64(ptr), uint64(len(value)))
}

// ResultZeroBlob sets the result of the function to a zero-filled, length n BLOB.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultZeroBlob(n int64) {
	ctx.c.call("sqlite3_result_zeroblob64",
		uint64(ctx.handle), uint64(n))
}

// ResultNull sets the result of the function to NULL.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultNull() {
	ctx.c.call("sqlite3_result_null",
		uint64(ctx.handle))
}

// ResultTime sets the result of the function to a [time.Time].
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultTime(value time.Time, format TimeFormat) {
	switch format {
	case TimeFormatDefault, TimeFormatAuto, time.RFC3339Nano:
		ctx.resultRFC3339Nano(value)
		return
	}
	switch v := format.Encode(value).(type) {
	case string:
		ctx.ResultText(v)
	case int64:
		ctx.ResultInt64(v)
	case float64:
		ctx.ResultFloat(v)
	default:
		panic(util.AssertErr())
	}
}

func (ctx Context) resultRFC3339Nano(value time.Time) {
	const maxlen = uint64(len(time.RFC3339Nano)) + 5

	ptr := ctx.c.new(maxlen)
	buf := util.View(ctx.c.mod, ptr, maxlen)
	buf = value.AppendFormat(buf[:0], time.RFC3339Nano)

	ctx.c.call("sqlite3_result_text_go",
		uint64(ctx.handle), uint64(ptr), uint64(len(buf)))
}

// ResultPointer sets the result of the function to NULL, just like [Context.ResultNull],
// except that it also associates ptr with that NULL value such that it can be retrieved
// within an application-defined SQL function using [Value.Pointer].
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultPointer(ptr any) {
	valPtr := util.AddHandle(ctx.c.ctx, ptr)
	ctx.c.call("sqlite3_result_pointer_go",
		uint64(ctx.handle), uint64(valPtr))
}

// ResultJSON sets the result of the function to the JSON encoding of value.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultJSON(value any) {
	data, err := json.Marshal(value)
	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
	ctx.ResultRawText(data)
}

// ResultValue sets the result of the function to a copy of [Value].
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultValue(value Value) {
	if value.c != ctx.c {
		ctx.ResultError(MISUSE)
		return
	}
	ctx.c.call("sqlite3_result_value",
		uint64(ctx.handle), uint64(value.handle))
}

// ResultError sets the result of the function an error.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultError(err error) {
	if errors.Is(err, NOMEM) {
		ctx.c.call("sqlite3_result_error_nomem", uint64(ctx.handle))
		return
	}

	if errors.Is(err, TOOBIG) {
		ctx.c.call("sqlite3_result_error_toobig", uint64(ctx.handle))
		return
	}

	msg, code := errorCode(err, _OK)
	if msg != "" {
		defer ctx.c.arena.mark()()
		ptr := ctx.c.arena.string(msg)
		ctx.c.call("sqlite3_result_error",
			uint64(ctx.handle), uint64(ptr), uint64(len(msg)))
	}
	if code != _OK {
		ctx.c.call("sqlite3_result_error_code",
			uint64(ctx.handle), uint64(code))
	}
}

// VTabNoChange may return true if a column is being fetched as part
// of an update during which the column value will not change.
//
// https://sqlite.org/c3ref/vtab_nochange.html
func (ctx Context) VTabNoChange() bool {
	r := ctx.c.call("sqlite3_vtab_nochange", uint64(ctx.handle))
	return r != 0
}
