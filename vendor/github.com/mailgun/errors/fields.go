package errors

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"github.com/mailgun/errors/callstack"
)

// HasFields Implement this interface to pass along unstructured context to the logger.
// It is the responsibility of Fields() implementation to unwrap the error chain and
// collect all errors that have `HasFields()` defined.
type HasFields interface {
	HasFields() map[string]any
}

// HasFormat True if the interface has the format method (from fmt package)
type HasFormat interface {
	Format(st fmt.State, verb rune)
}

// Fields Creates errors that conform to the `HasFields` interface
type Fields map[string]any

// Wrapf returns an error annotating err with a stack trace
// at the point Wrapf is call, and the format specifier.
// If err is nil, Wrapf returns nil.
func (f Fields) Wrapf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return &fields{
		stack:   callstack.New(1),
		fields:  f,
		wrapped: err,
		msg:     fmt.Sprintf(format, args...),
	}
}

// WrapFields returns a new error wrapping the provided error with fields and a message.
func WrapFields(err error, f Fields, msg string) error {
	if err == nil {
		return nil
	}
	return &fields{
		stack:   callstack.New(1),
		wrapped: err,
		msg:     msg,
		fields:  f,
	}
}

// WrapFieldsf is identical to WrapFields but with optional formatting
func WrapFieldsf(err error, f Fields, format string, args ...any) error {
	if err == nil {
		return nil
	}
	return &fields{
		msg:     fmt.Sprintf(format, args...),
		stack:   callstack.New(1),
		wrapped: err,
		fields:  f,
	}
}

// Wrap returns an error annotating err with a stack trace
// at the point Wrap is called, and the supplied message.
// If err is nil, Wrap returns nil.
func (f Fields) Wrap(err error, msg string) error {
	if err == nil {
		return nil
	}
	return &fields{
		stack:   callstack.New(1),
		fields:  f,
		wrapped: err,
		msg:     msg,
	}
}

// Stack returns an error annotating err with a stack trace
// at the point Stack is called. If err is nil, Stack returns nil.
func (f Fields) Stack(err error) error {
	if err == nil {
		return nil
	}
	return &fields{
		stack:   callstack.New(1),
		fields:  f,
		wrapped: err,
	}
}

func (f Fields) Error(msg string) error {
	return &fields{
		stack:   callstack.New(1),
		fields:  f,
		wrapped: errors.New(msg),
		msg:     "",
	}
}

func (f Fields) Errorf(format string, args ...any) error {
	return &fields{
		stack:   callstack.New(1),
		fields:  f,
		wrapped: fmt.Errorf(format, args...),
		msg:     "",
	}
}

type fields struct {
	fields  Fields
	msg     string
	wrapped error
	stack   *callstack.CallStack
}

func (c *fields) Unwrap() error {
	return c.wrapped
}

func (c *fields) Is(target error) bool {
	_, ok := target.(*fields)
	return ok
}

// Cause returns the wrapped error which was the original
// cause of the issue. We only support this because some code
// depends on github.com/pkg/errors.Cause() returning the cause
// of the error.
// Deprecated: use error.Is() or error.As() instead
func (c *fields) Cause() error { return c.wrapped }

func (c *fields) Error() string {
	if c.msg == NoMsg {
		return c.wrapped.Error()
	}
	return c.msg + ": " + c.wrapped.Error()
}

func (c *fields) StackTrace() callstack.StackTrace {
	if child, ok := c.wrapped.(callstack.HasStackTrace); ok {
		return child.StackTrace()
	}
	return c.stack.StackTrace()
}

func (c *fields) HasFields() map[string]any {
	result := make(map[string]any, len(c.fields))
	for key, value := range c.fields {
		result[key] = value
	}

	// child fields have precedence as they are closer to the cause
	var f HasFields
	if errors.As(c.wrapped, &f) {
		child := f.HasFields()
		if child == nil {
			return result
		}
		for key, value := range child {
			result[key] = value
		}
	}
	return result
}

func (c *fields) Format(s fmt.State, verb rune) {
	switch verb {
	case 'v':
		if s.Flag('+') {
			if c.msg == NoMsg {
				_, _ = fmt.Fprintf(s, "%+v (%s)", c.Unwrap(), c.FormatFields())
				return
			}
			_, _ = fmt.Fprintf(s, "%s: %+v (%s)", c.msg, c.Unwrap(), c.FormatFields())
			return
		}
		fallthrough
	case 's', 'q':
		_, _ = io.WriteString(s, c.Error())
		return
	}
}

func (c *fields) FormatFields() string {
	var buf bytes.Buffer
	var count int

	for key, value := range c.fields {
		if count > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(fmt.Sprintf("%+v=%+v", key, value))
		count++
	}
	return buf.String()
}

// ToMap Returns the fields for the underlying error as map[string]any
// If no fields are available returns nil
func ToMap(err error) map[string]any {
	if err == nil {
		return nil
	}

	result := map[string]any{
		"excValue": err.Error(),
		"excType":  fmt.Sprintf("%T", Unwrap(err)),
	}

	// Find any errors with StackTrace information if available
	var stack callstack.HasStackTrace
	if Last(err, &stack) {
		trace := stack.StackTrace()
		caller := callstack.GetLastFrame(trace)
		result["excFuncName"] = caller.Func
		result["excLineNum"] = caller.LineNo
		result["excFileName"] = caller.File
	}

	// Search the error chain for fields
	var f HasFields
	if errors.As(err, &f) {
		for key, value := range f.HasFields() {
			result[key] = value
		}
	}
	return result
}

// ToLogrus Returns the context and stacktrace information for the underlying error
// that could be used as logrus.Fields
//
//	logrus.Fields(errors.ToLogrus(err)).WithField("tid", 1).Error(err)
func ToLogrus(err error) map[string]any {
	return ToMap(err)
}
