package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorPEProviderSpec struct {
	operation AggressorPEOperation
	minimum   int
	maximum   int
}

func fixedAggressorPEProviderSpec(operation AggressorPEOperation, arguments int) aggressorPEProviderSpec {
	return aggressorPEProviderSpec{operation: operation, minimum: arguments, maximum: arguments}
}

// The current function reference states these positional shapes and
// value-returning contracts. pe_set_export_name deliberately accepts one or
// two arguments because its Arguments table lists only DLL content while both
// executable examples supply content and a second export-name value. This is
// an evidence-union routing policy, not a claim that a licensed runtime accepts
// both forms.
var aggressorPEProviderSpecs = map[string]aggressorPEProviderSpec{
	"pe_insert_rich_header":           fixedAggressorPEProviderSpec(AggressorPEInsertRichHeader, 2),
	"pe_mask_section":                 fixedAggressorPEProviderSpec(AggressorPEMaskSection, 3),
	"pe_patch_code":                   fixedAggressorPEProviderSpec(AggressorPEPatchCode, 3),
	"pe_remove_rich_header":           fixedAggressorPEProviderSpec(AggressorPERemoveRichHeader, 1),
	"pe_set_compile_time_with_string": fixedAggressorPEProviderSpec(AggressorPESetCompileTimeWithString, 2),
	"pe_set_export_name":              {operation: AggressorPESetExportName, minimum: 1, maximum: 2},
	"pe_set_value_at":                 fixedAggressorPEProviderSpec(AggressorPESetValueAt, 3),
	"pedump":                          fixedAggressorPEProviderSpec(AggressorPEDump, 1),
}

func (r *Runtime) aggressorPEProviderFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorPEProviderSpecs))
	for name := range aggressorPEProviderSpecs {
		functions[name] = r.aggressorPEProviderCall
	}
	return functions
}

func (r *Runtime) aggressorPEProviderCall(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorPEProviderSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor PE provider operation",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorPEProviderArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorPEProvider
	if isNilInterface(provider) {
		// Preserve the exact reference-bearing Invocation for existing Cobalt
		// adapters. Their result and error conventions remain unchanged.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	request := AggressorPERequest{
		Operation: spec.operation,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Arguments: invocation.Values(),
	}
	result, err := provider.HandleAggressorPE(ctx, request)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return result, nil
}

func requireAggressorPEProviderArguments(invocation Invocation, minimum, maximum int) error {
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
