package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorPreferenceSpec struct {
	operation    AggressorPreferenceOperation
	arguments    int
	returnsValue bool
	listValue    bool
}

// Arity and the pref_set_list array constraint come from the current Fortra
// Aggressor function reference. That reference does not publish further input
// coercion rules, so the provider receives the remaining Values unchanged.
var aggressorPreferenceSpecs = map[string]aggressorPreferenceSpec{
	"pref_get":      {operation: AggressorPreferenceGet, arguments: 2, returnsValue: true},
	"pref_get_list": {operation: AggressorPreferenceGetList, arguments: 1, returnsValue: true},
	"pref_set":      {operation: AggressorPreferenceSet, arguments: 2},
	"pref_set_list": {operation: AggressorPreferenceSetList, arguments: 2, listValue: true},
}

// aggressorPreferenceFunctions returns native wrappers around the
// importer-owned Cobalt preference store. With no provider, every valid call
// preserves the original reference-bearing Host invocation exactly once.
func (r *Runtime) aggressorPreferenceFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorPreferenceSpecs))
	for name := range aggressorPreferenceSpecs {
		functions[name] = r.aggressorPreference
	}
	return functions
}

func (r *Runtime) aggressorPreference(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorPreferenceSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor preference operation",
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

	provider := r.aggressorPreferenceProvider
	if isNilInterface(provider) {
		if spec.listValue {
			if _, ok := invocation.Arg(1).Array(); !ok {
				return Null(), fmt.Errorf("&%s: argument 2 must be an array", builtinName(invocation.Name))
			}
		}
		// Preserve the original Invocation and its reference-bearing Arguments.
		// Host compatibility must observe and mutate the caller's exact Cells.
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
	if spec.listValue {
		if _, ok := values[1].Array(); !ok {
			return Null(), fmt.Errorf("&%s: argument 2 must be an array", builtinName(invocation.Name))
		}
	}
	request := AggressorPreferenceRequest{
		Operation:       spec.operation,
		RuntimeID:       r.ID(),
		Script:          invocation.Script,
		Span:            invocation.Span,
		PreferenceName:  values[0],
		DefaultValue:    Null(),
		PreferenceValue: Null(),
	}
	switch spec.operation {
	case AggressorPreferenceGet:
		request.DefaultValue = values[1]
	case AggressorPreferenceSet, AggressorPreferenceSetList:
		request.PreferenceValue = values[1]
	}
	result, err := provider.HandleAggressorPreference(ctx, request)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if !spec.returnsValue {
		return Null(), nil
	}
	return result, nil
}
