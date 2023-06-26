package sqlite3

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
)

// Tx is an in-progress database transaction.
//
// https://www.sqlite.org/lang_transaction.html
type Tx struct {
	c *Conn
}

// Begin starts a deferred transaction.
//
// https://www.sqlite.org/lang_transaction.html
func (c *Conn) Begin() Tx {
	// BEGIN even if interrupted.
	err := c.txExecInterrupted(`BEGIN DEFERRED`)
	if err != nil {
		panic(err)
	}
	return Tx{c}
}

// BeginImmediate starts an immediate transaction.
//
// https://www.sqlite.org/lang_transaction.html
func (c *Conn) BeginImmediate() (Tx, error) {
	err := c.Exec(`BEGIN IMMEDIATE`)
	if err != nil {
		return Tx{}, err
	}
	return Tx{c}, nil
}

// BeginExclusive starts an exclusive transaction.
//
// https://www.sqlite.org/lang_transaction.html
func (c *Conn) BeginExclusive() (Tx, error) {
	err := c.Exec(`BEGIN EXCLUSIVE`)
	if err != nil {
		return Tx{}, err
	}
	return Tx{c}, nil
}

// End calls either [Tx.Commit] or [Tx.Rollback]
// depending on whether *error points to a nil or non-nil error.
//
// This is meant to be deferred:
//
//	func doWork(conn *sqlite3.Conn) (err error) {
//		tx := conn.Begin()
//		defer tx.End(&err)
//
//		// ... do work in the transaction
//	}
//
// https://www.sqlite.org/lang_transaction.html
func (tx Tx) End(errp *error) {
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
// https://www.sqlite.org/lang_transaction.html
func (tx Tx) Commit() error {
	return tx.c.Exec(`COMMIT`)
}

// Rollback rolls back the transaction,
// even if the connection has been interrupted.
//
// https://www.sqlite.org/lang_transaction.html
func (tx Tx) Rollback() error {
	return tx.c.txExecInterrupted(`ROLLBACK`)
}

// Savepoint is a marker within a transaction
// that allows for partial rollback.
//
// https://www.sqlite.org/lang_savepoint.html
type Savepoint struct {
	c    *Conn
	name string
}

// Savepoint establishes a new transaction savepoint.
//
// https://www.sqlite.org/lang_savepoint.html
func (c *Conn) Savepoint() Savepoint {
	name := "sqlite3.Savepoint"
	var pc [1]uintptr
	if n := runtime.Callers(2, pc[:]); n > 0 {
		frames := runtime.CallersFrames(pc[:n])
		frame, _ := frames.Next()
		if frame.Function != "" {
			name = frame.Function
		}
	}
	// Names can be reused; this makes catching bugs more likely.
	name += "#" + strconv.Itoa(int(rand.Int31()))

	err := c.txExecInterrupted(fmt.Sprintf("SAVEPOINT %q;", name))
	if err != nil {
		panic(err)
	}
	return Savepoint{c: c, name: name}
}

// Release releases the savepoint rolling back any changes
// if *error points to a non-nil error.
//
// This is meant to be deferred:
//
//	func doWork(conn *sqlite3.Conn) (err error) {
//		savept := conn.Savepoint()
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
		*errp = s.c.Exec(fmt.Sprintf("RELEASE %q;", s.name))
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
	err := s.c.txExecInterrupted(fmt.Sprintf(`
		ROLLBACK TO %[1]q;
		RELEASE %[1]q;
	`, s.name))
	if err != nil {
		panic(err)
	}
}

// Rollback rolls the transaction back to the savepoint,
// even if the connection has been interrupted.
// Rollback does not release the savepoint.
//
// https://www.sqlite.org/lang_transaction.html
func (s Savepoint) Rollback() error {
	// ROLLBACK even if interrupted.
	return s.c.txExecInterrupted(fmt.Sprintf("ROLLBACK TO %q;", s.name))
}

func (c *Conn) txExecInterrupted(sql string) error {
	err := c.Exec(sql)
	if errors.Is(err, INTERRUPT) {
		old := c.SetInterrupt(context.Background())
		defer c.SetInterrupt(old)
		err = c.Exec(sql)
	}
	return err
}
