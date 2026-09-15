package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorSiteSpec struct {
	kind         AggressorSiteKind
	minimum      int
	maximum      int
	returnsValue bool
}

var aggressorSiteSpecs = map[string]aggressorSiteSpec{
	"localip":   {kind: AggressorSiteLocalIP, minimum: 0, maximum: 0, returnsValue: true},
	"site_host": {kind: AggressorSiteHost, minimum: 6, maximum: 7, returnsValue: true},
	"site_kill": {kind: AggressorSiteKill, minimum: 2, maximum: 2},
	"sites":     {kind: AggressorSiteList, minimum: 0, maximum: 0, returnsValue: true},
}

// aggressorSiteFunctions returns native wrappers around the importer-owned
// Team Server site-delivery boundary. With no provider, every valid call
// preserves the original reference-bearing Host invocation exactly once.
func (r *Runtime) aggressorSiteFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorSiteSpecs))
	for name := range aggressorSiteSpecs {
		functions[name] = r.aggressorSite
	}
	return functions
}

func (r *Runtime) aggressorSite(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorSiteSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor site delivery",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorSiteArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorSiteProvider
	if isNilInterface(provider) {
		// Do not resolve, validate, copy, or replace any Argument on this
		// compatibility path. Existing Hosts retain the exact Invocation and its
		// pass-by-name or ordinary bare-variable Cell capabilities.
		result, err := r.host.Call(ctx, invocation)
		return result, preserveNativeBoundaryError(ctx, err)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	// Resolve every source reference exactly once. Value copies deliberately
	// retain binary provenance and compound identity.
	values := invocation.Values()
	request := AggressorSiteRequest{
		Kind:        spec.kind,
		Name:        invocation.Name,
		RuntimeID:   r.ID(),
		Script:      invocation.Script,
		Span:        invocation.Span,
		Bindings:    invocation.Bindings(),
		Host:        Null(),
		Port:        Null(),
		URI:         Null(),
		Content:     Null(),
		MIMEType:    Null(),
		Description: Null(),
		SSL:         Null(),
	}

	switch spec.kind {
	case AggressorSiteHost:
		request.Host = values[0]
		request.Port = values[1]
		request.URI = values[2]
		request.Content = values[3]
		request.MIMEType = values[4]
		request.Description = values[5]
		if len(values) == 7 {
			request.SSL = values[6]
			request.HasSSL = true
			request.SSLTruth = values[6].Truth()
		}
	case AggressorSiteKill:
		request.Port = values[0]
		request.URI = values[1]
	}

	result, err := provider.HandleAggressorSite(ctx, request)
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

func requireAggressorSiteArguments(invocation Invocation, minimum, maximum int) error {
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
