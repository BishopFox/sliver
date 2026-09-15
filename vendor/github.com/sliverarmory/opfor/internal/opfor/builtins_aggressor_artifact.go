package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorArtifactSpec struct {
	kind    AggressorArtifactKind
	minimum int
	maximum int
}

var aggressorArtifactSpecs = map[string]aggressorArtifactSpec{
	"artifact_payload":   {kind: AggressorArtifactPayload, minimum: 5, maximum: 9},
	"artifact_stageless": {kind: AggressorArtifactStageless, minimum: 5, maximum: 5},
}

// aggressorArtifactFunctions returns native wrappers around the importer-owned
// artifact generation boundary. With no provider, every valid call preserves
// the original reference-bearing Host invocation exactly once.
func (r *Runtime) aggressorArtifactFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorArtifactSpecs))
	for name := range aggressorArtifactSpecs {
		functions[name] = r.aggressorArtifact
	}
	return functions
}

func (r *Runtime) aggressorArtifact(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorArtifactSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor artifact generation",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorArtifactArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorArtifactProvider
	if isNilInterface(provider) {
		// Do not resolve, validate, copy, or replace Arguments on this
		// compatibility path. In particular, Host retains pass-by-name Cells and
		// the raw callback-shaped fifth argument exactly as supplied.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	// Resolve each reference exactly once so every request field describes the
	// same observed argument snapshot without exposing source Cells.
	values := invocation.Values()
	request := AggressorArtifactRequest{
		Kind:         spec.kind,
		Name:         invocation.Name,
		RuntimeID:    r.ID(),
		Script:       invocation.Script,
		Span:         invocation.Span,
		Bindings:     invocation.Bindings(),
		Listener:     values[0],
		ArtifactType: values[1],
		Architecture: values[2],
	}

	switch spec.kind {
	case AggressorArtifactPayload:
		request.ExitMethod = values[3]
		request.SystemCallMethod = values[4]
		if len(values) > 5 {
			request.HTTPLibrary = values[5]
			request.HasHTTPLibrary = true
		}
		if len(values) > 6 {
			request.DNSCommMode = values[6]
			request.HasDNSCommMode = true
		}
		if len(values) > 7 {
			request.MalleableProfileOverride = values[7]
			request.HasMalleableProfileOverride = true
		}
		if len(values) > 8 {
			request.PayloadStoreInfo = values[8]
			request.HasPayloadStoreInfo = true
		}
	case AggressorArtifactStageless:
		request.ProxyConfiguration = values[3]
		callback, err := invocation.RetainCallback(values[4])
		if err != nil {
			if errors.Is(err, ErrInvalidCallable) {
				return Null(), fmt.Errorf("&%s: argument 5 is not callable: %w",
					builtinName(invocation.Name), err)
			}
			return Null(), err
		}
		request.Callback = callback
	}

	result, err := provider.GenerateAggressorArtifact(ctx, request)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if spec.kind == AggressorArtifactStageless {
		return Null(), nil
	}
	return result, nil
}

func requireAggressorArtifactArguments(invocation Invocation, minimum, maximum int) error {
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
