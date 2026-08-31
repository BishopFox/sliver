package opfor

import (
	"context"
	"fmt"
)

func (r *Runtime) beaconElevatorDescribe(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniqueDescribe(invocation, AggressorBeaconTechniqueElevator)
}

func (r *Runtime) beaconElevatorRegister(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniqueRegister(invocation, AggressorBeaconTechniqueElevator)
}

func (r *Runtime) beaconElevators(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniquesList(invocation, AggressorBeaconTechniqueElevator)
}

func (r *Runtime) beaconExploitDescribe(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniqueDescribe(invocation, AggressorBeaconTechniqueExploit)
}

func (r *Runtime) beaconExploitRegister(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniqueRegister(invocation, AggressorBeaconTechniqueExploit)
}

func (r *Runtime) beaconExploits(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniquesList(invocation, AggressorBeaconTechniqueExploit)
}

func (r *Runtime) beaconRemoteExecMethodDescribe(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniqueDescribe(invocation, AggressorBeaconTechniqueRemoteExecMethod)
}

func (r *Runtime) beaconRemoteExecMethodRegister(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniqueRegister(invocation, AggressorBeaconTechniqueRemoteExecMethod)
}

func (r *Runtime) beaconRemoteExecMethods(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniquesList(invocation, AggressorBeaconTechniqueRemoteExecMethod)
}

func (r *Runtime) beaconRemoteExploitArch(_ context.Context, invocation Invocation) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 1); err != nil {
		return Null(), err
	}
	metadata, exists := r.aggressorBeaconTechniques.describe(
		AggressorBeaconTechniqueRemoteExploit,
		invocation.Arg(0).String(),
	)
	if !exists {
		// The official documentation does not specify missing-name behavior.
		// OPFOR's explicit provisional policy is to return Sleep's null scalar.
		return Null(), nil
	}
	return String(metadata.Architecture), nil
}

func (r *Runtime) beaconRemoteExploitDescribe(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniqueDescribe(invocation, AggressorBeaconTechniqueRemoteExploit)
}

func (r *Runtime) beaconRemoteExploitRegister(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniqueRegister(invocation, AggressorBeaconTechniqueRemoteExploit)
}

func (r *Runtime) beaconRemoteExploits(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorBeaconTechniquesList(invocation, AggressorBeaconTechniqueRemoteExploit)
}

func (r *Runtime) aggressorBeaconTechniqueDescribe(
	invocation Invocation,
	kind AggressorBeaconTechniqueKind,
) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 1); err != nil {
		return Null(), err
	}
	metadata, exists := r.aggressorBeaconTechniques.describe(kind, invocation.Arg(0).String())
	if !exists {
		return Null(), nil
	}
	return String(metadata.Description), nil
}

func (r *Runtime) aggressorBeaconTechniqueRegister(
	invocation Invocation,
	kind AggressorBeaconTechniqueKind,
) (Value, error) {
	want := 3
	callbackIndex := 2
	if kind == AggressorBeaconTechniqueRemoteExploit {
		want = 4
		callbackIndex = 3
	}
	if err := requireAggressorCommandArguments(invocation, want); err != nil {
		return Null(), err
	}
	metadata := AggressorBeaconTechniqueMetadata{
		Name: invocation.Arg(0).String(),
	}
	if kind == AggressorBeaconTechniqueRemoteExploit {
		metadata.Architecture = invocation.Arg(1).String()
		metadata.Description = invocation.Arg(2).String()
	} else {
		metadata.Description = invocation.Arg(1).String()
	}
	if err := validateAggressorBeaconTechniqueMetadata(kind, metadata); err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	callback, err := invocation.Callback(callbackIndex)
	if err != nil {
		return Null(), fmt.Errorf("&%s: argument %d is not callable: %w", builtinName(invocation.Name), callbackIndex+1, err)
	}
	if err := r.registerAggressorBeaconTechnique(invocation, kind, metadata, callback); err != nil {
		return Null(), err
	}
	// Registration functions have no documented return contract. OPFOR uses
	// the null scalar as its explicit provisional side-effect-only result.
	return Null(), nil
}

func (r *Runtime) aggressorBeaconTechniquesList(
	invocation Invocation,
	kind AggressorBeaconTechniqueKind,
) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 0); err != nil {
		return Null(), err
	}
	names := r.aggressorBeaconTechniques.names(kind)
	if err := reserveCollectionEntries(invocation.Runtime, len(names)); err != nil {
		return Null(), err
	}
	values := make([]Value, len(names))
	for index, name := range names {
		values[index] = String(name)
	}
	return ArrayValue(NewArray(values...)), nil
}
