package sqlite3

// Pointer returns a pointer to a value
// that can be used as an argument to
// [database/sql.DB.Exec] and similar methods.
//
// https://sqlite.org/bindptr.html
func Pointer[T any](val T) any {
	return pointer[T]{val}
}

type pointer[T any] struct{ val T }

func (p pointer[T]) Pointer() any { return p.val }
