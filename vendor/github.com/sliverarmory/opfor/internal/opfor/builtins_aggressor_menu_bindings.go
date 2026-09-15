package opfor

import (
	"context"
	"errors"
	"fmt"
)

// registerMenuItem implements item(description, callback), the function-form
// counterpart to `item description { ... }`. It is evaluated while a popup or
// submenu body is being composed and publishes the callback through the same
// script-owned BindingItem registry as the declaration form.
func (r *Runtime) registerMenuItem(ctx context.Context, invocation Invocation) (Value, error) {
	return r.registerMenuEntry(ctx, invocation, BindingItem, "item")
}

// registerSubmenu implements menu(description, callback), the function-form
// counterpart to `menu description { ... }`. Invoking the resulting
// BindingMenu lazily composes its callback beneath the captured parent menu.
func (r *Runtime) registerSubmenu(ctx context.Context, invocation Invocation) (Value, error) {
	return r.registerMenuEntry(ctx, invocation, BindingMenu, "menu")
}

func (r *Runtime) registerMenuEntry(
	ctx context.Context,
	invocation Invocation,
	kind BindingKind,
	keyword string,
) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 2); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	parent := currentBindingInvocation(ctx)
	if parent == nil || !isCompositionBinding(parent.Kind) {
		return Null(), fmt.Errorf("&%s: requires an active popup or menu composition", builtinName(invocation.Name))
	}

	description := invocation.Arg(0)
	name := description.String()
	if name == "" {
		return Null(), fmt.Errorf("&%s: menu description is empty", builtinName(invocation.Name))
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
		Lifetime:    BindingPersistent,
		Environment: EnvironmentOrdinary,
		Name:        name,
		Span:        invocation.Span,
		Selectors: []BindingSelector{{
			Raw:       name,
			Value:     description,
			Evaluated: true,
			Span:      invocation.Span,
		}},
	}
	if err := owner.registerBinding(ctx, binding, callback); err != nil {
		return Null(), err
	}
	return Null(), nil
}

// registerKeyBinding implements bind(shortcut, callback), the documented
// function-form alternate to `bind shortcut { ... }`. It publishes through the
// same layered, script-owned BindingKey registry as declaration form: the most
// recently registered active layer wins and unloading it reveals the layer
// below it.
func (r *Runtime) registerKeyBinding(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 2); err != nil {
		return Null(), err
	}
	values := invocation.Values()
	shortcut := values[0].String()
	if shortcut == "" {
		return Null(), fmt.Errorf("&%s: keyboard shortcut is empty", builtinName(invocation.Name))
	}
	callback, err := invocation.RetainCallback(values[1])
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
		Kind:        BindingKey,
		Keyword:     "bind",
		Lifetime:    BindingPersistent,
		Environment: EnvironmentOrdinary,
		Name:        shortcut,
		Span:        invocation.Span,
		Selectors: []BindingSelector{{
			Raw:       shortcut,
			Value:     values[0],
			Evaluated: true,
			Span:      invocation.Span,
		}},
	}
	if err := owner.registerBinding(ctx, binding, callback); err != nil {
		return Null(), err
	}
	return Null(), nil
}

// clearKeyBinding implements unbind(shortcut). Removing every active exact
// layer restores importer/default behavior while reverse-order notification
// mirrors the registry's ordinary unload semantics. A concurrent bind that
// publishes after the snapshot is a later operation and remains active.
func (r *Runtime) clearKeyBinding(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireAggressorCommandArguments(invocation, 1); err != nil {
		return Null(), err
	}
	shortcut := invocation.Arg(0).String()
	if shortcut == "" {
		return Null(), fmt.Errorf("&%s: keyboard shortcut is empty", builtinName(invocation.Name))
	}
	bindings := r.Bindings(BindingKey, shortcut)
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
	return Null(), result
}

// insertMenu implements insert_menu(popupHook, ...arguments). It is a purely
// runtime-local composition operation: every exact popup layer selected at the
// call site is pinned, then invoked in registration order beneath the current
// popup/menu invocation. Additional arguments become the child popup's $1...
// values without crossing Host or the client-UI provider boundary.
func (r *Runtime) insertMenu(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireSleepBuiltinArguments(invocation, 1); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	values := invocation.Values()
	hook := values[0].String()
	if hook == "" {
		return Null(), fmt.Errorf("&%s: popup hook is empty", builtinName(invocation.Name))
	}
	parent := cloneBindingInvocation(currentBindingInvocation(ctx))
	if parent == nil || !isCompositionBinding(parent.Kind) {
		return Null(), fmt.Errorf("&%s: requires an active popup or menu composition", builtinName(invocation.Name))
	}
	bindings := r.Bindings(BindingPopup, hook)
	if len(bindings) == 0 {
		return Null(), nil
	}
	composer := newAggressorPopupComposer(r, invocation, bindings, values[1:], parent)
	return Null(), composer.Compose(ctx)
}
