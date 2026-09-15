package opfor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"time"
)

// forkTask is the execution token stored on the parent's duplex I/O handle.
// Completion publishes one immutable result which wait may retrieve any number
// of times. The child Script remains runtime-owned until its parent unloads so
// closures returned from a fork continue to have a live execution instance.
type forkTask struct {
	child *Script

	parentReader *io.PipeReader
	parentWriter *io.PipeWriter
	childReader  *io.PipeReader
	childWriter  *io.PipeWriter

	cancel         context.CancelCauseFunc
	releaseContext func()
	done           chan struct{}
	once           sync.Once
	result         Value
	err            error

	traceMu     sync.Mutex
	launchTrace *forkLaunchTrace
}

type forkLaunchTrace struct {
	call   string
	span   Span
	result Value
}

type forkFunctionTemplate struct {
	name     string
	callable Callable
	inline   bool
}

type forkScriptSnapshot struct {
	debug        int32
	imports      map[string]string
	removedFuncs map[string]struct{}
	functions    []forkFunctionTemplate
	firstID      uint64
}

func (r *Runtime) fork(ctx context.Context, invocation Invocation) (Value, error) {
	parent := r.script(invocation.Script)
	if parent == nil || !parent.Active() {
		return Null(), ErrScriptUnloaded
	}
	if len(invocation.Arguments) == 0 {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument("expected &closure--received: " + Null().Describe())
		}
		return Null(), fmt.Errorf("&%s: expected &closure--received: %s", builtinName(invocation.Name), Null().Describe())
	}

	targetValue := invocation.Arg(0)
	callable, ok := targetValue.Function()
	if !ok {
		if currentFiber(ctx) != nil {
			return Null(), sleepBridgeIllegalArgument("expected &closure--received: " + targetValue.Describe())
		}
		return Null(), fmt.Errorf("&%s: expected &closure--received: %s", builtinName(invocation.Name), targetValue.Describe())
	}
	target, ok := callable.(*scriptClosure)
	if !ok || target == nil || target.function == nil {
		return Null(), fmt.Errorf("&%s: expected script closure--received: %s", builtinName(invocation.Name), targetValue.Describe())
	}

	type forkBinding struct {
		name  string
		value Value
	}
	bindings := make([]forkBinding, 0, len(invocation.Arguments)-1)
	for _, argument := range invocation.Arguments[1:] {
		name, value, ok := sleepNamedArgument(argument)
		if !ok {
			return Null(), fmt.Errorf("&%s: attempted to pass a malformed key value pair: %s", builtinName(invocation.Name), argument.Resolve().Describe())
		}
		bindings = append(bindings, forkBinding{name: name, value: value})
	}

	snapshot, err := snapshotForkScript(parent)
	if err != nil {
		return Null(), err
	}
	child, err := r.newForkScript(ctx, parent, snapshot, invocation.Span)
	if err != nil {
		return Null(), err
	}
	runnable := cloneForkFunctions(child, snapshot, target)

	toParent, fromChild := io.Pipe()
	toChild, fromParent := io.Pipe()
	parentHandle := newIOHandle("fork", toParent, fromParent, true, true, false).withRuntimeOutputAccount(r.resources)
	childHandle := newIOHandle("fork-source", toChild, fromChild, true, true, false).withRuntimeOutputAccount(r.resources)

	if err := child.setGlobalAt(ctx, "$source", ObjectValue(childHandle), invocation.Span); err != nil {
		_ = parentHandle.close()
		_ = childHandle.close()
		r.removeForkScript(child)
		return Null(), err
	}
	for _, binding := range bindings {
		// A fork argument always creates a new cell. Value itself is copied by
		// value, which deliberately preserves the backing identity of arrays,
		// hashes, closures, and opaque objects.
		if err := child.setGlobalAt(ctx, binding.name, binding.value, invocation.Span); err != nil {
			_ = parentHandle.close()
			_ = childHandle.close()
			r.removeForkScript(child)
			return Null(), err
		}
	}

	taskContext, releaseTaskContext, cancel := newAsynchronousExecutionTaskContext(ctx)
	task := &forkTask{
		child:        child,
		parentReader: toParent, parentWriter: fromParent,
		childReader: toChild, childWriter: fromChild,
		cancel: cancel, releaseContext: releaseTaskContext,
		done: make(chan struct{}), result: Null(),
	}
	parentHandle.setTask(task)
	childHandle.setTask(task)
	child.forkTask = task
	task.deferLaunchTrace(currentFiber(ctx), ObjectValue(parentHandle), builtinName(invocation.Name))

	parent.mu.Lock()
	if !parent.active {
		parent.mu.Unlock()
		task.cancelAndClose()
		r.removeForkScript(child)
		return Null(), ErrScriptUnloaded
	}
	if parent.forkTasks == nil {
		parent.forkTasks = make(map[*forkTask]struct{})
	}
	parent.forkTasks[task] = struct{}{}
	parent.mu.Unlock()

	go func() {
		// The fork goroutine inherits cancellation and quota state, but it is a
		// distinct ScriptInstance. Do not let the parent's active evaluator fiber
		// become the child's caller: that would let child callcc tracing mutate a
		// pending parent call trace (most visibly wait(...)).
		childContext := withCurrentFiber(taskContext, nil)
		result, runErr := runnable.invokeFresh(withExecutionMeter(childContext, r), nil)
		task.flushLaunchTrace(&fiber{closure: runnable})
		task.complete(result, runErr)
	}()

	return ObjectValue(parentHandle), nil
}

func (task *forkTask) deferLaunchTrace(caller *fiber, result Value, function string) {
	if task == nil || caller == nil || len(caller.callTraces) == 0 {
		return
	}
	frame := caller.callTraces[len(caller.callTraces)-1]
	if frame == nil || frame.deferred || !strings.HasPrefix(frame.call, "&"+strings.TrimPrefix(function, "&")+"(") {
		return
	}
	frame.deferred = true
	task.traceMu.Lock()
	task.launchTrace = &forkLaunchTrace{call: frame.call, span: frame.span, result: result}
	task.traceMu.Unlock()
}

func (task *forkTask) flushLaunchTrace(emitter *fiber) {
	if task == nil || emitter == nil {
		return
	}
	task.traceMu.Lock()
	trace := task.launchTrace
	task.launchTrace = nil
	task.traceMu.Unlock()
	if trace != nil {
		emitter.writeCallTrace(trace.call, trace.result, nil, trace.span)
	}
}

func (f *fiber) flushForkLaunchTraceBeforeCall(name string) {
	if f == nil || f.closure == nil || f.closure.script == nil || f.closure.script.forkTask == nil {
		return
	}
	switch strings.TrimPrefix(name, "&") {
	case "print", "println", "printf", "printAll", "warn":
		return
	}
	f.closure.script.forkTask.flushLaunchTrace(f)
}

func (f *fiber) flushForkLaunchTraceAfterCall() {
	if f == nil || f.closure == nil || f.closure.script == nil || f.closure.script.forkTask == nil {
		return
	}
	f.closure.script.forkTask.flushLaunchTrace(f)
}

func (r *Runtime) wait(ctx context.Context, invocation Invocation) (Value, error) {
	handle, ok := ioHandleValue(invocation.Arg(0))
	if !ok {
		if len(invocation.Arguments) == 0 && currentFiber(ctx) != nil {
			return Null(), sleepBridgeNullValue()
		}
		return Null(), fmt.Errorf("&%s: expected I/O handle argument, received: %s", builtinName(invocation.Name), invocation.Arg(0).Describe())
	}
	worker := handle.getWorker()
	workerCompleted, err := r.waitIOWorker(ctx, invocation, worker)
	workerErr := err
	if workerCompleted {
		if failure, ok := worker.(sleepIOWorkerFailure); ok {
			workerErr = errors.Join(workerErr, failure.sleepIOWorkerError())
		}
	}
	if process := handle.getProcess(); process != nil {
		// ProcessObject invokes IOObject.wait for its current reader/fork/socket
		// thread, then waits for the process regardless of an exception or soft
		// timeout from that first step. Preserve both errors after the child has
		// been reaped.
		result, processErr := process.wait(ctx, r, invocation)
		return result, errors.Join(workerErr, processErr)
	}
	if workerErr != nil {
		return Null(), workerErr
	}
	if !workerCompleted {
		return Null(), nil
	}

	// Fork completion stores an IOObject token independently from its mutable
	// Thread slot. A later read may replace that slot while the token remains
	// available once the fork eventually completes.
	if task := handle.getTask(); task != nil {
		select {
		case <-task.done:
			return task.result, task.err
		default:
		}
	}
	return Null(), nil
}

func (r *Runtime) waitIOWorker(ctx context.Context, invocation Invocation, worker sleepIOWorker) (bool, error) {
	if worker == nil || worker.sleepIOWorkerDone() == nil {
		return true, nil
	}
	done := worker.sleepIOWorkerDone()
	// Thread.join is only attempted for a live worker. Completed workers ignore
	// every timeout value, including negative values.
	select {
	case <-done:
		return true, nil
	default:
	}

	timeout := int64(0)
	if len(invocation.Arguments) > 1 {
		timeout = sleepInt64(invocation.Arg(1))
	}
	if timeout < 0 {
		_, err := r.flagSourceError(invocation, errors.New("java.lang.IllegalArgumentException: timeout value is negative"))
		return false, err
	}
	if timeout == 0 || timeout > int64((time.Duration(1<<63-1))/time.Millisecond) {
		// Java accepts every nonnegative long. A value beyond time.Duration's
		// range is effectively unbounded for this process.
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case <-done:
			return true, nil
		}
	}

	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-done:
		return true, nil
	case <-timer.C:
		// Prefer completion at the timeout boundary: IOObject checks isAlive
		// after join returns before it flags the soft timeout.
		select {
		case <-done:
			return true, nil
		default:
		}
		_, err := r.flagSourceError(invocation, errors.New("java.io.IOException: wait on object timed out"))
		return false, err
	}
}

func snapshotForkScript(parent *Script) (forkScriptSnapshot, error) {
	parent.mu.Lock()
	defer parent.mu.Unlock()
	if !parent.active {
		return forkScriptSnapshot{}, ErrScriptUnloaded
	}

	snapshot := forkScriptSnapshot{
		debug:        parent.debug,
		imports:      make(map[string]string, len(parent.imports)),
		removedFuncs: make(map[string]struct{}, len(parent.removedFuncs)),
	}
	for name, target := range parent.imports {
		snapshot.imports[name] = target
	}
	for name := range parent.removedFuncs {
		snapshot.removedFuncs[name] = struct{}{}
	}

	// Keep only the current value for each name, ordered by the declaration or
	// setf operation which installed that current value. This gives child
	// closures deterministic IDs and matches ScriptInstance.makeSafe's fresh
	// SleepClosure state without retaining a continuation from the parent.
	seen := make(map[string]struct{}, len(parent.functions))
	reversed := make([]string, 0, len(parent.functions))
	for index := len(parent.functionOrder) - 1; index >= 0; index-- {
		name := parent.functionOrder[index]
		if _, exists := seen[name]; exists {
			continue
		}
		if parent.functions[name] == nil {
			continue
		}
		seen[name] = struct{}{}
		reversed = append(reversed, name)
	}
	for left, right := 0, len(reversed)-1; left < right; left, right = left+1, right-1 {
		reversed[left], reversed[right] = reversed[right], reversed[left]
	}
	ordered := reversed
	remaining := make([]string, 0, len(parent.functions)-len(seen))
	for name, callable := range parent.functions {
		if callable == nil {
			continue
		}
		if _, exists := seen[name]; !exists {
			remaining = append(remaining, name)
		}
	}
	sort.Strings(remaining)
	ordered = append(ordered, remaining...)

	closureCount := uint64(0)
	for _, name := range ordered {
		callable := parent.functions[name]
		template := forkFunctionTemplate{name: name, callable: callable}
		if closure, isScript := callable.(*scriptClosure); isScript && closure != nil {
			template.inline = closure.inline
			if !closure.inline {
				closureCount++
			}
		}
		snapshot.functions = append(snapshot.functions, template)
	}

	snapshot.firstID = parent.nextClosure
	// Reserve IDs for every cloned named closure and for the fresh runnable.
	// SleepClosure's JVM counter is process-global; advancing the parent keeps
	// subsequent parent and sibling closure descriptions collision-free.
	parent.nextClosure += closureCount + 1
	return snapshot, nil
}

func (r *Runtime) newForkScript(ctx context.Context, parent *Script, snapshot forkScriptSnapshot, span Span) (*Script, error) {
	r.mu.Lock()
	if r.closing || r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	r.nextScript++
	id := r.nextScript
	r.mu.Unlock()
	globals, err := parent.globals.forkRootAt(ctx, r.ID(), id, span)
	if err != nil {
		return nil, err
	}
	child := &Script{
		runtime: r, program: parent.program, globals: globals, active: true,
		debug: snapshot.debug, functions: make(map[string]Callable, len(snapshot.functions)),
		removedFuncs: snapshot.removedFuncs, imports: snapshot.imports,
		nextClosure: snapshot.firstID, forkParent: parent,
	}
	initializeScriptExecution(child)
	r.mu.Lock()
	if r.closing || r.closed {
		r.mu.Unlock()
		child.mu.Lock()
		child.active = false
		child.mu.Unlock()
		child.executionCancel()
		return nil, ErrRuntimeClosed
	}
	child.id = id
	r.scripts[child.id] = child
	r.mu.Unlock()
	return child, nil
}

func cloneForkFunctions(child *Script, snapshot forkScriptSnapshot, target *scriptClosure) *scriptClosure {
	for _, template := range snapshot.functions {
		if source, ok := template.callable.(*scriptClosure); ok && source != nil {
			var clone *scriptClosure
			if template.inline {
				clone = child.newInline(source.function, child.globals)
			} else {
				clone = child.newClosure(source.function, child.globals)
			}
			child.functions[template.name] = clone
		} else if native, ok := template.callable.(*scriptNativeCallable); ok && native != nil {
			// Script-scoped native functions are capabilities of the cloned
			// environment, not retained callbacks into the parent Script. Rebind
			// their execution lease and Invocation provenance to the fork child.
			child.functions[template.name] = &scriptNativeCallable{
				owner:      child,
				generation: child.generation,
				name:       native.name,
				function:   native.function,
			}
		} else {
			child.functions[template.name] = template.callable
		}
		child.functionOrder = append(child.functionOrder, template.name)
	}
	// A fork runs the target's bytecode in the child instance; it does not copy
	// the target closure's captured variables, this-scope, or saved fibers.
	return child.newClosure(target.function, child.globals)
}

func (r *Runtime) removeForkScript(child *Script) {
	if r == nil || child == nil {
		return
	}
	child.mu.Lock()
	child.active = false
	child.mu.Unlock()
	r.mu.Lock()
	delete(r.scripts, child.id)
	r.mu.Unlock()
}

func (task *forkTask) complete(result Value, err error) {
	if task == nil {
		return
	}
	task.once.Do(func() {
		if task.releaseContext != nil {
			task.releaseContext()
		}
		task.result = result
		task.err = err
		// PipedInputStream notices a dead writer thread. Go's io.Pipe does not,
		// so explicitly close the child's endpoints before publishing done.
		_ = task.childWriter.Close()
		_ = task.childReader.Close()
		close(task.done)
	})
}

func (task *forkTask) cancelAndClose() {
	if task == nil {
		return
	}
	if task.cancel != nil {
		task.cancel(context.Canceled)
	}
	if task.releaseContext != nil {
		task.releaseContext()
	}
	cancelled := context.Canceled
	_ = task.parentReader.CloseWithError(cancelled)
	_ = task.parentWriter.CloseWithError(cancelled)
	_ = task.childReader.CloseWithError(cancelled)
	_ = task.childWriter.CloseWithError(cancelled)
}

func (task *forkTask) join(ctx context.Context) error {
	if task == nil || task.done == nil {
		return nil
	}
	if ctx == nil {
		<-task.done
		return nil
	}
	select {
	case <-task.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
