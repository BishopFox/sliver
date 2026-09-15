package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorBeaconExecutionSpec struct {
	kind         AggressorBeaconExecutionKind
	minimum      int
	maximum      int
	returnsValue bool
}

var aggressorBeaconExecutionSpecs = map[string]aggressorBeaconExecutionSpec{
	"beacon_execute_job":          {kind: AggressorBeaconExecuteJob, minimum: 4, maximum: 4},
	"beacon_execute_postex_job":   {kind: AggressorBeaconExecutePostexJob, minimum: 3, maximum: 6},
	"beacon_host_imported_script": {kind: AggressorBeaconHostImportedScript, minimum: 1, maximum: 1, returnsValue: true},
	"beacon_host_script":          {kind: AggressorBeaconHostScript, minimum: 2, maximum: 2, returnsValue: true},
	"beacon_inline_execute":       {kind: AggressorBeaconInlineExecute, minimum: 4, maximum: 5},
	"beacon_inline_execute_pe":    {kind: AggressorBeaconInlineExecutePE, minimum: 4, maximum: 5},
	"get_postex_kit_callback_id":  {kind: AggressorBeaconPostexKitCallbackID, minimum: 0, maximum: 0, returnsValue: true},
}

// aggressorBeaconExecutionFunctions returns native wrappers around the
// importer-owned low-level Beacon execution boundary. beacon_inline_execute
// uses its specialized wrapper so it can apply BEACON_INLINE_EXECUTE before
// either the typed provider or the compatibility Host sees the task.
func (r *Runtime) aggressorBeaconExecutionFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorBeaconExecutionSpecs))
	for name := range aggressorBeaconExecutionSpecs {
		functions[name] = r.aggressorBeaconExecution
	}
	functions["beacon_inline_execute"] = r.beaconInlineExecute
	return functions
}

func (r *Runtime) aggressorBeaconExecution(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorBeaconExecutionSpecs[invocation.Name]
	if !exists || spec.kind == AggressorBeaconInlineExecute {
		return Null(), &UnsupportedError{
			Operation: "Aggressor Beacon execution",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorBeaconExecutionArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorBeaconExecutionProvider
	if isNilInterface(provider) {
		// Preserve the original reference-bearing Invocation on the compatibility
		// path. No argument is resolved, copied, or callback-wrapped first.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	values := invocation.Values()
	request := AggressorBeaconExecutionRequest{
		Kind:      spec.kind,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Bindings:  invocation.Bindings(),
	}

	switch spec.kind {
	case AggressorBeaconExecuteJob:
		request.BeaconID = values[0]
		request.Command = values[1]
		request.CommandArguments = values[2]
		request.Flags = values[3]
	case AggressorBeaconExecutePostexJob:
		request.BeaconID = values[0]
		request.PID = values[1]
		request.Content = values[2]
		if len(values) > 3 {
			request.PackedArguments = values[3]
			request.HasPackedArguments = true
		}
		if len(values) > 4 {
			callback, state, err := retainAggressorBeaconExecutionCallback(invocation, values[4], 5)
			if err != nil {
				return Null(), err
			}
			request.Callback = callback
			request.CallbackState = state
		}
		if len(values) > 5 {
			request.MessageID = values[5]
			request.HasMessageID = true
		}
	case AggressorBeaconInlineExecutePE:
		request.BeaconID = values[0]
		request.Content = values[1]
		request.EntryPoint = values[2]
		request.PackedArguments = values[3]
		request.HasPackedArguments = true
		if len(values) > 4 {
			callback, state, err := retainAggressorBeaconExecutionCallback(invocation, values[4], 5)
			if err != nil {
				return Null(), err
			}
			request.Callback = callback
			request.CallbackState = state
		}
	case AggressorBeaconHostImportedScript:
		request.BeaconID = values[0]
	case AggressorBeaconHostScript:
		request.BeaconID = values[0]
		request.Content = values[1]
	case AggressorBeaconPostexKitCallbackID:
		// This query has no arguments.
	default:
		return Null(), &UnsupportedError{
			Operation: "Aggressor Beacon execution",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}

	result, err := provider.HandleAggressorBeaconExecution(ctx, request)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if spec.returnsValue {
		return result, nil
	}
	return Null(), nil
}

func retainAggressorBeaconExecutionCallback(
	invocation Invocation,
	value Value,
	position int,
) (Callable, AggressorCallbackState, error) {
	if value.IsNull() {
		return nil, AggressorCallbackNull, nil
	}
	callback, err := invocation.RetainCallback(value)
	if err != nil {
		if errors.Is(err, ErrInvalidCallable) {
			return nil, AggressorCallbackOmitted, fmt.Errorf(
				"&%s: argument %d is not callable or $null: %w",
				builtinName(invocation.Name), position, err,
			)
		}
		return nil, AggressorCallbackOmitted, err
	}
	return callback, AggressorCallbackCallable, nil
}

func requireAggressorBeaconExecutionArguments(invocation Invocation, minimum, maximum int) error {
	count := len(invocation.Arguments)
	if count >= minimum && count <= maximum {
		return nil
	}
	if minimum == maximum {
		return fmt.Errorf("&%s: expected %d argument(s), received %d",
			builtinName(invocation.Name), minimum, count)
	}
	if maximum == minimum+1 {
		return fmt.Errorf("&%s: expected %d or %d arguments, received %d",
			builtinName(invocation.Name), minimum, maximum, count)
	}
	return fmt.Errorf("&%s: expected %d to %d argument(s), received %d",
		builtinName(invocation.Name), minimum, maximum, count)
}
