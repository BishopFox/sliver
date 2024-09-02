//go:build (go1.23 || goexperiment.rangefunc) && !vet

package sqlite3

import "iter"

// Stmts returns an iterator for the prepared statements
// associated with the database connection.
//
// https://sqlite.org/c3ref/next_stmt.html
func (c *Conn) Stmts() iter.Seq[*Stmt] { return c.stmtsIter }
