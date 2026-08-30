package opfor

import (
	"context"
	"errors"
	"fmt"
)

// BindingByID returns one active registration by its owning script and
// script-local binding ID. The returned metadata is detached in the same way
// as Bindings; its callback remains owned by the script lifetime.
func (r *Runtime) BindingByID(scriptID ScriptID, bindingID uint64) (Binding, bool) {
	if r == nil || scriptID == 0 || bindingID == 0 {
		return Binding{}, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, byName := range r.bindings {
		for _, entries := range byName {
			for _, binding := range entries {
				if binding.Script == scriptID && binding.ID == bindingID {
					return cloneBinding(binding), true
				}
			}
		}
	}
	return Binding{}, false
}

// InvokeBindingByID invokes one exact active registration. This avoids the
// newest-by-name selection performed by InvokeBinding when an embedding host
// retained a specific Binding from Bindings or a BindingObserver callback.
func (r *Runtime) InvokeBindingByID(
	ctx context.Context,
	scriptID ScriptID,
	bindingID uint64,
	arguments ...Value,
) (Value, error) {
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	binding, ok := r.BindingByID(scriptID, bindingID)
	if !ok {
		return Null(), &UnsupportedError{
			Operation: "binding id",
			Name:      fmt.Sprintf("%d/%d", scriptID, bindingID),
		}
	}
	value, err, claimed := r.invokeRegisteredBinding(ctx, binding, arguments)
	if !claimed {
		return Null(), &UnsupportedError{
			Operation: "binding id",
			Name:      fmt.Sprintf("%d/%d", scriptID, bindingID),
		}
	}
	return value, err
}
