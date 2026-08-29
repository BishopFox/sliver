package opfor

import (
	"context"
	"errors"
)

// scriptGeneration is the opaque identity of one importer-capability epoch
// for a Script. A Script and its raw Sleep closures intentionally survive a
// portable ScriptLoader unload; importer-facing capabilities do not. Pointer
// identity therefore matters more than the monotonically increasing sequence.
//
// All mutable fields are protected by script.mu. The generation context is
// canceled as soon as retirement closes admission, while drained is closed
// only after every already-admitted generation execution has returned.
type scriptGeneration struct {
	script   *Script
	sequence uint64

	context context.Context
	cancel  context.CancelCauseFunc

	retiring bool
	leases   uint64
	drained  chan struct{}

	// cleanupDone is distinct from drained: drained closes when admitted
	// generation calls have returned, while cleanupDone closes only after the
	// retirement owner has run unload observers, revoked registrations, and
	// either opened the next generation or yielded to terminal Script unload.
	// Both fields are protected by script.mu; cleanupErr is immutable after
	// cleanupDone closes.
	cleanupDone chan struct{}
	cleanupErr  error

	// reservation is one ordinary Script execution count held from retirement
	// admission through generation cleanup. It prevents terminal Script unload
	// from starting finishUnload over the same callbacks and snapshots.
	reservation bool
	completed   bool
}

func initializeScriptGeneration(script *Script) {
	if script == nil {
		return
	}
	script.generation = newScriptGenerationLocked(script)
}

func newScriptGenerationLocked(script *Script) *scriptGeneration {
	script.nextGeneration++
	ctx, cancel := context.WithCancelCause(context.Background())
	return &scriptGeneration{
		script:   script,
		sequence: script.nextGeneration,
		context:  ctx,
		cancel:   cancel,
	}
}

// currentScriptGeneration returns the current capability generation. The
// returned token is immutable in identity and may be retained for comparison;
// callers must still use acquireGenerationExecution before exercising it.
func (script *Script) currentScriptGeneration() *scriptGeneration {
	if script == nil {
		return nil
	}
	script.mu.RLock()
	generation := script.generation
	script.mu.RUnlock()
	return generation
}

// generationAdmissibleLocked reports whether generation is the exact current
// capability epoch and still admits new importer-facing work. The caller must
// hold script.mu for either reading or writing.
func (script *Script) generationAdmissibleLocked(generation *scriptGeneration) bool {
	return script != nil &&
		generation != nil &&
		generation.script == script &&
		script.active &&
		script.generation == generation &&
		!generation.retiring &&
		!generation.completed
}

func (script *Script) generationAdmissible(generation *scriptGeneration) bool {
	if script == nil {
		return false
	}
	script.mu.RLock()
	admissible := script.generationAdmissibleLocked(generation)
	script.mu.RUnlock()
	return admissible
}

// acquireGenerationExecution atomically admits an importer-facing operation
// for the exact expected generation. In addition to a generation lease it
// holds an ordinary Script execution lease, so terminal unload cannot overlap
// the callback. The returned context is canceled by either terminal unload or
// logical generation retirement.
func (script *Script) acquireGenerationExecution(
	ctx context.Context,
	expected *scriptGeneration,
) (context.Context, func() error, error) {
	if script == nil || expected == nil || expected.script != script {
		return ctx, nil, ErrScriptUnloaded
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if canReuseClosureExecution(ctx, script) {
		return script.acquireNestedGenerationExecution(ctx, expected)
	}
	if err := executionContextError(ctx); err != nil {
		return ctx, nil, err
	}
	parent, _ := ctx.Value(scriptExecutionContextKey{}).(*scriptExecutionToken)

	script.mu.Lock()
	if !script.generationAdmissibleLocked(expected) {
		script.mu.Unlock()
		return ctx, nil, ErrScriptUnloaded
	}
	if script.runtime != nil {
		if err := script.runtime.outputLimitError(); err != nil {
			script.mu.Unlock()
			return ctx, nil, err
		}
	}
	script.executions++
	expected.leases++
	scriptContext := script.executionCtx
	generationContext := expected.context
	script.mu.Unlock()
	caller, releaseCaller := captureExecutionCallerLease(ctx)

	executionCtx, cancel := context.WithCancelCause(ctx)
	stopScriptCancel := context.AfterFunc(scriptContext, func() {
		cancel(errExecutionLeaseCancellation)
	})
	stopGenerationCancel := context.AfterFunc(generationContext, func() {
		cancel(ErrScriptUnloaded)
	})
	token := &scriptExecutionToken{
		script:          script,
		caller:          caller,
		parent:          parent,
		generation:      expected,
		generationLease: true,
	}
	token.active.Store(true)
	executionCtx = context.WithValue(executionCtx, scriptExecutionContextKey{}, token)
	release := func() error {
		if !token.active.Swap(false) {
			return nil
		}
		stopGenerationCancel()
		stopScriptCancel()
		cancel(errExecutionLeaseCancellation)
		releaseCaller()
		return script.releaseExecution(token)
	}
	return executionCtx, release, nil
}

// acquireNestedGenerationExecution admits an importer-facing callback made
// synchronously by an already-running fiber. The outer script lease already
// owns terminal-unload cancellation and caller-capture lifetime, so this path
// needs only the distinct generation cancellation and counters. Public,
// asynchronous, retained, and cross-script calls use the full path above.
func (script *Script) acquireNestedGenerationExecution(
	ctx context.Context,
	expected *scriptGeneration,
) (context.Context, func() error, error) {
	if err := executionContextError(ctx); err != nil {
		return ctx, nil, err
	}
	parent, _ := ctx.Value(scriptExecutionContextKey{}).(*scriptExecutionToken)
	var caller context.Context
	for token := parent; token != nil; token = token.parent {
		if token.active.Load() && token.script == script {
			caller = token.caller
			break
		}
	}

	script.mu.Lock()
	if !script.generationAdmissibleLocked(expected) {
		script.mu.Unlock()
		return ctx, nil, ErrScriptUnloaded
	}
	if script.runtime != nil && script.runtime.outputLimitEnabled() {
		if err := script.runtime.outputLimitError(); err != nil {
			script.mu.Unlock()
			return ctx, nil, err
		}
	}
	script.executions++
	expected.leases++
	generationContext := expected.context
	script.mu.Unlock()

	executionCtx, cancel := context.WithCancelCause(ctx)
	stopGenerationCancel := context.AfterFunc(generationContext, func() {
		cancel(ErrScriptUnloaded)
	})
	token := &scriptExecutionToken{
		script:          script,
		caller:          caller,
		parent:          parent,
		generation:      expected,
		generationLease: true,
	}
	token.active.Store(true)
	executionCtx = context.WithValue(executionCtx, scriptExecutionContextKey{}, token)
	release := func() error {
		if !token.active.Swap(false) {
			return nil
		}
		stopGenerationCancel()
		cancel(errExecutionLeaseCancellation)
		return script.releaseExecution(token)
	}
	return executionCtx, release, nil
}

// retireCurrentScriptGeneration closes admission for the current capability
// generation and reserves terminal Script finalization until the caller has
// completed generation cleanup. Exactly one caller receives started=true;
// concurrent callers join the same token and drained channel.
//
// This primitive deliberately does not open the next generation. The owner
// must wait for drained, perform its cleanup, and then call
// completeScriptGenerationRetirement.
func (script *Script) retireCurrentScriptGeneration() (
	generation *scriptGeneration,
	drained <-chan struct{},
	started bool,
	err error,
) {
	if script == nil {
		return nil, nil, false, ErrScriptUnloaded
	}

	var cancel context.CancelCauseFunc
	script.mu.Lock()
	generation = script.generation
	if !script.active || script.unloadRequested || generation == nil || generation.completed {
		script.mu.Unlock()
		return nil, nil, false, ErrScriptUnloaded
	}
	if generation.retiring {
		drained = generation.drained
		script.mu.Unlock()
		return generation, drained, false, nil
	}

	generation.retiring = true
	generation.drained = make(chan struct{})
	generation.cleanupDone = make(chan struct{})
	generation.reservation = true
	// Reserve terminal finalization without counting as a generation lease: it
	// must not prevent the generation's own drain channel from closing.
	script.executions++
	if generation.leases == 0 {
		close(generation.drained)
	}
	drained = generation.drained
	cancel = generation.cancel
	script.mu.Unlock()

	if cancel != nil {
		cancel(ErrScriptUnloaded)
	}
	return generation, drained, true, nil
}

// completeScriptGenerationRetirement releases the terminal-finalizer
// reservation and, when the Script is still terminally active, opens the next
// capability generation on the same Script. Cleanup must already be complete
// and the retired generation must be drained.
func (script *Script) completeScriptGenerationRetirement(
	retired *scriptGeneration,
) (*scriptGeneration, error) {
	if script == nil || retired == nil || retired.script != script {
		return nil, ErrScriptUnloaded
	}

	var next *scriptGeneration
	var finalizeTerminal bool
	var terminalCtx context.Context
	script.mu.Lock()
	if retired.completed {
		next = script.generation
		script.mu.Unlock()
		return next, nil
	}
	if script.generation != retired || !retired.retiring || retired.leases != 0 {
		script.mu.Unlock()
		return nil, errors.New("opfor: script generation retirement is not drained")
	}

	retired.completed = true
	if script.active && !script.unloadRequested {
		next = newScriptGenerationLocked(script)
		script.generation = next
	}
	if retired.reservation {
		retired.reservation = false
		if script.executions > 0 {
			script.executions--
		}
	}
	if script.executions == 0 && script.unloadRequested && !script.unloadFinalizing {
		script.unloadFinalizing = true
		finalizeTerminal = true
		terminalCtx = script.unloadContext
	}
	if retired.cleanupDone != nil && !channelClosed(retired.cleanupDone) {
		close(retired.cleanupDone)
	}
	script.mu.Unlock()

	if finalizeTerminal && script.runtime != nil {
		go script.runtime.finishUnload(terminalCtx, script)
	}
	return next, nil
}

// contextOwnsScriptGeneration reports whether ctx carries an admitted
// generation execution for generation. Logical unload uses this to avoid
// waiting synchronously on the provider callback which initiated retirement.
func contextOwnsScriptGeneration(ctx context.Context, generation *scriptGeneration) bool {
	if ctx == nil || generation == nil {
		return false
	}
	for token, _ := ctx.Value(scriptExecutionContextKey{}).(*scriptExecutionToken); token != nil; token = token.parent {
		if token.active.Load() && token.generationLease && token.generation == generation {
			return true
		}
	}
	return false
}

func scriptGenerationFromContext(ctx context.Context, script *Script) *scriptGeneration {
	if ctx == nil || script == nil {
		return nil
	}
	for token, _ := ctx.Value(scriptExecutionContextKey{}).(*scriptExecutionToken); token != nil; token = token.parent {
		if token.active.Load() && token.script == script && token.generation != nil {
			return token.generation
		}
	}
	return nil
}

func scriptGenerationForInvocation(ctx context.Context, runtime *Runtime, scriptID ScriptID) *scriptGeneration {
	if ctx != nil {
		for token, _ := ctx.Value(scriptExecutionContextKey{}).(*scriptExecutionToken); token != nil; token = token.parent {
			if !token.active.Load() || token.script == nil || token.script.runtime != runtime || token.generation == nil {
				continue
			}
			if scriptID == 0 || token.script.id == scriptID {
				return token.generation
			}
		}
	}
	if runtime == nil || scriptID == 0 {
		return nil
	}
	runtime.mu.RLock()
	script := runtime.scripts[scriptID]
	runtime.mu.RUnlock()
	if script == nil {
		return nil
	}
	return script.currentScriptGeneration()
}

func stampInvocationGeneration(ctx context.Context, runtime *Runtime, invocation *Invocation) *scriptGeneration {
	if invocation == nil {
		return nil
	}
	generation := invocation.generation
	if generation == nil {
		generation = scriptGenerationForInvocation(ctx, runtime, invocation.Script)
		invocation.generation = generation
	}
	if generation != nil && invocation.Script == 0 && generation.script != nil {
		invocation.Script = generation.script.id
	}
	return generation
}

func stampObjectInvocationGeneration(
	ctx context.Context,
	runtime *Runtime,
	invocation *ObjectInvocation,
) *scriptGeneration {
	if invocation == nil {
		return nil
	}
	var generation *scriptGeneration
	for _, argument := range invocation.Arguments {
		if argument.generation != nil {
			generation = argument.generation
			break
		}
	}
	if generation == nil {
		generation = scriptGenerationForInvocation(ctx, runtime, invocation.Script)
	}
	if generation == nil {
		return nil
	}
	if invocation.Runtime == nil {
		invocation.Runtime = runtime
	}
	if invocation.Script == 0 && generation.script != nil {
		invocation.Script = generation.script.id
	}
	if len(invocation.Arguments) != 0 {
		invocation.Arguments = append([]Argument(nil), invocation.Arguments...)
		for index := range invocation.Arguments {
			if invocation.Arguments[index].generation == nil {
				invocation.Arguments[index].generation = generation
			}
		}
	}
	return generation
}
