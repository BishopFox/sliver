package opfor

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// AggressorCommandKind selects one of Cobalt Strike's independent console
// help catalogs.
type AggressorCommandKind string

const (
	// AggressorCommandBeacon identifies Beacon console command help.
	AggressorCommandBeacon AggressorCommandKind = "beacon"
	// AggressorCommandSSH identifies SSH console command help.
	AggressorCommandSSH AggressorCommandKind = "ssh"
)

// AggressorCommandMetadata describes one command in a Beacon or SSH help
// catalog. GroupID is empty when the command is not assigned to a help group.
type AggressorCommandMetadata struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Detail      string `json:"detail"`
	GroupID     string `json:"group_id,omitempty"`
}

// AggressorCommandGroup describes one Beacon or SSH help group.
type AggressorCommandGroup struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// AggressorCommandCatalog is a static or effective command-help catalog.
// Command slice order is preserved and is the order returned by the
// corresponding commands function. An effective snapshot includes only groups
// referenced by an active command. The zero value is an empty catalog.
type AggressorCommandCatalog struct {
	Commands []AggressorCommandMetadata `json:"commands,omitempty"`
	Groups   []AggressorCommandGroup    `json:"groups,omitempty"`
}

// WithAggressorCommandCatalog seeds an importer-owned base help catalog for
// one console kind. Script registrations layer over this immutable base and
// unloading a script reveals the preceding script or base entry. Supplying
// the option again for the same kind replaces the earlier base catalog.
//
// The catalog is validated and defensively copied while New applies the
// option. Command and group names must be non-empty, entries must be unique,
// group IDs may not contain ',' or '@', and command group references must
// identify a group in the same catalog.
func WithAggressorCommandCatalog(kind AggressorCommandKind, catalog AggressorCommandCatalog) Option {
	return func(config *runtimeConfig) error {
		if err := validateAggressorCommandKind(kind); err != nil {
			return err
		}
		clone := cloneAggressorCommandCatalog(catalog)
		if err := validateAggressorCommandCatalog(kind, clone); err != nil {
			return err
		}
		if config.aggressorCommands == nil {
			config.aggressorCommands = make(map[AggressorCommandKind]AggressorCommandCatalog, 2)
		}
		config.aggressorCommands[kind] = clone
		return nil
	}
}

// SnapshotAggressorCommandCatalog returns a detached snapshot of the effective
// help catalog. The last registration for a name supplies its metadata while
// names retain deterministic first-insertion order. Groups without an active
// command are omitted. Mutating the returned slices does not affect the Runtime.
func (r *Runtime) SnapshotAggressorCommandCatalog(kind AggressorCommandKind) (AggressorCommandCatalog, error) {
	if r == nil {
		return AggressorCommandCatalog{}, errors.New("opfor: runtime is nil")
	}
	if err := validateAggressorCommandKind(kind); err != nil {
		return AggressorCommandCatalog{}, err
	}
	if r.aggressorCommands == nil {
		return AggressorCommandCatalog{}, nil
	}
	return r.aggressorCommands.snapshot(kind), nil
}

func validateAggressorCommandKind(kind AggressorCommandKind) error {
	switch kind {
	case AggressorCommandBeacon, AggressorCommandSSH:
		return nil
	default:
		return fmt.Errorf("opfor: invalid Aggressor command kind %q", kind)
	}
}

func cloneAggressorCommandCatalog(catalog AggressorCommandCatalog) AggressorCommandCatalog {
	clone := AggressorCommandCatalog{
		Commands: make([]AggressorCommandMetadata, len(catalog.Commands)),
		Groups:   make([]AggressorCommandGroup, len(catalog.Groups)),
	}
	copy(clone.Commands, catalog.Commands)
	copy(clone.Groups, catalog.Groups)
	return clone
}

func validateAggressorCommandCatalog(kind AggressorCommandKind, catalog AggressorCommandCatalog) error {
	groups := make(map[string]struct{}, len(catalog.Groups))
	for index, group := range catalog.Groups {
		if err := validateAggressorCommandGroup(group.ID, group.Name); err != nil {
			return fmt.Errorf("opfor: %s command catalog group %d: %w", kind, index, err)
		}
		if _, duplicate := groups[group.ID]; duplicate {
			return fmt.Errorf("opfor: %s command catalog has duplicate group %q", kind, group.ID)
		}
		groups[group.ID] = struct{}{}
	}
	commands := make(map[string]struct{}, len(catalog.Commands))
	for index, command := range catalog.Commands {
		if command.Name == "" {
			return fmt.Errorf("opfor: %s command catalog command %d has an empty name", kind, index)
		}
		if _, duplicate := commands[command.Name]; duplicate {
			return fmt.Errorf("opfor: %s command catalog has duplicate command %q", kind, command.Name)
		}
		commands[command.Name] = struct{}{}
		if command.GroupID != "" {
			if _, exists := groups[command.GroupID]; !exists {
				return fmt.Errorf("opfor: %s command %q references unknown group %q", kind, command.Name, command.GroupID)
			}
		}
	}
	return nil
}

func validateAggressorCommandGroup(id, name string) error {
	if id == "" {
		return errors.New("help group ID is empty")
	}
	if strings.ContainsAny(id, ",@") {
		return fmt.Errorf("help group ID %q contains ',' or '@'", id)
	}
	if name == "" {
		return fmt.Errorf("help group %q has an empty name", id)
	}
	return nil
}

type aggressorCommandLayer struct {
	owner      ScriptID
	generation *scriptGeneration
	metadata   AggressorCommandMetadata
}

type aggressorCommandGroupLayer struct {
	owner      ScriptID
	generation *scriptGeneration
	group      AggressorCommandGroup
}

type aggressorCommandNamespace struct {
	commands     map[string][]aggressorCommandLayer
	commandOrder []string
	groups       map[string][]aggressorCommandGroupLayer
	groupOrder   []string
}

type aggressorCommandState struct {
	mu         sync.RWMutex
	namespaces map[AggressorCommandKind]*aggressorCommandNamespace
}

func newAggressorCommandNamespace() *aggressorCommandNamespace {
	return &aggressorCommandNamespace{
		commands: make(map[string][]aggressorCommandLayer),
		groups:   make(map[string][]aggressorCommandGroupLayer),
	}
}

func newAggressorCommandState(catalogs map[AggressorCommandKind]AggressorCommandCatalog) *aggressorCommandState {
	state := &aggressorCommandState{namespaces: map[AggressorCommandKind]*aggressorCommandNamespace{
		AggressorCommandBeacon: newAggressorCommandNamespace(),
		AggressorCommandSSH:    newAggressorCommandNamespace(),
	}}
	for _, kind := range []AggressorCommandKind{AggressorCommandBeacon, AggressorCommandSSH} {
		catalog, exists := catalogs[kind]
		if !exists {
			continue
		}
		namespace := state.namespaces[kind]
		for _, group := range catalog.Groups {
			namespace.groupOrder = append(namespace.groupOrder, group.ID)
			namespace.groups[group.ID] = []aggressorCommandGroupLayer{{group: group}}
		}
		for _, command := range catalog.Commands {
			namespace.commandOrder = append(namespace.commandOrder, command.Name)
			namespace.commands[command.Name] = []aggressorCommandLayer{{metadata: command}}
		}
	}
	return state
}

func (state *aggressorCommandState) registerGroup(
	kind AggressorCommandKind,
	owner ScriptID,
	generation *scriptGeneration,
	group AggressorCommandGroup,
) {
	state.mu.Lock()
	defer state.mu.Unlock()
	namespace := state.namespaces[kind]
	if len(namespace.groups[group.ID]) == 0 {
		namespace.groupOrder = append(namespace.groupOrder, group.ID)
	}
	layers := namespace.groups[group.ID]
	coalesced := layers[:0]
	for _, layer := range layers {
		if layer.owner != owner || layer.generation != generation {
			coalesced = append(coalesced, layer)
		}
	}
	namespace.groups[group.ID] = append(coalesced, aggressorCommandGroupLayer{
		owner:      owner,
		generation: generation,
		group:      group,
	})
}

func (state *aggressorCommandState) registerCommand(
	kind AggressorCommandKind,
	owner ScriptID,
	generation *scriptGeneration,
	metadata AggressorCommandMetadata,
) {
	state.mu.Lock()
	defer state.mu.Unlock()
	namespace := state.namespaces[kind]
	if metadata.GroupID != "" && len(namespace.groups[metadata.GroupID]) == 0 {
		// The reference documentation explicitly says an unknown optional group
		// is ignored. It does not defer or retroactively create the association.
		metadata.GroupID = ""
	}
	if len(namespace.commands[metadata.Name]) == 0 {
		namespace.commandOrder = append(namespace.commandOrder, metadata.Name)
	}
	layers := namespace.commands[metadata.Name]
	coalesced := layers[:0]
	for _, layer := range layers {
		if layer.owner != owner || layer.generation != generation {
			coalesced = append(coalesced, layer)
		}
	}
	namespace.commands[metadata.Name] = append(coalesced, aggressorCommandLayer{
		owner:      owner,
		generation: generation,
		metadata:   metadata,
	})
}

func (state *aggressorCommandState) describe(kind AggressorCommandKind, name string) (AggressorCommandMetadata, bool) {
	state.mu.RLock()
	defer state.mu.RUnlock()
	layers := state.namespaces[kind].commands[name]
	if len(layers) == 0 {
		return AggressorCommandMetadata{}, false
	}
	return layers[len(layers)-1].metadata, true
}

func (state *aggressorCommandState) commandNames(kind AggressorCommandKind) []string {
	state.mu.RLock()
	defer state.mu.RUnlock()
	namespace := state.namespaces[kind]
	names := make([]string, 0, len(namespace.commandOrder))
	for _, name := range namespace.commandOrder {
		if len(namespace.commands[name]) != 0 {
			names = append(names, name)
		}
	}
	return names
}

func (state *aggressorCommandState) snapshot(kind AggressorCommandKind) AggressorCommandCatalog {
	state.mu.RLock()
	defer state.mu.RUnlock()
	namespace := state.namespaces[kind]
	catalog := AggressorCommandCatalog{
		Commands: make([]AggressorCommandMetadata, 0, len(namespace.commandOrder)),
		Groups:   make([]AggressorCommandGroup, 0, len(namespace.groupOrder)),
	}
	visibleGroups := make(map[string]struct{})
	for _, name := range namespace.commandOrder {
		layers := namespace.commands[name]
		if len(layers) != 0 {
			metadata := layers[len(layers)-1].metadata
			if metadata.GroupID != "" && len(namespace.groups[metadata.GroupID]) == 0 {
				// A group can disappear when its owning script unloads while a
				// command from another script remains. Do not expose a dangling
				// effective association; the stored accepted association may become
				// effective again if an earlier layer for that group is restored.
				metadata.GroupID = ""
			} else if metadata.GroupID != "" {
				visibleGroups[metadata.GroupID] = struct{}{}
			}
			catalog.Commands = append(catalog.Commands, metadata)
		}
	}
	for _, id := range namespace.groupOrder {
		layers := namespace.groups[id]
		if _, visible := visibleGroups[id]; visible && len(layers) != 0 {
			catalog.Groups = append(catalog.Groups, layers[len(layers)-1].group)
		}
	}
	return catalog
}

// baseSnapshot returns only immutable importer entries (owner zero). Portable
// ScriptLoader child runtimes inherit this base, never the parent runtime's
// script-owned overlays.
func (state *aggressorCommandState) baseSnapshot(kind AggressorCommandKind) AggressorCommandCatalog {
	state.mu.RLock()
	defer state.mu.RUnlock()
	namespace := state.namespaces[kind]
	catalog := AggressorCommandCatalog{
		Commands: make([]AggressorCommandMetadata, 0, len(namespace.commandOrder)),
		Groups:   make([]AggressorCommandGroup, 0, len(namespace.groupOrder)),
	}
	baseGroups := make(map[string]struct{})
	for _, id := range namespace.groupOrder {
		for _, layer := range namespace.groups[id] {
			if layer.owner == 0 {
				catalog.Groups = append(catalog.Groups, layer.group)
				baseGroups[id] = struct{}{}
				break
			}
		}
	}
	for _, name := range namespace.commandOrder {
		for _, layer := range namespace.commands[name] {
			if layer.owner != 0 {
				continue
			}
			metadata := layer.metadata
			if metadata.GroupID != "" {
				if _, exists := baseGroups[metadata.GroupID]; !exists {
					metadata.GroupID = ""
				}
			}
			catalog.Commands = append(catalog.Commands, metadata)
			break
		}
	}
	return catalog
}

func (state *aggressorCommandState) removeScript(owner ScriptID) {
	if state == nil || owner == 0 {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	for _, namespace := range state.namespaces {
		for name, layers := range namespace.commands {
			kept := layers[:0]
			for _, layer := range layers {
				if layer.owner != owner {
					kept = append(kept, layer)
				}
			}
			clear(layers[len(kept):])
			if len(kept) == 0 {
				delete(namespace.commands, name)
			} else {
				namespace.commands[name] = kept
			}
		}
		namespace.commandOrder = filterAggressorCommandOrder(namespace.commandOrder, func(name string) bool {
			return len(namespace.commands[name]) != 0
		})
		for id, layers := range namespace.groups {
			kept := layers[:0]
			for _, layer := range layers {
				if layer.owner != owner {
					kept = append(kept, layer)
				}
			}
			clear(layers[len(kept):])
			if len(kept) == 0 {
				delete(namespace.groups, id)
			} else {
				namespace.groups[id] = kept
			}
		}
		namespace.groupOrder = filterAggressorCommandOrder(namespace.groupOrder, func(id string) bool {
			return len(namespace.groups[id]) != 0
		})
	}
}

// removeGeneration removes only registrations created by one exact execution
// generation. A later run of the same Script has a different pointer token and
// therefore remains effective when an older portable ScriptLoader generation
// is explicitly retired.
func (state *aggressorCommandState) removeGeneration(owner ScriptID, generation *scriptGeneration) {
	if state == nil || owner == 0 || generation == nil {
		return
	}
	state.mu.Lock()
	defer state.mu.Unlock()
	for _, namespace := range state.namespaces {
		for name, layers := range namespace.commands {
			kept := layers[:0]
			for _, layer := range layers {
				if layer.owner != owner || layer.generation != generation {
					kept = append(kept, layer)
				}
			}
			clear(layers[len(kept):])
			if len(kept) == 0 {
				delete(namespace.commands, name)
			} else {
				namespace.commands[name] = kept
			}
		}
		namespace.commandOrder = filterAggressorCommandOrder(namespace.commandOrder, func(name string) bool {
			return len(namespace.commands[name]) != 0
		})
		for id, layers := range namespace.groups {
			kept := layers[:0]
			for _, layer := range layers {
				if layer.owner != owner || layer.generation != generation {
					kept = append(kept, layer)
				}
			}
			clear(layers[len(kept):])
			if len(kept) == 0 {
				delete(namespace.groups, id)
			} else {
				namespace.groups[id] = kept
			}
		}
		namespace.groupOrder = filterAggressorCommandOrder(namespace.groupOrder, func(id string) bool {
			return len(namespace.groups[id]) != 0
		})
	}
}

func filterAggressorCommandOrder(order []string, keep func(string) bool) []string {
	filtered := order[:0]
	for _, name := range order {
		if keep(name) {
			filtered = append(filtered, name)
		}
	}
	clear(order[len(filtered):])
	return filtered
}

func (r *Runtime) registerAggressorCommandGroup(
	invocation Invocation,
	kind AggressorCommandKind,
	group AggressorCommandGroup,
) error {
	if r == nil || r.aggressorCommands == nil {
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
	r.aggressorCommands.registerGroup(kind, invocation.Script, generation, group)
	return nil
}

func (r *Runtime) registerAggressorCommand(
	invocation Invocation,
	kind AggressorCommandKind,
	metadata AggressorCommandMetadata,
) error {
	if r == nil || r.aggressorCommands == nil {
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
	r.aggressorCommands.registerCommand(kind, invocation.Script, generation, metadata)
	return nil
}
