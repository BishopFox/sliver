package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorPayloadStoreSpec struct {
	operation         AggressorPayloadStoreOperation
	minimum           int
	maximum           int
	returns           bool
	optionalHashIndex int
}

var aggressorPayloadStoreSpecs = map[string]aggressorPayloadStoreSpec{
	"payloadstore_add":      {operation: AggressorPayloadStoreAdd, minimum: 5, maximum: 6, returns: true, optionalHashIndex: 5},
	"payloadstore_fetch":    {operation: AggressorPayloadStoreFetch, minimum: 1, maximum: 1, returns: true, optionalHashIndex: -1},
	"payloadstore_list":     {operation: AggressorPayloadStoreList, returns: true, optionalHashIndex: -1},
	"payloadstore_metadata": {operation: AggressorPayloadStoreMetadata, minimum: 1, maximum: 1, returns: true, optionalHashIndex: -1},
	"payloadstore_remove":   {operation: AggressorPayloadStoreRemove, minimum: 1, maximum: 1, optionalHashIndex: -1},
}

func (r *Runtime) aggressorPayloadStoreFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorPayloadStoreSpecs))
	for name := range aggressorPayloadStoreSpecs {
		functions[name] = r.aggressorPayloadStore
	}
	return functions
}

func (r *Runtime) aggressorPayloadStore(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorPayloadStoreSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor Payload Store operation",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorPayloadStoreArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorPayloadStoreProvider
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
	if spec.optionalHashIndex >= 0 && len(values) > spec.optionalHashIndex {
		if _, ok := values[spec.optionalHashIndex].Hash(); !ok {
			return Null(), fmt.Errorf("&%s: argument %d must be a hash",
				builtinName(invocation.Name), spec.optionalHashIndex+1)
		}
	}
	request := AggressorPayloadStoreRequest{
		Operation: spec.operation,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Arguments: values,
	}
	result, err := provider.HandleAggressorPayloadStore(ctx, request)
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

func requireAggressorPayloadStoreArguments(invocation Invocation, minimum, maximum int) error {
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
