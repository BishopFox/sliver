package opfor

import (
	"errors"
	"fmt"
	"sort"
)

// ProfileStatisticSnapshot is one detached profiler counter. Ticks are
// exclusive elapsed milliseconds, matching Sleep's ProfilerStatistic ticks;
// Calls is the number of completed calls recorded for FunctionName.
//
// The JSON field names are part of OPFOR's controller-facing compatibility
// surface and are shared by the opfor serve profile response.
type ProfileStatisticSnapshot struct {
	FunctionName string `json:"function_name"`
	Ticks        int64  `json:"ticks"`
	Calls        int64  `json:"calls"`
}

// ScriptProfileSnapshot is a detached, deterministically ordered profile for
// one exact loaded script. Mutating Statistics does not affect runtime state.
type ScriptProfileSnapshot struct {
	Script     ScriptID                   `json:"script"`
	Statistics []ProfileStatisticSnapshot `json:"statistics"`
}

type scriptNotLoadedError struct {
	id ScriptID
}

func (err *scriptNotLoadedError) Error() string {
	if err == nil {
		return ErrScriptUnloaded.Error()
	}
	return fmt.Sprintf("opfor: script %d is not loaded", err.id)
}

func (*scriptNotLoadedError) Unwrap() error { return ErrScriptUnloaded }

// ScriptByID resolves one exact, active runtime-local script identity. It does
// not fall back to a primary script, path, or newest load. A missing or already
// unloading identity returns an error matching ErrScriptUnloaded.
func (r *Runtime) ScriptByID(id ScriptID) (*Script, error) {
	if r == nil {
		return nil, errors.New("opfor: runtime is nil")
	}
	if id == 0 {
		return nil, errors.New("opfor: script ID must be nonzero")
	}
	r.mu.RLock()
	script := r.scripts[id]
	r.mu.RUnlock()
	if script == nil {
		return nil, &scriptNotLoadedError{id: id}
	}
	script.mu.RLock()
	active := script.active
	script.mu.RUnlock()
	if !active {
		return nil, &scriptNotLoadedError{id: id}
	}
	return script, nil
}

// DebugFlags returns the script's current Sleep debug bitmask without invoking
// script code. Unload admission takes precedence and returns ErrScriptUnloaded.
func (s *Script) DebugFlags() (int32, error) {
	if s == nil {
		return 0, ErrScriptUnloaded
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if !s.active {
		return 0, ErrScriptUnloaded
	}
	return s.debug, nil
}

// SetDebugFlags replaces the script's Sleep debug bitmask and returns the new
// value, matching debug(flags). The update is atomic with unload admission and
// does not execute script code.
func (s *Script) SetDebugFlags(flags int32) (int32, error) {
	if s == nil {
		return 0, ErrScriptUnloaded
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return 0, ErrScriptUnloaded
	}
	s.debug = flags
	return s.debug, nil
}

// SnapshotProfile returns a detached snapshot of the counters accumulated
// while Sleep's DEBUG_TRACE_CALLS bit was enabled. Entries are sorted by ticks
// descending and then function name, matching OPFOR's deterministic profile()
// ordering. The operation does not create profiler state or invoke script code.
func (s *Script) SnapshotProfile() (ScriptProfileSnapshot, error) {
	if s == nil {
		return ScriptProfileSnapshot{}, ErrScriptUnloaded
	}
	s.mu.RLock()
	if !s.active {
		s.mu.RUnlock()
		return ScriptProfileSnapshot{}, ErrScriptUnloaded
	}
	id := s.id
	profiler := s.profiler
	parent := s.forkParent
	s.mu.RUnlock()

	// Forks share the parent's profiler. Avoid profilerState here because a
	// read-only controller request must not allocate or publish runtime state.
	if profiler == nil && parent != nil {
		parent.mu.RLock()
		profiler = parent.profiler
		parent.mu.RUnlock()
	}
	report := ScriptProfileSnapshot{
		Script:     id,
		Statistics: make([]ProfileStatisticSnapshot, 0),
	}
	if profiler == nil {
		return report, nil
	}

	profiler.mu.RLock()
	report.Statistics = make([]ProfileStatisticSnapshot, 0, len(profiler.statistics))
	for _, statistic := range profiler.statistics {
		ticks, calls, functionName := statistic.snapshot()
		report.Statistics = append(report.Statistics, ProfileStatisticSnapshot{
			FunctionName: functionName,
			Ticks:        ticks,
			Calls:        calls,
		})
	}
	profiler.mu.RUnlock()
	sort.SliceStable(report.Statistics, func(left, right int) bool {
		if report.Statistics[left].Ticks != report.Statistics[right].Ticks {
			return report.Statistics[left].Ticks > report.Statistics[right].Ticks
		}
		return report.Statistics[left].FunctionName < report.Statistics[right].FunctionName
	})
	return report, nil
}
