package opfor

import (
	"context"
	"errors"
	"fmt"
)

// VariableContainerKind identifies the role of a VariableContainer. Sleep
// asks the installed global container to create local containers for closure
// invocations and pushl frames, and internal containers for closure state and
// fork globals.
type VariableContainerKind uint8

const (
	// VariableContainerGlobal is the root variable container of an ordinary
	// loaded Script.
	VariableContainerGlobal VariableContainerKind = iota
	// VariableContainerLocal is the current, temporary local level of one
	// closure invocation.
	VariableContainerLocal
	// VariableContainerInternal stores closure state or the globals of a forked
	// Script.
	VariableContainerInternal
)

func (kind VariableContainerKind) String() string {
	switch kind {
	case VariableContainerGlobal:
		return "global"
	case VariableContainerLocal:
		return "local"
	case VariableContainerInternal:
		return "internal"
	default:
		return fmt.Sprintf("VariableContainerKind(%d)", kind)
	}
}

// VariableContainerRequest describes one container creation. RuntimeID is the
// nonzero identity of the Runtime which will use the container. Script is the
// identity of the ordinary, fork, or ScriptLoader-child Script which owns it.
// Span is the source operation which first required the container; it is empty
// for a script root created before top-level execution.
type VariableContainerRequest struct {
	Kind      VariableContainerKind
	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
}

// VariableAccess describes one normalized Sleep variable operation. Name
// always includes its $, @, or % sigil. RuntimeID, Script, and Span identify
// the originating execution without exposing a *Runtime or *Script.
type VariableAccess struct {
	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
	Name      string
}

// VariableProvider creates the global VariableContainer for each ordinary
// loaded Script. A provider instance is shared with portable ScriptLoader child
// runtimes; requests retain the child's RuntimeID and Script identity.
//
// CreateGlobalVariableContainer is called synchronously once before the Script
// is published or any initial/reserved globals are installed. A returned error
// aborts the load and is authoritative. Implementations may be called
// concurrently and should observe ctx, but must not retain it after returning.
type VariableProvider interface {
	CreateGlobalVariableContainer(context.Context, VariableContainerRequest) (VariableContainer, error)
}

// VariableProviderFunc adapts a function to VariableProvider.
type VariableProviderFunc func(context.Context, VariableContainerRequest) (VariableContainer, error)

// CreateGlobalVariableContainer calls function.
func (function VariableProviderFunc) CreateGlobalVariableContainer(
	ctx context.Context,
	request VariableContainerRequest,
) (VariableContainer, error) {
	if function == nil {
		return nil, errors.New("opfor: variable provider is nil")
	}
	return function(ctx, request)
}

// VariableContainer is the Go equivalent of Sleep's
// sleep.interfaces.Variable bridge. OPFOR deliberately passes *Cell rather
// than copying Values: an existing cell returned by GetScalar remains the exact
// assignment/reference target used by the interpreter.
//
// ScalarExists is queried before GetScalar. If it returns true, GetScalar must
// return a non-nil *Cell. PutScalar must store the exact supplied *Cell; its
// return value is the previously stored cell, matching Sleep's putScalar ABI,
// and OPFOR otherwise ignores it. RemoveScalar removes only this container's
// binding.
//
// CreateLocalVariableContainer and CreateInternalVariableContainer are invoked
// on the Script's global container, not on the currently active local or
// closure container. They must return fresh usable containers unless sharing is
// an intentional importer policy. A nil container without an error is a
// provider contract violation.
//
// Calls are synchronous and errors are authoritative; variable operations
// never retry through Host or a native function. Implementations may be called
// concurrently and should observe ctx. They must not retain ctx, but may retain
// requests and cells subject to the normal capability lifetime of those cells.
type VariableContainer interface {
	ScalarExists(context.Context, VariableAccess) (bool, error)
	GetScalar(context.Context, VariableAccess) (*Cell, error)
	PutScalar(context.Context, VariableAccess, *Cell) (*Cell, error)
	RemoveScalar(context.Context, VariableAccess) error
	CreateLocalVariableContainer(context.Context, VariableContainerRequest) (VariableContainer, error)
	CreateInternalVariableContainer(context.Context, VariableContainerRequest) (VariableContainer, error)
}

// VariableProviderOperation identifies an importer call for error reporting.
type VariableProviderOperation string

const (
	VariableProviderCreateGlobal   VariableProviderOperation = "create global container"
	VariableProviderCreateLocal    VariableProviderOperation = "create local container"
	VariableProviderCreateInternal VariableProviderOperation = "create internal container"
	VariableProviderExists         VariableProviderOperation = "check variable"
	VariableProviderGet            VariableProviderOperation = "get variable"
	VariableProviderPut            VariableProviderOperation = "put variable"
	VariableProviderRemove         VariableProviderOperation = "remove variable"
)

// VariableProviderError adds the exact operation and provenance to an error
// returned by VariableProvider or VariableContainer. Unwrap preserves
// errors.Is/errors.As checks against the importer's original error.
type VariableProviderError struct {
	Operation VariableProviderOperation
	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
	Name      string
	Cause     error
}

func (err *VariableProviderError) Error() string {
	if err == nil {
		return "opfor: variable provider error"
	}
	where := ""
	if err.Name != "" {
		where = " " + err.Name
	}
	if err.Cause == nil {
		return fmt.Sprintf("opfor: variable provider %s%s", err.Operation, where)
	}
	return fmt.Sprintf("opfor: variable provider %s%s: %v", err.Operation, where, err.Cause)
}

// Unwrap returns the provider's original error.
func (err *VariableProviderError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

// WithVariableProvider installs the typed variable-container boundary. The
// provider controls globals for ordinary scripts and supplies the local and
// internal factories used by closures, pushl, forks, and portable ScriptLoader
// child runtimes. WithInitialGlobals and runtime-reserved globals are installed
// into the provider container before top-level execution.
func WithVariableProvider(provider VariableProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: variable provider is nil")
		}
		config.variableProvider = provider
		return nil
	}
}

func (r *Runtime) createGlobalScope(ctx context.Context, script ScriptID) (*scope, error) {
	if r == nil || isNilInterface(r.variableProvider) {
		root := newRootScope()
		if r != nil {
			root.runtimeID = r.ID()
		}
		root.scriptID = script
		return root, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	request := VariableContainerRequest{
		Kind: VariableContainerGlobal, RuntimeID: r.ID(), Script: script,
	}
	container, providerErr := r.variableProvider.CreateGlobalVariableContainer(ctx, request)
	err := joinExecutionContextError(ctx, variableProviderError(
		VariableProviderCreateGlobal, request.RuntimeID, request.Script, request.Span, "", providerErr,
	))
	if err != nil {
		return nil, err
	}
	if isNilInterface(container) {
		return nil, variableProviderError(VariableProviderCreateGlobal, request.RuntimeID, request.Script, request.Span, "", errors.New("provider returned a nil container"))
	}
	return newVariableRootScope(request.RuntimeID, request.Script, container), nil
}
