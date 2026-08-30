package opfor

import (
	"context"
	"errors"
)

type aggressorProcessInjectionSpec struct {
	operation    AggressorProcessInjectionOperation
	arguments    int
	returnsValue bool
}

// Arity and query/effect classification come from the current Fortra
// Aggressor function reference. It does not publish additional input coercion
// rules, so setter names reach the provider as their resolved Value.
var aggressorProcessInjectionSpecs = map[string]aggressorProcessInjectionSpec{
	"pi_explicit_get":            {operation: AggressorProcessInjectionExplicitGet, returnsValue: true},
	"pi_explicit_info":           {operation: AggressorProcessInjectionExplicitInfo, returnsValue: true},
	"pi_explicit_set":            {operation: AggressorProcessInjectionExplicitSet, arguments: 1},
	"pi_spawn_get":               {operation: AggressorProcessInjectionSpawnGet, returnsValue: true},
	"pi_spawn_info":              {operation: AggressorProcessInjectionSpawnInfo, returnsValue: true},
	"pi_spawn_set":               {operation: AggressorProcessInjectionSpawnSet, arguments: 1},
	"pi_user_explicit_clear":     {operation: AggressorProcessInjectionUserExplicitClear},
	"pi_user_explicit_get":       {operation: AggressorProcessInjectionUserExplicitGet, returnsValue: true},
	"pi_user_explicit_get_map":   {operation: AggressorProcessInjectionUserExplicitGetMap, returnsValue: true},
	"pi_user_explicit_get_names": {operation: AggressorProcessInjectionUserExplicitGetNames, returnsValue: true},
	"pi_user_explicit_set":       {operation: AggressorProcessInjectionUserExplicitSet, arguments: 1},
	"pi_user_spawn_clear":        {operation: AggressorProcessInjectionUserSpawnClear},
	"pi_user_spawn_get":          {operation: AggressorProcessInjectionUserSpawnGet, returnsValue: true},
	"pi_user_spawn_get_map":      {operation: AggressorProcessInjectionUserSpawnGetMap, returnsValue: true},
	"pi_user_spawn_get_names":    {operation: AggressorProcessInjectionUserSpawnGetNames, returnsValue: true},
	"pi_user_spawn_set":          {operation: AggressorProcessInjectionUserSpawnSet, arguments: 1},
}

// aggressorProcessInjectionFunctions returns native wrappers around the
// importer-owned process-injection configuration boundary. With no provider,
// every valid call preserves the original reference-bearing Host invocation
// exactly once.
func (r *Runtime) aggressorProcessInjectionFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorProcessInjectionSpecs))
	for name := range aggressorProcessInjectionSpecs {
		functions[name] = r.aggressorProcessInjection
	}
	return functions
}

func (r *Runtime) aggressorProcessInjection(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorProcessInjectionSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor process-injection operation",
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

	provider := r.aggressorProcessInjectionProvider
	if isNilInterface(provider) {
		// Preserve the raw Invocation on the compatibility route, including
		// pass-by-name and ordinary bare-variable Cell capabilities.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	request := AggressorProcessInjectionRequest{
		Operation: spec.operation,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Bindings:  invocation.Bindings(),
	}
	if spec.arguments != 0 {
		request.SelectionName = invocation.Arg(0)
	}
	result, err := provider.HandleAggressorProcessInjection(ctx, request)
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
