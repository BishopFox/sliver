//go:build goexperiment.jsonv2

package sqlite3

import (
	"encoding/json/v2"
	"strconv"

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
	w := bytesWriter{sqlite: ctx.c.sqlite}
	if err := json.MarshalWrite(&w, value); err != nil {
		ctx.c.free(w.ptr)
		ctx.ResultError(err)
		return // notest
	}
	ctx.c.call("sqlite3_result_text_go",
		stk_t(ctx.handle), stk_t(w.ptr), stk_t(len(w.buf)))
}

// BindJSON binds the JSON encoding of value to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindJSON(param int, value any) error {
	w := bytesWriter{sqlite: s.c.sqlite}
	if err := json.MarshalWrite(&w, value); err != nil {
		s.c.free(w.ptr)
		return err
	}
	rc := res_t(s.c.call("sqlite3_bind_text_go",
		stk_t(s.handle), stk_t(param),
		stk_t(w.ptr), stk_t(len(w.buf))))
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
		panic(util.AssertErr())
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
		panic(util.AssertErr())
	}
	return json.Unmarshal(data, ptr)
}

type bytesWriter struct {
	*sqlite
	buf []byte
	ptr ptr_t
}

func (b *bytesWriter) Write(p []byte) (n int, err error) {
	if len(p) > cap(b.buf)-len(b.buf) {
		want := int64(len(b.buf)) + int64(len(p))
		grow := int64(cap(b.buf))
		grow += grow >> 1
		want = max(want, grow)
		b.ptr = b.realloc(b.ptr, want)
		b.buf = util.View(b.mod, b.ptr, want)[:len(b.buf)]
	}
	b.buf = append(b.buf, p...)
	return len(p), nil
}
