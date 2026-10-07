package errutil

import (
	"runtime"
	"strconv"
)

type ErrorString string

func (e ErrorString) Error() string { return string(e) }

const (
	NilErr       = ErrorString("sqlite3: invalid memory address or null pointer dereference")
	OOMErr       = ErrorString("sqlite3: out of memory")
	NoNulErr     = ErrorString("sqlite3: missing NUL terminator")
	TimeErr      = ErrorString("sqlite3: invalid time value")
	WhenceErr    = ErrorString("sqlite3: invalid whence")
	OffsetErr    = ErrorString("sqlite3: invalid offset")
	TailErr      = ErrorString("sqlite3: multiple statements")
	IsolationErr = ErrorString("sqlite3: unsupported isolation level")
	ValueErr     = ErrorString("sqlite3: unsupported value")
	NoVFSErr     = ErrorString("sqlite3: no such vfs: ")
)

func AssertErr() ErrorString {
	msg := "sqlite3: assertion failed"
	if _, file, line, ok := runtime.Caller(1); ok {
		msg += " (" + file + ":" + strconv.Itoa(line) + ")"
	}
	return ErrorString(msg)
}

type ErrorJoiner []error

func (j *ErrorJoiner) Join(errs ...error) {
	for _, err := range errs {
		if err != nil {
			*j = append(*j, err)
		}
	}
}
