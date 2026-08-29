package opfor

import (
	"context"
	"errors"
)

type aggressorDataModelQuerySpec struct {
	kind  AggressorDataModelQueryKind
	keyed bool
}

var aggressorDataModelQuerySpecs = map[string]aggressorDataModelQuerySpec{
	"data_keys":  {kind: AggressorDataModelQueryKeys},
	"data_query": {kind: AggressorDataModelQueryValue, keyed: true},
	"pivots":     {kind: AggressorDataModelQueryPivots},
}

// aggressorDataModelQueryFunctions returns native wrappers around the
// importer-owned Cobalt data model. With no provider, a valid call preserves
// the pre-wrapper Host path exactly once.
func (r *Runtime) aggressorDataModelQueryFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorDataModelQuerySpecs))
	for name := range aggressorDataModelQuerySpecs {
		functions[name] = r.aggressorDataModelQuery
	}
	return functions
}

func (r *Runtime) aggressorDataModelQuery(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorDataModelQuerySpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor data-model query",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	arity := 0
	if spec.keyed {
		arity = 1
	}
	if err := requireExactAggressorClientArguments(invocation, arity); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorDataModelQueryProvider
	if isNilInterface(provider) {
		// Preserve the original reference-bearing invocation. Existing Hosts keep
		// their exact argument capabilities and result/error behavior.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	query := AggressorDataModelQuery{
		Kind:      spec.kind,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Key:       Null(),
	}
	if spec.keyed {
		query.Key = invocation.Arg(0)
	}

	result, err := provider.QueryAggressorDataModel(ctx, query)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return result, nil
}
