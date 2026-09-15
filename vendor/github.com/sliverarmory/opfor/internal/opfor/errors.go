package opfor

import (
	"errors"
	"fmt"
	"strings"
)

var (
	// ErrScriptUnloaded is returned when an importer retains and invokes a
	// callback after its owning script has been unloaded.
	ErrScriptUnloaded = errors.New("opfor: script unloaded")
	// ErrRuntimeClosed is returned when execution or script loading is requested
	// after Runtime.Close has begun. Closing is terminal: a Runtime is not a
	// reusable session once shutdown starts.
	ErrRuntimeClosed = errors.New("opfor: runtime closed")
	// ErrReadCancellationUnsupported reports that shutdown revoked an
	// asynchronous read callback but could not interrupt its in-flight Read.
	// This is possible only for a borrowed reader which neither has an
	// OPFOR-owned closer nor implements the optional context-aware ReadContext
	// method documented by WithStdin.
	ErrReadCancellationUnsupported = errors.New("opfor: borrowed reader does not support read cancellation")
	// ErrInvalidCallable is returned when a value that is not a function is
	// invoked.
	ErrInvalidCallable = errors.New("opfor: value is not callable")
	// ErrAggressorUIClosed is returned when an Aggressor dialog or prompt
	// responder has already been consumed, dismissed, or failed. A script
	// lifecycle revocation returns ErrScriptUnloaded instead.
	ErrAggressorUIClosed = errors.New("opfor: Aggressor UI responder is closed")
	// ErrInstructionLimit is returned when a configured VM instruction quota
	// is exhausted.
	ErrInstructionLimit = errors.New("opfor: instruction limit exceeded")
	// ErrResourceLimit is matched by every configured runtime resource quota
	// violation, including instruction limits. Use errors.As with *LimitError
	// to inspect the stable resource name and configured limit.
	ErrResourceLimit = errors.New("opfor: resource limit exceeded")
)

// LimitError describes a runtime resource quota violation. Resource is one of
// the exported LimitResource* constants.
type LimitError struct {
	Resource string
	Limit    uint64
}

// Error returns the formatted resource-limit failure.
func (e *LimitError) Error() string {
	if e == nil {
		return ErrResourceLimit.Error()
	}
	return fmt.Sprintf("opfor: %s limit exceeded (%d)", e.Resource, e.Limit)
}

// Unwrap returns ErrResourceLimit for generic errors.Is matching.
func (e *LimitError) Unwrap() error {
	if e == nil {
		return nil
	}
	return ErrResourceLimit
}

// Is keeps the historical ErrInstructionLimit match specific to instruction
// failures while allowing every LimitError to match ErrResourceLimit.
func (e *LimitError) Is(target error) bool {
	if e == nil {
		return false
	}
	if target == ErrResourceLimit {
		return true
	}
	return e.Resource == resourceInstruction && target == ErrInstructionLimit
}

// CompileError groups all lexer, parser, and compiler diagnostics produced for
// one source. Callers can use errors.As to inspect Diagnostics.
type CompileError struct {
	Diagnostics []Diagnostic
}

// Error returns the collected source diagnostics in display form.
func (e *CompileError) Error() string {
	if e == nil || len(e.Diagnostics) == 0 {
		return "opfor: compilation failed"
	}
	if len(e.Diagnostics) == 1 {
		return e.Diagnostics[0].Error()
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "opfor: compilation failed with %d diagnostics", len(e.Diagnostics))
	for _, diagnostic := range e.Diagnostics {
		builder.WriteString("\n  ")
		builder.WriteString(diagnostic.Error())
	}
	return builder.String()
}

// UnsupportedError identifies a syntactically valid operation for which the
// active importer supplied no implementation.
type UnsupportedError struct {
	Operation string
	Name      string
	Span      Span
}

// Error returns the unsupported operation and its source location.
func (e *UnsupportedError) Error() string {
	if e == nil {
		return "opfor: unsupported operation"
	}
	operation := e.Operation
	if operation == "" {
		operation = "operation"
	}
	location := ""
	if e.Span.Source != "" {
		location = " at " + e.Span.String()
	}
	if e.Name == "" {
		return fmt.Sprintf("opfor: unsupported %s%s", operation, location)
	}
	return fmt.Sprintf("opfor: unsupported %s %q%s", operation, e.Name, location)
}

// RuntimeError attaches source and script context to a runtime failure.
type RuntimeError struct {
	Script ScriptID
	Span   Span
	Cause  error
}

// Error returns the underlying failure with runtime source context.
func (e *RuntimeError) Error() string {
	if e == nil {
		return "opfor: runtime error"
	}
	message := "opfor: runtime error"
	if e.Cause != nil {
		message = e.Cause.Error()
	}
	if e.Span.Source != "" {
		message += " at " + e.Span.String()
	}
	return message
}

// Unwrap returns the underlying runtime cause.
func (e *RuntimeError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}
