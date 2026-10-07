package sqlite3

import (
	"math/rand"
	"runtime"
	"strconv"
	"strings"
)

// Txn is an in-progress database transaction.
//
// https://sqlite.org/lang_transaction.html
type Txn struct {
	c *Conn
}

// Begin starts a deferred transaction.
// It panics if a transaction is in-progress.
// For nested transactions, use [Conn.Savepoint].
//
// https://sqlite.org/lang_transaction.html
func (c *Conn) Begin() Txn {
	// BEGIN even if interrupted.
	err := c.exec(`BEGIN DEFERRED`)
	if err != nil {
		panic(err)
	}
	return Txn{c}
}

// BeginConcurrent starts a concurrent transaction.
//
// Experimental: requires a custom build of SQLite.
//
// https://sqlite.org/cgi/src/doc/begin-concurrent/doc/begin_concurrent.md
func (c *Conn) BeginConcurrent() (Txn, error) {
	err := c.Exec(`BEGIN CONCURRENT`)
	if err != nil {
		return Txn{}, err
	}
	return Txn{c}, nil
}

// BeginImmediate starts an immediate transaction.
//
// https://sqlite.org/lang_transaction.html
func (c *Conn) BeginImmediate() (Txn, error) {
	err := c.Exec(`BEGIN IMMEDIATE`)
	if err != nil {
		return Txn{}, err
	}
	return Txn{c}, nil
}

// BeginExclusive starts an exclusive transaction.
//
// https://sqlite.org/lang_transaction.html
func (c *Conn) BeginExclusive() (Txn, error) {
	err := c.Exec(`BEGIN EXCLUSIVE`)
	if err != nil {
		return Txn{}, err
	}
	return Txn{c}, nil
}

// End calls either [Txn.Commit] or [Txn.Rollback]
// depending on whether *error points to a nil or non-nil error.
//
// This is meant to be deferred:
//
//	func doWork(db *sqlite3.Conn) (err error) {
//		tx := db.Begin()
//		defer tx.End(&err)
//
//		// ... do work in the transaction
//	}
//
// https://sqlite.org/lang_transaction.html
func (tx Txn) End(errp *error) {
	recovered := recover()
	if recovered != nil {
		defer panic(recovered)
	}

	if *errp == nil && recovered == nil {
		// Success path.
		if tx.c.GetAutocommit() { // There is nothing to commit.
			return
		}
		*errp = tx.Commit()
		if *errp == nil {
			return
		}
		// Fall through to the error path.
	}

	// Error path.
	if tx.c.GetAutocommit() { // There is nothing to rollback.
		return
	}
	err := tx.Rollback()
	if err != nil {
		panic(err)
	}
}

// Commit commits the transaction.
//
// https://sqlite.org/lang_transaction.html
func (tx Txn) Commit() error {
	return tx.c.Exec(`COMMIT`)
}

// Rollback rolls back the transaction,
// even if the connection has been interrupted.
//
// https://sqlite.org/lang_transaction.html
func (tx Txn) Rollback() error {
	// ROLLBACK even if interrupted.
	return tx.c.exec(`ROLLBACK`)
}

// Savepoint is a marker within a transaction
// that allows for partial rollback.
//
// https://sqlite.org/lang_savepoint.html
type Savepoint struct {
	c    *Conn
	name string
}

// Savepoint establishes a new transaction savepoint.
//
// https://sqlite.org/lang_savepoint.html
func (c *Conn) Savepoint() Savepoint {
	name := callerName()
	if name == "" {
		name = "sqlite3.Savepoint"
	}
	// Names can be reused, but this makes catching bugs more likely.
	name = QuoteIdentifier(name + "_" + strconv.Itoa(int(rand.Int31())))

	err := c.exec(`SAVEPOINT ` + name)
	if err != nil {
		panic(err)
	}
	return Savepoint{c: c, name: name}
}

func callerName() (name string) {
	var pc [8]uintptr
	n := runtime.Callers(3, pc[:])
	if n <= 0 {
		return ""
	}
	frames := runtime.CallersFrames(pc[:n])
	frame, more := frames.Next()
	for more && (strings.HasPrefix(frame.Function, "database/sql.") ||
		strings.HasPrefix(frame.Function, "github.com/ncruces/go-sqlite3/driver.")) {
		frame, more = frames.Next()
	}
	return frame.Function
}

// Release releases the savepoint rolling back any changes
// if *error points to a non-nil error.
//
// This is meant to be deferred:
//
//	func doWork(db *sqlite3.Conn) (err error) {
//		savept := db.Savepoint()
//		defer savept.Release(&err)
//
//		// ... do work in the transaction
//	}
func (s Savepoint) Release(errp *error) {
	recovered := recover()
	if recovered != nil {
		defer panic(recovered)
	}

	if *errp == nil && recovered == nil {
		// Success path.
		if s.c.GetAutocommit() { // There is nothing to commit.
			return
		}
		*errp = s.c.Exec(`RELEASE ` + s.name)
		if *errp == nil {
			return
		}
		// Fall through to the error path.
	}

	// Error path.
	if s.c.GetAutocommit() { // There is nothing to rollback.
		return
	}
	// ROLLBACK and RELEASE even if interrupted.
	err := s.c.exec(`ROLLBACK TO ` + s.name + `; RELEASE ` + s.name)
	if err != nil {
		panic(err)
	}
}

// Rollback rolls the transaction back to the savepoint,
// even if the connection has been interrupted.
// Rollback does not release the savepoint.
//
// https://sqlite.org/lang_transaction.html
func (s Savepoint) Rollback() error {
	// ROLLBACK even if interrupted.
	return s.c.exec(`ROLLBACK TO ` + s.name)
}

// TxnState determines the transaction state of a database.
//
// https://sqlite.org/c3ref/txn_state.html
func (c *Conn) TxnState(schema string) TxnState {
	var ptr ptr_t
	if schema != "" {
		defer c.arena.Mark()()
		ptr = c.arena.String(schema)
	}
	return TxnState(c.wrp.Xsqlite3_txn_state(int32(c.handle), int32(ptr)))
}

// CommitHook registers a callback function to be invoked
// whenever a transaction is committed.
// Return true to allow the commit operation to continue normally.
//
// https://sqlite.org/c3ref/commit_hook.html
func (c *Conn) CommitHook(cb func() (ok bool)) {
	var enable int32
	if cb != nil {
		enable = 1
	}
	c.wrp.Xsqlite3_commit_hook_go(int32(c.handle), enable)
	c.commit = cb
}

// RollbackHook registers a callback function to be invoked
// whenever a transaction is rolled back.
//
// https://sqlite.org/c3ref/commit_hook.html
func (c *Conn) RollbackHook(cb func()) {
	var enable int32
	if cb != nil {
		enable = 1
	}
	c.wrp.Xsqlite3_rollback_hook_go(int32(c.handle), enable)
	c.rollback = cb
}

// UpdateHook registers a callback function to be invoked
// whenever a row is updated, inserted or deleted in a rowid table.
//
// https://sqlite.org/c3ref/update_hook.html
func (c *Conn) UpdateHook(cb func(op AuthorizerActionCode, schema, table string, rowid int64)) {
	var enable int32
	if cb != nil {
		enable = 1
	}
	c.wrp.Xsqlite3_update_hook_go(int32(c.handle), enable)
	c.update = cb
}

// PreUpdateHook registers a callback function that is invoked prior
// to each INSERT, UPDATE, and DELETE operation on a database table.
//
// https://sqlite.org/c3ref/preupdate_blobwrite.html
func (c *Conn) PreUpdateHook(cb func(PreUpdateData)) {
	var enable int32
	if cb != nil {
		enable = 1
	}
	c.wrp.Xsqlite3_preupdate_hook_go(int32(c.handle), enable)
	c.preupdate = cb
}

func (e env) Xgo_commit_hook(pDB int32) (rollback int32) {
	if c, ok := e.DB.(*Conn); ok && c.handle == ptr_t(pDB) && c.commit != nil {
		if !c.commit() {
			rollback = 1
		}
	}
	return rollback
}

func (e env) Xgo_rollback_hook(pDB int32) {
	if c, ok := e.DB.(*Conn); ok && c.handle == ptr_t(pDB) && c.rollback != nil {
		c.rollback()
	}
}

func (e env) Xgo_update_hook(pDB, op, zSchema, zTabName int32, rowid int64) {
	if c, ok := e.DB.(*Conn); ok && c.handle == ptr_t(pDB) && c.update != nil {
		schema := e.ReadString(ptr_t(zSchema), _MAX_NAME)
		table := e.ReadString(ptr_t(zTabName), _MAX_NAME)
		c.update(AuthorizerActionCode(op), schema, table, rowid)
	}
}

func (e env) Xgo_preupdate_hook(_, pDB, op, zSchema, zTabName int32, oldRowID, newRowID int64) {
	if c, ok := e.DB.(*Conn); ok && c.handle == ptr_t(pDB) && c.preupdate != nil {
		c.preupdate(PreUpdateData{
			c:        c,
			Op:       AuthorizerActionCode(op),
			Schema:   e.ReadString(ptr_t(zSchema), _MAX_NAME),
			Table:    e.ReadString(ptr_t(zTabName), _MAX_NAME),
			OldRowID: oldRowID,
			NewRowID: newRowID,
		})
	}
}

// CacheFlush flushes caches to disk mid-transaction.
//
// https://sqlite.org/c3ref/db_cacheflush.html
func (c *Conn) CacheFlush() error {
	rc := res_t(c.wrp.Xsqlite3_db_cacheflush(int32(c.handle)))
	return c.error(rc)
}

// PreUpdateData provides information about a preupdate event.
//
// https://sqlite.org/c3ref/preupdate_blobwrite.html
type PreUpdateData struct {
	c        *Conn
	Op       AuthorizerActionCode
	Schema   string
	Table    string
	OldRowID int64
	NewRowID int64
}

// Conn returns the database connection associated with the preupdate event.
func (pud *PreUpdateData) Conn() *Conn {
	return pud.c
}

// Count returns the number of columns in the row that is being inserted, updated, or deleted.
//
// https://sqlite.org/c3ref/preupdate_blobwrite.html
func (pud *PreUpdateData) Count() int {
	return int(pud.c.wrp.Xsqlite3_preupdate_count(int32(pud.c.handle)))
}

// Depth returns the trigger depth of the insert, update, or delete operation.
//
// https://sqlite.org/c3ref/preupdate_blobwrite.html
func (pud *PreUpdateData) Depth() int {
	return int(pud.c.wrp.Xsqlite3_preupdate_depth(int32(pud.c.handle)))
}

// BlobWrite returns the index of the column being written to using [Blob].
//
// https://sqlite.org/c3ref/preupdate_blobwrite.html
func (pud *PreUpdateData) BlobWrite() int {
	return int(pud.c.wrp.Xsqlite3_preupdate_blobwrite(int32(pud.c.handle)))
}

// Old returns the value of a column of the table row before it is updated.
//
// https://sqlite.org/c3ref/preupdate_blobwrite.html
func (pud *PreUpdateData) Old(column int) (Value, error) {
	return pud.c.columnValue(pud.c.wrp.Xsqlite3_preupdate_old, pud.c.handle, column)
}

// New returns the value of a column of the table row after it is updated.
//
// https://sqlite.org/c3ref/preupdate_blobwrite.html
func (pud *PreUpdateData) New(column int) (Value, error) {
	return pud.c.columnValue(pud.c.wrp.Xsqlite3_preupdate_new, pud.c.handle, column)
}
