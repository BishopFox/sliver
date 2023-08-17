package sqlite3

import (
	"math"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Value is any value that can be stored in a database table.
//
// https://www.sqlite.org/c3ref/value.html
type Value struct {
	*sqlite
	handle uint32
}

// Type returns the initial [Datatype] of the value.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v Value) Type() Datatype {
	r := v.call(v.api.valueType, uint64(v.handle))
	return Datatype(r)
}

// Bool returns the value as a bool.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are retrieved as integers,
// with 0 converted to false and any other value to true.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v Value) Bool() bool {
	if i := v.Int64(); i != 0 {
		return true
	}
	return false
}

// Int returns the value as an int.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v Value) Int() int {
	return int(v.Int64())
}

// Int64 returns the value as an int64.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v Value) Int64() int64 {
	r := v.call(v.api.valueInteger, uint64(v.handle))
	return int64(r)
}

// Float returns the value as a float64.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v Value) Float() float64 {
	r := v.call(v.api.valueFloat, uint64(v.handle))
	return math.Float64frombits(r)
}

// Time returns the value as a [time.Time].
//
// https://www.sqlite.org/c3ref/value_blob.html
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
// https://www.sqlite.org/c3ref/value_blob.html
func (v Value) Text() string {
	return string(v.RawText())
}

// Blob appends to buf and returns
// the value as a []byte.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v Value) Blob(buf []byte) []byte {
	return append(buf, v.RawBlob()...)
}

// RawText returns the value as a []byte.
// The []byte is owned by SQLite and may be invalidated by
// subsequent calls to [Value] methods.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v Value) RawText() []byte {
	r := v.call(v.api.valueText, uint64(v.handle))
	return v.rawBytes(uint32(r))
}

// RawBlob returns the value as a []byte.
// The []byte is owned by SQLite and may be invalidated by
// subsequent calls to [Value] methods.
//
// https://www.sqlite.org/c3ref/value_blob.html
func (v Value) RawBlob() []byte {
	r := v.call(v.api.valueBlob, uint64(v.handle))
	return v.rawBytes(uint32(r))
}

func (v Value) rawBytes(ptr uint32) []byte {
	if ptr == 0 {
		return nil
	}

	r := v.call(v.api.valueBytes, uint64(v.handle))
	return util.View(v.mod, ptr, r)
}
