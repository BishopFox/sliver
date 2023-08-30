package sqlite3

import (
	"math"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Stmt is a prepared statement object.
//
// https://www.sqlite.org/c3ref/stmt.html
type Stmt struct {
	c      *Conn
	err    error
	handle uint32
}

// Close destroys the prepared statement object.
//
// It is safe to close a nil, zero or closed Stmt.
//
// https://www.sqlite.org/c3ref/finalize.html
func (s *Stmt) Close() error {
	if s == nil || s.handle == 0 {
		return nil
	}

	r := s.c.call(s.c.api.finalize, uint64(s.handle))

	s.handle = 0
	return s.c.error(r)
}

// Reset resets the prepared statement object.
//
// https://www.sqlite.org/c3ref/reset.html
func (s *Stmt) Reset() error {
	r := s.c.call(s.c.api.reset, uint64(s.handle))
	s.err = nil
	return s.c.error(r)
}

// ClearBindings resets all bindings on the prepared statement.
//
// https://www.sqlite.org/c3ref/clear_bindings.html
func (s *Stmt) ClearBindings() error {
	r := s.c.call(s.c.api.clearBindings, uint64(s.handle))
	return s.c.error(r)
}

// Step evaluates the SQL statement.
// If the SQL statement being executed returns any data,
// then true is returned each time a new row of data is ready for processing by the caller.
// The values may be accessed using the Column access functions.
// Step is called again to retrieve the next row of data.
// If an error has occurred, Step returns false;
// call [Stmt.Err] or [Stmt.Reset] to get the error.
//
// https://www.sqlite.org/c3ref/step.html
func (s *Stmt) Step() bool {
	s.c.checkInterrupt()
	r := s.c.call(s.c.api.step, uint64(s.handle))
	if r == _ROW {
		return true
	}
	if r == _DONE {
		s.err = nil
	} else {
		s.err = s.c.error(r)
	}
	return false
}

// Err gets the last error occurred during [Stmt.Step].
// Err returns nil after [Stmt.Reset] is called.
//
// https://www.sqlite.org/c3ref/step.html
func (s *Stmt) Err() error {
	return s.err
}

// Exec is a convenience function that repeatedly calls [Stmt.Step] until it returns false,
// then calls [Stmt.Reset] to reset the statement and get any error that occurred.
func (s *Stmt) Exec() error {
	for s.Step() {
	}
	return s.Reset()
}

// BindCount returns the number of SQL parameters in the prepared statement.
//
// https://www.sqlite.org/c3ref/bind_parameter_count.html
func (s *Stmt) BindCount() int {
	r := s.c.call(s.c.api.bindCount,
		uint64(s.handle))
	return int(r)
}

// BindIndex returns the index of a parameter in the prepared statement
// given its name.
//
// https://www.sqlite.org/c3ref/bind_parameter_index.html
func (s *Stmt) BindIndex(name string) int {
	defer s.c.arena.reset()
	namePtr := s.c.arena.string(name)
	r := s.c.call(s.c.api.bindIndex,
		uint64(s.handle), uint64(namePtr))
	return int(r)
}

// BindName returns the name of a parameter in the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_parameter_name.html
func (s *Stmt) BindName(param int) string {
	r := s.c.call(s.c.api.bindName,
		uint64(s.handle), uint64(param))

	ptr := uint32(r)
	if ptr == 0 {
		return ""
	}
	return util.ReadString(s.c.mod, ptr, _MAX_STRING)
}

// BindBool binds a bool to the prepared statement.
// The leftmost SQL parameter has an index of 1.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are stored as integers 0 (false) and 1 (true).
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindBool(param int, value bool) error {
	if value {
		return s.BindInt64(param, 1)
	}
	return s.BindInt64(param, 0)
}

// BindInt binds an int to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindInt(param int, value int) error {
	return s.BindInt64(param, int64(value))
}

// BindInt64 binds an int64 to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindInt64(param int, value int64) error {
	r := s.c.call(s.c.api.bindInteger,
		uint64(s.handle), uint64(param), uint64(value))
	return s.c.error(r)
}

// BindFloat binds a float64 to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindFloat(param int, value float64) error {
	r := s.c.call(s.c.api.bindFloat,
		uint64(s.handle), uint64(param), math.Float64bits(value))
	return s.c.error(r)
}

// BindText binds a string to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindText(param int, value string) error {
	ptr := s.c.newString(value)
	r := s.c.call(s.c.api.bindText,
		uint64(s.handle), uint64(param),
		uint64(ptr), uint64(len(value)),
		uint64(s.c.api.destructor), _UTF8)
	return s.c.error(r)
}

// BindBlob binds a []byte to the prepared statement.
// The leftmost SQL parameter has an index of 1.
// Binding a nil slice is the same as calling [Stmt.BindNull].
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindBlob(param int, value []byte) error {
	ptr := s.c.newBytes(value)
	r := s.c.call(s.c.api.bindBlob,
		uint64(s.handle), uint64(param),
		uint64(ptr), uint64(len(value)),
		uint64(s.c.api.destructor))
	return s.c.error(r)
}

// BindZeroBlob binds a zero-filled, length n BLOB to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindZeroBlob(param int, n int64) error {
	r := s.c.call(s.c.api.bindZeroBlob,
		uint64(s.handle), uint64(param), uint64(n))
	return s.c.error(r)
}

// BindNull binds a NULL to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindNull(param int) error {
	r := s.c.call(s.c.api.bindNull,
		uint64(s.handle), uint64(param))
	return s.c.error(r)
}

// BindTime binds a [time.Time] to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://www.sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindTime(param int, value time.Time, format TimeFormat) error {
	if format == TimeFormatDefault {
		return s.bindRFC3339Nano(param, value)
	}
	switch v := format.Encode(value).(type) {
	case string:
		s.BindText(param, v)
	case int64:
		s.BindInt64(param, v)
	case float64:
		s.BindFloat(param, v)
	default:
		panic(util.AssertErr())
	}
	return nil
}

func (s *Stmt) bindRFC3339Nano(param int, value time.Time) error {
	const maxlen = uint64(len(time.RFC3339Nano))

	ptr := s.c.new(maxlen)
	buf := util.View(s.c.mod, ptr, maxlen)
	buf = value.AppendFormat(buf[:0], time.RFC3339Nano)

	r := s.c.call(s.c.api.bindText,
		uint64(s.handle), uint64(param),
		uint64(ptr), uint64(len(buf)),
		uint64(s.c.api.destructor), _UTF8)
	return s.c.error(r)
}

// ColumnCount returns the number of columns in a result set.
//
// https://www.sqlite.org/c3ref/column_count.html
func (s *Stmt) ColumnCount() int {
	r := s.c.call(s.c.api.columnCount,
		uint64(s.handle))
	return int(r)
}

// ColumnName returns the name of the result column.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_name.html
func (s *Stmt) ColumnName(col int) string {
	r := s.c.call(s.c.api.columnName,
		uint64(s.handle), uint64(col))

	ptr := uint32(r)
	if ptr == 0 {
		panic(util.OOMErr)
	}
	return util.ReadString(s.c.mod, ptr, _MAX_STRING)
}

// ColumnType returns the initial [Datatype] of the result column.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnType(col int) Datatype {
	r := s.c.call(s.c.api.columnType,
		uint64(s.handle), uint64(col))
	return Datatype(r)
}

// ColumnBool returns the value of the result column as a bool.
// The leftmost column of the result set has the index 0.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are retrieved as integers,
// with 0 converted to false and any other value to true.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnBool(col int) bool {
	if i := s.ColumnInt64(col); i != 0 {
		return true
	}
	return false
}

// ColumnInt returns the value of the result column as an int.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnInt(col int) int {
	return int(s.ColumnInt64(col))
}

// ColumnInt64 returns the value of the result column as an int64.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnInt64(col int) int64 {
	r := s.c.call(s.c.api.columnInteger,
		uint64(s.handle), uint64(col))
	return int64(r)
}

// ColumnFloat returns the value of the result column as a float64.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnFloat(col int) float64 {
	r := s.c.call(s.c.api.columnFloat,
		uint64(s.handle), uint64(col))
	return math.Float64frombits(r)
}

// ColumnTime returns the value of the result column as a [time.Time].
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnTime(col int, format TimeFormat) time.Time {
	var v any
	switch s.ColumnType(col) {
	case INTEGER:
		v = s.ColumnInt64(col)
	case FLOAT:
		v = s.ColumnFloat(col)
	case TEXT, BLOB:
		v = s.ColumnText(col)
	case NULL:
		return time.Time{}
	default:
		panic(util.AssertErr())
	}
	t, err := format.Decode(v)
	if err != nil {
		s.err = err
	}
	return t
}

// ColumnText returns the value of the result column as a string.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnText(col int) string {
	return string(s.ColumnRawText(col))
}

// ColumnBlob appends to buf and returns
// the value of the result column as a []byte.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnBlob(col int, buf []byte) []byte {
	return append(buf, s.ColumnRawBlob(col)...)
}

// ColumnRawText returns the value of the result column as a []byte.
// The []byte is owned by SQLite and may be invalidated by
// subsequent calls to [Stmt] methods.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnRawText(col int) []byte {
	r := s.c.call(s.c.api.columnText,
		uint64(s.handle), uint64(col))

	ptr := uint32(r)
	if ptr == 0 {
		r = s.c.call(s.c.api.errcode, uint64(s.c.handle))
		s.err = s.c.error(r)
		return nil
	}

	r = s.c.call(s.c.api.columnBytes,
		uint64(s.handle), uint64(col))

	return util.View(s.c.mod, ptr, r)
}

// ColumnRawBlob returns the value of the result column as a []byte.
// The []byte is owned by SQLite and may be invalidated by
// subsequent calls to [Stmt] methods.
// The leftmost column of the result set has the index 0.
//
// https://www.sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnRawBlob(col int) []byte {
	r := s.c.call(s.c.api.columnBlob,
		uint64(s.handle), uint64(col))

	ptr := uint32(r)
	if ptr == 0 {
		r = s.c.call(s.c.api.errcode, uint64(s.c.handle))
		s.err = s.c.error(r)
		return nil
	}

	r = s.c.call(s.c.api.columnBytes,
		uint64(s.handle), uint64(col))

	return util.View(s.c.mod, ptr, r)
}

// Return true if stmt is an empty SQL statement.
// This is used as an optimization.
// It's OK to always return false here.
func emptyStatement(stmt string) bool {
	for _, b := range []byte(stmt) {
		switch b {
		case ' ', '\n', '\r', '\t', '\v', '\f':
		case ';':
		default:
			return false
		}
	}
	return true
}
