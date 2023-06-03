package sqlite3

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

// Error wraps an SQLite Error Code.
//
// https://www.sqlite.org/c3ref/errcode.html
type Error struct {
	code uint64
	str  string
	msg  string
	sql  string
}

// Code returns the primary error code for this error.
//
// https://www.sqlite.org/rescode.html
func (e *Error) Code() ErrorCode {
	return ErrorCode(e.code)
}

// ExtendedCode returns the extended error code for this error.
//
// https://www.sqlite.org/rescode.html
func (e *Error) ExtendedCode() ExtendedErrorCode {
	return ExtendedErrorCode(e.code)
}

// Error implements the error interface.
func (e *Error) Error() string {
	var b strings.Builder
	b.WriteString("sqlite3: ")

	if e.str != "" {
		b.WriteString(e.str)
	} else {
		b.WriteString(strconv.Itoa(int(e.code)))
	}

	if e.msg != "" {
		b.WriteByte(':')
		b.WriteByte(' ')
		b.WriteString(e.msg)
	}

	return b.String()
}

// Is tests whether this error matches a given [ErrorCode] or [ExtendedErrorCode].
//
// It makes it possible to do:
//
//	if errors.Is(err, sqlite3.BUSY) {
//		// ... handle BUSY
//	}
func (e *Error) Is(err error) bool {
	switch c := err.(type) {
	case ErrorCode:
		return c == e.Code()
	case ExtendedErrorCode:
		return c == e.ExtendedCode()
	}
	return false
}

// Temporary returns true for [BUSY] errors.
func (e *Error) Temporary() bool {
	return e.Code() == BUSY
}

// Timeout returns true for [BUSY_TIMEOUT] errors.
func (e *Error) Timeout() bool {
	return e.ExtendedCode() == BUSY_TIMEOUT
}

// SQL returns the SQL starting at the token that triggered a syntax error.
func (e *Error) SQL() string {
	return e.sql
}

// Error implements the error interface.
func (e ErrorCode) Error() string {
	switch e {
	case _OK:
		return "sqlite3: not an error"
	case _ROW:
		return "sqlite3: another row available"
	case _DONE:
		return "sqlite3: no more rows available"

	case ERROR:
		return "sqlite3: SQL logic error"
	case INTERNAL:
		break
	case PERM:
		return "sqlite3: access permission denied"
	case ABORT:
		return "sqlite3: query aborted"
	case BUSY:
		return "sqlite3: database is locked"
	case LOCKED:
		return "sqlite3: database table is locked"
	case NOMEM:
		return "sqlite3: out of memory"
	case READONLY:
		return "sqlite3: attempt to write a readonly database"
	case INTERRUPT:
		return "sqlite3: interrupted"
	case IOERR:
		return "sqlite3: disk I/O error"
	case CORRUPT:
		return "sqlite3: database disk image is malformed"
	case NOTFOUND:
		return "sqlite3: unknown operation"
	case FULL:
		return "sqlite3: database or disk is full"
	case CANTOPEN:
		return "sqlite3: unable to open database file"
	case PROTOCOL:
		return "sqlite3: locking protocol"
	case FORMAT:
		break
	case SCHEMA:
		return "sqlite3: database schema has changed"
	case TOOBIG:
		return "sqlite3: string or blob too big"
	case CONSTRAINT:
		return "sqlite3: constraint failed"
	case MISMATCH:
		return "sqlite3: datatype mismatch"
	case MISUSE:
		return "sqlite3: bad parameter or other API misuse"
	case NOLFS:
		break
	case AUTH:
		return "sqlite3: authorization denied"
	case EMPTY:
		break
	case RANGE:
		return "sqlite3: column index out of range"
	case NOTADB:
		return "sqlite3: file is not a database"
	case NOTICE:
		return "sqlite3: notification message"
	case WARNING:
		return "sqlite3: warning message"
	}
	return "sqlite3: unknown error"
}

// Temporary returns true for [BUSY] errors.
func (e ErrorCode) Temporary() bool {
	return e == BUSY
}

// Error implements the error interface.
func (e ExtendedErrorCode) Error() string {
	switch x := ErrorCode(e); {
	case e == ABORT_ROLLBACK:
		return "sqlite3: abort due to ROLLBACK"
	case x < _ROW:
		return x.Error()
	case e == _ROW:
		return "sqlite3: another row available"
	case e == _DONE:
		return "sqlite3: no more rows available"
	}
	return "sqlite3: unknown error"
}

// Is tests whether this error matches a given [ErrorCode].
func (e ExtendedErrorCode) Is(err error) bool {
	c, ok := err.(ErrorCode)
	return ok && c == ErrorCode(e)
}

// Temporary returns true for [BUSY] errors.
func (e ExtendedErrorCode) Temporary() bool {
	return ErrorCode(e) == BUSY
}

// Timeout returns true for [BUSY_TIMEOUT] errors.
func (e ExtendedErrorCode) Timeout() bool {
	return e == BUSY_TIMEOUT
}

type errorString string

func (e errorString) Error() string { return string(e) }

const (
	nilErr      = errorString("sqlite3: invalid memory address or null pointer dereference")
	oomErr      = errorString("sqlite3: out of memory")
	rangeErr    = errorString("sqlite3: index out of range")
	noNulErr    = errorString("sqlite3: missing NUL terminator")
	noGlobalErr = errorString("sqlite3: could not find global: ")
	noFuncErr   = errorString("sqlite3: could not find function: ")
	binaryErr   = errorString("sqlite3: no SQLite binary embed/set/loaded")
	timeErr     = errorString("sqlite3: invalid time value")
	whenceErr   = errorString("sqlite3: invalid whence")
	offsetErr   = errorString("sqlite3: invalid offset")
)

func assertErr() errorString {
	msg := "sqlite3: assertion failed"
	if _, file, line, ok := runtime.Caller(1); ok {
		msg += " (" + file + ":" + strconv.Itoa(line) + ")"
	}
	return errorString(msg)
}

func finalizer[T any](skip int) func(*T) {
	msg := fmt.Sprintf("sqlite3: %T not closed", new(T))
	if _, file, line, ok := runtime.Caller(skip + 1); ok && skip >= 0 {
		msg += " (" + file + ":" + strconv.Itoa(line) + ")"
	}
	return func(*T) { panic(errorString(msg)) }
}
