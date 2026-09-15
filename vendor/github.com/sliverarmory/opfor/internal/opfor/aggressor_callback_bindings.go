package opfor

import (
	"context"
	"errors"
)

// ErrAggressorBindingsUnavailable is returned when a method is called on the
// zero value of AggressorBindings. A zero value carries no Runtime authority.
var ErrAggressorBindingsUnavailable = errors.New("opfor: Aggressor bindings are unavailable")

// AggressorBindings is an opaque capability for invoking the event, hook, and
// popup registrations owned by the Runtime which produced a typed Aggressor
// provider request. Script-originated capabilities are also bound to the exact
// generation which created the request. It deliberately does not expose the
// Runtime, evaluator, binding registry, script scopes, or generation token.
//
// A capability may be retained after the provider call. Retaining it also
// retains its Runtime. Calls acquire an ordinary Runtime execution lease and,
// when the request originated in a script, a lease for that exact script
// generation. They honor cancellation and instruction limits, reject execution
// after explicit unload or Runtime.Close, and never become valid again if the
// same Script is run in a later generation.
//
// The zero value is invalid. Use Valid before optional use, RuntimeID for
// provenance, and Same to compare two capabilities without exposing their
// underlying Runtime pointers.
type AggressorBindings struct {
	runtime    *Runtime
	owner      *Script
	generation *scriptGeneration
}

// aggressorBindingsFor preserves runtime-only authority for trusted legacy and
// scriptless invocation paths. Script-originated provider requests must use
// aggressorBindingsForInvocation so explicit unload revokes retained calls.
func aggressorBindingsFor(runtime *Runtime) AggressorBindings {
	return AggressorBindings{runtime: runtime}
}

func aggressorBindingsForInvocation(invocation Invocation) AggressorBindings {
	generation := invocation.generationToken()
	var owner *Script
	if generation != nil {
		owner = generation.script
	}
	return AggressorBindings{
		runtime:    invocation.Runtime,
		owner:      owner,
		generation: generation,
	}
}

// Valid reports whether bindings carries Runtime authority. It remains true
// after its script generation unloads or its Runtime closes; subsequent
// invocation methods return ErrScriptUnloaded or ErrRuntimeClosed.
func (bindings AggressorBindings) Valid() bool {
	return bindings.runtime != nil
}

// RuntimeID returns the process-local identity of the bound Runtime, or zero
// for an invalid capability.
func (bindings AggressorBindings) RuntimeID() RuntimeID {
	return bindings.runtime.ID()
}

// Same reports whether two valid capabilities address the same Runtime.
// Invalid zero values are never considered the same capability.
func (bindings AggressorBindings) Same(other AggressorBindings) bool {
	return bindings.runtime != nil && bindings.runtime == other.runtime
}

// DispatchEvent invokes every exact and wildcard event registration in load
// order and returns their successful results.
func (bindings AggressorBindings) DispatchEvent(
	ctx context.Context,
	name string,
	arguments ...Value,
) ([]Value, error) {
	runtime, executionCtx, release, err := bindings.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	results, invokeErr := runtime.DispatchEvent(executionCtx, name, arguments...)
	if invokeErr == nil {
		invokeErr = runtimeExecutionError(executionCtx)
	}
	return results, errors.Join(invokeErr, release())
}

// InvokeHook invokes the newest active set-hook registration with name.
func (bindings AggressorBindings) InvokeHook(
	ctx context.Context,
	name string,
	arguments ...Value,
) (Value, error) {
	return bindings.invoke(ctx, BindingHook, name, arguments...)
}

// InvokePopupHook invokes the newest active popup registration with name.
// Use DispatchPopupHook when every additive popup layer should contribute.
func (bindings AggressorBindings) InvokePopupHook(
	ctx context.Context,
	name string,
	arguments ...Value,
) (Value, error) {
	return bindings.invoke(ctx, BindingPopup, name, arguments...)
}

// DispatchPopupHook invokes every exact popup registration in load order and
// returns their successful results.
func (bindings AggressorBindings) DispatchPopupHook(
	ctx context.Context,
	name string,
	arguments ...Value,
) ([]Value, error) {
	runtime, executionCtx, release, err := bindings.acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	results, invokeErr := runtime.DispatchPopupHook(executionCtx, name, arguments...)
	if invokeErr == nil {
		invokeErr = runtimeExecutionError(executionCtx)
	}
	return results, errors.Join(invokeErr, release())
}

func (bindings AggressorBindings) invoke(
	ctx context.Context,
	kind BindingKind,
	name string,
	arguments ...Value,
) (Value, error) {
	runtime, executionCtx, release, err := bindings.acquire(ctx)
	if err != nil {
		return Null(), err
	}
	defer release()
	value, invokeErr := runtime.InvokeBinding(executionCtx, kind, name, arguments...)
	if invokeErr == nil {
		invokeErr = runtimeExecutionError(executionCtx)
	}
	return value, errors.Join(invokeErr, release())
}

func (bindings AggressorBindings) acquire(
	ctx context.Context,
) (*Runtime, context.Context, func() error, error) {
	if bindings.runtime == nil {
		return nil, ctx, nil, ErrAggressorBindingsUnavailable
	}
	executionCtx, release, err := bindings.runtime.acquireRuntimeExecution(ctx)
	if err != nil {
		return nil, ctx, nil, err
	}
	if bindings.generation == nil {
		return bindings.runtime, withExecutionMeter(executionCtx, bindings.runtime), release, nil
	}
	generationCtx, releaseGeneration, err := bindings.owner.acquireGenerationExecution(executionCtx, bindings.generation)
	if err != nil {
		return nil, ctx, nil, errors.Join(err, release())
	}
	releaseBoth := func() error {
		return errors.Join(releaseGeneration(), release())
	}
	return bindings.runtime, withExecutionMeter(generationCtx, bindings.runtime), releaseBoth, nil
}
