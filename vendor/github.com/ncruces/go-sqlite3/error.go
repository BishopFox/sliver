package sqlite3

import (
	"errors"
	"strconv"
	"strings"

	"github.com/ncruces/go-sqlite3/internal/util"
)

// Error wraps an SQLite Error Code.
//
// https://sqlite.org/c3ref/errcode.html
type Error struct {
	str  string
	msg  string
	sql  string
	code uint64
}

// Code returns the primary error code for this error.
//
// https://sqlite.org/rescode.html
func (e *Error) Code() ErrorCode {
	return ErrorCode(e.code)
}

// ExtendedCode returns the extended error code for this error.
//
// https://sqlite.org/rescode.html
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
		b.WriteString(": ")
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

// As converts this error to an [ErrorCode] or [ExtendedErrorCode].
func (e *Error) As(err any) bool {
	switch c := err.(type) {
	case *ErrorCode:
		*c = e.Code()
		return true
	case *ExtendedErrorCode:
		*c = e.ExtendedCode()
		return true
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

// As converts this error to an [ErrorCode].
func (e ExtendedErrorCode) As(err any) bool {
	c, ok := err.(*ErrorCode)
	if ok {
		*c = ErrorCode(e)
	}
	return ok
}

// Temporary returns true for [BUSY] errors.
func (e ExtendedErrorCode) Temporary() bool {
	return ErrorCode(e) == BUSY
}

// Timeout returns true for [BUSY_TIMEOUT] errors.
func (e ExtendedErrorCode) Timeout() bool {
	return e == BUSY_TIMEOUT
}

func errorCode(err error, def ErrorCode) (msg string, code uint32) {
	switch code := err.(type) {
	case nil:
		return "", _OK
	case ErrorCode:
		return "", uint32(code)
	case xErrorCode:
		return "", uint32(code)
	case *Error:
		return code.msg, uint32(code.code)
	}

	var ecode ErrorCode
	var xcode xErrorCode
	switch {
	case errors.As(err, &xcode):
		code = uint32(xcode)
	case errors.As(err, &ecode):
		code = uint32(ecode)
	default:
		code = uint32(def)
	}
	return err.Error(), code
}
