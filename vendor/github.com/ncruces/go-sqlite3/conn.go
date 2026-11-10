package sqlite3

import (
	"context"
	"fmt"
	"iter"
	"math"
	"math/rand"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/internal/sqlite3_wrap"
	"github.com/ncruces/go-sqlite3/vfs"
)

// Conn is a database connection handle.
// A Conn is not safe for concurrent use by multiple goroutines.
//
// https://sqlite.org/c3ref/sqlite3.html
type Conn struct {
	wrp *sqlite3_wrap.Wrapper

	interrupt  context.Context
	stmts      []*Stmt
	busy       func(context.Context, int) bool
	log        func(xErrorCode, string)
	collation  func(*Conn, string)
	wal        func(*Conn, string, int) error
	trace      func(TraceEvent, any, any) error
	authorizer func(AuthorizerActionCode, string, string, string, string) AuthorizerReturnCode
	update     func(AuthorizerActionCode, string, string, int64)
	commit     func() bool
	rollback   func()
	preupdate  func(PreUpdateData)

	busy1st time.Time
	busylst time.Time
	arena   sqlite3_wrap.Arena
	handle  ptr_t
	gosched uint8
}

// Open calls [OpenFlags] with [OPEN_READWRITE], [OPEN_CREATE] and [OPEN_URI].
func Open(filename string) (*Conn, error) {
	return newConn(context.Background(), filename, OPEN_READWRITE|OPEN_CREATE|OPEN_URI)
}

// OpenContext is like [Open] but includes a context,
// which is used to interrupt the process of opening the connection.
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

func newConn(ctx context.Context, filename string, flags OpenFlag) (ret *Conn, _ error) {
	err := ctx.Err()
	if err != nil {
		return nil, err
	}

	c := &Conn{interrupt: ctx}
	c.wrp, err = createWrapper(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if ret == nil {
			c.Close()
		} else {
			c.interrupt = context.Background()
		}
	}()

	c.wrp.DB = c
	if logger := defaultLogger.Load(); logger != nil {
		c.ConfigLog(*logger)
	}
	c.arena = c.wrp.NewArena()
	c.handle, err = c.openDB(filename, flags)
	if err == nil {
		err = initExtensions(c)
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Conn) openDB(filename string, flags OpenFlag) (ptr_t, error) {
	defer c.arena.Mark()()
	connPtr := c.arena.New(ptrlen)
	namePtr := c.arena.String(filename)

	flags |= OPEN_EXRESCODE
	rc := res_t(c.wrp.Xsqlite3_open_v2(int32(namePtr), int32(connPtr), int32(flags), 0))

	handle := ptr_t(c.wrp.Read32(connPtr))
	if err := c.errorFor(handle, rc); err != nil {
		c.closeDB(handle)
		return 0, err
	}

	c.wrp.Xsqlite3_progress_handler_go(int32(handle), 1000)
	if flags|OPEN_URI != 0 && strings.HasPrefix(filename, "file:") {
		var pragmas strings.Builder
		if u, err := url.Parse(filename); err == nil {
			query := u.Query()
			for _, p := range query["_pragma"] {
				pragmas.WriteString(`PRAGMA `)
				pragmas.WriteString(p)
				pragmas.WriteString(`;`)
			}
		}
		if pragmas.Len() != 0 {
			pragmaPtr := c.arena.String(pragmas.String())
			rc := res_t(c.wrp.Xsqlite3_exec(int32(handle), int32(pragmaPtr), 0, 0, 0))
			if err := c.errorFor(handle, rc, pragmas.String()); err != nil {
				err = fmt.Errorf("sqlite3: invalid _pragma: %w", err)
				c.closeDB(handle)
				return 0, err
			}
		}
	}
	return handle, nil
}

func (c *Conn) closeDB(handle ptr_t) {
	rc := res_t(c.wrp.Xsqlite3_close_v2(int32(handle)))
	if err := c.errorFor(handle, rc); err != nil {
		panic(err) // MISUSE
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

	rc := res_t(c.wrp.Xsqlite3_close(int32(c.handle)))
	if err := c.error(rc); err != nil {
		return err
	}

	c.handle = 0
	return c.wrp.Close()
}

// Exec is a convenience function that allows an application to run
// multiple statements of SQL without having to use a lot of code.
//
// https://sqlite.org/c3ref/exec.html
func (c *Conn) Exec(sql string) error {
	if c.interrupt.Err() != nil {
		return INTERRUPT
	}
	return c.exec(sql)
}

func (c *Conn) exec(sql string) error {
	defer c.arena.Mark()()
	textPtr := c.arena.String(sql)
	rc := res_t(c.wrp.Xsqlite3_exec(int32(c.handle), int32(textPtr), 0, 0, 0))
	return c.error(rc, sql)
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
	if c.interrupt.Err() != nil {
		return nil, "", INTERRUPT
	}

	defer c.arena.Mark()()
	stmtPtr := c.arena.New(ptrlen)
	tailPtr := c.arena.New(ptrlen)
	textPtr := c.arena.String(sql)

	rc := res_t(c.wrp.Xsqlite3_prepare_v3(int32(c.handle),
		int32(textPtr), int32(len(sql)+1), int32(flags),
		int32(stmtPtr), int32(tailPtr)))

	stmt = &Stmt{c: c, sql: sql}
	stmt.handle = ptr_t(c.wrp.Read32(stmtPtr))
	if sql := sql[ptr_t(c.wrp.Read32(tailPtr))-textPtr:]; sql != "" {
		tail = sql
	}

	if err := c.error(rc, sql); err != nil {
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
	ptr := ptr_t(c.wrp.Xsqlite3_db_name(int32(c.handle), int32(n)))
	if ptr == 0 {
		return ""
	}
	return c.wrp.ReadString(ptr, _MAX_NAME)
}

// Filename returns the filename for a database.
//
// https://sqlite.org/c3ref/db_filename.html
func (c *Conn) Filename(schema string) *vfs.Filename {
	var ptr ptr_t
	if schema != "" {
		defer c.arena.Mark()()
		ptr = c.arena.String(schema)
	}
	ptr = ptr_t(c.wrp.Xsqlite3_db_filename(int32(c.handle), int32(ptr)))
	return vfs.GetFilename(c.wrp, ptr, vfs.OPEN_MAIN_DB)
}

// ReadOnly determines if a database is read-only.
//
// https://sqlite.org/c3ref/db_readonly.html
func (c *Conn) ReadOnly(schema string) (ro, ok bool) {
	var ptr ptr_t
	if schema != "" {
		defer c.arena.Mark()()
		ptr = c.arena.String(schema)
	}
	b := c.wrp.Xsqlite3_db_readonly(int32(c.handle), int32(ptr))
	return b > 0, b < 0
}

// GetAutocommit tests the connection for auto-commit mode.
//
// https://sqlite.org/c3ref/get_autocommit.html
func (c *Conn) GetAutocommit() bool {
	b := c.wrp.Xsqlite3_get_autocommit(int32(c.handle))
	return b != 0
}

// LastInsertRowID returns the rowid of the most recent successful INSERT
// on the database connection.
//
// https://sqlite.org/c3ref/last_insert_rowid.html
func (c *Conn) LastInsertRowID() int64 {
	return c.wrp.Xsqlite3_last_insert_rowid(int32(c.handle))
}

// SetLastInsertRowID allows the application to set the value returned by
// [Conn.LastInsertRowID].
//
// https://sqlite.org/c3ref/set_last_insert_rowid.html
func (c *Conn) SetLastInsertRowID(id int64) {
	c.wrp.Xsqlite3_set_last_insert_rowid(int32(c.handle), id)
}

// Changes returns the number of rows modified, inserted or deleted
// by the most recently completed INSERT, UPDATE or DELETE statement
// on the database connection.
//
// https://sqlite.org/c3ref/changes.html
func (c *Conn) Changes() int64 {
	return c.wrp.Xsqlite3_changes64(int32(c.handle))
}

// TotalChanges returns the number of rows modified, inserted or deleted
// by all INSERT, UPDATE or DELETE statements completed
// since the database connection was opened.
//
// https://sqlite.org/c3ref/total_changes.html
func (c *Conn) TotalChanges() int64 {
	return c.wrp.Xsqlite3_total_changes64(int32(c.handle))
}

// ReleaseMemory frees memory used by a database connection.
//
// https://sqlite.org/c3ref/db_release_memory.html
func (c *Conn) ReleaseMemory() error {
	rc := res_t(c.wrp.Xsqlite3_db_release_memory(int32(c.handle)))
	return c.error(rc)
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
	if ctx == nil {
		panic("nil Context")
	}
	old = c.interrupt
	c.interrupt = ctx
	return old
}

func (e env) Xgo_progress_handler(_ int32) (interrupt int32) {
	if c, ok := e.DB.(*Conn); ok {
		if c.gosched++; c.gosched%16 == 0 {
			runtime.Gosched()
		}
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
	rc := res_t(c.wrp.Xsqlite3_busy_timeout(int32(c.handle), int32(ms)))
	return c.error(rc)
}

func (e env) Xgo_busy_timeout(count, tmout int32) (retry int32) {
	// https://fractaledmind.github.io/2024/04/15/sqlite-on-rails-the-how-and-why-of-optimal-performance/
	if c, ok := e.DB.(*Conn); ok && c.interrupt.Err() == nil {
		switch {
		case count == 0:
			c.busy1st = time.Now()
		case time.Since(c.busy1st) >= time.Duration(tmout)*time.Millisecond:
			return 0
		}
		if time.Since(c.busylst) < time.Millisecond {
			const sleepIncrement = 2*1024*1024 - 1 // power of two, ~2ms
			time.Sleep(time.Duration(rand.Int63() & sleepIncrement))
		}
		c.busylst = time.Now()
		return 1
	}
	return 0
}

// BusyHandler registers a callback to handle [BUSY] errors.
//
// https://sqlite.org/c3ref/busy_handler.html
func (c *Conn) BusyHandler(cb func(ctx context.Context, count int) (retry bool)) error {
	var enable int32
	if cb != nil {
		enable = 1
	}
	rc := res_t(c.wrp.Xsqlite3_busy_handler_go(int32(c.handle), enable))
	if err := c.error(rc); err != nil {
		return err
	}
	c.busy = cb
	return nil
}

func (e env) Xgo_busy_handler(pDB, count int32) (retry int32) {
	if c, ok := e.DB.(*Conn); ok && c.handle == ptr_t(pDB) && c.busy != nil {
		if interrupt := c.interrupt; interrupt.Err() == nil &&
			c.busy(interrupt, int(count)) {
			retry = 1
		}
	}
	return retry
}

// Status retrieves runtime status information about a database connection.
//
// https://sqlite.org/c3ref/db_status.html
func (c *Conn) Status(op DBStatus, reset bool) (current, highwater int64, err error) {
	defer c.wrp.StackMark()()
	hiPtr := c.wrp.StackAlloc(8)
	curPtr := c.wrp.StackAlloc(8)

	var i int32
	if reset {
		i = 1
	}

	rc := res_t(c.wrp.Xsqlite3_db_status64(int32(c.handle),
		int32(op), int32(curPtr), int32(hiPtr), i))
	if err = c.error(rc); err == nil {
		current = int64(c.wrp.Read64(curPtr))
		highwater = int64(c.wrp.Read64(hiPtr))
	}
	return
}

// TableColumnMetadata extracts metadata about a column of a table.
//
// https://sqlite.org/c3ref/table_column_metadata.html
func (c *Conn) TableColumnMetadata(schema, table, column string) (declType, collSeq string, notNull, primaryKey, autoInc bool, err error) {
	defer c.arena.Mark()()
	var (
		declTypePtr   ptr_t
		collSeqPtr    ptr_t
		notNullPtr    ptr_t
		primaryKeyPtr ptr_t
		autoIncPtr    ptr_t
		columnPtr     ptr_t
		schemaPtr     ptr_t
	)
	if column != "" {
		declTypePtr = c.arena.New(ptrlen)
		collSeqPtr = c.arena.New(ptrlen)
		notNullPtr = c.arena.New(ptrlen)
		primaryKeyPtr = c.arena.New(ptrlen)
		autoIncPtr = c.arena.New(ptrlen)
		columnPtr = c.arena.String(column)
	}
	if schema != "" {
		schemaPtr = c.arena.String(schema)
	}
	tablePtr := c.arena.String(table)

	rc := res_t(c.wrp.Xsqlite3_table_column_metadata(int32(c.handle),
		int32(schemaPtr), int32(tablePtr), int32(columnPtr),
		int32(declTypePtr), int32(collSeqPtr),
		int32(notNullPtr), int32(primaryKeyPtr), int32(autoIncPtr)))
	if err = c.error(rc); err == nil && column != "" {
		if ptr := ptr_t(c.wrp.Read32(declTypePtr)); ptr != 0 {
			declType = c.wrp.ReadString(ptr, _MAX_NAME)
		}
		if ptr := ptr_t(c.wrp.Read32(collSeqPtr)); ptr != 0 {
			collSeq = c.wrp.ReadString(ptr, _MAX_NAME)
		}
		notNull = c.wrp.ReadBool(notNullPtr)
		autoInc = c.wrp.ReadBool(autoIncPtr)
		primaryKey = c.wrp.ReadBool(primaryKeyPtr)
	}
	return
}

func (c *Conn) error(rc res_t, sql ...string) error {
	return c.errorFor(c.handle, rc, sql...)
}

func (c *Conn) errorFor(handle ptr_t, rc res_t, sql ...string) error {
	if rc == _OK {
		return nil
	}

	if ErrorCode(rc) == NOMEM || xErrorCode(rc) == IOERR_NOMEM {
		panic(errutil.OOMErr)
	}

	var msg, query string
	if handle != 0 {
		if ptr := ptr_t(c.wrp.Xsqlite3_errmsg(int32(handle))); ptr != 0 {
			msg = c.wrp.ReadString(ptr, _MAX_LENGTH)
			msg = strings.TrimPrefix(msg, "sqlite3: ")
			msg = strings.TrimPrefix(msg, sqlite3_wrap.ErrorCodeString(rc)[len("sqlite3: "):])
			msg = strings.TrimPrefix(msg, ": ")
			if msg == "" || msg == "not an error" {
				msg = ""
			}
		}

		if len(sql) != 0 {
			if i := c.wrp.Xsqlite3_error_offset(int32(handle)); i != -1 {
				query = sql[0][i:]
			}
		}
	}

	var sys error
	switch ErrorCode(rc) {
	case CANTOPEN, IOERR:
		sys = c.wrp.SysError
	}

	if sys != nil || msg != "" || query != "" {
		return &Error{code: rc, sys: sys, msg: msg, sql: query}
	}
	return xErrorCode(rc)
}

// Stmts returns an iterator for the prepared statements
// associated with the database connection.
//
// https://sqlite.org/c3ref/next_stmt.html
func (c *Conn) Stmts() iter.Seq[*Stmt] {
	return func(yield func(*Stmt) bool) {
		for _, s := range c.stmts {
			if !yield(s) {
				break
			}
		}
	}
}
