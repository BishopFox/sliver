package sqlite3

import (
	"time"

	"github.com/ncruces/go-sqlite3/internal/errutil"
)

// Value is any value that can be stored in a database table.
//
// https://sqlite.org/c3ref/value.html
type Value struct {
	c      *Conn
	handle ptr_t
}

// Dup makes a copy of the SQL value and returns a pointer to that copy.
//
// https://sqlite.org/c3ref/value_dup.html
func (v Value) Dup() *Value {
	ptr := ptr_t(v.c.wrp.Xsqlite3_value_dup(int32(v.handle)))
	return &Value{
		c:      v.c,
		handle: ptr,
	}
}

// Close frees an SQL value previously obtained by [Value.Dup].
//
// https://sqlite.org/c3ref/value_dup.html
func (v *Value) Close() error {
	v.c.wrp.Xsqlite3_value_free(int32(v.handle))
	v.handle = 0
	return nil
}

// Type returns the initial datatype of the value.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) Type() Datatype {
	return Datatype(v.c.wrp.Xsqlite3_value_type(int32(v.handle)))
}

// Subtype returns the subtype of the value.
//
// https://sqlite.org/c3ref/value_subtype.html
func (v Value) Subtype() uint {
	return uint(uint32(v.c.wrp.Xsqlite3_value_subtype(int32(v.handle))))
}

// NumericType returns the numeric datatype of the value.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) NumericType() Datatype {
	return Datatype(v.c.wrp.Xsqlite3_value_numeric_type(int32(v.handle)))
}

// Bool returns the value as a bool.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are retrieved as numbers,
// with 0 converted to false and any other value to true.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) Bool() bool {
	return v.Float() != 0
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
	return v.c.wrp.Xsqlite3_value_int64(int32(v.handle))
}

// Float returns the value as a float64.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) Float() float64 {
	return v.c.wrp.Xsqlite3_value_double(int32(v.handle))
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
		panic(errutil.AssertErr())
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
	ptr := ptr_t(v.c.wrp.Xsqlite3_value_text(int32(v.handle)))
	return v.rawBytes(ptr, 1)
}

// RawBlob returns the value as a []byte.
// The []byte is owned by SQLite and may be invalidated by
// subsequent calls to [Value] methods.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) RawBlob() []byte {
	ptr := ptr_t(v.c.wrp.Xsqlite3_value_blob(int32(v.handle)))
	return v.rawBytes(ptr, 0)
}

func (v Value) rawBytes(ptr ptr_t, nul int32) []byte {
	if ptr == 0 {
		return nil
	}

	n := v.c.wrp.Xsqlite3_value_bytes(int32(v.handle))
	return v.c.wrp.Bytes(ptr, int64(n+nul))[:n]
}

// Pointer gets the pointer associated with this value,
// or nil if it has no associated pointer.
func (v Value) Pointer() any {
	ptr := ptr_t(v.c.wrp.Xsqlite3_value_pointer_go(int32(v.handle)))
	return v.c.wrp.GetHandle(ptr)
}

// NoChange returns true if and only if the value is unchanged
// in a virtual table update operatiom.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) NoChange() bool {
	b := v.c.wrp.Xsqlite3_value_nochange(int32(v.handle))
	return b != 0
}

// FromBind returns true if value originated from a bound parameter.
//
// https://sqlite.org/c3ref/value_blob.html
func (v Value) FromBind() bool {
	b := v.c.wrp.Xsqlite3_value_frombind(int32(v.handle))
	return b != 0
}

// InFirst returns the first element
// on the right-hand side of an IN constraint.
//
// https://sqlite.org/c3ref/vtab_in_first.html
func (v Value) InFirst() (Value, error) {
	return v.c.returnValue(v.c.wrp.Xsqlite3_vtab_in_first, v.handle)
}

// InNext returns the next element
// on the right-hand side of an IN constraint.
//
// https://sqlite.org/c3ref/vtab_in_first.html
func (v Value) InNext() (Value, error) {
	return v.c.returnValue(v.c.wrp.Xsqlite3_vtab_in_next, v.handle)
}

func (c *Conn) returnValue(fn func(_, _ int32) int32, handle ptr_t) (Value, error) {
	defer c.wrp.StackMark()()
	valPtr := c.wrp.StackAlloc(ptrlen)
	rc := res_t(fn(int32(handle), int32(valPtr)))
	if err := c.error(rc); err != nil {
		return Value{}, err
	}
	return Value{c: c, handle: ptr_t(c.wrp.Read32(valPtr))}, nil
}

func (c *Conn) columnValue(fn func(_, _, _ int32) int32, handle ptr_t, column int) (Value, error) {
	defer c.wrp.StackMark()()
	valPtr := c.wrp.StackAlloc(ptrlen)
	rc := res_t(fn(int32(handle), int32(column), int32(valPtr)))
	if err := c.error(rc); err != nil {
		return Value{}, err
	}
	return Value{c: c, handle: ptr_t(c.wrp.Read32(valPtr))}, nil
}
