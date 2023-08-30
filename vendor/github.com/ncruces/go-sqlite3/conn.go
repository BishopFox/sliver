package sqlite3

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"net/url"
	"runtime"
	"strings"
	"sync/atomic"
	"unsafe"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Conn is a database connection handle.
// A Conn is not safe for concurrent use by multiple goroutines.
//
// https://www.sqlite.org/c3ref/sqlite3.html
type Conn struct {
	*module

	interrupt context.Context
	waiter    chan struct{}
	pending   *Stmt
	arena     arena

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
//	sqlite3.Open("file:demo.db?_pragma=busy_timeout(10000)&_pragma=locking_mode(normal)")
//
// https://www.sqlite.org/c3ref/open.html
func OpenFlags(filename string, flags OpenFlag) (*Conn, error) {
	if flags&(OPEN_READONLY|OPEN_READWRITE|OPEN_CREATE) == 0 {
		flags |= OPEN_READWRITE | OPEN_CREATE
	}
	return newConn(filename, flags)
}

func newConn(filename string, flags OpenFlag) (conn *Conn, err error) {
	mod, err := instantiateModule()
	if err != nil {
		return nil, err
	}
	defer func() {
		if conn == nil {
			mod.close()
		} else {
			runtime.SetFinalizer(conn, util.Finalizer[Conn](3))
		}
	}()

	c := &Conn{module: mod}
	c.arena = c.newArena(1024)
	c.handle, err = c.openDB(filename, flags)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Conn) openDB(filename string, flags OpenFlag) (uint32, error) {
	defer c.arena.reset()
	connPtr := c.arena.new(ptrlen)
	namePtr := c.arena.string(filename)

	flags |= OPEN_EXRESCODE
	r := c.call(c.api.open, uint64(namePtr), uint64(connPtr), uint64(flags), 0)

	handle := util.ReadUint32(c.mod, connPtr)
	if err := c.module.error(r, handle); err != nil {
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
				pragmas.WriteByte(';')
			}
		}

		c.arena.reset()
		pragmaPtr := c.arena.string(pragmas.String())
		r := c.call(c.api.exec, uint64(handle), uint64(pragmaPtr), 0, 0, 0)
		if err := c.module.error(r, handle, pragmas.String()); err != nil {
			if errors.Is(err, ERROR) {
				err = fmt.Errorf("sqlite3: invalid _pragma: %w", err)
			}
			c.closeDB(handle)
			return 0, err
		}
	}

	return handle, nil
}

func (c *Conn) closeDB(handle uint32) {
	r := c.call(c.api.closeZombie, uint64(handle))
	if err := c.module.error(r, handle); err != nil {
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
// https://www.sqlite.org/c3ref/close.html
func (c *Conn) Close() error {
	if c == nil || c.handle == 0 {
		return nil
	}

	c.SetInterrupt(context.Background())
	c.pending.Close()
	c.pending = nil

	r := c.call(c.api.close, uint64(c.handle))
	if err := c.error(r); err != nil {
		return err
	}

	c.handle = 0
	runtime.SetFinalizer(c, nil)
	return c.module.close()
}

// Exec is a convenience function that allows an application to run
// multiple statements of SQL without having to use a lot of code.
//
// https://www.sqlite.org/c3ref/exec.html
func (c *Conn) Exec(sql string) error {
	c.checkInterrupt()
	defer c.arena.reset()
	sqlPtr := c.arena.string(sql)

	r := c.call(c.api.exec, uint64(c.handle), uint64(sqlPtr), 0, 0, 0)
	return c.error(r)
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
// https://www.sqlite.org/c3ref/prepare.html
func (c *Conn) PrepareFlags(sql string, flags PrepareFlag) (stmt *Stmt, tail string, err error) {
	if emptyStatement(sql) {
		return nil, "", nil
	}

	defer c.arena.reset()
	stmtPtr := c.arena.new(ptrlen)
	tailPtr := c.arena.new(ptrlen)
	sqlPtr := c.arena.string(sql)

	r := c.call(c.api.prepare, uint64(c.handle),
		uint64(sqlPtr), uint64(len(sql)+1), uint64(flags),
		uint64(stmtPtr), uint64(tailPtr))

	stmt = &Stmt{c: c}
	stmt.handle = util.ReadUint32(c.mod, stmtPtr)
	i := util.ReadUint32(c.mod, tailPtr)
	tail = sql[i-sqlPtr:]

	if err := c.error(r, sql); err != nil {
		return nil, "", err
	}
	if stmt.handle == 0 {
		return nil, "", nil
	}
	return
}

// GetAutocommit tests the connection for auto-commit mode.
//
// https://www.sqlite.org/c3ref/get_autocommit.html
func (c *Conn) GetAutocommit() bool {
	r := c.call(c.api.autocommit, uint64(c.handle))
	return r != 0
}

// LastInsertRowID returns the rowid of the most recent successful INSERT
// on the database connection.
//
// https://www.sqlite.org/c3ref/last_insert_rowid.html
func (c *Conn) LastInsertRowID() int64 {
	r := c.call(c.api.lastRowid, uint64(c.handle))
	return int64(r)
}

// Changes returns the number of rows modified, inserted or deleted
// by the most recently completed INSERT, UPDATE or DELETE statement
// on the database connection.
//
// https://www.sqlite.org/c3ref/changes.html
func (c *Conn) Changes() int64 {
	r := c.call(c.api.changes, uint64(c.handle))
	return int64(r)
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
// https://www.sqlite.org/c3ref/interrupt.html
func (c *Conn) SetInterrupt(ctx context.Context) (old context.Context) {
	// Is a waiter running?
	if c.waiter != nil {
		c.waiter <- struct{}{} // Cancel the waiter.
		<-c.waiter             // Wait for it to finish.
		c.waiter = nil
	}
	// Reset the pending statement.
	if c.pending != nil {
		c.pending.Reset()
	}

	old = c.interrupt
	c.interrupt = ctx
	if ctx == nil || ctx.Done() == nil {
		return old
	}

	// Creating an uncompleted SQL statement prevents SQLite from ignoring
	// an interrupt that comes before any other statements are started.
	if c.pending == nil {
		c.pending, _, _ = c.Prepare(`SELECT 1 UNION ALL SELECT 2`)
	}
	c.pending.Step()

	// Don't create the goroutine if we're already interrupted.
	// This happens frequently while restoring to a previously interrupted state.
	if c.checkInterrupt() {
		return old
	}

	waiter := make(chan struct{})
	c.waiter = waiter
	go func() {
		select {
		case <-waiter: // Waiter was cancelled.
			break

		case <-ctx.Done(): // Done was closed.
			const isInterruptedOffset = 280
			buf := util.View(c.mod, c.handle+isInterruptedOffset, 4)
			(*atomic.Uint32)(unsafe.Pointer(&buf[0])).Store(1)
			// Wait for the next call to SetInterrupt.
			<-waiter
		}

		// Signal that the waiter has finished.
		waiter <- struct{}{}
	}()
	return old
}

func (c *Conn) checkInterrupt() bool {
	if c.interrupt == nil || c.interrupt.Err() == nil {
		return false
	}
	const isInterruptedOffset = 280
	buf := util.View(c.mod, c.handle+isInterruptedOffset, 4)
	(*atomic.Uint32)(unsafe.Pointer(&buf[0])).Store(1)
	return true
}

// Pragma executes a PRAGMA statement and returns any results.
//
// https://www.sqlite.org/pragma.html
func (c *Conn) Pragma(str string) ([]string, error) {
	stmt, _, err := c.Prepare(`PRAGMA ` + str)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	var pragmas []string
	for stmt.Step() {
		pragmas = append(pragmas, stmt.ColumnText(0))
	}
	return pragmas, stmt.Close()
}

func (c *Conn) error(rc uint64, sql ...string) error {
	return c.module.error(rc, c.handle, sql...)
}

// DriverConn is implemented by the SQLite [database/sql] driver connection.
//
// It can be used to access advanced SQLite features like
// [savepoints], [online backup] and [incremental BLOB I/O].
//
// [savepoints]: https://www.sqlite.org/lang_savepoint.html
// [online backup]: https://www.sqlite.org/backup.html
// [incremental BLOB I/O]: https://www.sqlite.org/c3ref/blob_open.html
type DriverConn interface {
	driver.Conn
	driver.ConnBeginTx
	driver.ExecerContext
	driver.ConnPrepareContext

	SetInterrupt(ctx context.Context) (old context.Context)

	Savepoint() Savepoint
	Backup(srcDB, dstURI string) error
	Restore(dstDB, srcURI string) error
	OpenBlob(db, table, column string, row int64, write bool) (*Blob, error)
}
