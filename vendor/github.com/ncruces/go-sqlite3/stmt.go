package sqlite3

import (
	"encoding/json"
	"math"
	"strconv"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Stmt is a prepared statement object.
//
// https://sqlite.org/c3ref/stmt.html
type Stmt struct {
	c      *Conn
	err    error
	sql    string
	handle uint32
}

// Close destroys the prepared statement object.
//
// It is safe to close a nil, zero or closed Stmt.
//
// https://sqlite.org/c3ref/finalize.html
func (s *Stmt) Close() error {
	if s == nil || s.handle == 0 {
		return nil
	}

	r := s.c.call("sqlite3_finalize", uint64(s.handle))
	stmts := s.c.stmts
	for i := range stmts {
		if s == stmts[i] {
			l := len(stmts) - 1
			stmts[i] = stmts[l]
			stmts[l] = nil
			s.c.stmts = stmts[:l]
			break
		}
	}

	s.handle = 0
	return s.c.error(r)
}

// Conn returns the database connection to which the prepared statement belongs.
//
// https://sqlite.org/c3ref/db_handle.html
func (s *Stmt) Conn() *Conn {
	return s.c
}

// SQL returns the SQL text used to create the prepared statement.
//
// https://sqlite.org/c3ref/expanded_sql.html
func (s *Stmt) SQL() string {
	return s.sql
}

// ExpandedSQL returns the SQL text of the prepared statement
// with bound parameters expanded.
//
// https://sqlite.org/c3ref/expanded_sql.html
func (s *Stmt) ExpandedSQL() string {
	r := s.c.call("sqlite3_expanded_sql", uint64(s.handle))
	sql := util.ReadString(s.c.mod, uint32(r), _MAX_SQL_LENGTH)
	s.c.free(uint32(r))
	return sql
}

// ReadOnly returns true if and only if the statement
// makes no direct changes to the content of the database file.
//
// https://sqlite.org/c3ref/stmt_readonly.html
func (s *Stmt) ReadOnly() bool {
	r := s.c.call("sqlite3_stmt_readonly", uint64(s.handle))
	return r != 0
}

// Reset resets the prepared statement object.
//
// https://sqlite.org/c3ref/reset.html
func (s *Stmt) Reset() error {
	r := s.c.call("sqlite3_reset", uint64(s.handle))
	s.err = nil
	return s.c.error(r)
}

// Busy determines if a prepared statement has been reset.
//
// https://sqlite.org/c3ref/stmt_busy.html
func (s *Stmt) Busy() bool {
	r := s.c.call("sqlite3_stmt_busy", uint64(s.handle))
	return r != 0
}

// Step evaluates the SQL statement.
// If the SQL statement being executed returns any data,
// then true is returned each time a new row of data is ready for processing by the caller.
// The values may be accessed using the Column access functions.
// Step is called again to retrieve the next row of data.
// If an error has occurred, Step returns false;
// call [Stmt.Err] or [Stmt.Reset] to get the error.
//
// https://sqlite.org/c3ref/step.html
func (s *Stmt) Step() bool {
	s.c.checkInterrupt(s.c.handle)
	r := s.c.call("sqlite3_step", uint64(s.handle))
	switch r {
	case _ROW:
		s.err = nil
		return true
	case _DONE:
		s.err = nil
	default:
		s.err = s.c.error(r)
	}
	return false
}

// Err gets the last error occurred during [Stmt.Step].
// Err returns nil after [Stmt.Reset] is called.
//
// https://sqlite.org/c3ref/step.html
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

// Status monitors the performance characteristics of prepared statements.
//
// https://sqlite.org/c3ref/stmt_status.html
func (s *Stmt) Status(op StmtStatus, reset bool) int {
	if op > STMTSTATUS_FILTER_HIT && op != STMTSTATUS_MEMUSED {
		return 0
	}
	var i uint64
	if reset {
		i = 1
	}
	r := s.c.call("sqlite3_stmt_status", uint64(s.handle),
		uint64(op), i)
	return int(int32(r))
}

// ClearBindings resets all bindings on the prepared statement.
//
// https://sqlite.org/c3ref/clear_bindings.html
func (s *Stmt) ClearBindings() error {
	r := s.c.call("sqlite3_clear_bindings", uint64(s.handle))
	return s.c.error(r)
}

// BindCount returns the number of SQL parameters in the prepared statement.
//
// https://sqlite.org/c3ref/bind_parameter_count.html
func (s *Stmt) BindCount() int {
	r := s.c.call("sqlite3_bind_parameter_count",
		uint64(s.handle))
	return int(int32(r))
}

// BindIndex returns the index of a parameter in the prepared statement
// given its name.
//
// https://sqlite.org/c3ref/bind_parameter_index.html
func (s *Stmt) BindIndex(name string) int {
	defer s.c.arena.mark()()
	namePtr := s.c.arena.string(name)
	r := s.c.call("sqlite3_bind_parameter_index",
		uint64(s.handle), uint64(namePtr))
	return int(int32(r))
}

// BindName returns the name of a parameter in the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_parameter_name.html
func (s *Stmt) BindName(param int) string {
	r := s.c.call("sqlite3_bind_parameter_name",
		uint64(s.handle), uint64(param))

	ptr := uint32(r)
	if ptr == 0 {
		return ""
	}
	return util.ReadString(s.c.mod, ptr, _MAX_NAME)
}

// BindBool binds a bool to the prepared statement.
// The leftmost SQL parameter has an index of 1.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are stored as integers 0 (false) and 1 (true).
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindBool(param int, value bool) error {
	var i int64
	if value {
		i = 1
	}
	return s.BindInt64(param, i)
}

// BindInt binds an int to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindInt(param int, value int) error {
	return s.BindInt64(param, int64(value))
}

// BindInt64 binds an int64 to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindInt64(param int, value int64) error {
	r := s.c.call("sqlite3_bind_int64",
		uint64(s.handle), uint64(param), uint64(value))
	return s.c.error(r)
}

// BindFloat binds a float64 to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindFloat(param int, value float64) error {
	r := s.c.call("sqlite3_bind_double",
		uint64(s.handle), uint64(param), math.Float64bits(value))
	return s.c.error(r)
}

// BindText binds a string to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindText(param int, value string) error {
	if len(value) > _MAX_LENGTH {
		return TOOBIG
	}
	ptr := s.c.newString(value)
	r := s.c.call("sqlite3_bind_text_go",
		uint64(s.handle), uint64(param),
		uint64(ptr), uint64(len(value)))
	return s.c.error(r)
}

// BindRawText binds a []byte to the prepared statement as text.
// The leftmost SQL parameter has an index of 1.
// Binding a nil slice is the same as calling [Stmt.BindNull].
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindRawText(param int, value []byte) error {
	if len(value) > _MAX_LENGTH {
		return TOOBIG
	}
	ptr := s.c.newBytes(value)
	r := s.c.call("sqlite3_bind_text_go",
		uint64(s.handle), uint64(param),
		uint64(ptr), uint64(len(value)))
	return s.c.error(r)
}

// BindBlob binds a []byte to the prepared statement.
// The leftmost SQL parameter has an index of 1.
// Binding a nil slice is the same as calling [Stmt.BindNull].
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindBlob(param int, value []byte) error {
	if len(value) > _MAX_LENGTH {
		return TOOBIG
	}
	ptr := s.c.newBytes(value)
	r := s.c.call("sqlite3_bind_blob_go",
		uint64(s.handle), uint64(param),
		uint64(ptr), uint64(len(value)))
	return s.c.error(r)
}

// BindZeroBlob binds a zero-filled, length n BLOB to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindZeroBlob(param int, n int64) error {
	r := s.c.call("sqlite3_bind_zeroblob64",
		uint64(s.handle), uint64(param), uint64(n))
	return s.c.error(r)
}

// BindNull binds a NULL to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindNull(param int) error {
	r := s.c.call("sqlite3_bind_null",
		uint64(s.handle), uint64(param))
	return s.c.error(r)
}

// BindTime binds a [time.Time] to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindTime(param int, value time.Time, format TimeFormat) error {
	switch format {
	case TimeFormatDefault, TimeFormatAuto, time.RFC3339Nano:
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
	const maxlen = uint64(len(time.RFC3339Nano)) + 5

	ptr := s.c.new(maxlen)
	buf := util.View(s.c.mod, ptr, maxlen)
	buf = value.AppendFormat(buf[:0], time.RFC3339Nano)

	r := s.c.call("sqlite3_bind_text_go",
		uint64(s.handle), uint64(param),
		uint64(ptr), uint64(len(buf)))
	return s.c.error(r)
}

// BindPointer binds a NULL to the prepared statement, just like [Stmt.BindNull],
// but it also associates ptr with that NULL value such that it can be retrieved
// within an application-defined SQL function using [Value.Pointer].
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindPointer(param int, ptr any) error {
	valPtr := util.AddHandle(s.c.ctx, ptr)
	r := s.c.call("sqlite3_bind_pointer_go",
		uint64(s.handle), uint64(param), uint64(valPtr))
	return s.c.error(r)
}

// BindJSON binds the JSON encoding of value to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindJSON(param int, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return s.BindRawText(param, data)
}

// BindValue binds a copy of value to the prepared statement.
// The leftmost SQL parameter has an index of 1.
//
// https://sqlite.org/c3ref/bind_blob.html
func (s *Stmt) BindValue(param int, value Value) error {
	if value.c != s.c {
		return MISUSE
	}
	r := s.c.call("sqlite3_bind_value",
		uint64(s.handle), uint64(param), uint64(value.handle))
	return s.c.error(r)
}

// DataCount resets the number of columns in a result set.
//
// https://sqlite.org/c3ref/data_count.html
func (s *Stmt) DataCount() int {
	r := s.c.call("sqlite3_data_count",
		uint64(s.handle))
	return int(int32(r))
}

// ColumnCount returns the number of columns in a result set.
//
// https://sqlite.org/c3ref/column_count.html
func (s *Stmt) ColumnCount() int {
	r := s.c.call("sqlite3_column_count",
		uint64(s.handle))
	return int(int32(r))
}

// ColumnName returns the name of the result column.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_name.html
func (s *Stmt) ColumnName(col int) string {
	r := s.c.call("sqlite3_column_name",
		uint64(s.handle), uint64(col))
	if r == 0 {
		panic(util.OOMErr)
	}
	return util.ReadString(s.c.mod, uint32(r), _MAX_NAME)
}

// ColumnType returns the initial [Datatype] of the result column.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnType(col int) Datatype {
	r := s.c.call("sqlite3_column_type",
		uint64(s.handle), uint64(col))
	return Datatype(r)
}

// ColumnDeclType returns the declared datatype of the result column.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_decltype.html
func (s *Stmt) ColumnDeclType(col int) string {
	r := s.c.call("sqlite3_column_decltype",
		uint64(s.handle), uint64(col))
	if r == 0 {
		return ""
	}
	return util.ReadString(s.c.mod, uint32(r), _MAX_NAME)
}

// ColumnDatabaseName returns the name of the database
// that is the origin of a particular result column.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_database_name.html
func (s *Stmt) ColumnDatabaseName(col int) string {
	r := s.c.call("sqlite3_column_database_name",
		uint64(s.handle), uint64(col))
	if r == 0 {
		return ""
	}
	return util.ReadString(s.c.mod, uint32(r), _MAX_NAME)
}

// ColumnTableName returns the name of the table
// that is the origin of a particular result column.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_database_name.html
func (s *Stmt) ColumnTableName(col int) string {
	r := s.c.call("sqlite3_column_table_name",
		uint64(s.handle), uint64(col))
	if r == 0 {
		return ""
	}
	return util.ReadString(s.c.mod, uint32(r), _MAX_NAME)
}

// ColumnOriginName returns the name of the table column
// that is the origin of a particular result column.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_database_name.html
func (s *Stmt) ColumnOriginName(col int) string {
	r := s.c.call("sqlite3_column_origin_name",
		uint64(s.handle), uint64(col))
	if r == 0 {
		return ""
	}
	return util.ReadString(s.c.mod, uint32(r), _MAX_NAME)
}

// ColumnBool returns the value of the result column as a bool.
// The leftmost column of the result set has the index 0.
// SQLite does not have a separate boolean storage class.
// Instead, boolean values are retrieved as numbers,
// with 0 converted to false and any other value to true.
//
// https://sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnBool(col int) bool {
	return s.ColumnFloat(col) != 0
}

// ColumnInt returns the value of the result column as an int.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnInt(col int) int {
	return int(s.ColumnInt64(col))
}

// ColumnInt64 returns the value of the result column as an int64.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnInt64(col int) int64 {
	r := s.c.call("sqlite3_column_int64",
		uint64(s.handle), uint64(col))
	return int64(r)
}

// ColumnFloat returns the value of the result column as a float64.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnFloat(col int) float64 {
	r := s.c.call("sqlite3_column_double",
		uint64(s.handle), uint64(col))
	return math.Float64frombits(r)
}

// ColumnTime returns the value of the result column as a [time.Time].
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_blob.html
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
// https://sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnText(col int) string {
	return string(s.ColumnRawText(col))
}

// ColumnBlob appends to buf and returns
// the value of the result column as a []byte.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnBlob(col int, buf []byte) []byte {
	return append(buf, s.ColumnRawBlob(col)...)
}

// ColumnRawText returns the value of the result column as a []byte.
// The []byte is owned by SQLite and may be invalidated by
// subsequent calls to [Stmt] methods.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnRawText(col int) []byte {
	r := s.c.call("sqlite3_column_text",
		uint64(s.handle), uint64(col))
	return s.columnRawBytes(col, uint32(r))
}

// ColumnRawBlob returns the value of the result column as a []byte.
// The []byte is owned by SQLite and may be invalidated by
// subsequent calls to [Stmt] methods.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnRawBlob(col int) []byte {
	r := s.c.call("sqlite3_column_blob",
		uint64(s.handle), uint64(col))
	return s.columnRawBytes(col, uint32(r))
}

func (s *Stmt) columnRawBytes(col int, ptr uint32) []byte {
	if ptr == 0 {
		r := s.c.call("sqlite3_errcode", uint64(s.c.handle))
		s.err = s.c.error(r)
		return nil
	}

	r := s.c.call("sqlite3_column_bytes",
		uint64(s.handle), uint64(col))
	return util.View(s.c.mod, ptr, r)
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
		data = strconv.AppendFloat(nil, s.ColumnFloat(col), 'g', -1, 64)
	default:
		panic(util.AssertErr())
	}
	return json.Unmarshal(data, ptr)
}

// ColumnValue returns the unprotected value of the result column.
// The leftmost column of the result set has the index 0.
//
// https://sqlite.org/c3ref/column_blob.html
func (s *Stmt) ColumnValue(col int) Value {
	r := s.c.call("sqlite3_column_value",
		uint64(s.handle), uint64(col))
	return Value{
		c:      s.c,
		unprot: true,
		handle: uint32(r),
	}
}

// Columns populates result columns into the provided slice.
// The slice must have [Stmt.ColumnCount] length.
//
// [INTEGER] columns will be retrieved as int64 values,
// [FLOAT] as float64, [NULL] as nil,
// [TEXT] as string, and [BLOB] as []byte.
// Any []byte are owned by SQLite and may be invalidated by
// subsequent calls to [Stmt] methods.
func (s *Stmt) Columns(dest []any) error {
	defer s.c.arena.mark()()
	count := uint64(len(dest))
	typePtr := s.c.arena.new(count)
	dataPtr := s.c.arena.new(count * 8)

	r := s.c.call("sqlite3_columns_go",
		uint64(s.handle), count, uint64(typePtr), uint64(dataPtr))
	if err := s.c.error(r); err != nil {
		return err
	}

	types := util.View(s.c.mod, typePtr, count)

	// Avoid bounds checks on types below.
	if len(types) != len(dest) {
		panic(util.AssertErr())
	}

	for i := range dest {
		switch types[i] {
		case byte(INTEGER):
			dest[i] = int64(util.ReadUint64(s.c.mod, dataPtr))
		case byte(FLOAT):
			dest[i] = util.ReadFloat64(s.c.mod, dataPtr)
		case byte(NULL):
			dest[i] = nil
		default:
			ptr := util.ReadUint32(s.c.mod, dataPtr+0)
			len := util.ReadUint32(s.c.mod, dataPtr+4)
			buf := util.View(s.c.mod, ptr, uint64(len))
			if types[i] == byte(TEXT) {
				dest[i] = string(buf)
			} else {
				dest[i] = buf
			}
		}
		dataPtr += 8
	}
	return nil
}
