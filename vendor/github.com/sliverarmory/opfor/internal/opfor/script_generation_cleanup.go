package opfor

import (
	"context"
	"errors"
	"sort"
)

// scriptGenerationCleanupToken prevents a Loadable or binding observer from
// synchronously joining the cleanup operation which is currently invoking it.
// It is deliberately separate from terminal Script unload ancestry: logical
// ScriptLoader unload keeps the Script active and emits no lifecycle event.
type scriptGenerationCleanupToken struct {
	generation *scriptGeneration
	parent     *scriptGenerationCleanupToken
}

type scriptGenerationCleanupContextKey struct{}

func withScriptGenerationCleanup(
	ctx context.Context,
	generation *scriptGeneration,
) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	parent, _ := ctx.Value(scriptGenerationCleanupContextKey{}).(*scriptGenerationCleanupToken)
	return context.WithValue(ctx, scriptGenerationCleanupContextKey{}, &scriptGenerationCleanupToken{
		generation: generation,
		parent:     parent,
	})
}

func cleaningScriptGeneration(ctx context.Context, generation *scriptGeneration) bool {
	if ctx == nil || generation == nil {
		return false
	}
	for token, _ := ctx.Value(scriptGenerationCleanupContextKey{}).(*scriptGenerationCleanupToken); token != nil; token = token.parent {
		if token.generation == generation {
			return true
		}
	}
	return false
}

type retiredNativePublication struct {
	name              string
	expected          Callable
	previousShared    Value
	hadPreviousShared bool
}

type scriptGenerationCleanupSnapshot struct {
	bindings          []Binding
	descendants       []Binding
	loadableUses      []scriptLoadableUse
	uiResources       []aggressorUIResource
	sharedEnvironment *portableScriptSharedEnvironment

	nativePublications []retiredNativePublication
}

// retireScriptGeneration closes one importer-capability epoch while retaining
// the Script, its globals, and raw Sleep closures. Registry visibility is
// revoked synchronously; provider teardown waits for already-admitted
// generation calls to drain. A reentrant caller cannot wait for its own lease,
// so it starts the same cleanup worker and returns after revocation.
func (r *Runtime) retireScriptGeneration(ctx context.Context, script *Script) error {
	if r == nil || script == nil || script.runtime != r {
		return ErrScriptUnloaded
	}
	if ctx == nil {
		ctx = context.Background()
	}

	generation, drained, started, err := script.retireCurrentScriptGeneration()
	if err != nil {
		// Terminal loader/Runtime ownership may already be closing the child. Its
		// terminal finalizer owns every remaining provider callback in that case.
		if errors.Is(err, ErrScriptUnloaded) {
			return nil
		}
		return err
	}
	if cleaningScriptGeneration(ctx, generation) || contextOwnsScriptGeneration(ctx, generation) {
		if started {
			snapshot := r.detachScriptGeneration(generation)
			r.startScriptGenerationCleanup(ctx, script, generation, drained, snapshot)
		}
		return nil
	}
	if started {
		snapshot := r.detachScriptGeneration(generation)
		r.startScriptGenerationCleanup(ctx, script, generation, drained, snapshot)
	}
	return waitScriptGenerationCleanup(ctx, script, generation)
}

func waitScriptGenerationCleanup(
	ctx context.Context,
	script *Script,
	generation *scriptGeneration,
) error {
	if ctx == nil {
		ctx = context.Background()
	}
	script.mu.RLock()
	done := generation.cleanupDone
	script.mu.RUnlock()
	if done == nil {
		return errors.New("opfor: script generation cleanup was not initialized")
	}
	select {
	case <-done:
		script.mu.RLock()
		err := generation.cleanupErr
		script.mu.RUnlock()
		return err
	default:
	}
	select {
	case <-done:
		script.mu.RLock()
		err := generation.cleanupErr
		script.mu.RUnlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (r *Runtime) startScriptGenerationCleanup(
	ctx context.Context,
	script *Script,
	generation *scriptGeneration,
	drained <-chan struct{},
	snapshot scriptGenerationCleanupSnapshot,
) {
	detachedContext, releaseDetached := detachExecutionLeaseCancellationLease(ctx)
	cleanupCtx := withScriptGenerationCleanup(detachedContext, generation)
	go func() {
		defer releaseDetached()
		<-drained
		cleanupErr := r.finishScriptGenerationCleanup(cleanupCtx, script, snapshot)
		script.mu.Lock()
		generation.cleanupErr = cleanupErr
		script.mu.Unlock()
		if _, completionErr := script.completeScriptGenerationRetirement(generation); completionErr != nil {
			script.mu.Lock()
			generation.cleanupErr = errors.Join(generation.cleanupErr, completionErr)
			// An invariant failure must not strand explicit unload callers. Keep
			// the error observable even though terminal finalization may still
			// diagnose the leaked reservation in subsequent tests.
			if generation.cleanupDone != nil && !channelClosed(generation.cleanupDone) {
				close(generation.cleanupDone)
			}
			script.mu.Unlock()
		}
	}()
}

// detachScriptGeneration makes every generation-owned registry entry
// unavailable at the unload admission boundary. It intentionally leaves raw
// script functions, globals, forks, processes, and sockets alone: upstream
// ScriptInstance retains those across ScriptLoader registry unload.
func (r *Runtime) detachScriptGeneration(generation *scriptGeneration) scriptGenerationCleanupSnapshot {
	var snapshot scriptGenerationCleanupSnapshot
	if r == nil || generation == nil || generation.script == nil {
		return snapshot
	}
	script := generation.script

	script.mu.Lock()
	keptBindings := make([]Binding, 0, len(script.bindings))
	for _, binding := range script.bindings {
		if generationForBinding(binding) == generation {
			snapshot.bindings = append(snapshot.bindings, cloneBinding(binding))
			continue
		}
		keptBindings = append(keptBindings, binding)
	}
	clear(script.bindings[len(keptBindings):])
	script.bindings = keptBindings
	if len(snapshot.bindings) != 0 {
		r.mu.Lock()
		for _, binding := range snapshot.bindings {
			r.removeRuntimeBindingLocked(binding)
		}
		r.mu.Unlock()
	}

	keptUses := make([]scriptLoadableUse, 0, len(script.loadableUses))
	for _, use := range script.loadableUses {
		if use.generation == generation {
			snapshot.loadableUses = append(snapshot.loadableUses, use)
			continue
		}
		keptUses = append(keptUses, use)
	}
	clear(script.loadableUses[len(keptUses):])
	script.loadableUses = keptUses
	snapshot.uiResources = takeAggressorUIResourcesForGenerationLocked(script, generation)
	snapshot.sharedEnvironment = script.sharedEnvironment
	snapshot.nativePublications = detachGenerationNativeFunctionsLocked(script, generation)
	script.mu.Unlock()

	for _, publication := range snapshot.nativePublications {
		if snapshot.sharedEnvironment != nil {
			snapshot.sharedEnvironment.restoreFunctionIfCurrent(
				publication.name,
				publication.expected,
				publication.previousShared,
				publication.hadPreviousShared,
			)
		}
	}
	if r.aggressorCommands != nil {
		r.aggressorCommands.removeGeneration(script.id, generation)
	}
	if r.aggressorBeaconTechniques != nil {
		r.aggressorBeaconTechniques.removeGeneration(script.id, generation)
	}
	revokeAggressorUIResources(snapshot.uiResources)
	snapshot.descendants = r.detachRetiredBindingDescendants(snapshot.bindings)
	return snapshot
}

// detachGenerationNativeFunctionsLocked unwinds only native bridge layers
// installed during generation. The bottom layer retains the exact shared-table
// value it shadowed, which may be an object-valued stock Sleep bridge.
func detachGenerationNativeFunctionsLocked(
	script *Script,
	generation *scriptGeneration,
) []retiredNativePublication {
	publications := make([]retiredNativePublication, 0)
	for name, head := range script.functions {
		current := head
		var bottom *scriptNativeCallable
		for {
			native, ok := current.(*scriptNativeCallable)
			if !ok || native == nil || native.owner != script || native.generation != generation {
				break
			}
			bottom = native
			if native.hadPrevious {
				current = native.previous
			} else {
				current = nil
			}
		}
		if bottom == nil {
			continue
		}
		publications = append(publications, retiredNativePublication{
			name:              name,
			expected:          head,
			previousShared:    bottom.previousShared,
			hadPreviousShared: bottom.hadPreviousShared,
		})
		if current == nil {
			delete(script.functions, name)
		} else {
			script.functions[name] = current
		}
		delete(script.removedFuncs, name)
	}
	return publications
}

func (r *Runtime) detachRetiredBindingDescendants(ancestors []Binding) []Binding {
	if r == nil || len(ancestors) == 0 {
		return nil
	}
	r.mu.RLock()
	var candidates []Binding
	for _, ordered := range r.bindingOrder {
		for _, candidate := range ordered {
			for _, ancestor := range ancestors {
				if bindingHasAncestor(candidate, ancestor) {
					candidates = append(candidates, cloneBinding(candidate))
					break
				}
			}
		}
	}
	r.mu.RUnlock()
	sort.SliceStable(candidates, func(left, right int) bool {
		leftDepth := bindingInvocationDepth(candidates[left].Parent)
		rightDepth := bindingInvocationDepth(candidates[right].Parent)
		if leftDepth != rightDepth {
			return leftDepth > rightDepth
		}
		if candidates[left].Script != candidates[right].Script {
			return candidates[left].Script > candidates[right].Script
		}
		return candidates[left].ID > candidates[right].ID
	})

	detached := make([]Binding, 0, len(candidates))
	for _, candidate := range candidates {
		r.mu.RLock()
		owner := r.scripts[candidate.Script]
		r.mu.RUnlock()
		if owner != nil && owner.removeBindingIfPresent(candidate) {
			detached = append(detached, candidate)
		}
	}
	return detached
}

func (r *Runtime) finishScriptGenerationCleanup(
	ctx context.Context,
	script *Script,
	snapshot scriptGenerationCleanupSnapshot,
) error {
	var result error
	for index := len(snapshot.loadableUses) - 1; index >= 0; index-- {
		bridge := snapshot.loadableUses[index].bridge
		if isNilInterface(bridge) {
			continue
		}
		if err := bridge.ScriptUnloaded(ctx, script); err != nil {
			result = errors.Join(result, preserveNativeBoundaryError(ctx, err))
		}
	}
	if r.observer != nil {
		for index := len(snapshot.bindings) - 1; index >= 0; index-- {
			if err := r.observer.Unregistered(ctx, cloneBinding(snapshot.bindings[index])); err != nil {
				result = errors.Join(result, preserveNativeBoundaryError(ctx, err))
			}
		}
		for _, binding := range snapshot.descendants {
			if err := r.observer.Unregistered(ctx, cloneBinding(binding)); err != nil {
				result = errors.Join(result, preserveNativeBoundaryError(ctx, err))
			}
		}
	}
	return result
}
