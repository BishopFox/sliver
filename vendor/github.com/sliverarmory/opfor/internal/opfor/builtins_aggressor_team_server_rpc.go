package opfor

import (
	"context"
	"errors"
	"fmt"
)

const aggressorTeamServerRPCMinimumArguments = 3

// aggressorTeamServerRPCFunctions returns the native wrapper around the
// importer-owned Team Server RPC boundary. With no provider, a valid call
// preserves the original reference-bearing Host invocation exactly once.
func (r *Runtime) aggressorTeamServerRPCFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{"call": r.aggressorTeamServerRPC}
}

func (r *Runtime) aggressorTeamServerRPC(ctx context.Context, invocation Invocation) (Value, error) {
	if invocation.Name != "call" {
		return Null(), &UnsupportedError{
			Operation: "Aggressor Team Server RPC",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if len(invocation.Arguments) < aggressorTeamServerRPCMinimumArguments {
		return Null(), fmt.Errorf("&%s: expected at least %d argument(s), received %d",
			builtinName(invocation.Name), aggressorTeamServerRPCMinimumArguments, len(invocation.Arguments))
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorTeamServerRPCProvider
	if isNilInterface(provider) {
		// Preserve the complete raw Invocation on the compatibility path. Host
		// receives the original pass-by-name Cells and callback-shaped argument;
		// the typed wrapper performs no resolution or validation first.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	// Resolve every source reference exactly once. Value copies retain binary
	// provenance and compound identity, while the provider receives a detached
	// top-level payload slice rather than Invocation.Arguments.
	values := invocation.Values()
	request := AggressorTeamServerRPCRequest{
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Command:   values[0],
		Arguments: append([]Value(nil), values[2:]...),
	}
	if !values[1].IsNull() {
		callback, err := invocation.RetainCallback(values[1])
		if err != nil {
			if errors.Is(err, ErrInvalidCallable) {
				return Null(), fmt.Errorf("&%s: argument 2 is not callable or $null: %w",
					builtinName(invocation.Name), err)
			}
			return Null(), err
		}
		request.Callback = AggressorTeamServerRPCCallback{
			command:  values[0],
			callable: callback,
		}
	}

	if err := provider.CallAggressorTeamServerRPC(ctx, request); err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return Null(), nil
}
