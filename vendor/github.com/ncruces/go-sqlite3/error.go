package sqlite3

import (
	"strconv"
	"strings"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Error wraps an SQLite Error Code.
//
// https://www.sqlite.org/c3ref/errcode.html
type Error struct {
	str  string
	msg  string
	sql  string
	code uint64
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
	return util.ErrorCodeString(uint32(e))
}

// Temporary returns true for [BUSY] errors.
func (e ErrorCode) Temporary() bool {
	return e == BUSY
}

// Error implements the error interface.
func (e ExtendedErrorCode) Error() string {
	return util.ErrorCodeString(uint32(e))
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
