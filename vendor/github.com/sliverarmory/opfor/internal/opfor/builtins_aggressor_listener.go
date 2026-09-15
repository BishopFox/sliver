package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorListenerSpec struct {
	operation AggressorListenerOperation
	minimum   int
	maximum   int
	returns   bool
	hashIndex int
}

// listener_create's argument list publishes five positions while the official
// foreign-listener example supplies four, so the beacon-host list is optional.
var aggressorListenerSpecs = map[string]aggressorListenerSpec{
	"listener_create":       {operation: AggressorListenerCreate, minimum: 4, maximum: 5, hashIndex: -1},
	"listener_create_ext":   {operation: AggressorListenerCreateExtended, minimum: 3, maximum: 3, hashIndex: 2},
	"listener_delete":       {operation: AggressorListenerDelete, minimum: 1, maximum: 1, hashIndex: -1},
	"listener_describe":     {operation: AggressorListenerDescribe, minimum: 1, maximum: 2, returns: true, hashIndex: -1},
	"listener_info":         {operation: AggressorListenerInfo, minimum: 1, maximum: 2, returns: true, hashIndex: -1},
	"listener_pivot_create": {operation: AggressorListenerPivotCreate, minimum: 5, maximum: 5, hashIndex: -1},
	"listener_restart":      {operation: AggressorListenerRestart, minimum: 1, maximum: 1, hashIndex: -1},
	"listeners":             {operation: AggressorListenerList, returns: true, hashIndex: -1},
	"listeners_local":       {operation: AggressorListenerListLocal, returns: true, hashIndex: -1},
	"listeners_stageless":   {operation: AggressorListenerListStageless, returns: true, hashIndex: -1},
}

func (r *Runtime) aggressorListenerFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorListenerSpecs))
	for name := range aggressorListenerSpecs {
		functions[name] = r.aggressorListener
	}
	return functions
}

func (r *Runtime) aggressorListener(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorListenerSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor listener operation",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorListenerArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorListenerProvider
	if isNilInterface(provider) {
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
	if spec.hashIndex >= 0 {
		if _, ok := values[spec.hashIndex].Hash(); !ok {
			return Null(), fmt.Errorf("&%s: argument %d must be a hash",
				builtinName(invocation.Name), spec.hashIndex+1)
		}
	}
	request := AggressorListenerRequest{
		Operation: spec.operation,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Bindings:  invocation.Bindings(),
		Arguments: values,
	}
	result, err := provider.HandleAggressorListener(ctx, request)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if !spec.returns {
		return Null(), nil
	}
	return result, nil
}

func requireAggressorListenerArguments(invocation Invocation, minimum, maximum int) error {
	count := len(invocation.Arguments)
	if count >= minimum && count <= maximum {
		return nil
	}
	if minimum == maximum {
		return requireExactAggressorClientArguments(invocation, minimum)
	}
	return fmt.Errorf("&%s: expected %d to %d argument(s), received %d",
		builtinName(invocation.Name), minimum, maximum, count)
}
