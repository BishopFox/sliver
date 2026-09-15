package opfor

import (
	"context"
	"errors"
	"time"
)

// AggressorBreakpointStackFrame is one active Sleep execution frame, ordered
// from the frame containing brk() toward the script entry point. Function is
// the user-facing function name, SourceLocation is the instruction active in
// that frame, and LocalVariables is a detached snapshot of its current local
// level.
//
// Maps and compound Values belong to the snapshot. A retained frame does not
// expose interpreter scopes or variable Cells. Function and opaque object
// Values necessarily retain their ordinary capability identity.
type AggressorBreakpointStackFrame struct {
	Function       string
	SourceLocation Span
	LocalVariables map[string]Value
}

// AggressorBreakpointSnapshot is the interpreter-built state captured by one
// brk() call. Timestamp comes from the Runtime Clock. SourceLocation is the
// exact brk call span, while ScriptName and the script-facing location string
// use Sleep's basename-oriented display policy.
//
// LocalVariables describes the active local level. GlobalVariables describes
// the script root. ClosureVariables flattens the current closure state and its
// captured lexical scopes from nearest to farthest, excluding the active local
// level and globals. StackFrames and CallStack are ordered innermost-first.
//
// Every map, slice, array, and hash graph is detached from live evaluator
// storage, including cycles and shared compound identity. Functions and opaque
// objects remain ordinary Values and may retain capabilities. Snapshot values
// may be safely retained after the provider returns, but callers must
// synchronize their own concurrent mutations of retained compound Values.
type AggressorBreakpointSnapshot struct {
	RuntimeID RuntimeID
	Script    ScriptID

	ScriptName     string
	SourceLocation Span
	Timestamp      time.Time

	LocalVariables   map[string]Value
	GlobalVariables  map[string]Value
	ClosureVariables map[string]Value
	StackFrames      []AggressorBreakpointStackFrame
	CallStack        []string
	CurrentFunction  string
}

// Clone returns an independently mutable copy of snapshot. Maps, slices,
// arrays, and hashes are recursively detached while preserving cycles and
// shared identities within the clone. Function and object Values preserve
// their original identities because OPFOR cannot clone importer capabilities.
func (snapshot AggressorBreakpointSnapshot) Clone() AggressorBreakpointSnapshot {
	return cloneAggressorBreakpointSnapshot(snapshot)
}

// AggressorBreakpointProvider presents an already-built brk() snapshot to an
// importer UI. OPFOR calls HandleAggressorBreakpoint synchronously exactly
// once for each valid brk() call when a provider is installed. The method may
// block until an operator chooses Continue; it should observe ctx so script
// unload, Runtime.Close, and caller cancellation can end that wait.
//
// A provider error is authoritative: brk() returns $null with that error and
// OPFOR neither prints the headless default nor retries through Host. Provider
// implementations may be called concurrently by independent script
// executions and must provide any synchronization they require. They may
// retain the detached snapshot, but must not retain ctx after the method
// returns.
type AggressorBreakpointProvider interface {
	HandleAggressorBreakpoint(context.Context, AggressorBreakpointSnapshot) error
}

// AggressorBreakpointProviderFunc adapts a function to
// AggressorBreakpointProvider.
type AggressorBreakpointProviderFunc func(context.Context, AggressorBreakpointSnapshot) error

// HandleAggressorBreakpoint calls function.
func (function AggressorBreakpointProviderFunc) HandleAggressorBreakpoint(
	ctx context.Context,
	snapshot AggressorBreakpointSnapshot,
) error {
	if function == nil {
		return errors.New("opfor: Aggressor breakpoint provider is nil")
	}
	return function(ctx, snapshot)
}

// WithAggressorBreakpointProvider installs the synchronous importer boundary
// for brk(). A nil or typed-nil provider is rejected. WithFunction("brk", ...)
// retains precedence regardless of option order.
func WithAggressorBreakpointProvider(provider AggressorBreakpointProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor breakpoint provider is nil")
		}
		config.aggressorBreakpointProvider = provider
		return nil
	}
}
