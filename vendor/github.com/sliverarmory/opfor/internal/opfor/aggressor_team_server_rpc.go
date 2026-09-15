package opfor

import (
	"context"
	"errors"
)

// AggressorTeamServerRPCRequest is one resolved call(...) request to the
// Cobalt Strike Team Server. Name is the exact normalized Aggressor function
// spelling used by the script. RuntimeID is the nonzero process-local identity
// of the originating Runtime; Script and Span identify the call site without
// exposing a *Runtime.
//
// Command is the first call argument. Arguments contains the third and later
// arguments in source order; the second argument is represented by Callback.
// The wrapper resolves every source argument exactly once and detaches the
// top-level Arguments slice, so no pass-by-name Cell or Invocation crosses the
// typed boundary. Values are otherwise unmodified: binary provenance and
// compound/object/function reference identity are retained. Providers which
// retain a request also retain capabilities reachable through those Values
// and should snapshot or detach them when that lifetime is undesirable.
//
// Callback is invalid when the script supplied $null. A valid callback is a
// retained, script-owned, multi-shot response capability. It rejects responses
// after the creating Script generation retires, its Script unloads, or its
// Runtime closes.
type AggressorTeamServerRPCRequest struct {
	Name string

	RuntimeID RuntimeID
	Script    ScriptID
	Span      Span

	Command   Value
	Arguments []Value
	Callback  AggressorTeamServerRPCCallback
}

// AggressorTeamServerRPCCallback is the response capability supplied by the
// second argument to call(...). Its zero value is safe and represents an
// explicit $null callback.
//
// A valid callback is multi-shot. Respond always invokes the script callback
// with exactly two positional arguments: the original Command Value followed
// by the supplied response Value. It honors ctx and the creating Script's
// lifecycle.
type AggressorTeamServerRPCCallback struct {
	command  Value
	callable Callable
}

// Valid reports whether callback contains a retained script-owned function.
func (callback AggressorTeamServerRPCCallback) Valid() bool {
	return !isNilInterface(callback.callable)
}

// Respond delivers one Team Server response to the retained script callback.
// It returns ErrInvalidCallable for the zero value.
func (callback AggressorTeamServerRPCCallback) Respond(
	ctx context.Context,
	response Value,
) (Value, error) {
	if !callback.Valid() {
		return Null(), ErrInvalidCallable
	}
	return callback.callable.Invoke(ctx, callback.command, response)
}

// AggressorTeamServerRPCProvider dispatches typed call(...) requests. OPFOR
// calls CallAggressorTeamServerRPC synchronously exactly once for each valid
// invocation when a provider is configured. A nil error means the request was
// accepted; the native wrapper then returns $null. Returning an error rejects
// the invocation and is authoritative: OPFOR never retries through Host,
// because doing so could duplicate a Team Server effect.
//
// Implementations may be called concurrently and should observe ctx. A
// provider may retain Request Values and Callback subject to the documented
// capability lifetimes, but must not retain ctx after this method returns.
type AggressorTeamServerRPCProvider interface {
	CallAggressorTeamServerRPC(context.Context, AggressorTeamServerRPCRequest) error
}

// AggressorTeamServerRPCProviderFunc adapts a function to
// AggressorTeamServerRPCProvider.
type AggressorTeamServerRPCProviderFunc func(context.Context, AggressorTeamServerRPCRequest) error

// CallAggressorTeamServerRPC calls function.
func (function AggressorTeamServerRPCProviderFunc) CallAggressorTeamServerRPC(
	ctx context.Context,
	request AggressorTeamServerRPCRequest,
) error {
	if function == nil {
		return errors.New("opfor: Aggressor Team Server RPC provider is nil")
	}
	return function(ctx, request)
}

// WithAggressorTeamServerRPCProvider installs the typed importer boundary for
// call. Provider errors are authoritative and never fall back to Host.
// Importer-defined WithFunction callbacks retain precedence over the native
// wrapper.
func WithAggressorTeamServerRPCProvider(provider AggressorTeamServerRPCProvider) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(provider) {
			return errors.New("opfor: Aggressor Team Server RPC provider is nil")
		}
		config.aggressorTeamServerRPCProvider = provider
		return nil
	}
}
