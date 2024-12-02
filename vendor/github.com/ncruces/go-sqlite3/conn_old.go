//go:build !go1.23

package sqlite3

// Stmts returns an iterator for the prepared statements
// associated with the database connection.
//
// https://sqlite.org/c3ref/next_stmt.html
func (c *Conn) Stmts() func(func(*Stmt) bool) { return c.stmtsIter }
