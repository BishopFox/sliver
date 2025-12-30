//go:build !goexperiment.jsonv2

package sqlite3

import (
	"encoding/json"
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
	err := json.NewEncoder(callbackWriter(func(p []byte) (int, error) {
		ctx.ResultRawText(p[:len(p)-1]) // remove the newline
		return 0, nil
	})).Encode(value)

	if err != nil {
		ctx.ResultError(err)
		return // notest
	}
}

// BindJSON binds the JSON encoding of value to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindJSON(param int, value any) error {
	return json.NewEncoder(callbackWriter(func(p []byte) (int, error) {
		return 0, s.BindRawText(param, p[:len(p)-1]) // remove the newline
	})).Encode(value)
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

type callbackWriter func(p []byte) (int, error)

func (fn callbackWriter) Write(p []byte) (int, error) { return fn(p) }
