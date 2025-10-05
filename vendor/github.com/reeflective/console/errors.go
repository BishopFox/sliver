package console

import (
	"fmt"
	"os"
)

type (
	// ErrorHandler is a function that handles errors.
	//
	// The handler can choose not to bubble up the error by returning nil.
	ErrorHandler func(err error) error

	// Err is the Console base error type.
	//
	// All errors that bubble up to the error handler should be
	// wrapped in this error type.
	//
	// There are more concrete error types that wrap this one defined below
	// this allow for easy use of errors.As.
	Err struct {
		err     error
		message string
	}

	// PreReadError is an error that occurs during the pre-read phase.
	PreReadError struct{ Err }

	// ParseError is an error that occurs during the parsing phase.
	ParseError struct{ Err }

	// LineHookError is an error that occurs during the line hook phase.
	LineHookError struct{ Err }

	// ExecutionError is an error that occurs during the execution phase.
	ExecutionError struct{ Err }
)

func defaultErrorHandler(err error) error {
	fmt.Fprintf(os.Stderr, "Error: %s\n", err)

	return nil
}

// newError creates a new Err.
func newError(err error, message string) Err {
	return Err{
		err:     err,
		message: message,
	}
}

// Error returns the error message with an optional
// message prefix.
func (e Err) Error() string {
	if len(e.message) > 0 {
		return fmt.Sprintf("%s: %s", e.message, e.err.Error())
	}

	return e.err.Error()
}

// Unwrap implements the errors Unwrap interface.
func (e Err) Unwrap() error {
	return e.err
}
