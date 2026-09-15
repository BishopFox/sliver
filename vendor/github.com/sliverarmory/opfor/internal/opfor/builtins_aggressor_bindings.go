package opfor

import (
	"context"
	"errors"
	"fmt"
)

// namedBindingCallable is the private bridge used by console bindings. Sleep
// keeps $0 as a named closure message rather than an ordinary positional
// argument; both compiled closures and invocation-retained callbacks preserve
// that distinction.
type namedBindingCallable interface {
	invokeNamed(context.Context, string, ...Value) (Value, error)
}

// registerEvent implements the documented on(name, callback) function form.
// It shares the same script-owned registry as the on name { ... } environment
// form, so event ordering, observer notification, and unload revocation remain
// identical.
func (r *Runtime) registerEvent(ctx context.Context, invocation Invocation) (Value, error) {
	return r.registerAggressorCallback(ctx, invocation, BindingEvent, "on", BindingPersistent)
}

// registerWhen implements both the historical Cortana and current Aggressor
// when(name, callback) function form. It uses the ordinary event registry, but
// marks the registration for atomic consumption by the next matching event.
func (r *Runtime) registerWhen(ctx context.Context, invocation Invocation) (Value, error) {
	return r.registerAggressorCallback(ctx, invocation, BindingEvent, "when", BindingOnce)
}

// registerAlias implements the documented alias(name, callback) function
// form. The callback is retained as a capability owned by the calling script;
// it therefore cannot outlive that script even when its underlying Callable
// came from an importer.
func (r *Runtime) registerAlias(ctx context.Context, invocation Invocation) (Value, error) {
	return r.registerAggressorCallback(ctx, invocation, BindingAlias, "alias", BindingPersistent)
}

// registerSSHAlias implements the function form permitted by the official SSH
// sessions guide. The guide documents its callback ABI but does not list the
// function arguments separately; OPFOR infers the symmetric
// ssh_alias(name, callback) mapping from alias. It deliberately shares the same
// script-owned registry and invocation ABI as ssh_alias name { ... }.
func (r *Runtime) registerSSHAlias(ctx context.Context, invocation Invocation) (Value, error) {
	return r.registerAggressorCallback(ctx, invocation, BindingSSHAlias, "ssh_alias", BindingPersistent)
}

// clearAlias implements alias_clear(name). The reference documentation says
// this restores default Beacon command behavior when one exists. OPFOR removes
// every active Beacon alias layer for the exact name, allowing an embedding
// console to resume its own default dispatch; SSH aliases and help metadata are
// independent and deliberately untouched.
func (r *Runtime) clearAlias(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 1); err != nil {
		return Null(), err
	}
	name := invocation.Arg(0).String()
	if name == "" {
		return Null(), fmt.Errorf("&%s: alias name is empty", builtinName(invocation.Name))
	}
	bindings := r.Bindings(BindingAlias, name)
	var result error
	for index := len(bindings) - 1; index >= 0; index-- {
		binding := bindings[index]
		r.mu.RLock()
		owner := r.scripts[binding.Script]
		r.mu.RUnlock()
		if owner == nil || !owner.removeBindingIfPresent(binding) {
			continue
		}
		if r.observer != nil {
			if err := r.observer.Unregistered(ctx, cloneBinding(binding)); err != nil {
				result = errors.Join(result, preserveNativeBoundaryError(ctx, err))
			}
		}
	}
	// alias_clear has no documented return contract. OPFOR provisionally uses
	// the null scalar for this side-effect-only function.
	return Null(), result
}

func (r *Runtime) registerAggressorCallback(
	ctx context.Context,
	invocation Invocation,
	kind BindingKind,
	keyword string,
	lifetime BindingLifetime,
) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 2); err != nil {
		return Null(), err
	}
	name := invocation.Arg(0).String()
	if name == "" {
		return Null(), fmt.Errorf("&%s: binding name is empty", builtinName(invocation.Name))
	}
	callback, err := invocation.Callback(1)
	if err != nil {
		if errors.Is(err, ErrInvalidCallable) {
			return Null(), fmt.Errorf("&%s: argument 2 is not callable: %w", builtinName(invocation.Name), err)
		}
		return Null(), err
	}
	owner := r.script(invocation.Script)
	if owner == nil {
		return Null(), ErrScriptUnloaded
	}
	binding := Binding{
		Kind:        kind,
		Keyword:     keyword,
		Lifetime:    lifetime,
		Environment: EnvironmentOrdinary,
		Name:        name,
		Span:        invocation.Span,
		Selectors: []BindingSelector{{
			Raw:       name,
			Value:     invocation.Arg(0),
			Evaluated: true,
			Span:      invocation.Span,
		}},
	}
	if err := owner.registerBinding(ctx, binding, callback); err != nil {
		return Null(), err
	}
	// The official function reference specifies the registration side effect
	// but no return contract. OPFOR follows the other Aggressor registration
	// helpers and returns Sleep's empty scalar.
	return Null(), nil
}

// fireAlias implements fireAlias(beaconID, aliasName, arguments). The third
// argument is the same raw argument tail a user would type after the alias
// name; InvokeConsole then applies the ordinary alias ABI ($0 is the complete
// line, $1 is the session ID, and parsed tokens begin at $2).
func (r *Runtime) fireAlias(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 3); err != nil {
		return Null(), err
	}
	name := invocation.Arg(1).String()
	if name == "" {
		return Null(), fmt.Errorf("&%s: alias name is empty", builtinName(invocation.Name))
	}
	rawInput := name
	if arguments := invocation.Arg(2).String(); arguments != "" {
		rawInput += " " + arguments
	}
	_, err := r.InvokeConsole(ctx, ConsoleInvocation{
		Kind:      BindingAlias,
		Name:      name,
		RawInput:  rawInput,
		SessionID: invocation.Arg(0),
	})
	// The official function reference documents execution, not a return value.
	// Match fireEvent's side-effect-only contract and discard the callback's
	// result.
	return Null(), err
}
