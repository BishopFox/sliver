package errors

import (
	"errors"
	"fmt"
	"io"

	"github.com/mailgun/errors/callstack"
)

// Stack annotates err with a stack trace at the point Stack was called.
// If err is nil, Stack returns nil.
func Stack(err error) error {
	if err == nil {
		return nil
	}
	return &stack{
		err,
		callstack.New(1),
	}
}

type stack struct {
	error
	*callstack.CallStack
}

func (w *stack) Unwrap() error { return w.error }

func (w *stack) Is(target error) bool {
	_, ok := target.(*stack)
	return ok
}

// Cause returns the wrapped error which was the original
// cause of the issue. We only support this because some code
// depends on github.com/pkg/errors.Cause() returning the cause
// of the error.
// Deprecated: use error.Is() or error.As() instead
func (w *stack) Cause() error { return w.error }

func (w *stack) HasFields() map[string]any {
	if child, ok := w.error.(HasFields); ok {
		return child.HasFields()
	}

	var f HasFields
	if errors.As(w.error, &f) {
		return f.HasFields()
	}

	return nil
}

func (w *stack) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			_, _ = fmt.Fprintf(s, "%+v", w.Unwrap())
			w.CallStack.Format(s, verb)
			return
		}
		fallthrough
	case 's':
		_, _ = io.WriteString(s, w.Error())
	case 'q':
		_, _ = fmt.Fprintf(s, "%q", w.Error())
	}
}
