package opfor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// ScriptLifecycleObserver receives the load and unload boundary of each
// independently loaded script. ScriptLoaded runs after initial and
// runtime-owned globals are installed, but before the top-level body executes.
// It may inspect or mutate globals through Script.Get and Script.Set.
//
// Once ScriptLoaded begins, ScriptUnloaded is called exactly once, including
// when ScriptLoaded itself fails, the top-level body fails, Execute completes,
// Script.Unload is called, or Runtime.Close unloads the script. Observers may be
// called concurrently for independent scripts and must be concurrency-safe.
// Runtime.Eval owns one persistent script: its first evaluation starts the
// lifecycle, later evaluations reuse it, and Runtime.Close ends it.
// ScriptUnloaded sees an inactive Script after runtime-owned resources and
// registrations have been revoked. Its globals remain readable for final-state
// inspection, while Script.Set and callback invocation return ErrScriptUnloaded.
// As with ordinary Go references, importer-retained mutable Values and raw
// Argument.Reference cells remain trusted capabilities and can still change
// their backing data; lifecycle revocation does not make shared Go objects
// immutable.
//
// Sleep fork instances are not independent loads: they receive neither fresh
// initial globals nor lifecycle notifications. This matches Sleep 2.1, whose
// fork path creates an internal ScriptInstance without running ScriptLoader
// bridges. Source-backed portable ScriptLoader children are independent loads
// and inherit the parent runtime's observer.
type ScriptLifecycleObserver interface {
	ScriptLoaded(context.Context, *Script) error
	ScriptUnloaded(context.Context, *Script) error
}

// ScriptLifecycleFuncs adapts optional functions to ScriptLifecycleObserver.
// A nil function is a no-op.
type ScriptLifecycleFuncs struct {
	Loaded   func(context.Context, *Script) error
	Unloaded func(context.Context, *Script) error
}

// ScriptLoaded invokes Loaded when configured.
func (functions ScriptLifecycleFuncs) ScriptLoaded(ctx context.Context, script *Script) error {
	if functions.Loaded == nil {
		return nil
	}
	return functions.Loaded(ctx, script)
}

// ScriptUnloaded invokes Unloaded when configured.
func (functions ScriptLifecycleFuncs) ScriptUnloaded(ctx context.Context, script *Script) error {
	if functions.Unloaded == nil {
		return nil
	}
	return functions.Unloaded(ctx, script)
}

// WithInitialGlobals installs importer-owned values into every independently
// loaded script before ScriptLifecycleObserver.ScriptLoaded and before the
// script's top-level body. The persistent Runtime.Eval session receives them
// once when it is first created. A name without a sigil is normalized to a
// scalar name (for example, "client" becomes "$client"). Names beginning with
// $, @, or % retain that sigil.
//
// Scalar values are copied. Arrays, hashes, functions, and objects retain their
// reference identity, so reusing one Value intentionally shares its backing
// value across loaded scripts. Use ScriptLoaded to install a fresh container per
// script when sharing is not desired. The input map is copied before New
// returns and later map changes have no effect.
//
// $__SCRIPT__, $__SCRIPT_NAME__, and @ARGV are reserved because the runtime
// defines them for each load. Duplicate normalized names and malformed names
// make New fail instead of silently choosing one value.
func WithInitialGlobals(globals map[string]Value) Option {
	return func(config *runtimeConfig) error {
		if len(globals) == 0 {
			return nil
		}
		if config.initialGlobals == nil {
			config.initialGlobals = make(map[string]Value, len(globals))
		}
		for name, value := range globals {
			normalized, err := normalizeInitialGlobalName(name)
			if err != nil {
				return err
			}
			if _, duplicate := config.initialGlobals[normalized]; duplicate {
				return fmt.Errorf("opfor: duplicate initial global %q", normalized)
			}
			config.initialGlobals[normalized] = value
		}
		return nil
	}
}

// WithScriptLifecycleObserver installs per-script load and unload callbacks.
// Callback errors are returned by the operation that crossed the lifecycle
// boundary. The observer is inherited by source-backed portable ScriptLoader
// runtimes.
func WithScriptLifecycleObserver(observer ScriptLifecycleObserver) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(observer) {
			return errors.New("opfor: script lifecycle observer is nil")
		}
		config.lifecycle = observer
		return nil
	}
}

var reservedInitialGlobals = map[string]struct{}{
	"$__SCRIPT__":      {},
	"$__SCRIPT_NAME__": {},
	"@ARGV":            {},
}

func normalizeInitialGlobalName(name string) (string, error) {
	if name == "" {
		return "", errors.New("opfor: initial global name is empty")
	}
	if strings.TrimSpace(name) != name {
		return "", fmt.Errorf("opfor: initial global name %q has surrounding whitespace", name)
	}
	for _, character := range name {
		if unicode.IsSpace(character) || unicode.IsControl(character) {
			return "", fmt.Errorf("opfor: initial global name %q contains whitespace or a control character", name)
		}
	}
	if name[0] != '$' && name[0] != '@' && name[0] != '%' {
		name = "$" + name
	}
	if len(name) == 1 {
		return "", fmt.Errorf("opfor: initial global name %q has no identifier", name)
	}
	if _, reserved := reservedInitialGlobals[name]; reserved {
		return "", fmt.Errorf("opfor: initial global %q is reserved by the runtime", name)
	}
	return name, nil
}

func cloneInitialGlobals(globals map[string]Value) map[string]Value {
	if len(globals) == 0 {
		return nil
	}
	clone := make(map[string]Value, len(globals))
	for name, value := range globals {
		clone[name] = value
	}
	return clone
}

func (r *Runtime) installInitialGlobals(ctx context.Context, script *Script) error {
	if r == nil || script == nil || script.globals == nil || len(r.initialGlobals) == 0 {
		return nil
	}
	names := make([]string, 0, len(r.initialGlobals))
	for name := range r.initialGlobals {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := script.setGlobalAt(ctx, name, r.initialGlobals[name], Span{}); err != nil {
			return err
		}
	}
	return nil
}

func (r *Runtime) notifyScriptLoaded(ctx context.Context, script *Script) error {
	if r == nil || script == nil || r.lifecycle == nil {
		return nil
	}
	// Mark the callback as begun before invoking importer code. If importer setup
	// fails or unloads the script reentrantly, unload still delivers exactly one
	// matching cleanup notification. Serializing the active check and marker
	// against unload's Script lock also prevents a concurrent Close from
	// removing the script immediately before this marker is installed.
	script.mu.RLock()
	if !script.active {
		script.mu.RUnlock()
		return ErrScriptUnloaded
	}
	r.mu.Lock()
	if r.lifecycleScripts == nil {
		r.lifecycleScripts = make(map[ScriptID]struct{})
	}
	r.lifecycleScripts[script.id] = struct{}{}
	r.mu.Unlock()
	script.mu.RUnlock()
	observerErr := r.lifecycle.ScriptLoaded(ctx, script)
	if observerErr != nil {
		observerErr = fmt.Errorf("opfor: script %d load observer: %w", script.id, observerErr)
	}
	observerErr = joinExecutionContextError(ctx, observerErr)
	if !script.Active() {
		// Concurrent unload wins the lifecycle classification even though the
		// execution context it canceled also reports context.Canceled. Preserve
		// any observer or fatal resource error alongside that terminal state.
		return errors.Join(ErrScriptUnloaded, observerErr)
	}
	return observerErr
}
