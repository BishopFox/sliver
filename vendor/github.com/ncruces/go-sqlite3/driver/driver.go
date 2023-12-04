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
//	sql.Open("sqlite3", "file:demo.db?_pragma=busy_timeout(10000)")
//
// If no PRAGMAs are specified, a busy timeout of 1 minute is set.
//
// Order matters:
// busy timeout and locking mode should be the first PRAGMAs set, in that order.
//
// [URI]: https://sqlite.org/uri.html
// [PRAGMA]: https://sqlite.org/pragma.html
// [TRANSACTION]: https://sqlite.org/lang_transaction.html#deferred_immediate_and_exclusive_transactions
package driver

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strings"
	"time"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// This variable can be replaced with -ldflags:
//
//	go build -ldflags="-X github.com/ncruces/go-sqlite3.driverName=sqlite"
var driverName = "sqlite3"

func init() {
	if driverName != "" {
		sql.Register(driverName, sqlite{})
	}
}

// Open opens the SQLite database specified by dataSourceName as a [database/sql.DB].
//
// The init function is called by the driver on new connections.
// The conn can be used to execute queries, register functions, etc.
// Any error return closes the conn and passes the error to [database/sql].
func Open(dataSourceName string, init func(*sqlite3.Conn) error) (*sql.DB, error) {
	c, err := newConnector(dataSourceName, init)
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(c), nil
}

type sqlite struct{}

func (sqlite) Open(name string) (driver.Conn, error) {
	c, err := newConnector(name, nil)
	if err != nil {
		return nil, err
	}
	return c.Connect(context.Background())
}

func (sqlite) OpenConnector(name string) (driver.Connector, error) {
	return newConnector(name, nil)
}

func newConnector(name string, init func(*sqlite3.Conn) error) (*connector, error) {
	c := connector{name: name, init: init}
	if strings.HasPrefix(name, "file:") {
		if _, after, ok := strings.Cut(name, "?"); ok {
			query, err := url.ParseQuery(after)
			if err != nil {
				return nil, err
			}
			c.txlock = query.Get("_txlock")
			c.pragmas = len(query["_pragma"]) > 0
		}
	}
	return &c, nil
}

type connector struct {
	init    func(*sqlite3.Conn) error
	name    string
	txlock  string
	pragmas bool
}

func (n *connector) Driver() driver.Driver {
	return sqlite{}
}

func (n *connector) Connect(ctx context.Context) (_ driver.Conn, err error) {
	var c conn
	c.Conn, err = sqlite3.Open(n.name)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			c.Close()
		}
	}()

	old := c.Conn.SetInterrupt(ctx)
	defer c.Conn.SetInterrupt(old)

	switch n.txlock {
	case "":
		c.txBegin = "BEGIN"
	case "deferred", "immediate", "exclusive":
		c.txBegin = "BEGIN " + n.txlock
	default:
		return nil, fmt.Errorf("sqlite3: invalid _txlock: %s", n.txlock)
	}
	if !n.pragmas {
		err = c.Conn.Exec(`PRAGMA busy_timeout=60000`)
		if err != nil {
			return nil, err
		}
	}
	if n.init != nil {
		err = n.init(c.Conn)
		if err != nil {
			return nil, err
		}
	}
	if n.pragmas || n.init != nil {
		s, _, err := c.Conn.Prepare(`PRAGMA query_only`)
		if err != nil {
			return nil, err
		}
		if s.Step() && s.ColumnBool(0) {
			c.readOnly = '1'
		} else {
			c.readOnly = '0'
		}
		err = s.Close()
		if err != nil {
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
	readOnly   byte
}

var (
	// Ensure these interfaces are implemented:
	_ driver.ConnPrepareContext = &conn{}
	_ driver.ExecerContext      = &conn{}
	_ driver.ConnBeginTx        = &conn{}
	_ sqlite3.DriverConn        = &conn{}
)

func (c *conn) Raw() *sqlite3.Conn {
	return c.Conn
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
		c.txRollback = `
			ROLLBACK;
			PRAGMA query_only=` + string(c.readOnly)
		c.txCommit = c.txRollback
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
	if err != nil && !c.Conn.GetAutocommit() {
		c.Rollback()
	}
	return err
}

func (c *conn) Rollback() error {
	err := c.Conn.Exec(c.txRollback)
	if errors.Is(err, sqlite3.INTERRUPT) {
		old := c.Conn.SetInterrupt(context.Background())
		defer c.Conn.SetInterrupt(old)
		err = c.Conn.Exec(c.txRollback)
	}
	return err
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

	if savept, ok := ctx.(*saveptCtx); ok {
		// Called from driver.Savepoint.
		savept.Savepoint = c.Savepoint()
		return resultRowsAffected(0), nil
	}

	old := c.Conn.SetInterrupt(ctx)
	defer c.Conn.SetInterrupt(old)

	err := c.Conn.Exec(query)
	if err != nil {
		return nil, err
	}

	return newResult(c.Conn), nil
}

func (*conn) CheckNamedValue(arg *driver.NamedValue) error {
	return nil
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
	err := s.setupBindings(args)
	if err != nil {
		return nil, err
	}

	old := s.Conn.SetInterrupt(ctx)
	defer s.Conn.SetInterrupt(old)

	err = s.Stmt.Exec()
	if err != nil {
		return nil, err
	}

	return newResult(s.Conn), nil
}

func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	err := s.setupBindings(args)
	if err != nil {
		return nil, err
	}
	return &rows{ctx, s.Stmt, s.Conn}, nil
}

func (s *stmt) setupBindings(args []driver.NamedValue) error {
	err := s.Stmt.ClearBindings()
	if err != nil {
		return err
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
			case interface{ Pointer() any }:
				err = s.Stmt.BindPointer(id, a.Pointer())
			case interface{ JSON() any }:
				err = s.Stmt.BindJSON(id, a.JSON())
			case nil:
				err = s.Stmt.BindNull(id)
			default:
				panic(util.AssertErr())
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *stmt) CheckNamedValue(arg *driver.NamedValue) error {
	switch arg.Value.(type) {
	case bool, int, int64, float64, string, []byte,
		sqlite3.ZeroBlob, time.Time,
		interface{ Pointer() any },
		interface{ JSON() any },
		nil:
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
			dest[i] = nil
		default:
			panic(util.AssertErr())
		}
	}

	return r.Stmt.Err()
}
