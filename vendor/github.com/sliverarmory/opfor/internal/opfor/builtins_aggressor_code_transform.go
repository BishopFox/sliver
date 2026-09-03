package opfor

import (
	"context"
	"errors"
)

const aggressorPowerShellCompressHook = "POWERSHELL_COMPRESS"

type aggressorCodeTransformSpec struct {
	operation AggressorCodeTransformOperation
	arguments int
}

// The current Fortra function reference documents these exact positional
// shapes and value-producing contracts. It does not publish the concrete
// encoder bytes, PowerShell deflator template, or VBS output grammar, so those
// results remain importer-owned.
var aggressorCodeTransformSpecs = map[string]aggressorCodeTransformSpec{
	"encode":              {operation: AggressorCodeTransformEncode, arguments: 3},
	"powershell_compress": {operation: AggressorCodeTransformPowerShellCompress, arguments: 1},
	"transform_vbs":       {operation: AggressorCodeTransformVBS, arguments: 2},
}

// aggressorCodeTransformFunctions returns native wrappers around Cobalt-owned
// code and script transformations. With no provider, every valid call keeps
// the original reference-bearing Host invocation. powershell_compress first
// honors the newest active POWERSHELL_COMPRESS hook.
func (r *Runtime) aggressorCodeTransformFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorCodeTransformSpecs))
	for name := range aggressorCodeTransformSpecs {
		functions[name] = r.aggressorCodeTransform
	}
	return functions
}

func (r *Runtime) aggressorCodeTransform(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorCodeTransformSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor code-transform operation",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireExactAggressorClientArguments(invocation, spec.arguments); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	if invocation.Name == "powershell_compress" {
		if result, handled, err := r.invokeAggressorPowerShellCompressHook(ctx, invocation); handled {
			return result, err
		}
	}

	provider := r.aggressorCodeTransformProvider
	if isNilInterface(provider) {
		// Preserve the exact reference-bearing Invocation for existing Cobalt
		// adapters. The hook path above is the only documented rewrite.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	request := AggressorCodeTransformRequest{
		Operation: spec.operation,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Arguments: invocation.Values(),
	}
	result, err := provider.HandleAggressorCodeTransform(ctx, request)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return result, nil
}

// invokeAggressorPowerShellCompressHook invokes the exact newest binding
// snapshot once. If registration or unload races after selection,
// prepareBindingInvocation either retains that owner for the call or rejects
// it; OPFOR never silently retargets an older hook or falls through after a
// selected hook may have run.
func (r *Runtime) invokeAggressorPowerShellCompressHook(
	ctx context.Context,
	invocation Invocation,
) (Value, bool, error) {
	hooks := r.Bindings(BindingHook, aggressorPowerShellCompressHook)
	if len(hooks) == 0 {
		return Null(), false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), true, err
	}

	hook := hooks[len(hooks)-1]
	values := invocation.Values()
	hookArguments := []Value{values[0]}
	hookContext, release, err := r.prepareBindingInvocation(ctx, hook, hookArguments)
	if err != nil {
		return Null(), true, err
	}
	defer release()
	result, invokeErr := hook.Callback.Invoke(hookContext, hookArguments...)
	if err := joinExecutionError(invokeErr, release); err != nil {
		return Null(), true, err
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), true, err
	}
	return result, true, nil
}
