package opfor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// profilerStatistic is the pure-Go counterpart of
// ScriptInstance.ProfilerStatistic. Statistics returned by profile remain
// live: later calls to an already-recorded function update the same object.
type profilerStatistic struct {
	mu           sync.RWMutex
	functionName string
	ticks        int64
	calls        int64
}

func (statistic *profilerStatistic) snapshot() (ticks, calls int64, functionName string) {
	if statistic == nil {
		return 0, 0, ""
	}
	statistic.mu.RLock()
	defer statistic.mu.RUnlock()
	return statistic.ticks, statistic.calls, statistic.functionName
}

func (statistic *profilerStatistic) String() string {
	ticks, calls, functionName := statistic.snapshot()
	return fmt.Sprintf("%ss %d %s", formatDouble(float64(ticks)/1000.0), calls, functionName)
}

func (statistic *profilerStatistic) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := strings.ReplaceAll(invocation.Class, "$", ".")
		return Bool(class == "Object" || class == "java.lang.Object" ||
			class == "ProfilerStatistic" || class == "ScriptInstance.ProfilerStatistic" ||
			class == "sleep.runtime.ScriptInstance.ProfilerStatistic"), true, nil
	}
	if invocation.Op != ObjectInvoke && invocation.Op != ObjectGet {
		return Null(), false, nil
	}
	ticks, calls, functionName := statistic.snapshot()
	switch invocation.Message {
	case "ticks":
		if len(invocation.Arguments) == 0 {
			return Long(ticks), true, nil
		}
	case "calls":
		if len(invocation.Arguments) == 0 {
			return Long(calls), true, nil
		}
	case "functionName":
		if len(invocation.Arguments) == 0 {
			return String(functionName), true, nil
		}
	case "toString":
		if len(invocation.Arguments) == 0 {
			return String(statistic.String()), true, nil
		}
	case "compareTo":
		if len(invocation.Arguments) == 1 {
			object, ok := invocation.Arg(0).Object()
			other, okStatistic := object.(*profilerStatistic)
			if !ok || !okStatistic || other == nil {
				return Null(), true, fmt.Errorf("java.lang.ClassCastException: expected sleep.runtime.ScriptInstance$ProfilerStatistic")
			}
			otherTicks, _, _ := other.snapshot()
			// ProfilerStatistic.compareTo returns (int)(other.ticks - ticks),
			// including Java's narrowing conversion.
			return Int(int32(otherTicks - ticks)), true, nil
		}
	}
	return Null(), false, nil
}

type scriptProfiler struct {
	mu         sync.RWMutex
	statistics map[string]*profilerStatistic
	total      int64
}

type profileCallFrame struct {
	profiler *scriptProfiler
	runtime  *Runtime
	function string
	started  time.Time
	total    int64
}

func (s *Script) profilerState() *scriptProfiler {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	profiler := s.profiler
	parent := s.forkParent
	s.mu.RUnlock()
	if profiler != nil {
		return profiler
	}
	// Sleep's forked ScriptInstances share metadata, including profiler
	// statistics. Resolve the parent's state lazily so fork construction does
	// not need a second synchronization path.
	if parent != nil {
		profiler = parent.profilerState()
	} else {
		profiler = &scriptProfiler{statistics: make(map[string]*profilerStatistic)}
	}
	s.mu.Lock()
	if s.profiler == nil {
		s.profiler = profiler
	}
	profiler = s.profiler
	s.mu.Unlock()
	return profiler
}

func (f *fiber) beginProfileCall(function string) *profileCallFrame {
	if f == nil || f.closure == nil || f.closure.script == nil || function == "" {
		return nil
	}
	if !f.profileCallsEnabled() {
		return nil
	}
	return f.beginEnabledProfileCall(function)
}

// beginClosureProfileCall defers closure name rendering until profiling is
// enabled. closure.String walks instructions and formats a source location;
// doing that for every ordinary Sleep call used to allocate even when the
// profiler was disabled.
func (f *fiber) beginClosureProfileCall(closure *scriptClosure, arguments []Argument) *profileCallFrame {
	if !f.profileCallsEnabled() {
		return nil
	}
	return f.beginEnabledProfileCall(closureInvocationProfileName(closure, arguments))
}

func (f *fiber) profileCallsEnabled() bool {
	if f == nil || f.closure == nil || f.closure.script == nil {
		return false
	}
	script := f.closure.script
	script.mu.RLock()
	enabled := script.debug&debugTraceCalls == debugTraceCalls
	script.mu.RUnlock()
	return enabled
}

func (f *fiber) beginEnabledProfileCall(function string) *profileCallFrame {
	if f == nil || f.closure == nil || f.closure.script == nil || function == "" {
		return nil
	}
	script := f.closure.script
	profiler := script.profilerState()
	if profiler == nil || script.runtime == nil || script.runtime.clock == nil {
		return nil
	}
	profiler.mu.RLock()
	total := profiler.total
	profiler.mu.RUnlock()
	return &profileCallFrame{
		profiler: profiler,
		runtime:  script.runtime,
		function: function,
		started:  script.runtime.clock.Now(),
		total:    total,
	}
}

func (f *fiber) finishProfileCall(frame *profileCallFrame, callErr error) {
	if frame == nil || frame.profiler == nil || frame.runtime == nil || !profileCallCompleted(callErr) {
		return
	}
	elapsed := frame.runtime.clock.Now().Sub(frame.started).Milliseconds()
	profiler := frame.profiler
	profiler.mu.Lock()
	nested := profiler.total - frame.total
	exclusive := elapsed - nested
	// Concurrent forks share profiler metadata just as Sleep does. Their
	// unrelated totals can overlap this call's wall time; never let that race
	// manufacture a negative statistic.
	if exclusive < 0 {
		exclusive = 0
	}
	statistic := profiler.statistics[frame.function]
	if statistic == nil {
		statistic = &profilerStatistic{functionName: frame.function}
		profiler.statistics[frame.function] = statistic
	}
	statistic.mu.Lock()
	statistic.ticks += exclusive
	statistic.calls++
	statistic.mu.Unlock()
	profiler.total += exclusive
	profiler.mu.Unlock()
}

func profileCallCompleted(err error) bool {
	if err == nil {
		return true
	}
	// Sleep represents language control and bridge warnings in the
	// ScriptEnvironment rather than throwing a Java RuntimeException. OPFOR
	// carries those states as errors while unwinding Go frames, so regard them
	// as completed calls for profiler purposes.
	var thrown *scriptThrow
	var exited *scriptExit
	var transfer *callCCTransfer
	var returned *inlineReturn
	var yielded *inlineYield
	var controlled *loopControl
	var warning *uncaughtScriptWarning
	return errors.As(err, &thrown) || errors.As(err, &exited) ||
		errors.As(err, &transfer) || errors.As(err, &returned) ||
		errors.As(err, &yielded) || errors.As(err, &controlled) || errors.As(err, &warning)
}

func closureProfileName(value Value) string {
	callable, ok := value.Function()
	if !ok {
		return ""
	}
	closure, ok := callable.(*scriptClosure)
	if !ok || closure == nil {
		return ""
	}
	name := closure.String()
	if marker := strings.LastIndexByte(name, '#'); marker >= 0 {
		name = name[:marker]
	}
	return name
}

func closureInvocationProfileName(closure *scriptClosure, arguments []Argument) string {
	for _, argument := range arguments {
		if argument.Name != "$0" {
			continue
		}
		name := argument.Resolve().String()
		if strings.HasPrefix(name, "&") && len(name) > 1 {
			return name
		}
	}
	return closureProfileName(FunctionValue(closure))
}

func (s *Script) profilerStatistics() []*profilerStatistic {
	profiler := s.profilerState()
	if profiler == nil {
		return nil
	}
	profiler.mu.RLock()
	statistics := make([]*profilerStatistic, 0, len(profiler.statistics))
	for _, statistic := range profiler.statistics {
		statistics = append(statistics, statistic)
	}
	profiler.mu.RUnlock()
	sort.SliceStable(statistics, func(left, right int) bool {
		leftTicks, _, leftName := statistics[left].snapshot()
		rightTicks, _, rightName := statistics[right].snapshot()
		if leftTicks != rightTicks {
			return leftTicks > rightTicks
		}
		// HashMap iteration is not a language guarantee. A tie-break keeps the
		// pure-Go result deterministic without changing Sleep's tick ordering.
		return leftName < rightName
	})
	return statistics
}

func (r *Runtime) profile(_ context.Context, invocation Invocation) (Value, error) {
	script := r.script(invocation.Script)
	if script == nil {
		return ArrayValue(NewReadOnlyArray()), nil
	}
	statistics := script.profilerStatistics()
	if err := reserveCollectionEntries(invocation.Runtime, len(statistics)); err != nil {
		return Null(), err
	}
	values := make([]Value, len(statistics))
	for index, statistic := range statistics {
		values[index] = ObjectValue(statistic)
	}
	return ArrayValue(NewReadOnlyArray(values...)), nil
}

func portableObjectProfileName(invocation ObjectInvocation) string {
	if invocation.Op != ObjectInvoke {
		return ""
	}
	if invocation.Target.Kind() == KindString && invocation.Message == "length" && len(invocation.Arguments) == 0 {
		return "public int java.lang.String.length()"
	}
	return ""
}
