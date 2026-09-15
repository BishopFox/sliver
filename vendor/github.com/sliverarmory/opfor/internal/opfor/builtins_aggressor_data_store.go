package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorDataStoreSpec struct {
	operation     AggressorDataStoreOperation
	minimum       int
	maximum       int
	discardResult bool
}

// Arity comes from the current Fortra Aggressor function reference. The pinned
// official examples independently exercise credential_add with five arguments
// and tokenToEmail with one; no pinned example calls the remaining operations.
var aggressorDataStoreSpecs = map[string]aggressorDataStoreSpec{
	"credential_add": {operation: AggressorDataStoreCredentialAdd, minimum: 2, maximum: 7},
	"credentials":    {operation: AggressorDataStoreCredentials},
	"tokenToEmail":   {operation: AggressorDataStoreTokenToEmail, minimum: 1, maximum: 1},
	"applications":   {operation: AggressorDataStoreApplications},
	"archives":       {operation: AggressorDataStoreArchives},
	"downloads":      {operation: AggressorDataStoreDownloads},
	"highlight":      {operation: AggressorDataStoreHighlight, minimum: 3, maximum: 3},
	"keystrokes":     {operation: AggressorDataStoreKeystrokes},
	"screenshots":    {operation: AggressorDataStoreScreenshots},
	"services":       {operation: AggressorDataStoreServices},
	"targets":        {operation: AggressorDataStoreTargets},
	"hosts":          {operation: AggressorDataStoreHosts},
	"host_info":      {operation: AggressorDataStoreHostInfo, minimum: 1, maximum: 2},
	"host_update":    {operation: AggressorDataStoreHostUpdate, minimum: 4, maximum: 5},
	"host_delete":    {operation: AggressorDataStoreHostDelete, minimum: 1, maximum: 1},
	"resetData":      {operation: AggressorDataStoreResetData},
	"redactobject":   {operation: AggressorDataStoreRedactObject, minimum: 1, maximum: 1, discardResult: true},
}

// aggressorDataStoreFunctions returns native wrappers around the
// importer-owned Cobalt application data stores. With no provider, every
// valid call preserves the original reference-bearing Host invocation exactly
// once.
func (r *Runtime) aggressorDataStoreFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorDataStoreSpecs))
	for name := range aggressorDataStoreSpecs {
		functions[name] = r.aggressorDataStore
	}
	return functions
}

func (r *Runtime) aggressorDataStore(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorDataStoreSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor data-store operation",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorDataStoreArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorDataStoreProvider
	if isNilInterface(provider) {
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

	request := AggressorDataStoreRequest{
		Operation: spec.operation,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Arguments: invocation.Values(),
	}
	result, err := provider.HandleAggressorDataStore(ctx, request)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if spec.discardResult {
		return Null(), nil
	}
	return result, nil
}

func requireAggressorDataStoreArguments(invocation Invocation, minimum, maximum int) error {
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
