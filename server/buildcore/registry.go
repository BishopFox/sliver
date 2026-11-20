package buildcore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"github.com/bishopfox/sliver/server/buildplan"
)

// Result is the server-owned output of one core build. The adapter returns
// exact final bytes and stable identifiers after its own build work. Sliver's
// adapter also saves its legacy database metadata for compatibility callers.
type Result struct {
	BuildID           string
	ImplantName       string
	FileName          string
	Data              []byte
	ToolchainIdentity string
}

// CoreProduct holds one accepted core artifact until the ordered transform
// pipeline has produced its final bytes. SaveFinal is an installed core's
// compatibility hook (for example Sliver's legacy build database). Close
// releases private build inputs even when a later transform fails. A core
// without legacy storage may leave either hook nil.
type CoreProduct struct {
	Result    Result
	SaveFinal func(context.Context, buildplan.Artifact, string, []byte) (string, error)
	Close     func() error
}

// CoreStepAdapter compiles only the first, core step of a resolved plan.
// Transform execution and final artifact storage are server-owned.
type CoreStepAdapter interface {
	BuildCore(context.Context, buildplan.Plan) (CoreProduct, error)
}

// Adapter is a preinstalled compiler integration. The generic executor sees
// only an immutable plan and output bytes; Sliver-only configuration stays in
// the Sliver adapter. A future core can register another executor ID without
// changing the planner or this interface.
type Adapter interface {
	ExecutorID() string
	Build(context.Context, buildplan.Plan) (Result, error)
}

// Availability reports whether a registered core can execute one declared
// target and toolchain. The planner asks this through the registry; it never
// imports core-specific source packages or assumes one compiler matrix.
type Availability interface {
	Supports(buildplan.Target, buildplan.Toolchain) bool
}

// PeakMemoryEstimator is optional. A zero result means the installed executor
// has no per-step estimate, so the scheduler must use the spec estimate or
// reserve a builder exclusively.
type PeakMemoryEstimator interface {
	EstimatedPeakMemoryBytes(buildplan.Target, buildplan.Toolchain) uint64
}

type Registry struct {
	mu       sync.RWMutex
	adapters map[string]Adapter
}

func NewRegistry() *Registry {
	return &Registry{adapters: make(map[string]Adapter)}
}

func (registry *Registry) Register(adapter Adapter) error {
	if registry == nil || adapter == nil {
		return errors.New("missing build core registry or adapter")
	}
	id := adapter.ExecutorID()
	if id == "" {
		return errors.New("build core adapter has no executor ID")
	}
	registry.mu.Lock()
	defer registry.mu.Unlock()
	if _, exists := registry.adapters[id]; exists {
		return fmt.Errorf("build core executor %q is already registered", id)
	}
	registry.adapters[id] = adapter
	return nil
}

func (registry *Registry) Get(executorID string) (Adapter, bool) {
	if registry == nil {
		return nil, false
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	adapter, ok := registry.adapters[executorID]
	return adapter, ok
}

func (registry *Registry) InstalledExecutorIDs() []string {
	if registry == nil {
		return nil
	}
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	ids := make([]string, 0, len(registry.adapters))
	for id := range registry.adapters {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

var defaultRegistry = NewRegistry()

func RegisterDefault(adapter Adapter) error { return defaultRegistry.Register(adapter) }

func DefaultAdapter(executorID string) (Adapter, bool) { return defaultRegistry.Get(executorID) }

func InstalledExecutorIDs() []string { return defaultRegistry.InstalledExecutorIDs() }
