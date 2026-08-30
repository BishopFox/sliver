package opfor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type aggressorPayloadSpec struct {
	operation           AggressorPayloadOperation
	minimum             int
	maximum             int
	predicate           bool
	optionalHashIndex   int
	argumentConstraints []AggressorArgumentConstraint
}

var aggressorAllPayloadsArgumentConstraints = []AggressorArgumentConstraint{
	{Position: 3, Kind: "enum", Values: []string{"None", "Direct", "Indirect"}},
	{Position: 4, Kind: "enum", Values: []string{"wininet", "winhttp", "$null", ""}},
	{Position: 5, Kind: "enum", Values: []string{"dns", "dns_over_https", "$null", ""}},
}

// Arity comes from the current Fortra function reference. Two entries are
// internally inconsistent: artifact's argument table has four positions while
// its official example supplies two, and powershell's table has three while
// its official example supplies two. Treating the trailing positions as
// optional accepts every published form without inventing omitted values.
var aggressorPayloadSpecs = map[string]aggressorPayloadSpec{
	"-hasbootstraphint":      {operation: AggressorPayloadHasBootstrapHint, minimum: 1, maximum: 1, predicate: true, optionalHashIndex: -1},
	"all_payloads":           {operation: AggressorPayloadGenerateAll, minimum: 3, maximum: 6, optionalHashIndex: -1, argumentConstraints: aggressorAllPayloadsArgumentConstraints},
	"artifact":               {operation: AggressorPayloadArtifact, minimum: 2, maximum: 4, optionalHashIndex: -1},
	"artifact_general":       {operation: AggressorPayloadArtifactGeneral, minimum: 3, maximum: 3, optionalHashIndex: -1},
	"artifact_sign":          {operation: AggressorPayloadArtifactSign, minimum: 1, maximum: 1, optionalHashIndex: -1},
	"artifact_stager":        {operation: AggressorPayloadArtifactStager, minimum: 3, maximum: 4, optionalHashIndex: 3},
	"payload":                {operation: AggressorPayloadExport, minimum: 4, maximum: 7, optionalHashIndex: -1},
	"payload_bootstrap_hint": {operation: AggressorPayloadBootstrapHint, minimum: 2, maximum: 2, optionalHashIndex: -1},
	"payload_local":          {operation: AggressorPayloadExportLocal, minimum: 5, maximum: 6, optionalHashIndex: -1},
	"powershell":             {operation: AggressorPayloadPowerShell, minimum: 2, maximum: 3, optionalHashIndex: -1},
	"shellcode":              {operation: AggressorPayloadShellcode, minimum: 3, maximum: 3, optionalHashIndex: -1},
	"stager":                 {operation: AggressorPayloadStager, minimum: 2, maximum: 2, optionalHashIndex: -1},
	"stager_bind_pipe":       {operation: AggressorPayloadStagerBindPipe, minimum: 1, maximum: 1, optionalHashIndex: -1},
	"stager_bind_tcp":        {operation: AggressorPayloadStagerBindTCP, minimum: 3, maximum: 3, optionalHashIndex: -1},
}

// aggressorPayloadFunctions returns native wrappers around importer-owned
// payload generation. With no provider, every valid call preserves the raw,
// reference-bearing Host invocation exactly once.
func (r *Runtime) aggressorPayloadFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorPayloadSpecs))
	for name := range aggressorPayloadSpecs {
		functions[name] = r.aggressorPayload
	}
	return functions
}

func (r *Runtime) aggressorPayload(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorPayloadSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor payload operation",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorPayloadArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorPayloadProvider
	if isNilInterface(provider) {
		// Do not resolve, validate, copy, or replace any Argument on the
		// compatibility path. Existing Hosts retain pass-by-name Cells exactly.
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
	if err := validateAggressorPayloadArgumentConstraints(invocation.Name, values, spec.argumentConstraints); err != nil {
		return Null(), err
	}
	if spec.optionalHashIndex >= 0 && len(values) > spec.optionalHashIndex {
		if _, ok := values[spec.optionalHashIndex].Hash(); !ok {
			return Null(), fmt.Errorf("&%s: argument %d must be a hash",
				builtinName(invocation.Name), spec.optionalHashIndex+1)
		}
	}
	request := AggressorPayloadRequest{
		Operation: spec.operation,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
		Bindings:  invocation.Bindings(),
		Arguments: values,
	}
	result, err := provider.HandleAggressorPayload(ctx, request)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if spec.predicate {
		return Bool(result.Truth()), nil
	}
	return result, nil
}

func validateAggressorPayloadArgumentConstraints(
	name string,
	values []Value,
	constraints []AggressorArgumentConstraint,
) error {
	for _, constraint := range constraints {
		if constraint.Kind != "enum" {
			return fmt.Errorf("opfor: unsupported Aggressor payload constraint kind %q", constraint.Kind)
		}
		index := constraint.Position - 1
		if index < 0 || index >= len(values) {
			continue
		}
		value := values[index]
		matched := false
		for _, expected := range constraint.Values {
			switch expected {
			case "$null":
				matched = value.IsNull()
			case "":
				matched = !value.IsNull() && value.String() == ""
			default:
				matched = !value.IsNull() && value.String() == expected
			}
			if matched {
				break
			}
		}
		if matched {
			continue
		}

		allowed := make([]string, len(constraint.Values))
		for index, expected := range constraint.Values {
			switch expected {
			case "$null":
				allowed[index] = "$null"
			case "":
				allowed[index] = "blank string"
			default:
				allowed[index] = strconv.Quote(expected)
			}
		}
		received := value.Describe()
		return fmt.Errorf("&%s: argument %d must be one of %s; received %s",
			builtinName(name), constraint.Position, strings.Join(allowed, ", "), received)
	}
	return nil
}

func requireAggressorPayloadArguments(invocation Invocation, minimum, maximum int) error {
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
