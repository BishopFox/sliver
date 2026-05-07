package transport

import "fmt"

// Error wraps a low-level transport error in a concrete type.
type Error struct {
	Err error
}

func (e *Error) Error() string {
	return fmt.Sprintf("transport error: %v", e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}

func NewError(err error) *Error {
	return &Error{
		Err: err,
	}
}
