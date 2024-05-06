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
// Possible values are: "deferred", "immediate", "exclusive".
// A [read-only] transaction is always "deferred", regardless of "_txlock".
//
// The time encoding/decoding format can be specified using "_timefmt":
//
//	sql.Open("sqlite3", "file:demo.db?_timefmt=sqlite")
//
// Possible values are: "auto" (the default), "sqlite", "rfc3339";
// "auto" encodes as RFC 3339 and decodes any [format] supported by SQLite;
// "sqlite" encodes as SQLite and decodes any [format] supported by SQLite;
// "rfc3339" encodes and decodes RFC 3339 only.
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
// [format]: https://sqlite.org/lang_datefunc.html#time_values
// [TRANSACTION]: https://sqlite.org/lang_transaction.html#deferred_immediate_and_exclusive_transactions
// [read-only]: https://pkg.go.dev/database/sql#TxOptions
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
	"unsafe"

	"github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/internal/util"
)

// This variable can be replaced with -ldflags:
//
//	go build -ldflags="-X github.com/ncruces/go-sqlite3/driver.driverName=sqlite"
var driverName = "sqlite3"

func init() {
	if driverName != "" {
		sql.Register(driverName, &SQLite{})
	}
}

// Open opens the SQLite database specified by dataSourceName as a [database/sql.DB].
//
// The init function is called by the driver on new connections.
// The [sqlite3.Conn] can be used to execute queries, register functions, etc.
// Any error returned closes the connection and is returned to [database/sql].
func Open(dataSourceName string, init func(*sqlite3.Conn) error) (*sql.DB, error) {
	c, err := (&SQLite{Init: init}).OpenConnector(dataSourceName)
	if err != nil {
		return nil, err
	}
	return sql.OpenDB(c), nil
}

// SQLite implements [database/sql/driver.Driver].
type SQLite struct {
	// Init function is called by the driver on new connections.
	// The [sqlite3.Conn] can be used to execute queries, register functions, etc.
	// Any error returned closes the connection and is returned to [database/sql].
	Init func(*sqlite3.Conn) error
}

// Open implements [database/sql/driver.Driver].
func (d *SQLite) Open(name string) (driver.Conn, error) {
	c, err := d.newConnector(name)
	if err != nil {
		return nil, err
	}
	return c.Connect(context.Background())
}

// OpenConnector implements [database/sql/driver.DriverContext].
func (d *SQLite) OpenConnector(name string) (driver.Connector, error) {
	return d.newConnector(name)
}

func (d *SQLite) newConnector(name string) (*connector, error) {
	c := connector{driver: d, name: name}

	var txlock, timefmt string
	if strings.HasPrefix(name, "file:") {
		if _, after, ok := strings.Cut(name, "?"); ok {
			query, err := url.ParseQuery(after)
			if err != nil {
				return nil, err
			}
			txlock = query.Get("_txlock")
			timefmt = query.Get("_timefmt")
			c.pragmas = query.Has("_pragma")
		}
	}

	switch txlock {
	case "":
		c.txBegin = "BEGIN"
	case "deferred", "immediate", "exclusive":
		c.txBegin = "BEGIN " + txlock
	default:
		return nil, fmt.Errorf("sqlite3: invalid _txlock: %s", txlock)
	}

	switch timefmt {
	case "":
		c.tmRead = sqlite3.TimeFormatAuto
		c.tmWrite = sqlite3.TimeFormatDefault
	case "sqlite":
		c.tmRead = sqlite3.TimeFormatAuto
		c.tmWrite = sqlite3.TimeFormat3
	case "rfc3339":
		c.tmRead = sqlite3.TimeFormatDefault
		c.tmWrite = sqlite3.TimeFormatDefault
	default:
		c.tmRead = sqlite3.TimeFormat(timefmt)
		c.tmWrite = sqlite3.TimeFormat(timefmt)
	}
	return &c, nil
}

type connector struct {
	driver  *SQLite
	name    string
	txBegin string
	tmRead  sqlite3.TimeFormat
	tmWrite sqlite3.TimeFormat
	pragmas bool
}

func (n *connector) Driver() driver.Driver {
	return n.driver
}

func (n *connector) Connect(ctx context.Context) (_ driver.Conn, err error) {
	c := &conn{
		txBegin: n.txBegin,
		tmRead:  n.tmRead,
		tmWrite: n.tmWrite,
	}

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

	if !n.pragmas {
		err = c.Conn.BusyTimeout(60 * time.Second)
		if err != nil {
			return nil, err
		}
	}
	if n.driver.Init != nil {
		err = n.driver.Init(c.Conn)
		if err != nil {
			return nil, err
		}
	}
	if n.pragmas || n.driver.Init != nil {
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
	return c, nil
}

type conn struct {
	*sqlite3.Conn
	txBegin    string
	txCommit   string
	txRollback string
	tmRead     sqlite3.TimeFormat
	tmWrite    sqlite3.TimeFormat
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
		s.Close()
		return nil, util.TailErr
	}
	return &stmt{Stmt: s, tmRead: c.tmRead, tmWrite: c.tmWrite}, nil
}

func (c *conn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if len(args) != 0 {
		// Slow path.
		return nil, driver.ErrSkip
	}

	if savept, ok := ctx.(*saveptCtx); ok {
		// Called from driver.Savepoint.
		savept.Savepoint = c.Conn.Savepoint()
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

func (c *conn) CheckNamedValue(arg *driver.NamedValue) error {
	return nil
}

type stmt struct {
	*sqlite3.Stmt
	tmWrite sqlite3.TimeFormat
	tmRead  sqlite3.TimeFormat
}

var (
	// Ensure these interfaces are implemented:
	_ driver.StmtExecContext   = &stmt{}
	_ driver.StmtQueryContext  = &stmt{}
	_ driver.NamedValueChecker = &stmt{}
)

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

	old := s.Stmt.Conn().SetInterrupt(ctx)
	defer s.Stmt.Conn().SetInterrupt(old)

	err = s.Stmt.Exec()
	if err != nil {
		return nil, err
	}

	return newResult(s.Stmt.Conn()), nil
}

func (s *stmt) QueryContext(ctx context.Context, args []driver.NamedValue) (driver.Rows, error) {
	err := s.setupBindings(args)
	if err != nil {
		return nil, err
	}
	return &rows{ctx: ctx, stmt: s}, nil
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
				err = s.Stmt.BindTime(id, a, s.tmWrite)
			case util.JSON:
				err = s.Stmt.BindJSON(id, a.Value)
			case util.PointerUnwrap:
				err = s.Stmt.BindPointer(id, util.UnwrapPointer(a))
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
		time.Time, sqlite3.ZeroBlob,
		util.JSON, util.PointerUnwrap,
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
	ctx context.Context
	*stmt
	names []string
	types []string
}

func (r *rows) Close() error {
	r.Stmt.ClearBindings()
	return r.Stmt.Reset()
}

func (r *rows) Columns() []string {
	if r.names == nil {
		count := r.Stmt.ColumnCount()
		r.names = make([]string, count)
		for i := range r.names {
			r.names[i] = r.Stmt.ColumnName(i)
		}
	}
	return r.names
}

func (r *rows) declType(index int) string {
	if r.types == nil {
		count := r.Stmt.ColumnCount()
		r.types = make([]string, count)
		for i := range r.types {
			r.types[i] = strings.ToUpper(r.Stmt.ColumnDeclType(i))
		}
	}
	return r.types[index]
}

func (r *rows) ColumnTypeDatabaseTypeName(index int) string {
	decltype := r.declType(index)
	if len := len(decltype); len > 0 && decltype[len-1] == ')' {
		if i := strings.LastIndexByte(decltype, '('); i >= 0 {
			decltype = decltype[:i]
		}
	}
	return strings.TrimSpace(decltype)
}

func (r *rows) Next(dest []driver.Value) error {
	old := r.Stmt.Conn().SetInterrupt(r.ctx)
	defer r.Stmt.Conn().SetInterrupt(old)

	if !r.Stmt.Step() {
		if err := r.Stmt.Err(); err != nil {
			return err
		}
		return io.EOF
	}

	data := unsafe.Slice((*any)(unsafe.SliceData(dest)), len(dest))
	err := r.Stmt.Columns(data)
	for i := range dest {
		if t, ok := r.decodeTime(i, dest[i]); ok {
			dest[i] = t
			continue
		}
		if s, ok := dest[i].(string); ok {
			t, ok := maybeTime(s)
			if ok {
				dest[i] = t
			}
		}
	}
	return err
}

func (r *rows) decodeTime(i int, v any) (_ time.Time, _ bool) {
	if r.tmRead == sqlite3.TimeFormatDefault {
		return
	}
	switch r.declType(i) {
	case "DATE", "TIME", "DATETIME", "TIMESTAMP":
		// maybe
	default:
		return
	}
	switch v.(type) {
	case int64, float64, string:
		// maybe
	default:
		return
	}
	t, err := r.tmRead.Decode(v)
	return t, err == nil
}
