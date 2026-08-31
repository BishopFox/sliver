package opfor

import (
	"context"
	"errors"
	"fmt"
)

type aggressorVPNSpec struct {
	operation    AggressorVPNOperation
	minimum      int
	maximum      int
	returnsValue bool
}

var aggressorVPNSpecs = map[string]aggressorVPNSpec{
	"vpn_interface_info": {operation: AggressorVPNInterfaceInfo, minimum: 1, maximum: 2, returnsValue: true},
	"vpn_interfaces":     {operation: AggressorVPNInterfaces, minimum: 0, maximum: 0, returnsValue: true},
	"vpn_tap_create":     {operation: AggressorVPNTAPCreate, minimum: 5, maximum: 5},
	"vpn_tap_delete":     {operation: AggressorVPNTAPDelete, minimum: 1, maximum: 1},
}

// aggressorVPNFunctions returns native wrappers around the importer-owned
// Team Server Covert VPN boundary. With no provider, every valid call preserves
// the original reference-bearing Host invocation exactly once.
func (r *Runtime) aggressorVPNFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorVPNSpecs))
	for name := range aggressorVPNSpecs {
		functions[name] = r.aggressorVPN
	}
	return functions
}

func (r *Runtime) aggressorVPN(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorVPNSpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor VPN",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireAggressorVPNArguments(invocation, spec.minimum, spec.maximum); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorVPNProvider
	if isNilInterface(provider) {
		// Preserve source references and pass-by-name cells on the generic
		// compatibility route. A Host may implement a legacy adapter whose own
		// return convention differs from the typed side-effect-only route.
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
	request := AggressorVPNRequest{
		Operation: spec.operation,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
	}
	switch spec.operation {
	case AggressorVPNInterfaceInfo:
		request.Interface = values[0]
		if len(values) == 2 {
			request.Key = values[1]
			request.HasKey = true
		}
	case AggressorVPNTAPCreate:
		request.Interface = values[0]
		request.MACAddress = values[1]
		request.Reserved = values[2]
		request.Port = values[3]
		request.Channel = values[4]
	case AggressorVPNTAPDelete:
		request.Interface = values[0]
	}
	result, err := provider.HandleAggressorVPN(ctx, request)
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

func requireAggressorVPNArguments(invocation Invocation, minimum, maximum int) error {
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
