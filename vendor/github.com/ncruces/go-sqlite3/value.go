package sqlite3

import (
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Value is any value that can be stored in a database table.
//
// https://sqlite.org/c3ref/value.html
type Value struct {
	*sqlite
	handle uint32
	unprot bool
	copied bool
}

func (v Value) protected() uint64 {
	if v.unprot {
		panic(util.ValueErr)
	}
	return uint64(v.handle)
}

// Dup makes a copy of the SQL value and returns a pointer to that copy.
//
// https://sqlite.org/c3ref/value_dup.html
func (v Value) Dup() *Value {
	r := v.call("sqlite3_value_dup", uint64(v.handle))
	return &Value{
		copied: true,
		sqlite: v.sqlite,
		handle: uint32(r),
	}
}

// Close frees an SQL value previously obtained by [Value.Dup].
//
// https://sqlite.org/c3ref/value_dup.html
func (dup *Value) Close() error {
	if !dup.copied {
		panic(util.ValueErr)
	}
	dup.call("sqlite3_value_free", uint64(dup.handle))
	dup.handle = 0
	return nil
}

// Type returns the initial [Datatype] of the value.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) Type() Datatype {
	r := v.call("sqlite3_value_type", v.protected())
	return Datatype(r)
}

// Bool returns the value as a bool.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are retrieved as integers,
// with 0 converted to false and any other value to true.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) Bool() bool {
	if i := v.Int64(); i != 0 {
		return true
	}
	return false
}

// Int returns the value as an int.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) Int() int {
	return int(v.Int64())
}

// Int64 returns the value as an int64.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) Int64() int64 {
	r := v.call("sqlite3_value_int64", v.protected())
	return int64(r)
}

// Float returns the value as a float64.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) Float() float64 {
	r := v.call("sqlite3_value_double", v.protected())
	return math.Float64frombits(r)
}

// Time returns the value as a [time.Time].
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) Time(format TimeFormat) time.Time {
	var a any
	switch v.Type() {
	case INTEGER:
		a = v.Int64()
	case FLOAT:
		a = v.Float()
	case TEXT, BLOB:
		a = v.Text()
	case NULL:
		return time.Time{}
	default:
		panic(util.AssertErr())
	}
	t, _ := format.Decode(a)
	return t
}

// Text returns the value as a string.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) Text() string {
	return string(v.RawText())
}

// Blob appends to buf and returns
// the value as a []byte.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) Blob(buf []byte) []byte {
	return append(buf, v.RawBlob()...)
}

// RawText returns the value as a []byte.
// The []byte is owned by SQLite and may be invalidated by
// subsequent calls to [Value] methods.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) RawText() []byte {
	r := v.call("sqlite3_value_text", v.protected())
	return v.rawBytes(uint32(r))
}

// RawBlob returns the value as a []byte.
// The []byte is owned by SQLite and may be invalidated by
// subsequent calls to [Value] methods.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) RawBlob() []byte {
	r := v.call("sqlite3_value_blob", v.protected())
	return v.rawBytes(uint32(r))
}

func (v Value) rawBytes(ptr uint32) []byte {
	if ptr == 0 {
		return nil
	}

	r := v.call("sqlite3_value_bytes", v.protected())
	return util.View(v.mod, ptr, r)
}

// Pointer gets the pointer associated with this value,
// or nil if it has no associated pointer.
func (v Value) Pointer() any {
	r := v.call("sqlite3_value_pointer_go", v.protected())
	return util.GetHandle(v.ctx, uint32(r))
}

// JSON parses a JSON-encoded value
// and stores the result in the value pointed to by ptr.
func (v Value) JSON(ptr any) error {
	var data []byte
	switch v.Type() {
	case NULL:
		data = append(data, "null"...)
	case TEXT:
		data = v.RawText()
	case BLOB:
		data = v.RawBlob()
	case INTEGER:
		data = strconv.AppendInt(nil, v.Int64(), 10)
	case FLOAT:
		data = strconv.AppendFloat(nil, v.Float(), 'g', -1, 64)
	default:
		panic(util.AssertErr())
	}
	return json.Unmarshal(data, ptr)
}
