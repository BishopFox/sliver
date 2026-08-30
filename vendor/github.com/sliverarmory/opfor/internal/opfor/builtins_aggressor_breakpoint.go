package opfor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/sliverarmory/opfor/internal/bytecode"
)

// aggressorBreakpoint is interpreter-owned because only the active fiber can
// distinguish current locals, closure captures, globals, and caller frames.
// The optional provider receives data only after those private structures have
// been converted into a detached public snapshot.
func (r *Runtime) aggressorBreakpoint(ctx context.Context, invocation Invocation) (Value, error) {
	if err := requireExactAggressorClientArguments(invocation, 0); err != nil {
		return Null(), err
	}
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	fiber := currentFiber(ctx)
	if fiber == nil || fiber.closure == nil || fiber.closure.script == nil || fiber.closure.script.runtime != r {
		return Null(), errors.New("&brk: requires active script execution")
	}
	snapshot, err := r.buildAggressorBreakpointSnapshot(ctx, invocation, fiber)
	if err != nil {
		return Null(), err
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}

	if provider := r.aggressorBreakpointProvider; !isNilInterface(provider) {
		// The provider owns an independent copy. A UI may annotate or retain it
		// without changing the hash brk returns to the paused script.
		providerSnapshot, err := cloneAggressorBreakpointSnapshotAtRuntime(r, snapshot)
		if err != nil {
			return Null(), err
		}
		if err := provider.HandleAggressorBreakpoint(ctx, providerSnapshot); err != nil {
			return Null(), preserveNativeBoundaryError(ctx, err)
		}
		if err := executionContextError(ctx); err != nil {
			return Null(), err
		}
		return aggressorBreakpointSnapshotValue(r, snapshot)
	}

	result, err := aggressorBreakpointSnapshotValue(r, snapshot)
	if err != nil {
		return Null(), err
	}
	if _, err := io.WriteString(r.consoleOutputWriter(), result.Describe()+"\n"); err != nil {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return result, nil
}

func (r *Runtime) buildAggressorBreakpointSnapshot(
	ctx context.Context,
	invocation Invocation,
	active *fiber,
) (AggressorBreakpointSnapshot, error) {
	script := active.closure.script
	scriptName := invocation.Span.Source
	if script.program != nil && script.program.source.Name != "" {
		scriptName = script.program.source.Name
	}
	scriptName = sleepSourceDisplayName(scriptName)

	type scopeSnapshot struct {
		values map[string]Value
		err    error
	}
	scopes := make(map[*scope]scopeSnapshot)
	snapshotScope := func(level *scope, span Span) (map[string]Value, error) {
		if level == nil {
			return map[string]Value{}, nil
		}
		if cached, ok := scopes[level]; ok {
			return cached.values, cached.err
		}
		values, err := level.snapshotOwnAt(ctx, span)
		if values == nil {
			values = map[string]Value{}
		}
		scopes[level] = scopeSnapshot{values: values, err: err}
		return values, err
	}

	locals, err := snapshotScope(active.scope, invocation.Span)
	if err != nil {
		return AggressorBreakpointSnapshot{}, err
	}
	root := active.scope.root
	globals, err := snapshotScope(root, invocation.Span)
	if err != nil {
		return AggressorBreakpointSnapshot{}, err
	}

	closureVariables := make(map[string]Value)
	seenScopes := make(map[*scope]struct{})
	for level := active.scope.parent; level != nil && level != root; level = level.parent {
		if err := executionContextError(ctx); err != nil {
			return AggressorBreakpointSnapshot{}, err
		}
		if _, duplicate := seenScopes[level]; duplicate {
			break
		}
		seenScopes[level] = struct{}{}
		values, scopeErr := snapshotScope(level, invocation.Span)
		if scopeErr != nil {
			return AggressorBreakpointSnapshot{}, scopeErr
		}
		for name, value := range values {
			if _, nearer := closureVariables[name]; !nearer {
				closureVariables[name] = value
			}
		}
	}

	frames := make([]AggressorBreakpointStackFrame, 0, 4)
	callStack := make([]string, 0, 4)
	seenFibers := make(map[*fiber]struct{})
	for current := active; current != nil; current = current.caller {
		if err := executionContextError(ctx); err != nil {
			return AggressorBreakpointSnapshot{}, err
		}
		if _, duplicate := seenFibers[current]; duplicate {
			break
		}
		seenFibers[current] = struct{}{}
		span := breakpointFiberLocation(current)
		if current == active {
			span = invocation.Span
		}
		frameLocals, scopeErr := snapshotScope(current.scope, span)
		if scopeErr != nil {
			return AggressorBreakpointSnapshot{}, scopeErr
		}
		function := breakpointFunctionName(current.function)
		frames = append(frames, AggressorBreakpointStackFrame{
			Function: function, SourceLocation: span, LocalVariables: frameLocals,
		})
		callStack = append(callStack, function)
	}

	snapshot := AggressorBreakpointSnapshot{
		RuntimeID:        r.ID(),
		Script:           script.id,
		ScriptName:       scriptName,
		SourceLocation:   invocation.Span,
		Timestamp:        r.clock.Now(),
		LocalVariables:   locals,
		GlobalVariables:  globals,
		ClosureVariables: closureVariables,
		StackFrames:      frames,
		CallStack:        callStack,
		CurrentFunction:  breakpointFunctionName(active.function),
	}
	// Detach variable Values only after every provider-backed read succeeds.
	// No partially built snapshot escapes on an authoritative provider error.
	return cloneAggressorBreakpointSnapshotAtRuntime(r, snapshot)
}

func breakpointFiberLocation(frame *fiber) Span {
	if frame == nil || frame.function == nil || frame.pc < 0 || frame.pc >= len(frame.function.Instructions) {
		return Span{}
	}
	return frame.function.Instructions[frame.pc].Span
}

func breakpointFunctionName(function *bytecode.Function) string {
	if function == nil {
		return "<anonymous>"
	}
	name := strings.TrimSpace(function.Name)
	if name == "" {
		return "<anonymous>"
	}
	for _, prefix := range []string{"sub ", "inline "} {
		if strings.HasPrefix(name, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(name, prefix))
		}
	}
	return name
}

func breakpointSourceLocation(span Span) string {
	name := sleepSourceDisplayName(span.Source)
	line := sleepDisplayLine(span)
	if name == "" {
		if line <= 0 {
			return ""
		}
		return fmt.Sprintf("%d", line)
	}
	if line <= 0 {
		return name
	}
	return fmt.Sprintf("%s:%d", name, line)
}

func aggressorBreakpointSnapshotValue(runtime *Runtime, snapshot AggressorBreakpointSnapshot) (Value, error) {
	entries := uint64(9 + len(snapshot.LocalVariables) + len(snapshot.GlobalVariables) + len(snapshot.ClosureVariables))
	entries += uint64(len(snapshot.StackFrames) + len(snapshot.CallStack))
	for _, frame := range snapshot.StackFrames {
		entries += uint64(3 + len(frame.LocalVariables))
	}
	if err := reserveCollectionEntryAmount(runtime, entries); err != nil {
		return Null(), err
	}
	result := NewOrderedHash()
	result.Set("script_name", String(snapshot.ScriptName))
	result.Set("source_location", String(breakpointSourceLocation(snapshot.SourceLocation)))
	result.Set("timestamp", Long(snapshot.Timestamp.UnixMilli()))
	result.Set("local_variables", breakpointVariablesValue(snapshot.LocalVariables))
	result.Set("global_variables", breakpointVariablesValue(snapshot.GlobalVariables))
	result.Set("closure_variables", breakpointVariablesValue(snapshot.ClosureVariables))

	frames := make([]Value, len(snapshot.StackFrames))
	for index, frame := range snapshot.StackFrames {
		value := NewOrderedHash()
		value.Set("function", String(frame.Function))
		value.Set("source_location", String(breakpointSourceLocation(frame.SourceLocation)))
		value.Set("local_variables", breakpointVariablesValue(frame.LocalVariables))
		frames[index] = HashValue(value)
	}
	result.Set("stack_frames", ArrayValue(NewArray(frames...)))

	stack := make([]Value, len(snapshot.CallStack))
	for index, function := range snapshot.CallStack {
		stack[index] = String(function)
	}
	result.Set("call_stack", ArrayValue(NewArray(stack...)))
	result.Set("current_function", String(snapshot.CurrentFunction))
	return HashValue(result), nil
}

func breakpointVariablesValue(variables map[string]Value) Value {
	hash := NewOrderedHash()
	names := make([]string, 0, len(variables))
	for name := range variables {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		hash.Set(name, variables[name])
	}
	return HashValue(hash)
}

func cloneAggressorBreakpointSnapshot(snapshot AggressorBreakpointSnapshot) AggressorBreakpointSnapshot {
	clone, _ := cloneAggressorBreakpointSnapshotAtRuntime(nil, snapshot)
	return clone
}

func cloneAggressorBreakpointSnapshotAtRuntime(runtime *Runtime, snapshot AggressorBreakpointSnapshot) (AggressorBreakpointSnapshot, error) {
	clone := snapshot
	cloner := breakpointValueCloner{
		runtime: runtime,
		arrays:  make(map[*Array]*Array),
		hashes:  make(map[*Hash]*Hash),
	}
	clone.LocalVariables = cloner.values(snapshot.LocalVariables)
	clone.GlobalVariables = cloner.values(snapshot.GlobalVariables)
	clone.ClosureVariables = cloner.values(snapshot.ClosureVariables)
	clone.StackFrames = make([]AggressorBreakpointStackFrame, len(snapshot.StackFrames))
	for index, frame := range snapshot.StackFrames {
		clone.StackFrames[index] = frame
		clone.StackFrames[index].LocalVariables = cloner.values(frame.LocalVariables)
	}
	clone.CallStack = append([]string(nil), snapshot.CallStack...)
	return clone, cloner.err
}

type breakpointValueCloner struct {
	runtime *Runtime
	err     error
	arrays  map[*Array]*Array
	hashes  map[*Hash]*Hash
}

func (cloner *breakpointValueCloner) values(values map[string]Value) map[string]Value {
	if values == nil {
		return nil
	}
	result := make(map[string]Value, len(values))
	for name, value := range values {
		result[name] = cloner.value(value)
	}
	return result
}

func (cloner *breakpointValueCloner) value(value Value) Value {
	if cloner.err != nil {
		return Null()
	}
	result := value
	result.stringUnits = append([]uint16(nil), value.stringUnits...)
	result.stringRaw = append([]bool(nil), value.stringRaw...)
	switch value.kind {
	case KindArray:
		source, _ := value.Array()
		if source == nil {
			return Null()
		}
		if existing := cloner.arrays[source]; existing != nil {
			result.data = existing
			return result
		}
		items := source.Values()
		if err := reserveCollectionEntries(cloner.runtime, len(items)); err != nil {
			cloner.err = err
			return Null()
		}
		target := NewArray()
		cloner.arrays[source] = target
		for _, item := range items {
			target.Append(cloner.value(item))
			if cloner.err != nil {
				return Null()
			}
		}
		result.data = target
	case KindHash:
		source, _ := value.Hash()
		if source == nil {
			return Null()
		}
		if existing := cloner.hashes[source]; existing != nil {
			result.data = existing
			return result
		}
		entries, ordered, accessOrdered, err := breakpointHashSnapshot(source, cloner.runtime)
		if err != nil {
			cloner.err = err
			return Null()
		}
		if err := reserveCollectionEntries(cloner.runtime, len(entries)); err != nil {
			cloner.err = err
			return Null()
		}
		target := NewHash()
		cloner.hashes[source] = target
		for _, entry := range entries {
			key, item := entry.key, entry.value
			target.SetValue(cloner.value(key), cloner.value(item))
			if cloner.err != nil {
				return Null()
			}
		}
		target.mu.Lock()
		target.ordered = ordered
		target.accessOrdered = accessOrdered
		target.mu.Unlock()
		result.data = target
	}
	return result
}

// breakpointHashSnapshot reads ordinary hash cells without calling GetValue:
// access-ordered hashes mutate their traversal order on GetValue, so debugger
// inspection must stay observational. Read-only wrapper backends expose an
// explicit detached dataSnapshot boundary; materialize that first rather than
// pretending their live, non-owning storage is an ordinary Hash.
func breakpointHashSnapshot(source *Hash, runtime *Runtime) ([]hashBackendEntry, bool, bool, error) {
	if source == nil {
		return nil, false, false, nil
	}
	if source.backend != nil {
		detached, err := source.backend.dataSnapshotReserved(func(count int) error {
			return reserveCollectionEntries(runtime, count)
		})
		if err != nil {
			return nil, false, false, err
		}
		if detached == nil || detached == source {
			return nil, false, false, nil
		}
		return breakpointHashSnapshot(detached, runtime)
	}
	source.mu.RLock()
	// Recreate the source's insertion/access order, not its currently rendered
	// bucket traversal. Re-inserting Java-bucket order into an ordinary hash can
	// reverse colliding keys a second time and would make the clone observably
	// different even though the live hash stayed untouched.
	keys := append([]string(nil), source.order...)
	entries := make([]hashBackendEntry, 0, len(keys))
	for _, keyText := range keys {
		cell := source.items[keyText]
		if cell == nil {
			continue
		}
		entries = append(entries, hashBackendEntry{
			key: source.keyValueLocked(keyText), value: cell.Get(),
		})
	}
	ordered, accessOrdered := source.ordered, source.accessOrdered
	source.mu.RUnlock()
	return entries, ordered, accessOrdered, nil
}
