package opfor

import (
	"context"
	"fmt"
)

const aggressorBeaconInlineExecuteHook = "BEACON_INLINE_EXECUTE"

// beaconInlineExecute implements the portable portion of the documented
// beacon_inline_execute contract. OPFOR applies the script hook and retains an
// optional callback, then sends the actual Beacon task to the configured typed
// provider or, for compatibility, Host. It never loads or executes a BOF in
// the local Go process.
func (r *Runtime) beaconInlineExecute(ctx context.Context, invocation Invocation) (Value, error) {
	if r == nil {
		return Null(), fmt.Errorf("&%s: runtime is nil", builtinName(invocation.Name))
	}
	spec, exists := aggressorBeaconExecutionSpecs[invocation.Name]
	if !exists || spec.kind != AggressorBeaconInlineExecute {
		return Null(), &UnsupportedError{
			Operation: "Aggressor Beacon execution",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorBeaconExecutionArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	provider := r.aggressorBeaconExecutionProvider
	if isNilInterface(provider) {
		return r.beaconInlineExecuteHost(ctx, invocation)
	}

	// Resolve each source argument exactly once for the typed boundary. The BOF
	// hook consumes the captured content, so mutations performed by the hook
	// cannot make the rest of this request observe a second argument snapshot.
	values := invocation.Values()
	bof, err := r.applyBeaconInlineExecuteHook(ctx, values[1])
	if err != nil {
		return Null(), err
	}
	request := AggressorBeaconExecutionRequest{
		Kind:               spec.kind,
		Name:               invocation.Name,
		RuntimeID:          r.ID(),
		Script:             invocation.Script,
		Span:               invocation.Span,
		Bindings:           invocation.Bindings(),
		BeaconID:           values[0],
		Content:            bof,
		EntryPoint:         values[2],
		PackedArguments:    values[3],
		HasPackedArguments: true,
	}
	if len(values) == 5 {
		callback, state, err := retainAggressorBeaconExecutionCallback(invocation, values[4], 5)
		if err != nil {
			return Null(), err
		}
		request.Callback = callback
		request.CallbackState = state
	}

	if _, err := provider.HandleAggressorBeaconExecution(ctx, request); err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return Null(), nil
}

// beaconInlineExecuteHost preserves the original specialized Host bridge for
// importers that have not installed the typed provider. The active hook may
// replace the BOF and a callable is lifecycle-guarded before Host receives it;
// every other source Argument retains its original pass-by-name identity.
func (r *Runtime) beaconInlineExecuteHost(ctx context.Context, invocation Invocation) (Value, error) {
	bof, err := r.applyBeaconInlineExecuteHook(ctx, invocation.Arg(1))
	if err != nil {
		return Null(), err
	}

	forwarded := invocation
	forwarded.Runtime = r
	forwarded.Arguments = append([]Argument(nil), invocation.Arguments...)
	// The hook result is a detached task payload. Do not retain the caller's
	// original pass-by-name Cell, whose current value would hide this rewrite.
	forwarded.Arguments[1] = Argument{Value: bof}
	if len(forwarded.Arguments) == 5 {
		// Resolve a reference exactly once and always detach it. This binds the
		// tested value and retained identity to one instant and prevents a null
		// callback cell from becoming an unguarded Host-visible function later.
		callbackValue := invocation.Arg(4)
		forwarded.Arguments[4] = Argument{Value: callbackValue}
		if !callbackValue.IsNull() {
			callback, err := invocation.RetainCallback(callbackValue)
			if err != nil {
				return Null(), fmt.Errorf("&%s: invalid callback: %w", builtinName(invocation.Name), err)
			}
			// Supplying the guarded capability as the Host-visible function also
			// protects simple Host implementations which retain Arg(4) directly
			// instead of asking Invocation.Callback for a second wrapper.
			forwarded.Arguments[4] = Argument{Value: FunctionValue(callback)}
		}
	}

	if _, err := r.host.Call(ctx, forwarded); err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	// The official function documents tasking and an optional result callback,
	// but no synchronous result. Keep the script-visible operation side-effect
	// only even if an importer returns internal task metadata.
	return Null(), nil
}

func (r *Runtime) applyBeaconInlineExecuteHook(ctx context.Context, content Value) (Value, error) {
	// BOFs are byte strings at this boundary. Normalizing through Sleep's low
	// byte coercion preserves BinaryString octets and gives ordinary textual
	// input the same deterministic byte policy as the other Aggressor binary
	// helpers.
	bof := BinaryString(sleepStringLowBytes(content))
	if hooks := r.Bindings(BindingHook, aggressorBeaconInlineExecuteHook); len(hooks) != 0 {
		// Invoke the exact newest snapshot we observed. A second name lookup could
		// race registration/unload and silently select a different hook.
		hook := hooks[len(hooks)-1]
		hookArguments := []Value{bof, Null()}
		hookContext, release, err := r.prepareBindingInvocation(ctx, hook, hookArguments)
		if err != nil {
			return Null(), err
		}
		defer release()
		updated, invokeErr := hook.Callback.Invoke(hookContext, hookArguments...)
		if err := joinExecutionError(invokeErr, release); err != nil {
			return Null(), err
		}
		if !updated.IsNull() {
			bof = BinaryString(sleepStringLowBytes(updated))
		}
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return bof, nil
}
