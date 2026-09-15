package opfor

import (
	"context"
	"fmt"
)

// belevateCommand dispatches an elevator callback for each supplied Beacon ID.
// The callback owns all tasking and logging; OPFOR only supplies the documented
// registry-to-consumer bridge.
func (r *Runtime) belevateCommand(ctx context.Context, invocation Invocation) (Value, error) {
	return r.dispatchAggressorBeaconTechniqueAction(ctx, invocation, aggressorBeaconTechniqueAction{
		kind:          AggressorBeaconTechniqueElevator,
		arity:         3,
		callbackTail:  []int{2},
		techniqueName: 1,
	})
}

// belevate dispatches a local-exploit callback for each supplied Beacon ID.
func (r *Runtime) belevate(ctx context.Context, invocation Invocation) (Value, error) {
	return r.dispatchAggressorBeaconTechniqueAction(ctx, invocation, aggressorBeaconTechniqueAction{
		kind:          AggressorBeaconTechniqueExploit,
		arity:         3,
		callbackTail:  []int{2},
		techniqueName: 1,
	})
}

// bremoteExec dispatches a remote-exec-method callback for each supplied
// Beacon ID.
func (r *Runtime) bremoteExec(ctx context.Context, invocation Invocation) (Value, error) {
	return r.dispatchAggressorBeaconTechniqueAction(ctx, invocation, aggressorBeaconTechniqueAction{
		kind:          AggressorBeaconTechniqueRemoteExecMethod,
		arity:         4,
		callbackTail:  []int{2, 3},
		techniqueName: 1,
	})
}

// bjump dispatches a remote-exploit callback for each supplied Beacon ID.
func (r *Runtime) bjump(ctx context.Context, invocation Invocation) (Value, error) {
	return r.dispatchAggressorBeaconTechniqueAction(ctx, invocation, aggressorBeaconTechniqueAction{
		kind:          AggressorBeaconTechniqueRemoteExploit,
		arity:         4,
		callbackTail:  []int{2, 3},
		techniqueName: 1,
	})
}

type aggressorBeaconTechniqueAction struct {
	kind          AggressorBeaconTechniqueKind
	arity         int
	techniqueName int
	callbackTail  []int
}

func (r *Runtime) dispatchAggressorBeaconTechniqueAction(
	ctx context.Context,
	invocation Invocation,
	action aggressorBeaconTechniqueAction,
) (Value, error) {
	if r == nil {
		return Null(), fmt.Errorf("&%s: runtime is nil", builtinName(invocation.Name))
	}
	if err := requireAggressorCommandArguments(invocation, action.arity); err != nil {
		return Null(), err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	// A missing entry and an importer base entry are indistinguishable at the
	// executable boundary: both have no local callback and must be offered to
	// Host exactly once with the original invocation intact. Calling r.Invoke
	// here would redispatch this native wrapper recursively.
	techniqueName := invocation.Arg(action.techniqueName).String()
	var callback Callable
	var exists bool
	if r.aggressorBeaconTechniques != nil {
		callback, exists = r.aggressorBeaconTechniques.callback(action.kind, techniqueName)
	}
	if !exists || callback == nil {
		if _, err := r.host.Call(ctx, invocation); err != nil {
			return Null(), preserveNativeBoundaryError(ctx, err)
		}
		return Null(), nil
	}

	// Runtime.Invoke does not otherwise install an instruction meter before a
	// native function. Install one here (or reuse the caller's) before entering
	// a retained callback so native callbacks cannot gain a fresh budget through
	// recursive Runtime.Invoke calls.
	ctx = withExecutionMeter(ctx, r)

	idsValue := invocation.Arg(0)
	ids := []Value{idsValue}
	if array, ok := idsValue.Array(); ok {
		// Snapshot only this top-level array. Nested arrays are individual Beacon
		// ID values and are deliberately not flattened.
		ids = array.Values()
	}
	tail := make([]Value, len(action.callbackTail))
	for index, argumentIndex := range action.callbackTail {
		tail[index] = invocation.Arg(argumentIndex)
	}
	for _, id := range ids {
		// Give every invocation its own slice. Callable is an importer-capable
		// interface, so a callback may legally retain or mutate its variadic
		// argument slice after Invoke returns.
		callbackArguments := make([]Value, 1+len(tail))
		callbackArguments[0] = id
		copy(callbackArguments[1:], tail)
		if _, err := callback.Invoke(ctx, callbackArguments...); err != nil {
			// Once a local callback is selected, revocation or execution failure is
			// authoritative. Falling through to Host would duplicate tasking and
			// violate script ownership.
			return Null(), err
		}
		if err := executionContextError(ctx); err != nil {
			return Null(), err
		}
	}
	return Null(), nil
}
