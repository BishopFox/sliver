package opfor

import (
	"context"
	"errors"
)

// AggressorListenerOperation identifies one documented listener query or
// mutation. String values are the exact Aggressor function names.
type AggressorListenerOperation string

const (
	// AggressorListenerCreate identifies the deprecated listener_create.
	AggressorListenerCreate AggressorListenerOperation = "listener_create"
	// AggressorListenerCreateExtended identifies listener_create_ext.
	AggressorListenerCreateExtended AggressorListenerOperation = "listener_create_ext"
	// AggressorListenerDelete identifies listener_delete.
	AggressorListenerDelete AggressorListenerOperation = "listener_delete"
	// AggressorListenerDescribe identifies listener_describe.
	AggressorListenerDescribe AggressorListenerOperation = "listener_describe"
	// AggressorListenerInfo identifies listener_info.
	AggressorListenerInfo AggressorListenerOperation = "listener_info"
	// AggressorListenerPivotCreate identifies listener_pivot_create.
	AggressorListenerPivotCreate AggressorListenerOperation = "listener_pivot_create"
	// AggressorListenerRestart identifies listener_restart.
	AggressorListenerRestart AggressorListenerOperation = "listener_restart"
	// AggressorListenerList identifies listeners.
	AggressorListenerList AggressorListenerOperation = "listeners"
	// AggressorListenerListLocal identifies listeners_local.
	AggressorListenerListLocal AggressorListenerOperation = "listeners_local"
	// AggressorListenerListStageless identifies listeners_stageless.
	AggressorListenerListStageless AggressorListenerOperation = "listeners_stageless"
)

// AggressorListenerRequest is one resolved request against importer-owned
// listener state. Name retains the exact normalized function spelling.
// RuntimeID, Script, and Span provide call-site provenance without exposing a
// *Runtime.
// Bindings is the opaque callback capability for that exact Runtime; retaining
// it retains the Runtime but does not expose its evaluator or binding registry.
//
// Arguments is an exact positional snapshot with these documented shapes:
//
//   - listener_create: name, payload, host, port, then optional beacon hosts
//   - listener_create_ext: name, payload, and options hash
//   - listener_delete/listener_restart: listener name
//   - listener_describe/listener_info: listener name and an optional target/key
//   - listener_pivot_create: Beacon ID, name, payload, host, and port
//   - listeners/listeners_local/listeners_stageless: no positions
//
// The top-level slice is detached from Invocation. Values are resolved once
// and retain binary provenance and compound identity. Length distinguishes an
// omitted optional argument from explicit $null.
type AggressorListenerRequest struct {
	Operation AggressorListenerOperation
	Name      string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span
	Bindings  AggressorBindings

	Arguments []Value
}

// Arg returns a resolved positional argument or $null when index is absent.
func (request AggressorListenerRequest) Arg(index int) Value {
	if index < 0 || index >= len(request.Arguments) {
		return Null()
	}
	return request.Arguments[index]
}

// HasArgument reports whether a positional argument was supplied.
func (request AggressorListenerRequest) HasArgument(index int) bool {
	return index >= 0 && index < len(request.Arguments)
}

// AggressorListenerProvider supplies Cobalt-owned listener state. It is
// called synchronously exactly once for every valid invocation when installed.
// Errors are authoritative and never fall back to Host because a mutation may
// already have taken effect.
//
// listener_describe, listener_info, listeners, listeners_local, and
// listeners_stageless transfer the provider's successful Value directly to
// script code. The five create/delete/restart operations are documented only
// as effects; OPFOR discards their provider result and returns $null. Record
// schemas, ordering, remote-listener resolution, and missing-listener behavior
// are not fully specified by the public reference and remain provider-owned.
//
// Implementations may be called concurrently and should observe ctx. They may
// retain request Values, but must not retain ctx after HandleAggressorListener
// returns.
type AggressorListenerProvider interface {
	HandleAggressorListener(context.Context, AggressorListenerRequest) (Value, error)
}

// AggressorListenerProviderFunc adapts a function to AggressorListenerProvider.
type AggressorListenerProviderFunc func(context.Context, AggressorListenerRequest) (Value, error)

// HandleAggressorListener calls function.
func (function AggressorListenerProviderFunc) HandleAggressorListener(
	ctx context.Context,
	request AggressorListenerRequest,
) (Value, error) {
	if function == nil {
		return Null(), errors.New("opfor: Aggressor listener provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorListenerProvider installs the typed importer boundary for the
// ten documented listener functions. Provider errors are authoritative and
// importer-defined WithFunction callbacks retain precedence.
func WithAggressorListenerProvider(provider AggressorListenerProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor listener provider is nil")
		}
		config.aggressorListenerProvider = provider
		return nil
	}
}
