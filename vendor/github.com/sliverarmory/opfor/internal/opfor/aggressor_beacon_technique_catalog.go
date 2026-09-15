package opfor

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// AggressorBeaconTechniqueKind selects one of Cobalt Strike's independent
// Beacon technique registries.
type AggressorBeaconTechniqueKind string

const (
	// AggressorBeaconTechniqueElevator identifies the belevate_command/runasadmin
	// elevator registry.
	AggressorBeaconTechniqueElevator AggressorBeaconTechniqueKind = "elevator"
	// AggressorBeaconTechniqueExploit identifies the belevate local-exploit
	// registry.
	AggressorBeaconTechniqueExploit AggressorBeaconTechniqueKind = "exploit"
	// AggressorBeaconTechniqueRemoteExecMethod identifies the bremote_exec
	// remote-exec-method registry.
	AggressorBeaconTechniqueRemoteExecMethod AggressorBeaconTechniqueKind = "remote-exec-method"
	// AggressorBeaconTechniqueRemoteExploit identifies the bjump remote-exploit
	// registry.
	AggressorBeaconTechniqueRemoteExploit AggressorBeaconTechniqueKind = "remote-exploit"
)

var aggressorBeaconTechniqueKinds = [...]AggressorBeaconTechniqueKind{
	AggressorBeaconTechniqueElevator,
	AggressorBeaconTechniqueExploit,
	AggressorBeaconTechniqueRemoteExecMethod,
	AggressorBeaconTechniqueRemoteExploit,
}

// AggressorBeaconTechniqueMetadata describes one registered Beacon technique.
// Architecture is required for remote exploits and must be "x86" or "x64";
// it must be empty for every other kind.
//
// Callbacks are deliberately absent. A catalog is safe to expose to importers,
// while executable callbacks remain private, script-owned capabilities.
type AggressorBeaconTechniqueMetadata struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Architecture string `json:"architecture,omitempty"`
}

// AggressorBeaconTechniqueCatalog is a static or effective technique catalog.
// Technique order is deterministic first-insertion order. The zero value is
// an empty catalog.
type AggressorBeaconTechniqueCatalog struct {
	Techniques []AggressorBeaconTechniqueMetadata `json:"techniques,omitempty"`
}

// WithAggressorBeaconTechniqueCatalog seeds an importer-owned base catalog for
// one technique kind. Script registrations layer over this immutable base and
// unloading a script reveals the preceding script or base entry. Supplying the
// option again for the same kind replaces the earlier base catalog.
//
// The catalog is validated and defensively copied while New applies the
// option. Names must be non-empty and unique using exact case-sensitive
// comparison. Remote-exploit architectures must be exactly "x86" or "x64";
// architecture is rejected for every other kind.
func WithAggressorBeaconTechniqueCatalog(
	kind AggressorBeaconTechniqueKind,
	catalog AggressorBeaconTechniqueCatalog,
) Option {
	return func(config *runtimeConfig) error {
		if err := validateAggressorBeaconTechniqueKind(kind); err != nil {
			return err
		}
		clone := cloneAggressorBeaconTechniqueCatalog(catalog)
		if err := validateAggressorBeaconTechniqueCatalog(kind, clone); err != nil {
			return err
		}
		if config.aggressorBeaconTechniques == nil {
			config.aggressorBeaconTechniques = make(map[AggressorBeaconTechniqueKind]AggressorBeaconTechniqueCatalog, len(aggressorBeaconTechniqueKinds))
		}
		config.aggressorBeaconTechniques[kind] = clone
		return nil
	}
}

// SnapshotAggressorBeaconTechniqueCatalog returns a detached snapshot of the
// effective metadata catalog. The last registration for a name supplies its
// metadata while names retain deterministic first-insertion order. Mutating
// the returned slice does not affect the Runtime.
func (r *Runtime) SnapshotAggressorBeaconTechniqueCatalog(
	kind AggressorBeaconTechniqueKind,
) (AggressorBeaconTechniqueCatalog, error) {
	if r == nil {
		return AggressorBeaconTechniqueCatalog{}, errors.New("opfor: runtime is nil")
	}
	if err := validateAggressorBeaconTechniqueKind(kind); err != nil {
		return AggressorBeaconTechniqueCatalog{}, err
	}
	if r.aggressorBeaconTechniques == nil {
		return AggressorBeaconTechniqueCatalog{}, nil
	}
	return r.aggressorBeaconTechniques.snapshot(kind), nil
}

// InvokeAggressorBeaconTechnique invokes the effective script-owned callback
// for an exact name. It never performs Beacon tasking or calls Host. Importer
// base entries carry metadata only, so a base-only or missing name returns a
// typed UnsupportedError. The retained callback rejects invocation once its
// owning script reaches unload admission.
//
// Arguments follow the documented callback ABI: elevator and exploit receive
// two values; remote-exec-method and remote-exploit receive three. For the two
// command callbacks the importer supplies the complete command and arguments
// as one unmodified Value.
func (r *Runtime) InvokeAggressorBeaconTechnique(
	ctx context.Context,
	kind AggressorBeaconTechniqueKind,
	name string,
	arguments ...Value,
) (Value, error) {
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if err := validateAggressorBeaconTechniqueKind(kind); err != nil {
		return Null(), err
	}
	if name == "" {
		return Null(), errors.New("opfor: Aggressor Beacon technique name is empty")
	}
	want := aggressorBeaconTechniqueCallbackArity(kind)
	if len(arguments) != want {
		return Null(), fmt.Errorf("opfor: %s Beacon technique callback expects exactly %d argument(s), received %d", kind, want, len(arguments))
	}
	executionCtx, release, err := r.acquireRuntimeExecution(ctx)
	if err != nil {
		return Null(), err
	}
	defer release()
	executionCtx = withExecutionMeter(executionCtx, r)
	var result Value
	var invokeErr error
	if r.aggressorBeaconTechniques == nil {
		invokeErr = unsupportedAggressorBeaconTechnique(kind, name)
		return Null(), errors.Join(invokeErr, release())
	}
	callback, exists := r.aggressorBeaconTechniques.callback(kind, name)
	if !exists || callback == nil {
		invokeErr = unsupportedAggressorBeaconTechnique(kind, name)
		return Null(), errors.Join(invokeErr, release())
	}
	result, invokeErr = callback.Invoke(executionCtx, arguments...)
	if invokeErr == nil {
		invokeErr = runtimeExecutionError(executionCtx)
	}
	return result, errors.Join(invokeErr, release())
}

func unsupportedAggressorBeaconTechnique(kind AggressorBeaconTechniqueKind, name string) error {
	return &UnsupportedError{
		Operation: fmt.Sprintf("Aggressor Beacon %s callback", kind),
		Name:      name,
	}
}

func validateAggressorBeaconTechniqueKind(kind AggressorBeaconTechniqueKind) error {
	switch kind {
	case AggressorBeaconTechniqueElevator,
		AggressorBeaconTechniqueExploit,
		AggressorBeaconTechniqueRemoteExecMethod,
		AggressorBeaconTechniqueRemoteExploit:
		return nil
	default:
		return fmt.Errorf("opfor: invalid Aggressor Beacon technique kind %q", kind)
	}
}

func aggressorBeaconTechniqueCallbackArity(kind AggressorBeaconTechniqueKind) int {
	switch kind {
	case AggressorBeaconTechniqueElevator, AggressorBeaconTechniqueExploit:
		return 2
	case AggressorBeaconTechniqueRemoteExecMethod, AggressorBeaconTechniqueRemoteExploit:
		return 3
	default:
		return 0
	}
}

func cloneAggressorBeaconTechniqueCatalog(catalog AggressorBeaconTechniqueCatalog) AggressorBeaconTechniqueCatalog {
	clone := AggressorBeaconTechniqueCatalog{
		Techniques: make([]AggressorBeaconTechniqueMetadata, len(catalog.Techniques)),
	}
	copy(clone.Techniques, catalog.Techniques)
	return clone
}

func validateAggressorBeaconTechniqueCatalog(
	kind AggressorBeaconTechniqueKind,
	catalog AggressorBeaconTechniqueCatalog,
) error {
	names := make(map[string]struct{}, len(catalog.Techniques))
	for index, metadata := range catalog.Techniques {
		if err := validateAggressorBeaconTechniqueMetadata(kind, metadata); err != nil {
			return fmt.Errorf("opfor: %s Beacon technique catalog entry %d: %w", kind, index, err)
		}
		if _, duplicate := names[metadata.Name]; duplicate {
			return fmt.Errorf("opfor: %s Beacon technique catalog has duplicate name %q", kind, metadata.Name)
		}
		names[metadata.Name] = struct{}{}
	}
	return nil
}

func validateAggressorBeaconTechniqueMetadata(
	kind AggressorBeaconTechniqueKind,
	metadata AggressorBeaconTechniqueMetadata,
) error {
	if metadata.Name == "" {
		return errors.New("technique name is empty")
	}
	if kind == AggressorBeaconTechniqueRemoteExploit {
		if metadata.Architecture != "x86" && metadata.Architecture != "x64" {
			return fmt.Errorf("remote-exploit architecture %q is invalid; expected x86 or x64", metadata.Architecture)
		}
		return nil
	}
	if metadata.Architecture != "" {
		return fmt.Errorf("architecture is not valid for %s techniques", kind)
	}
	return nil
}

type aggressorBeaconTechniqueLayer struct {
	owner      ScriptID
	generation *scriptGeneration
	metadata   AggressorBeaconTechniqueMetadata
	callback   Callable
}

type aggressorBeaconTechniqueNamespace struct {
	techniques map[string][]aggressorBeaconTechniqueLayer
	order      []string
}

type aggressorBeaconTechniqueState struct {
	mu         sync.RWMutex
	namespaces map[AggressorBeaconTechniqueKind]*aggressorBeaconTechniqueNamespace
}

func newAggressorBeaconTechniqueNamespace() *aggressorBeaconTechniqueNamespace {
	return &aggressorBeaconTechniqueNamespace{
		techniques: make(map[string][]aggressorBeaconTechniqueLayer),
	}
}

func newAggressorBeaconTechniqueState(
	catalogs map[AggressorBeaconTechniqueKind]AggressorBeaconTechniqueCatalog,
) *aggressorBeaconTechniqueState {
	state := &aggressorBeaconTechniqueState{
		namespaces: make(map[AggressorBeaconTechniqueKind]*aggressorBeaconTechniqueNamespace, len(aggressorBeaconTechniqueKinds)),
	}
	for _, kind := range aggressorBeaconTechniqueKinds {
		namespace := newAggressorBeaconTechniqueNamespace()
		state.namespaces[kind] = namespace
		for _, metadata := range catalogs[kind].Techniques {
			namespace.order = append(namespace.order, metadata.Name)
			namespace.techniques[metadata.Name] = []aggressorBeaconTechniqueLayer{{metadata: metadata}}
		}
	}
	return state
}

func (state *aggressorBeaconTechniqueState) register(
	kind AggressorBeaconTechniqueKind,
	owner ScriptID,
	generation *scriptGeneration,
	metadata AggressorBeaconTechniqueMetadata,
	callback Callable,
) {
	state.mu.Lock()
	defer state.mu.Unlock()
	namespace := state.namespaces[kind]
	if len(namespace.techniques[metadata.Name]) == 0 {
		namespace.order = append(namespace.order, metadata.Name)
	}
	layers := namespace.techniques[metadata.Name]
	coalesced := layers[:0]
	for _, layer := range layers {
		if layer.owner != owner || layer.generation != generation {
			coalesced = append(coalesced, layer)
		}
	}
	clear(layers[len(coalesced):])
	namespace.techniques[metadata.Name] = append(coalesced, aggressorBeaconTechniqueLayer{
		owner:      owner,
		generation: generation,
		metadata:   metadata,
		callback:   callback,
	})
}

func (state *aggressorBeaconTechniqueState) describe(
	kind AggressorBeaconTechniqueKind,
	name string,
) (AggressorBeaconTechniqueMetadata, bool) {
	state.mu.RLock()
	defer state.mu.RUnlock()
	layers := state.namespaces[kind].techniques[name]
	if len(layers) == 0 {
		return AggressorBeaconTechniqueMetadata{}, false
	}
	return layers[len(layers)-1].metadata, true
}

func (state *aggressorBeaconTechniqueState) names(kind AggressorBeaconTechniqueKind) []string {
	state.mu.RLock()
	defer state.mu.RUnlock()
	namespace := state.namespaces[kind]
	names := make([]string, 0, len(namespace.order))
	for _, name := range namespace.order {
		if len(namespace.techniques[name]) != 0 {
			names = append(names, name)
		}
	}
	return names
}

func (state *aggressorBeaconTechniqueState) callback(
	kind AggressorBeaconTechniqueKind,
	name string,
) (Callable, bool) {
	state.mu.RLock()
	defer state.mu.RUnlock()
	layers := state.namespaces[kind].techniques[name]
	if len(layers) == 0 {
		return nil, false
	}
	return layers[len(layers)-1].callback, true
}

func (state *aggressorBeaconTechniqueState) snapshot(
	kind AggressorBeaconTechniqueKind,
) AggressorBeaconTechniqueCatalog {
	state.mu.RLock()
	defer state.mu.RUnlock()
	namespace := state.namespaces[kind]
	catalog := AggressorBeaconTechniqueCatalog{
		Techniques: make([]AggressorBeaconTechniqueMetadata, 0, len(namespace.order)),
	}
	for _, name := range namespace.order {
		layers := namespace.techniques[name]
		if len(layers) != 0 {
			catalog.Techniques = append(catalog.Techniques, layers[len(layers)-1].metadata)
		}
	}
	return catalog
}

// baseSnapshot returns only immutable importer entries (owner zero). Portable
// ScriptLoader child runtimes inherit this base, never the parent runtime's
// script-owned metadata or callbacks.
func (state *aggressorBeaconTechniqueState) baseSnapshot(
	kind AggressorBeaconTechniqueKind,
) AggressorBeaconTechniqueCatalog {
	state.mu.RLock()
	defer state.mu.RUnlock()
	namespace := state.namespaces[kind]
	catalog := AggressorBeaconTechniqueCatalog{
		Techniques: make([]AggressorBeaconTechniqueMetadata, 0, len(namespace.order)),
	}
	for _, name := range namespace.order {
		for _, layer := range namespace.techniques[name] {
			if layer.owner == 0 {
				catalog.Techniques = append(catalog.Techniques, layer.metadata)
				break
			}
		}
	}
	return catalog
}

func (state *aggressorBeaconTechniqueState) removeScript(owner ScriptID) {
	if state == nil || owner == 0 {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	for _, namespace := range state.namespaces {
		for name, layers := range namespace.techniques {
			kept := layers[:0]
			for _, layer := range layers {
				if layer.owner != owner {
					kept = append(kept, layer)
				}
			}
			clear(layers[len(kept):])
			if len(kept) == 0 {
				delete(namespace.techniques, name)
			} else {
				namespace.techniques[name] = kept
			}
		}
		namespace.order = filterAggressorCommandOrder(namespace.order, func(name string) bool {
			return len(namespace.techniques[name]) != 0
		})
	}
}

// removeGeneration retires one exact script execution generation without
// disturbing registrations created by a later run of the same Script.
func (state *aggressorBeaconTechniqueState) removeGeneration(
	owner ScriptID,
	generation *scriptGeneration,
) {
	if state == nil || owner == 0 || generation == nil {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	for _, namespace := range state.namespaces {
		for name, layers := range namespace.techniques {
			kept := layers[:0]
			for _, layer := range layers {
				if layer.owner != owner || layer.generation != generation {
					kept = append(kept, layer)
				}
			}
			clear(layers[len(kept):])
			if len(kept) == 0 {
				delete(namespace.techniques, name)
			} else {
				namespace.techniques[name] = kept
			}
		}
		namespace.order = filterAggressorCommandOrder(namespace.order, func(name string) bool {
			return len(namespace.techniques[name]) != 0
		})
	}
}

func (r *Runtime) registerAggressorBeaconTechnique(
	invocation Invocation,
	kind AggressorBeaconTechniqueKind,
	metadata AggressorBeaconTechniqueMetadata,
	callback Callable,
) error {
	if r == nil || r.aggressorBeaconTechniques == nil {
		return errors.New("opfor: runtime is nil")
	}
	script := r.script(invocation.Script)
	if script == nil {
		return ErrScriptUnloaded
	}
	generation := invocation.generationToken()
	script.mu.Lock()
	defer script.mu.Unlock()
	if !script.generationAdmissibleLocked(generation) {
		return ErrScriptUnloaded
	}
	r.aggressorBeaconTechniques.register(kind, invocation.Script, generation, metadata, callback)
	return nil
}
