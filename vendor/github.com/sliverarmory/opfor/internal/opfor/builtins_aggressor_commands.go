package opfor

import (
	"context"
	"fmt"
)

func (r *Runtime) beaconCommandDescribe(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorCommandDescribe(invocation, AggressorCommandBeacon)
}

func (r *Runtime) beaconCommandDetail(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorCommandDetail(invocation, AggressorCommandBeacon)
}

func (r *Runtime) beaconCommandGroup(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorCommandGroup(invocation, AggressorCommandBeacon)
}

func (r *Runtime) beaconCommandRegister(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorCommandRegister(invocation, AggressorCommandBeacon)
}

func (r *Runtime) beaconCommands(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorCommandsList(invocation, AggressorCommandBeacon)
}

func (r *Runtime) sshCommandDescribe(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorCommandDescribe(invocation, AggressorCommandSSH)
}

func (r *Runtime) sshCommandDetail(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorCommandDetail(invocation, AggressorCommandSSH)
}

func (r *Runtime) sshCommandGroup(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorCommandGroup(invocation, AggressorCommandSSH)
}

func (r *Runtime) sshCommandRegister(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorCommandRegister(invocation, AggressorCommandSSH)
}

func (r *Runtime) sshCommands(_ context.Context, invocation Invocation) (Value, error) {
	return r.aggressorCommandsList(invocation, AggressorCommandSSH)
}

func (r *Runtime) aggressorCommandDescribe(invocation Invocation, kind AggressorCommandKind) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 1); err != nil {
		return Null(), err
	}
	metadata, exists := r.aggressorCommands.describe(kind, invocation.Arg(0).String())
	if !exists {
		// The official documentation does not specify missing-name behavior.
		// OPFOR's explicit provisional policy is to return Sleep's null scalar.
		return Null(), nil
	}
	return String(metadata.Description), nil
}

func (r *Runtime) aggressorCommandDetail(invocation Invocation, kind AggressorCommandKind) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 1); err != nil {
		return Null(), err
	}
	metadata, exists := r.aggressorCommands.describe(kind, invocation.Arg(0).String())
	if !exists {
		return Null(), nil
	}
	return String(metadata.Detail), nil
}

func (r *Runtime) aggressorCommandGroup(invocation Invocation, kind AggressorCommandKind) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 3); err != nil {
		return Null(), err
	}
	group := AggressorCommandGroup{
		ID:          invocation.Arg(0).String(),
		Name:        invocation.Arg(1).String(),
		Description: invocation.Arg(2).String(),
	}
	if err := validateAggressorCommandGroup(group.ID, group.Name); err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	if err := r.registerAggressorCommandGroup(invocation, kind, group); err != nil {
		return Null(), err
	}
	// Registration functions have no documented return contract. OPFOR uses
	// the null scalar as its explicit provisional side-effect-only result.
	return Null(), nil
}

func (r *Runtime) aggressorCommandRegister(invocation Invocation, kind AggressorCommandKind) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 3, 4); err != nil {
		return Null(), err
	}
	metadata := AggressorCommandMetadata{
		Name:        invocation.Arg(0).String(),
		Description: invocation.Arg(1).String(),
		Detail:      invocation.Arg(2).String(),
	}
	if metadata.Name == "" {
		return Null(), fmt.Errorf("&%s: command name is empty", builtinName(invocation.Name))
	}
	if len(invocation.Arguments) == 4 {
		metadata.GroupID = invocation.Arg(3).String()
	}
	if err := r.registerAggressorCommand(invocation, kind, metadata); err != nil {
		return Null(), err
	}
	return Null(), nil
}

func (r *Runtime) aggressorCommandsList(invocation Invocation, kind AggressorCommandKind) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 0); err != nil {
		return Null(), err
	}
	names := r.aggressorCommands.commandNames(kind)
	if err := reserveCollectionEntries(invocation.Runtime, len(names)); err != nil {
		return Null(), err
	}
	values := make([]Value, len(names))
	for index, name := range names {
		values[index] = String(name)
	}
	return ArrayValue(NewArray(values...)), nil
}

func requireAggressorCommandArguments(invocation Invocation, allowed ...int) error {
	// The documentation publishes these call shapes but does not specify how
	// extra or missing arguments are handled. Exact rejection is OPFOR's
	// explicit provisional policy rather than a licensed-runtime parity claim.
	for _, count := range allowed {
		if len(invocation.Arguments) == count {
			return nil
		}
	}
	if len(allowed) == 1 {
		return fmt.Errorf("&%s: expected exactly %d argument(s), received %d",
			builtinName(invocation.Name), allowed[0], len(invocation.Arguments))
	}
	return fmt.Errorf("&%s: expected %d or %d arguments, received %d",
		builtinName(invocation.Name), allowed[0], allowed[1], len(invocation.Arguments))
}
