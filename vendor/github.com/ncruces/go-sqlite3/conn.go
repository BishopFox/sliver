package sqlite3

import (
	"context"
	"fmt"
	"math"
	"net/url"
	"strings"
	"time"

	"github.com/tetratelabs/wazero/api"

	"github.com/ncruces/go-sqlite3/internal/util"
	"github.com/ncruces/go-sqlite3/vfs"
)

// Conn is a database connection handle.
// A Conn is not safe for concurrent use by multiple goroutines.
//
// https://sqlite.org/c3ref/sqlite3.html
type Conn struct {
	*sqlite

	interrupt  context.Context
	pending    *Stmt
	stmts      []*Stmt
	timer      *time.Timer
	busy       func(context.Context, int) bool
	log        func(xErrorCode, string)
	collation  func(*Conn, string)
	wal        func(*Conn, string, int) error
	trace      func(TraceEvent, any, any) error
	authorizer func(AuthorizerActionCode, string, string, string, string) AuthorizerReturnCode
	update     func(AuthorizerActionCode, string, string, int64)
	commit     func() bool
	rollback   func()
	arena      arena

	handle uint32
}

// Open calls [OpenFlags] with [OPEN_READWRITE], [OPEN_CREATE] and [OPEN_URI].
func Open(filename string) (*Conn, error) {
	return newConn(context.Background(), filename, OPEN_READWRITE|OPEN_CREATE|OPEN_URI)
}

// OpenContext is like [Open] but includes a context,
// which is used to interrupt the process of opening the connectiton.
func OpenContext(ctx context.Context, filename string) (*Conn, error) {
	return newConn(ctx, filename, OPEN_READWRITE|OPEN_CREATE|OPEN_URI)
}

// OpenFlags opens an SQLite database file as specified by the filename argument.
//
// If none of the required flags are used, a combination of [OPEN_READWRITE] and [OPEN_CREATE] is used.
// If a URI filename is used, PRAGMA statements to execute can be specified using "_pragma":
//
//	sqlite3.Open("file:demo.db?_pragma=busy_timeout(10000)")
//
// https://sqlite.org/c3ref/open.html
func OpenFlags(filename string, flags OpenFlag) (*Conn, error) {
	if flags&(OPEN_READONLY|OPEN_READWRITE|OPEN_CREATE) == 0 {
		flags |= OPEN_READWRITE | OPEN_CREATE
	}
	return newConn(context.Background(), filename, flags)
}

type connKey struct{}

func newConn(ctx context.Context, filename string, flags OpenFlag) (res *Conn, _ error) {
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	c := &Conn{interrupt: ctx}
	c.sqlite, err = instantiateSQLite()
	if err != nil {
		return nil, err
	}
	defer func() {
		if res == nil {
			c.Close()
			c.sqlite.close()
		} else {
			c.interrupt = context.Background()
		}
	}()

	c.ctx = context.WithValue(c.ctx, connKey{}, c)
	c.arena = c.newArena(1024)
	c.handle, err = c.openDB(filename, flags)
	if err == nil {
		err = initExtensions(c)
	}
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

	c.call("sqlite3_progress_handler_go", uint64(handle), 100)
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
			c.checkInterrupt(handle)
			pragmaPtr := c.arena.string(pragmas.String())
			r := c.call("sqlite3_exec", uint64(handle), uint64(pragmaPtr), 0, 0, 0)
			if err := c.sqlite.error(r, handle, pragmas.String()); err != nil {
				err = fmt.Errorf("sqlite3: invalid _pragma: %w", err)
				c.closeDB(handle)
				return 0, err
			}
		}
	}
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
	defer c.arena.mark()()
	sqlPtr := c.arena.string(sql)

	c.checkInterrupt(c.handle)
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

	c.checkInterrupt(c.handle)
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
	c.stmts = append(c.stmts, stmt)
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
	return vfs.GetFilename(c.ctx, c.mod, uint32(r), vfs.OPEN_MAIN_DB)
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

// GetInterrupt gets the context set with [Conn.SetInterrupt].
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
	old = c.interrupt
	c.interrupt = ctx

	if ctx == old || ctx.Done() == old.Done() {
		return old
	}

	// A busy SQL statement prevents SQLite from ignoring an interrupt
	// that comes before any other statements are started.
	if c.pending == nil {
		defer c.arena.mark()()
		stmtPtr := c.arena.new(ptrlen)
		loopPtr := c.arena.string(`WITH RECURSIVE c(x) AS (VALUES(0) UNION ALL SELECT x FROM c) SELECT x FROM c`)
		c.call("sqlite3_prepare_v3", uint64(c.handle), uint64(loopPtr), math.MaxUint64,
			uint64(PREPARE_PERSISTENT), uint64(stmtPtr), 0)
		c.pending = &Stmt{c: c}
		c.pending.handle = util.ReadUint32(c.mod, stmtPtr)
	}

	if old.Done() != nil && ctx.Err() == nil {
		c.pending.Reset()
	}
	if ctx.Done() != nil {
		c.pending.Step()
	}
	return old
}

func (c *Conn) checkInterrupt(handle uint32) {
	if c.interrupt.Err() != nil {
		c.call("sqlite3_interrupt", uint64(handle))
	}
}

func progressCallback(ctx context.Context, mod api.Module, _ uint32) (interrupt uint32) {
	if c, ok := ctx.Value(connKey{}).(*Conn); ok && c.interrupt.Err() != nil {
		interrupt = 1
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

func timeoutCallback(ctx context.Context, mod api.Module, count, tmout int32) (retry uint32) {
	if c, ok := ctx.Value(connKey{}).(*Conn); ok && c.interrupt.Err() == nil {
		const delays = "\x01\x02\x05\x0a\x0f\x14\x19\x19\x19\x32\x32\x64"
		const totals = "\x00\x01\x03\x08\x12\x21\x35\x4e\x67\x80\xb2\xe4"
		const ndelay = int32(len(delays) - 1)

		var delay, prior int32
		if count <= ndelay {
			delay = int32(delays[count])
			prior = int32(totals[count])
		} else {
			delay = int32(delays[ndelay])
			prior = int32(totals[ndelay]) + delay*(count-ndelay)
		}

		if delay = min(delay, tmout-prior); delay > 0 {
			delay := time.Duration(delay) * time.Millisecond
			if c.interrupt.Done() == nil {
				time.Sleep(delay)
				return 1
			}
			if c.timer == nil {
				c.timer = time.NewTimer(delay)
			} else {
				c.timer.Reset(delay)
			}
			select {
			case <-c.interrupt.Done():
				c.timer.Stop()
			case <-c.timer.C:
				return 1
			}
		}
	}
	return 0
}

// BusyHandler registers a callback to handle [BUSY] errors.
//
// https://sqlite.org/c3ref/busy_handler.html
func (c *Conn) BusyHandler(cb func(ctx context.Context, count int) (retry bool)) error {
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
		interrupt := c.interrupt
		if interrupt == nil {
			interrupt = context.Background()
		}
		if interrupt.Err() == nil && c.busy(interrupt, int(count)) {
			retry = 1
		}
	}
	return retry
}

// Status retrieves runtime status information about a database connection.
//
// https://sqlite.org/c3ref/db_status.html
func (c *Conn) Status(op DBStatus, reset bool) (current, highwater int, err error) {
	defer c.arena.mark()()
	hiPtr := c.arena.new(intlen)
	curPtr := c.arena.new(intlen)

	var i uint64
	if reset {
		i = 1
	}

	r := c.call("sqlite3_db_status", uint64(c.handle),
		uint64(op), uint64(curPtr), uint64(hiPtr), i)
	if err = c.error(r); err == nil {
		current = int(util.ReadUint32(c.mod, curPtr))
		highwater = int(util.ReadUint32(c.mod, hiPtr))
	}
	return
}

// TableColumnMetadata extracts metadata about a column of a table.
//
// https://sqlite.org/c3ref/table_column_metadata.html
func (c *Conn) TableColumnMetadata(schema, table, column string) (declType, collSeq string, notNull, primaryKey, autoInc bool, err error) {
	defer c.arena.mark()()

	var schemaPtr, columnPtr uint32
	declTypePtr := c.arena.new(ptrlen)
	collSeqPtr := c.arena.new(ptrlen)
	notNullPtr := c.arena.new(ptrlen)
	autoIncPtr := c.arena.new(ptrlen)
	primaryKeyPtr := c.arena.new(ptrlen)
	if schema != "" {
		schemaPtr = c.arena.string(schema)
	}
	tablePtr := c.arena.string(table)
	if column != "" {
		columnPtr = c.arena.string(column)
	}

	r := c.call("sqlite3_table_column_metadata", uint64(c.handle),
		uint64(schemaPtr), uint64(tablePtr), uint64(columnPtr),
		uint64(declTypePtr), uint64(collSeqPtr),
		uint64(notNullPtr), uint64(primaryKeyPtr), uint64(autoIncPtr))
	if err = c.error(r); err == nil && column != "" {
		declType = util.ReadString(c.mod, util.ReadUint32(c.mod, declTypePtr), _MAX_NAME)
		collSeq = util.ReadString(c.mod, util.ReadUint32(c.mod, collSeqPtr), _MAX_NAME)
		notNull = util.ReadUint32(c.mod, notNullPtr) != 0
		autoInc = util.ReadUint32(c.mod, autoIncPtr) != 0
		primaryKey = util.ReadUint32(c.mod, primaryKeyPtr) != 0
	}
	return
}

func (c *Conn) error(rc uint64, sql ...string) error {
	return c.sqlite.error(rc, c.handle, sql...)
}

func (c *Conn) stmtsIter(yield func(*Stmt) bool) {
	for _, s := range c.stmts {
		if !yield(s) {
			break
		}
	}
}
