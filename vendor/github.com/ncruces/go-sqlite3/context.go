package sqlite3

import (
	"errors"
	"math"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Context is the context in which an SQL function executes.
// An SQLite [Context] is in no way related to a Go [context.Context].
//
// https://www.sqlite.org/c3ref/context.html
type Context struct {
	*sqlite
	handle uint32
}

// SetAuxData saves metadata for argument n of the function.
//
// https://www.sqlite.org/c3ref/get_auxdata.html
func (c Context) SetAuxData(n int, data any) {
	ptr := util.AddHandle(c.ctx, data)
	c.call(c.api.setAuxData, uint64(c.handle), uint64(n), uint64(ptr))
}

// GetAuxData returns metadata for argument n of the function.
//
// https://www.sqlite.org/c3ref/get_auxdata.html
func (c Context) GetAuxData(n int) any {
	ptr := uint32(c.call(c.api.getAuxData, uint64(c.handle), uint64(n)))
	return util.GetHandle(c.ctx, ptr)
}

// ResultBool sets the result of the function to a bool.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are stored as integers 0 (false) and 1 (true).
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c Context) ResultBool(value bool) {
	var i int64
	if value {
		i = 1
	}
	c.ResultInt64(i)
}

// ResultInt sets the result of the function to an int.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c Context) ResultInt(value int) {
	c.ResultInt64(int64(value))
}

// ResultInt64 sets the result of the function to an int64.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c Context) ResultInt64(value int64) {
	c.call(c.api.resultInteger,
		uint64(c.handle), uint64(value))
}

// ResultFloat sets the result of the function to a float64.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c Context) ResultFloat(value float64) {
	c.call(c.api.resultFloat,
		uint64(c.handle), math.Float64bits(value))
}

// ResultText sets the result of the function to a string.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c Context) ResultText(value string) {
	ptr := c.newString(value)
	c.call(c.api.resultText,
		uint64(c.handle), uint64(ptr), uint64(len(value)),
		uint64(c.api.destructor), _UTF8)
}

// ResultBlob sets the result of the function to a []byte.
// Returning a nil slice is the same as calling [Context.ResultNull].
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c Context) ResultBlob(value []byte) {
	ptr := c.newBytes(value)
	c.call(c.api.resultBlob,
		uint64(c.handle), uint64(ptr), uint64(len(value)),
		uint64(c.api.destructor))
}

// BindZeroBlob sets the result of the function to a zero-filled, length n BLOB.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c Context) ResultZeroBlob(n int64) {
	c.call(c.api.resultZeroBlob,
		uint64(c.handle), uint64(n))
}

// ResultNull sets the result of the function to NULL.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c Context) ResultNull() {
	c.call(c.api.resultNull,
		uint64(c.handle))
}

// ResultTime sets the result of the function to a [time.Time].
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c Context) ResultTime(value time.Time, format TimeFormat) {
	if format == TimeFormatDefault {
		c.resultRFC3339Nano(value)
		return
	}
	switch v := format.Encode(value).(type) {
	case string:
		c.ResultText(v)
	case int64:
		c.ResultInt64(v)
	case float64:
		c.ResultFloat(v)
	default:
		panic(util.AssertErr())
	}
}

func (c Context) resultRFC3339Nano(value time.Time) {
	const maxlen = uint64(len(time.RFC3339Nano))

	ptr := c.new(maxlen)
	buf := util.View(c.mod, ptr, maxlen)
	buf = value.AppendFormat(buf[:0], time.RFC3339Nano)

	c.call(c.api.resultText,
		uint64(c.handle), uint64(ptr), uint64(len(buf)),
		uint64(c.api.destructor), _UTF8)
}

// ResultError sets the result of the function an error.
//
// https://www.sqlite.org/c3ref/result_blob.html
func (c Context) ResultError(err error) {
	if errors.Is(err, NOMEM) {
		c.call(c.api.resultErrorMem, uint64(c.handle))
		return
	}

	if errors.Is(err, TOOBIG) {
		c.call(c.api.resultErrorBig, uint64(c.handle))
		return
	}

	str := err.Error()
	ptr := c.newString(str)
	c.call(c.api.resultError,
		uint64(c.handle), uint64(ptr), uint64(len(str)))
	c.free(ptr)

	var code uint64
	var ecode ErrorCode
	var xcode xErrorCode
	switch {
	case errors.As(err, &xcode):
		code = uint64(xcode)
	case errors.As(err, &ecode):
		code = uint64(ecode)
	}
	if code != 0 {
		c.call(c.api.resultErrorCode,
			uint64(c.handle), code)
	}
}
