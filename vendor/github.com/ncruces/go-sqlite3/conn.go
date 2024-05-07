package sqlite3

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs"
	"github.com/tetratelabs/wazero/api"
)

// Conn is a database connection handle.
// A Conn is not safe for concurrent use by multiple goroutines.
//
// https://sqlite.org/c3ref/sqlite3.html
type Conn struct {
	*sqlite

	interrupt  context.Context
	pending    *Stmt
	busy       func(int) bool
	log        func(xErrorCode, string)
	collation  func(*Conn, string)
	authorizer func(AuthorizerActionCode, string, string, string, string) AuthorizerReturnCode
	update     func(AuthorizerActionCode, string, string, int64)
	commit     func() bool
	rollback   func()
	wal        func(*Conn, string, int) error
	arena      arena

	handle uint32
}

// Open calls [OpenFlags] with [OPEN_READWRITE], [OPEN_CREATE], [OPEN_URI] and [OPEN_NOFOLLOW].
func Open(filename string) (*Conn, error) {
	return newConn(filename, OPEN_READWRITE|OPEN_CREATE|OPEN_URI|OPEN_NOFOLLOW)
}

// OpenFlags opens an SQLite database file as specified by the filename argument.
//
// If none of the required flags is used, a combination of [OPEN_READWRITE] and [OPEN_CREATE] is used.
// If a URI filename is used, PRAGMA statements to execute can be specified using "_pragma":
//
//	sqlite3.Open("file:demo.db?_pragma=busy_timeout(10000)")
//
// https://sqlite.org/c3ref/open.html
func OpenFlags(filename string, flags OpenFlag) (*Conn, error) {
	if flags&(OPEN_READONLY|OPEN_READWRITE|OPEN_CREATE) == 0 {
		flags |= OPEN_READWRITE | OPEN_CREATE
	}
	return newConn(filename, flags)
}

type connKey struct{}

func newConn(filename string, flags OpenFlag) (conn *Conn, err error) {
	sqlite, err := instantiateSQLite()
	if err != nil {
		return nil, err
	}
	defer func() {
		if conn == nil {
			sqlite.close()
		}
	}()

	c := &Conn{sqlite: sqlite}
	c.arena = c.newArena(1024)
	c.ctx = context.WithValue(c.ctx, connKey{}, c)
	c.handle, err = c.openDB(filename, flags)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Conn) openDB(filename string, flags OpenFlag) (uint32, error) {
	defer c.arena.mark()()
	connPtr := c.arena.new(ptrlen)
	namePtr := c.arena.string(filename)

	flags |= OPEN_EXRESCODE
	r := c.call("sqlite3_open_v2", uint64(namePtr), uint64(connPtr), uint64(flags), 0)

	handle := util.ReadUint32(c.mod, connPtr)
	if err := c.sqlite.error(r, handle); err != nil {
		c.closeDB(handle)
		return 0, err
	}

	if flags|OPEN_URI != 0 && strings.HasPrefix(filename, "file:") {
		var pragmas strings.Builder
		if _, after, ok := strings.Cut(filename, "?"); ok {
			query, _ := url.ParseQuery(after)
			for _, p := range query["_pragma"] {
				pragmas.WriteString(`PRAGMA `)
				pragmas.WriteString(p)
				pragmas.WriteString(`;`)
			}
		}
		if pragmas.Len() != 0 {
			pragmaPtr := c.arena.string(pragmas.String())
			r := c.call("sqlite3_exec", uint64(handle), uint64(pragmaPtr), 0, 0, 0)
			if err := c.sqlite.error(r, handle, pragmas.String()); err != nil {
				err = fmt.Errorf("sqlite3: invalid _pragma: %w", err)
				c.closeDB(handle)
				return 0, err
			}
		}
	}
	c.call("sqlite3_progress_handler_go", uint64(handle), 100)
	return handle, nil
}

func (c *Conn) closeDB(handle uint32) {
	r := c.call("sqlite3_close_v2", uint64(handle))
	if err := c.sqlite.error(r, handle); err != nil {
		panic(err)
	}
}

// Close closes the database connection.
//
// If the database connection is associated with unfinalized prepared statements,
// open blob handles, and/or unfinished backup objects,
// Close will leave the database connection open and return [BUSY].
//
// It is safe to close a nil, zero or closed Conn.
//
// https://sqlite.org/c3ref/close.html
func (c *Conn) Close() error {
	if c == nil || c.handle == 0 {
		return nil
	}

	c.pending.Close()
	c.pending = nil

	r := c.call("sqlite3_close", uint64(c.handle))
	if err := c.error(r); err != nil {
		return err
	}

	c.handle = 0
	return c.close()
}

// Exec is a convenience function that allows an application to run
// multiple statements of SQL without having to use a lot of code.
//
// https://sqlite.org/c3ref/exec.html
func (c *Conn) Exec(sql string) error {
	c.checkInterrupt()
	defer c.arena.mark()()
	sqlPtr := c.arena.string(sql)

	r := c.call("sqlite3_exec", uint64(c.handle), uint64(sqlPtr), 0, 0, 0)
	return c.error(r, sql)
}

// Prepare calls [Conn.PrepareFlags] with no flags.
func (c *Conn) Prepare(sql string) (stmt *Stmt, tail string, err error) {
	return c.PrepareFlags(sql, 0)
}

// PrepareFlags compiles the first SQL statement in sql;
// tail is left pointing to what remains uncompiled.
// If the input text contains no SQL (if the input is an empty string or a comment),
// both stmt and err will be nil.
//
// https://sqlite.org/c3ref/prepare.html
func (c *Conn) PrepareFlags(sql string, flags PrepareFlag) (stmt *Stmt, tail string, err error) {
	if len(sql) > _MAX_SQL_LENGTH {
		return nil, "", TOOBIG
	}

	defer c.arena.mark()()
	stmtPtr := c.arena.new(ptrlen)
	tailPtr := c.arena.new(ptrlen)
	sqlPtr := c.arena.string(sql)

	r := c.call("sqlite3_prepare_v3", uint64(c.handle),
		uint64(sqlPtr), uint64(len(sql)+1), uint64(flags),
		uint64(stmtPtr), uint64(tailPtr))

	stmt = &Stmt{c: c}
	stmt.handle = util.ReadUint32(c.mod, stmtPtr)
	if sql := sql[util.ReadUint32(c.mod, tailPtr)-sqlPtr:]; sql != "" {
		tail = sql
	}

	if err := c.error(r, sql); err != nil {
		return nil, "", err
	}
	if stmt.handle == 0 {
		return nil, "", nil
	}
	return stmt, tail, nil
}

// DBName returns the schema name for n-th database on the database connection.
//
// https://sqlite.org/c3ref/db_name.html
func (c *Conn) DBName(n int) string {
	r := c.call("sqlite3_db_name", uint64(c.handle), uint64(n))

	ptr := uint32(r)
	if ptr == 0 {
		return ""
	}
	return util.ReadString(c.mod, ptr, _MAX_NAME)
}

// Filename returns the filename for a database.
//
// https://sqlite.org/c3ref/db_filename.html
func (c *Conn) Filename(schema string) *vfs.Filename {
	var ptr uint32
	if schema != "" {
		defer c.arena.mark()()
		ptr = c.arena.string(schema)
	}

	r := c.call("sqlite3_db_filename", uint64(c.handle), uint64(ptr))
	return vfs.OpenFilename(c.ctx, c.mod, uint32(r), vfs.OPEN_MAIN_DB)
}

// ReadOnly determines if a database is read-only.
//
// https://sqlite.org/c3ref/db_readonly.html
func (c *Conn) ReadOnly(schema string) (ro bool, ok bool) {
	var ptr uint32
	if schema != "" {
		defer c.arena.mark()()
		ptr = c.arena.string(schema)
	}
	r := c.call("sqlite3_db_readonly", uint64(c.handle), uint64(ptr))
	return int32(r) > 0, int32(r) < 0
}

// GetAutocommit tests the connection for auto-commit mode.
//
// https://sqlite.org/c3ref/get_autocommit.html
func (c *Conn) GetAutocommit() bool {
	r := c.call("sqlite3_get_autocommit", uint64(c.handle))
	return r != 0
}

// LastInsertRowID returns the rowid of the most recent successful INSERT
// on the database connection.
//
// https://sqlite.org/c3ref/last_insert_rowid.html
func (c *Conn) LastInsertRowID() int64 {
	r := c.call("sqlite3_last_insert_rowid", uint64(c.handle))
	return int64(r)
}

// SetLastInsertRowID allows the application to set the value returned by
// [Conn.LastInsertRowID].
//
// https://sqlite.org/c3ref/set_last_insert_rowid.html
func (c *Conn) SetLastInsertRowID(id int64) {
	c.call("sqlite3_set_last_insert_rowid", uint64(c.handle), uint64(id))
}

// Changes returns the number of rows modified, inserted or deleted
// by the most recently completed INSERT, UPDATE or DELETE statement
// on the database connection.
//
// https://sqlite.org/c3ref/changes.html
func (c *Conn) Changes() int64 {
	r := c.call("sqlite3_changes64", uint64(c.handle))
	return int64(r)
}

// TotalChanges returns the number of rows modified, inserted or deleted
// by all INSERT, UPDATE or DELETE statements completed
// since the database connection was opened.
//
// https://sqlite.org/c3ref/total_changes.html
func (c *Conn) TotalChanges() int64 {
	r := c.call("sqlite3_total_changes64", uint64(c.handle))
	return int64(r)
}

// ReleaseMemory frees memory used by a database connection.
//
// https://sqlite.org/c3ref/db_release_memory.html
func (c *Conn) ReleaseMemory() error {
	r := c.call("sqlite3_db_release_memory", uint64(c.handle))
	return c.error(r)
}

// GetInterrupt gets the context set with [Conn.SetInterrupt],
// or nil if none was set.
func (c *Conn) GetInterrupt() context.Context {
	return c.interrupt
}

// SetInterrupt interrupts a long-running query when a context is done.
//
// Subsequent uses of the connection will return [INTERRUPT]
// until the context is reset by another call to SetInterrupt.
//
// To associate a timeout with a connection:
//
//	ctx, cancel := context.WithTimeout(context.TODO(), 100*time.Millisecond)
//	conn.SetInterrupt(ctx)
//	defer cancel()
//
// SetInterrupt returns the old context assigned to the connection.
//
// https://sqlite.org/c3ref/interrupt.html
func (c *Conn) SetInterrupt(ctx context.Context) (old context.Context) {
	// Is it the same context?
	if ctx == c.interrupt {
		return ctx
	}

	// A busy SQL statement prevents SQLite from ignoring an interrupt
	// that comes before any other statements are started.
	if c.pending == nil {
		c.pending, _, _ = c.Prepare(`WITH RECURSIVE c(x) AS (VALUES(0) UNION ALL SELECT x FROM c) SELECT x FROM c`)
	}

	old = c.interrupt
	c.interrupt = ctx

	if old != nil && old.Done() != nil && (ctx == nil || ctx.Err() == nil) {
		c.pending.Reset()
	}
	if ctx != nil && ctx.Done() != nil {
		c.pending.Step()
	}
	return old
}

func (c *Conn) checkInterrupt() {
	if c.interrupt != nil && c.interrupt.Err() != nil {
		c.call("sqlite3_interrupt", uint64(c.handle))
	}
}

func progressCallback(ctx context.Context, mod api.Module, pDB uint32) (interrupt uint32) {
	if c, ok := ctx.Value(connKey{}).(*Conn); ok && c.handle == pDB && c.interrupt != nil {
		if c.interrupt.Err() != nil {
			interrupt = 1
		}
	}
	return interrupt
}

// BusyTimeout sets a busy timeout.
//
// https://sqlite.org/c3ref/busy_timeout.html
func (c *Conn) BusyTimeout(timeout time.Duration) error {
	ms := min((timeout+time.Millisecond-1)/time.Millisecond, math.MaxInt32)
	r := c.call("sqlite3_busy_timeout", uint64(c.handle), uint64(ms))
	return c.error(r)
}

// BusyHandler registers a callback to handle [BUSY] errors.
//
// https://sqlite.org/c3ref/busy_handler.html
func (c *Conn) BusyHandler(cb func(count int) (retry bool)) error {
	var enable uint64
	if cb != nil {
		enable = 1
	}
	r := c.call("sqlite3_busy_handler_go", uint64(c.handle), enable)
	if err := c.error(r); err != nil {
		return err
	}
	c.busy = cb
	return nil
}

func busyCallback(ctx context.Context, mod api.Module, pDB uint32, count int32) (retry uint32) {
	if c, ok := ctx.Value(connKey{}).(*Conn); ok && c.handle == pDB && c.busy != nil {
		if c.busy(int(count)) {
			retry = 1
		}
	}
	return retry
}

func (c *Conn) error(rc uint64, sql ...string) error {
	return c.sqlite.error(rc, c.handle, sql...)
}

// DriverConn is implemented by the SQLite [database/sql] driver connection.
//
// It can be used to access SQLite features like [online backup].
//
// [online backup]: https://sqlite.org/backup.html
type DriverConn interface {
	Raw() *Conn
}
