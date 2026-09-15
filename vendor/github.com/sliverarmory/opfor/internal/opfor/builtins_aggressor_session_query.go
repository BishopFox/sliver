package opfor

import (
	"context"
	"errors"
)

type aggressorSessionQuerySpec struct {
	kind        AggressorSessionQueryKind
	arity       int
	predicate   bool
	sessionID   bool
	metadataKey bool
}

var aggressorSessionQuerySpecs = map[string]aggressorSessionQuerySpec{
	"beacons":     {kind: AggressorSessionQueryBeacons, arity: 0},
	"beacon_ids":  {kind: AggressorSessionQueryBeaconIDs, arity: 0},
	"bdata":       {kind: AggressorSessionQueryBeaconData, arity: 1, sessionID: true},
	"beacon_data": {kind: AggressorSessionQueryBeaconData, arity: 1, sessionID: true},
	"binfo":       {kind: AggressorSessionQueryBeaconInfo, arity: 2, sessionID: true, metadataKey: true},
	"beacon_info": {kind: AggressorSessionQueryBeaconInfo, arity: 2, sessionID: true, metadataKey: true},
	"barch":       {kind: AggressorSessionQueryBeaconArchitecture, arity: 1, sessionID: true},
	"-is64":       {kind: AggressorSessionQueryIs64, arity: 1, predicate: true, sessionID: true},
	"-isactive":   {kind: AggressorSessionQueryIsActive, arity: 1, predicate: true, sessionID: true},
	"-isadmin":    {kind: AggressorSessionQueryIsAdmin, arity: 1, predicate: true, sessionID: true},
	"-isbeacon":   {kind: AggressorSessionQueryIsBeacon, arity: 1, predicate: true, sessionID: true},
	"-isssh":      {kind: AggressorSessionQueryIsSSH, arity: 1, predicate: true, sessionID: true},
}

// aggressorSessionQueryFunctions returns the native wrappers around the
// importer-owned Beacon/session metadata provider. With no provider, every
// valid call preserves the pre-wrapper Host path exactly once.
func (r *Runtime) aggressorSessionQueryFunctions() map[string]NativeFunc {
	functions := make(map[string]NativeFunc, len(aggressorSessionQuerySpecs))
	for name := range aggressorSessionQuerySpecs {
		functions[name] = r.aggressorSessionQuery
	}
	return functions
}

func (r *Runtime) aggressorSessionQuery(ctx context.Context, invocation Invocation) (Value, error) {
	spec, exists := aggressorSessionQuerySpecs[invocation.Name]
	if !exists {
		return Null(), &UnsupportedError{
			Operation: "Aggressor session query",
			Name:      invocation.Name,
			Span:      invocation.Span,
		}
	}
	if err := requireExactAggressorClientArguments(invocation, spec.arity); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}

	provider := r.aggressorSessionQueryProvider
	if isNilInterface(provider) {
		// Do not snapshot references or otherwise rewrite the invocation on the
		// compatibility path. Existing Hosts retain their exact arguments and
		// result/error behavior.
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
	query := AggressorSessionQuery{
		Kind:      spec.kind,
		Name:      invocation.Name,
		RuntimeID: r.ID(),
		Script:    invocation.Script,
		Span:      invocation.Span,
	}
	if spec.sessionID {
		query.SessionID = values[0]
	}
	if spec.metadataKey {
		query.Key = values[1]
	}

	result, err := provider.QueryAggressorSession(ctx, query)
	if err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if spec.predicate {
		return Bool(result.Truth()), nil
	}
	if spec.kind == AggressorSessionQueryBeaconArchitecture && (result.IsNull() || result.String() == "") {
		return String("x86"), nil
	}
	return result, nil
}
