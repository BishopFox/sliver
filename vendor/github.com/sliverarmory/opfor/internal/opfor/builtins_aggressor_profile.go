package opfor

import (
	"context"
	"errors"
)

type aggressorProfileSpec struct {
	operation AggressorProfileOperation
	arguments int
}

var aggressorProfileSpecs = map[string]aggressorProfileSpec{
	"killdate":              {operation: AggressorProfileKillDate, arguments: 0},
	"setup_strings":         {operation: AggressorProfileSetupStrings, arguments: 1},
	"setup_transformations": {operation: AggressorProfileSetupTransformations, arguments: 2},
}

// aggressorProfileFunctions returns native wrappers around the importer-owned
// Team Server/profile boundary. With no provider, every valid call preserves
// the original reference-bearing Host invocation exactly once.
func (r *Runtime) aggressorProfileFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorProfileSpecs))
	for name := range aggressorProfileSpecs {
		functions[name] = r.aggressorProfile
	}
	return functions
}

func (r *Runtime) aggressorProfile(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorProfileSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor profile",
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

	provider := r.aggressorProfileProvider
	if isNilInterface(provider) {
		// Preserve the raw Invocation on the compatibility path. Existing Hosts
		// continue to receive pass-by-name and ordinary bare-variable Cell
		// capabilities, and their result/error behavior is unchanged.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	request := AggressorProfileRequest{
		Operation: spec.operation,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
	}
	values := invocation.Values()
	switch spec.operation {
	case AggressorProfileSetupStrings:
		request.Payload = values[0]
	case AggressorProfileSetupTransformations:
		request.Payload = values[0]
		request.Architecture = values[1]
	}
	result, err := provider.HandleAggressorProfile(ctx, request)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return result, nil
}
