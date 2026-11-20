//go:build go1.27

package sqlite3

import (
	"encoding/json/v2"
	"strconv"

	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// JSON returns a value that can be used as an argument to
// [database/sql.DB.Exec], [database/sql.Row.Scan] and similar methods to
// store value as JSON, or decode JSON into value.
// JSON should NOT be used with [Stmt.BindJSON], [Stmt.ColumnJSON],
// [Value.JSON], or [Context.ResultJSON].
func JSON(value any) any {
	return util.JSON{Value: value}
}

// ResultJSON sets the result of the function to the JSON encoding of value.
//
// https://sqlite.org/c3ref/result_blob.html
func (ctx Context) ResultJSON(value any) {
	w := bytesWriter{wrp: ctx.c.wrp}
	if err := json.MarshalWrite(&w, value); err != nil {
		ctx.c.wrp.Free(w.ptr)
		ctx.ResultError(err)
		return // notest
	}
	ctx.c.wrp.Xsqlite3_result_text_go(
		int32(ctx.handle), int32(w.ptr), int64(len(w.buf)))
}

// BindJSON binds the JSON encoding of value to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindJSON(param int, value any) error {
	w := bytesWriter{wrp: s.c.wrp}
	if err := json.MarshalWrite(&w, value); err != nil {
		s.c.wrp.Free(w.ptr)
		return err // notest
	}
	rc := res_t(s.c.wrp.Xsqlite3_bind_text_go(
		int32(s.handle), int32(param),
		int32(w.ptr), int64(len(w.buf))))
	return s.c.error(rc)
}

// ColumnJSON parses the JSON-encoded value of the result column
// and stores it in the value pointed to by ptr.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnJSON(col int, ptr any) error {
	var data []byte
	switch s.ColumnType(col) {
	case NULL:
		data = []byte("null")
	case TEXT:
		data = s.ColumnRawText(col)
	case BLOB:
		data = s.ColumnRawBlob(col)
	case INTEGER:
		data = strconv.AppendInt(nil, s.ColumnInt64(col), 10)
	case FLOAT:
		data = util.AppendNumber(nil, s.ColumnFloat(col))
	default:
		panic(errutil.AssertErr())
	}
	return json.Unmarshal(data, ptr)
}

// JSON parses a JSON-encoded value
// and stores the result in the value pointed to by ptr.
func (v Value) JSON(ptr any) error {
	var data []byte
	switch v.Type() {
	case NULL:
		data = []byte("null")
	case TEXT:
		data = v.RawText()
	case BLOB:
		data = v.RawBlob()
	case INTEGER:
		data = strconv.AppendInt(nil, v.Int64(), 10)
	case FLOAT:
		data = util.AppendNumber(nil, v.Float())
	default:
		panic(errutil.AssertErr())
	}
	return json.Unmarshal(data, ptr)
}

type bytesWriter struct {
	wrp *sqlite3_wrap.Wrapper
	buf []byte
	ptr ptr_t
}

func (b *bytesWriter) Write(p []byte) (n int, err error) {
	if len(p) > cap(b.buf)-len(b.buf)-1 {
		want := int64(len(b.buf)) + int64(len(p)) + 1
		grow := int64(cap(b.buf))
		grow += grow >> 1
		want = max(want, grow)
		b.ptr = b.wrp.Realloc(b.ptr, want)
		b.buf = b.wrp.Bytes(b.ptr, want)[:len(b.buf)]
	}
	b.buf = append(b.buf, p...)
	_ = append(b.buf, 0)
	return len(p), nil
}
