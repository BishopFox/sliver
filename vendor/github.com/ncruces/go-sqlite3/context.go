package sqlite3

import (
	"errors"
	"time"

	"github.com/ncruces/go-sqlite3/internal/errutil"
)

// Context is the context in which an SQL function executes.
// It is in no way related to a Go [context.Context].
//
// https://sqlite.org/c3ref/context.html
type Context struct {
	c      *Conn
	handle ptr_t
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
	ptr := ctx.c.wrp.AddHandle(data)
	ctx.c.wrp.Xsqlite3_set_auxdata_go(int32(ctx.handle), int32(n), int32(ptr))
}

// GetAuxData returns metadata for argument n of the function.
//
// https://sqlite.org/c3ref/get_auxdata.html
func (ctx Context) GetAuxData(n int) any {
	ptr := ptr_t(ctx.c.wrp.Xsqlite3_get_auxdata(int32(ctx.handle), int32(n)))
	return ctx.c.wrp.GetHandle(ptr)
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
	ctx.c.wrp.Xsqlite3_result_int64(
		int32(ctx.handle), value)
}

// ResultFloat sets the result of the function to a float64.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultFloat(value float64) {
	ctx.c.wrp.Xsqlite3_result_double(
		int32(ctx.handle), value)
}

// ResultText sets the result of the function to a string.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultText(value string) {
	ptr := ctx.c.wrp.NewString(value)
	ctx.c.wrp.Xsqlite3_result_text_go(
		int32(ctx.handle), int32(ptr), int64(len(value)))
}

// ResultRawText sets the text result of the function to a []byte.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultRawText(value []byte) {
	ctx.ResultText(string(value)) // does not escape
}

// ResultBlob sets the result of the function to a []byte.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultBlob(value []byte) {
	if len(value) == 0 {
		ctx.ResultZeroBlob(0)
		return
	}
	ptr := ctx.c.wrp.NewBytes(value)
	ctx.c.wrp.Xsqlite3_result_blob_go(
		int32(ctx.handle), int32(ptr), int64(len(value)))
}

// ResultZeroBlob sets the result of the function to a zero-filled, length n BLOB.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultZeroBlob(n int64) {
	ctx.c.wrp.Xsqlite3_result_zeroblob64(
		int32(ctx.handle), n)
}

// ResultNull sets the result of the function to NULL.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultNull() {
	ctx.c.wrp.Xsqlite3_result_null(
		int32(ctx.handle))
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
		panic(errutil.AssertErr())
	}
}

func (ctx Context) resultRFC3339Nano(value time.Time) {
	const maxlen = 48
	ptr := ctx.c.wrp.New(maxlen)
	buf := ctx.c.wrp.Bytes(ptr, maxlen)
	buf = value.AppendFormat(buf[:0], time.RFC3339Nano)
	_ = append(buf, 0)

	ctx.c.wrp.Xsqlite3_result_text_go(
		int32(ctx.handle), int32(ptr), int64(len(buf)))
}

// ResultPointer sets the result of the function to NULL, just like [Context.ResultNull],
// except that it also associates ptr with that NULL value such that it can be retrieved
// within an application-defined SQL function using [Value.Pointer].
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultPointer(ptr any) {
	valPtr := ctx.c.wrp.AddHandle(ptr)
	ctx.c.wrp.Xsqlite3_result_pointer_go(
		int32(ctx.handle), int32(valPtr))
}

// ResultValue sets the result of the function to a copy of [Value].
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultValue(value Value) {
	if value.c != ctx.c {
		ctx.ResultError(MISUSE)
		return
	}
	ctx.c.wrp.Xsqlite3_result_value(
		int32(ctx.handle), int32(value.handle))
}

// ResultError sets the result of the function an error.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultError(err error) {
	if errors.Is(err, NOMEM) {
		ctx.c.wrp.Xsqlite3_result_error_nomem(int32(ctx.handle))
		return
	}

	if errors.Is(err, TOOBIG) {
		ctx.c.wrp.Xsqlite3_result_error_toobig(int32(ctx.handle))
		return
	}

	msg, code := errorCode(err, ERROR)
	if msg != "" {
		defer ctx.c.arena.Mark()()
		ptr := ctx.c.arena.String(msg)
		ctx.c.wrp.Xsqlite3_result_error(
			int32(ctx.handle), int32(ptr), int32(len(msg)))
	}
	if code != res_t(ERROR) {
		ctx.c.wrp.Xsqlite3_result_error_code(
			int32(ctx.handle), int32(code))
	}
}

// ResultSubtype sets the subtype of the result of the function.
//
// https://sqlite.org/c3ref/result_subtype.html
func (ctx Context) ResultSubtype(t uint) {
	ctx.c.wrp.Xsqlite3_result_subtype(
		int32(ctx.handle), int32(t))
}

// VTabNoChange may return true if a column is being fetched as part
// of an update during which the column value will not change.
//
// https://sqlite.org/c3ref/vtab_nochange.html
func (ctx Context) VTabNoChange() bool {
	b := ctx.c.wrp.Xsqlite3_vtab_nochange(int32(ctx.handle))
	return b != 0
}
