// Package driver provides a database/sql driver for SQLite.
//
// Importing package driver registers a [database/sql] driver named "sqlite3".
// You may also need to import package embed.
//
//	import _ "github.com/ncruces/go-sqlite3/driver"
//	import _ "github.com/ncruces/go-sqlite3/embed"
//
// The data source name for "sqlite3" databases can be a filename or a "file:" [URI].
//
// The [TRANSACTION] mode can be specified using "_txlock":
//
//	sql.Open("sqlite3", "file:demo.db?_txlock=immediate")
//
// [PRAGMA] statements can be specified using "_pragma":
//
//	sql.Open("sqlite3", "file:demo.db?_pragma=busy_timeout(10000)&_pragma=locking_mode(normal)")
//
// If no PRAGMAs are specified, a busy timeout of 1 minute
// and normal locking mode are used.
//
// Order matters:
// busy timeout and locking mode should be the first PRAGMAs set, in that order.
//
// [URI]: https://www.sqlite.org/uri.html
// [PRAGMA]: https://www.sqlite.org/pragma.html
// [TRANSACTION]: https://www.sqlite.org/lang_transaction.html#deferred_immediate_and_exclusive_transactions
package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

func init() {
	sql.Register("sqlite3", sqlite{})
}

type sqlite struct{}

func (sqlite) Open(name string) (_ driver.Conn, err error) {
	var c conn
	c.Conn, err = sqlite3.Open(name)
	if err != nil {
		return nil, err
	}

	c.txBegin = "BEGIN"
	var pragmas []string
	if strings.HasPrefix(name, "file:") {
		if _, after, ok := strings.Cut(name, "?"); ok {
			query, _ := url.ParseQuery(after)

			switch s := query.Get("_txlock"); s {
			case "":
				c.txBegin = "BEGIN"
			case "deferred", "immediate", "exclusive":
				c.txBegin = "BEGIN " + s
			default:
				c.Close()
				return nil, fmt.Errorf("sqlite3: invalid _txlock: %s", s)
			}

			pragmas = query["_pragma"]
		}
	}
	if len(pragmas) == 0 {
		err := c.Conn.Exec(`
			PRAGMA busy_timeout=60000;
			PRAGMA locking_mode=normal;
		`)
		if err != nil {
			c.Close()
			return nil, err
		}
		c.reusable = true
	} else {
		s, _, err := c.Conn.Prepare(`
			SELECT * FROM
				PRAGMA_locking_mode,
				PRAGMA_query_only;
		`)
		if err != nil {
			c.Close()
			return nil, err
		}
		if s.Step() {
			c.reusable = s.ColumnText(0) == "normal"
			c.readOnly = s.ColumnRawText(1)[0] // 0 or 1
		}
		err = s.Close()
		if err != nil {
			c.Close()
			return nil, err
		}
	}
	return &c, nil
}

type conn struct {
	*sqlite3.Conn
	txBegin    string
	txCommit   string
	txRollback string
	reusable   bool
	readOnly   byte
}

var (
	// Ensure these interfaces are implemented:
	_ driver.ExecerContext = &conn{}
	_ driver.ConnBeginTx   = &conn{}
	_ driver.Validator     = &conn{}
	_ sqlite3.DriverConn   = &conn{}
)

func (c *conn) IsValid() bool {
	return c.reusable
}

func (c *conn) Begin() (driver.Tx, error) {
	return c.BeginTx(context.Background(), driver.TxOptions{})
}

func (c *conn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	txBegin := c.txBegin
	c.txCommit = `COMMIT`
	c.txRollback = `ROLLBACK`

	if opts.ReadOnly {
		txBegin = `
			BEGIN deferred;
			PRAGMA query_only=on`
		c.txCommit = `
			ROLLBACK;
			PRAGMA query_only=` + string(c.readOnly)
		c.txRollback = c.txCommit
	}

	switch opts.Isolation {
	default:
		return nil, util.IsolationErr
	case
		driver.IsolationLevel(sql.LevelDefault),
		driver.IsolationLevel(sql.LevelSerializable):
		break
	}

	old := c.Conn.SetInterrupt(ctx)
	defer c.Conn.SetInterrupt(old)

	err := c.Conn.Exec(txBegin)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *conn) Commit() error {
	err := c.Conn.Exec(c.txCommit)
	if err != nil && !c.GetAutocommit() {
		c.Rollback()
	}
	return err
}

func (c *conn) Rollback() error {
	return c.Conn.Exec(c.txRollback)
}

func (c *conn) Prepare(query string) (driver.Stmt, error) {
	return c.PrepareContext(context.Background(), query)
}

func (c *conn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	old := c.Conn.SetInterrupt(ctx)
	defer c.Conn.SetInterrupt(old)

	s, tail, err := c.Conn.Prepare(query)
	if err != nil {
		return nil, err
	}
	if tail != "" {
		// Check if the tail contains any SQL.
		st, _, err := c.Conn.Prepare(tail)
		if err != nil {
			s.Close()
			return nil, err
		}
		if st != nil {
			s.Close()
			st.Close()
			return nil, util.TailErr
		}
	}
	return &stmt{s, c.Conn}, nil
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 0 {
		// Slow path.
		return nil, driver.ErrSkip
	}

	old := c.Conn.SetInterrupt(ctx)
	defer c.Conn.SetInterrupt(old)

	err := c.Conn.Exec(query)
	if err != nil {
		return nil, err
	}

	return newResult(c.Conn), nil
}

type stmt struct {
	Stmt *sqlite3.Stmt
	Conn *sqlite3.Conn
}

var (
	// Ensure these interfaces are implemented:
	_ driver.StmtExecContext   = &stmt{}
	_ driver.StmtQueryContext  = &stmt{}
	_ driver.NamedValueChecker = &stmt{}
)

func (s *stmt) Close() error {
	return s.Stmt.Close()
}

func (s *stmt) NumInput() int {
	n := s.Stmt.BindCount()
	for i := 1; i <= n; i++ {
		if s.Stmt.BindName(i) != "" {
			return -1
		}
	}
	return n
}

// Deprecated: use ExecContext instead.
func (s *stmt) Exec(args []driver.Value) (driver.Result, error) {
	return s.ExecContext(context.Background(), namedValues(args))
}

// Deprecated: use QueryContext instead.
func (s *stmt) Query(args []driver.Value) (driver.Rows, error) {
	return s.QueryContext(context.Background(), namedValues(args))
}

func (s *stmt) ExecContext(ctx context.Context, args []driver.NamedValue) (driver.Result, error) {
	// Use QueryContext to setup bindings.
	// No need to close rows: that simply resets the statement, exec does the same.
	_, err := s.QueryContext(ctx, args)
	if err != nil {
		return nil, err
	}

	err = s.Stmt.Exec()
	if err != nil {
		return nil, err
	}

	return newResult(s.Conn), nil
}

func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	err := s.Stmt.ClearBindings()
	if err != nil {
		return nil, err
	}

	var ids [3]int
	for _, arg := range args {
		ids := ids[:0]
		if arg.Name == "" {
			ids = append(ids, arg.Ordinal)
		} else {
			for _, prefix := range []string{":", "@", "$"} {
				if id := s.Stmt.BindIndex(prefix + arg.Name); id != 0 {
					ids = append(ids, id)
				}
			}
		}

		for _, id := range ids {
			switch a := arg.Value.(type) {
			case bool:
				err = s.Stmt.BindBool(id, a)
			case int:
				err = s.Stmt.BindInt(id, a)
			case int64:
				err = s.Stmt.BindInt64(id, a)
			case float64:
				err = s.Stmt.BindFloat(id, a)
			case string:
				err = s.Stmt.BindText(id, a)
			case []byte:
				err = s.Stmt.BindBlob(id, a)
			case sqlite3.ZeroBlob:
				err = s.Stmt.BindZeroBlob(id, int64(a))
			case time.Time:
				err = s.Stmt.BindTime(id, a, sqlite3.TimeFormatDefault)
			case nil:
				err = s.Stmt.BindNull(id)
			default:
				panic(util.AssertErr())
			}
		}
		if err != nil {
			return nil, err
		}
	}

	return &rows{ctx, s.Stmt, s.Conn}, nil
}

func (s *stmt) CheckNamedValue(arg *driver.NamedValue) error {
	switch arg.Value.(type) {
	case bool, int, int64, float64, string, []byte,
		sqlite3.ZeroBlob, time.Time, nil:
		return nil
	default:
		return driver.ErrSkip
	}
}

func newResult(c *sqlite3.Conn) driver.Result {
	rows := c.Changes()
	if rows != 0 {
		id := c.LastInsertRowID()
		if id != 0 {
			return result{id, rows}
		}
	}
	return resultRowsAffected(rows)
}

type result struct{ lastInsertId, rowsAffected int64 }

func (r result) LastInsertId() (int64, error) {
	return r.lastInsertId, nil
}

func (r result) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}

type resultRowsAffected int64

func (r resultRowsAffected) LastInsertId() (int64, error) {
	return 0, nil
}

func (r resultRowsAffected) RowsAffected() (int64, error) {
	return int64(r), nil
}

type rows struct {
	ctx  context.Context
	Stmt *sqlite3.Stmt
	Conn *sqlite3.Conn
}

func (r *rows) Close() error {
	return r.Stmt.Reset()
}

func (r *rows) Columns() []string {
	count := r.Stmt.ColumnCount()
	columns := make([]string, count)
	for i := range columns {
		columns[i] = r.Stmt.ColumnName(i)
	}
	return columns
}

func (r *rows) Next(dest []driver.Value) error {
	old := r.Conn.SetInterrupt(r.ctx)
	defer r.Conn.SetInterrupt(old)

	if !r.Stmt.Step() {
		if err := r.Stmt.Err(); err != nil {
			return err
		}
		return io.EOF
	}

	for i := range dest {
		switch r.Stmt.ColumnType(i) {
		case sqlite3.INTEGER:
			dest[i] = r.Stmt.ColumnInt64(i)
		case sqlite3.FLOAT:
			dest[i] = r.Stmt.ColumnFloat(i)
		case sqlite3.BLOB:
			dest[i] = r.Stmt.ColumnRawBlob(i)
		case sqlite3.TEXT:
			dest[i] = stringOrTime(r.Stmt.ColumnRawText(i))
		case sqlite3.NULL:
			if buf, ok := dest[i].([]byte); ok {
				dest[i] = buf[0:0]
			} else {
				dest[i] = nil
			}
		default:
			panic(util.AssertErr())
		}
	}

	return r.Stmt.Err()
}
