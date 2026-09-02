package opfor

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/sliverarmory/opfor/internal/bytecode"
	"github.com/sliverarmory/opfor/internal/envspec"
)

// Script is one loaded program and its persistent globals and registrations.
// A Script remains usable until Unload is called.
type Script struct {
	runtime *Runtime
	id      ScriptID
	program *Program
	globals *scope

	mu                   sync.RWMutex
	active               bool
	generation           *scriptGeneration
	nextGeneration       uint64
	executionCtx         context.Context
	executionCancel      context.CancelFunc
	executions           uint64
	unloadRequested      bool
	unloadFinalizing     bool
	unloadDone           chan struct{}
	unloadContext        context.Context
	unloadContextRelease func()
	unloadErr            error
	unloadErrDelivered   bool
	unloadWaiters        uint64
	unloadRecipient      *scriptExecutionToken
	unloadRecipientState unloadRecipientState
	unloadRecipientDone  chan struct{}
	result               Value
	debug                int32
	lastError            Value
	nextClosure          uint64
	nextBinding          uint64
	bindings             []Binding
	functions            map[string]Callable
	removedFuncs         map[string]struct{}
	functionOrder        []string
	imports              map[string]string
	importPackages       []string
	stackTrace           []string
	profiler             *scriptProfiler
	forkTasks            map[*forkTask]struct{}
	socketTasks          map[*sleepSocketTask]struct{}
	readTasks            map[*sleepReadTask]struct{}
	processes            map[*processObject]struct{}
	scriptLoaders        map[*portableScriptLoader]struct{}
	loadables            map[string]*scriptLoadableResolution
	loadableUses         []scriptLoadableUse
	aggressorUIResources map[aggressorUIResource]struct{}
	sharedEnvironment    *portableScriptSharedEnvironment
	forkParent           *Script
	forkTask             *forkTask
}

type unloadRecipientState uint8

const (
	unloadRecipientNone unloadRecipientState = iota
	unloadRecipientReserved
	unloadRecipientWaiting
	unloadRecipientDelivered
	unloadRecipientAbandoned
)

// ID returns the runtime-local identity of the loaded script.
func (s *Script) ID() ScriptID {
	if s == nil {
		return 0
	}
	return s.id
}

// Program returns the immutable program used to create the script.
func (s *Script) Program() *Program {
	if s == nil {
		return nil
	}
	return s.program
}

// Active reports whether callbacks owned by this script may still run.
func (s *Script) Active() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.active
}

// Result returns the value produced by the script's top-level body.
func (s *Script) Result() Value {
	if s == nil {
		return Null()
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.result
}

// Get returns a global variable. Names without a sigil are scalar names.
// Provider failures cannot be represented by this legacy convenience method;
// use GetContext when a VariableProvider is installed and the error matters.
func (s *Script) Get(name string) Value {
	value, _ := s.GetContext(context.Background(), name)
	return value
}

// GetContext returns a global variable while preserving variable-provider
// errors and cancellation. A missing scalar reads as $null; missing @/% names
// are installed with their ordinary empty container value.
func (s *Script) GetContext(ctx context.Context, name string) (Value, error) {
	if s == nil || s.globals == nil {
		return Null(), nil
	}
	return s.globals.getAt(ctx, name, Span{})
}

// Set replaces a global variable. Names without a sigil are scalar names.
func (s *Script) Set(name string, value Value) error {
	return s.SetContext(context.Background(), name, value)
}

// SetContext replaces a global variable and preserves VariableProvider errors.
// If the name already exists, its provider-owned *Cell identity is retained.
func (s *Script) SetContext(ctx context.Context, name string, value Value) (resultErr error) {
	if s == nil {
		return ErrScriptUnloaded
	}
	// Preserve the legacy built-in scope's single active-check/write critical
	// section. Besides avoiding unnecessary lease machinery, this guarantees
	// that an Unload admitted after Set owns Script.mu cannot turn a committed
	// write into ErrScriptUnloaded. Provider lookups stay on the lease-backed
	// path below because importer code may block or reenter OPFOR and therefore
	// must never run while Script.mu is held.
	if s.globals != nil && s.globals.root != nil && s.globals.root.container == nil {
		if ctx == nil {
			ctx = context.Background()
		}
		if err := executionContextError(ctx); err != nil {
			return err
		}
		s.mu.Lock()
		defer s.mu.Unlock()
		if !s.active {
			return ErrScriptUnloaded
		}
		if s.runtime != nil {
			if err := s.runtime.outputLimitError(); err != nil {
				return err
			}
		}
		cell, err := s.globals.globalAt(ctx, name, Span{})
		if err != nil {
			return err
		}
		cell.Set(value)
		return nil
	}
	executionCtx, release, err := s.acquireExecution(ctx)
	if err != nil {
		return err
	}
	defer func() { resultErr = joinExecutionError(resultErr, release) }()
	cell, err := s.globals.globalAt(executionCtx, name, Span{})
	if err != nil {
		return err
	}
	// Preserve Set's established active-check/write atomicity against unload.
	// The potentially reentrant provider lookup above deliberately occurs
	// outside Script.mu; only the final Cell mutation is serialized here.
	s.mu.Lock()
	if !s.active {
		s.mu.Unlock()
		return ErrScriptUnloaded
	}
	if s.runtime != nil {
		if err := s.runtime.outputLimitError(); err != nil {
			s.mu.Unlock()
			return err
		}
	}
	cell.Set(value)
	s.mu.Unlock()
	// Once the write commits while active, a concurrently admitted Unload may
	// cancel executionCtx before this goroutine is scheduled again. Report the
	// committed mutation as successful; returning ErrScriptUnloaded here would
	// claim that a write which is already visible did not occur.
	return nil
}

// BindVariable installs cell as the exact global slot for name. This is the
// public pass-by-name/bootstrap counterpart to VariableContainer.PutScalar.
func (s *Script) BindVariable(ctx context.Context, name string, cell *Cell) (resultErr error) {
	if s == nil {
		return ErrScriptUnloaded
	}
	if cell == nil {
		return errors.New("opfor: variable cell is nil")
	}
	executionCtx, release, err := s.acquireExecution(ctx)
	if err != nil {
		return err
	}
	defer func() {
		resultErr = joinExecutionContextError(executionCtx, resultErr)
		resultErr = joinExecutionError(resultErr, release)
	}()
	return s.globals.root.putCellAt(executionCtx, name, cell, Span{})
}

// UnsetVariable removes name from the Script's global container. It does not
// search or mutate a currently executing closure/local level.
func (s *Script) UnsetVariable(ctx context.Context, name string) (resultErr error) {
	if s == nil {
		return ErrScriptUnloaded
	}
	executionCtx, release, err := s.acquireExecution(ctx)
	if err != nil {
		return err
	}
	defer func() {
		resultErr = joinExecutionContextError(executionCtx, resultErr)
		resultErr = joinExecutionError(resultErr, release)
	}()
	return s.globals.root.removeOwnAt(executionCtx, name, Span{})
}

// Globals snapshots the script's root variable table.
// Provider failures cannot be represented by this legacy convenience method;
// use GlobalsContext when a VariableProvider is installed and the error matters.
func (s *Script) Globals() map[string]Value {
	values, _ := s.GlobalsContext(context.Background())
	return values
}

// GlobalsContext snapshots globals known to OPFOR and preserves provider
// errors. VariableContainer has no enumeration operation, matching upstream
// Sleep, so provider-only names which OPFOR has never accessed are omitted.
func (s *Script) GlobalsContext(ctx context.Context) (map[string]Value, error) {
	if s == nil {
		return nil, nil
	}
	return s.globals.snapshotRootAt(ctx)
}

func (s *Script) setGlobalAt(ctx context.Context, name string, value Value, span Span) error {
	if s == nil || s.globals == nil {
		return ErrScriptUnloaded
	}
	cell, err := s.globals.globalAt(ctx, name, span)
	if err != nil {
		return err
	}
	cell.SetAt(value, span)
	return nil
}

// Call invokes a sub or inline declaration owned by this script.
func (s *Script) Call(ctx context.Context, name string, arguments ...Value) (Value, error) {
	executionCtx, release, err := s.acquireExecution(ctx)
	if err != nil {
		return Null(), err
	}
	defer release()
	// Script.Call is a public evaluator entry. Independent calls receive a new
	// instruction budget, while reentrant calls preserve the meter carried by
	// their active evaluator context.
	executionCtx = withExecutionMeter(executionCtx, s.runtime)
	name = strings.TrimPrefix(strings.TrimSpace(name), "&")
	s.mu.RLock()
	closure := s.functions[name]
	s.mu.RUnlock()
	if closure == nil {
		return Null(), joinExecutionError(&UnsupportedError{Operation: "script function", Name: name}, release)
	}
	var value Value
	if scriptClosure, ok := closure.(*scriptClosure); ok {
		value, err = scriptClosure.invoke(executionCtx, []Argument{{Name: "$0", Value: String("&" + name)}}, arguments...)
	} else {
		value, err = closure.Invoke(executionCtx, arguments...)
	}
	err = joinExecutionContextError(executionCtx, err)
	return value, joinExecutionError(err, release)
}

// Bindings returns a stable snapshot of registrations owned by the script.
func (s *Script) Bindings() []Binding {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Binding, len(s.bindings))
	for index, binding := range s.bindings {
		result[index] = cloneBinding(binding)
	}
	return result
}

// Unload removes all registrations and invalidates retained callbacks. If an
// asynchronous read is already blocked in an uncancelable borrowed reader,
// Unload returns ErrReadCancellationUnsupported without falsely reporting the
// read worker as complete; see WithStdin.
func (s *Script) Unload(ctx context.Context) error {
	if s == nil {
		return nil
	}
	return s.runtime.unload(ctx, s)
}

// Load executes a Program and retains its globals, functions, events, aliases,
// commands, and hooks until the returned Script is unloaded. Arguments populate
// Sleep's @ARGV launcher array; they are not top-level subroutine arguments, so
// @_ is empty and $1 through $n are null in the program's top-level body.
func (r *Runtime) Load(ctx context.Context, program *Program, arguments ...Value) (*Script, error) {
	if r == nil {
		return nil, errors.New("opfor: runtime is nil")
	}
	if program == nil || program.function == nil {
		return nil, errors.New("opfor: program is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	// Reject terminal runtimes before charging standalone or foreign Program
	// admission. The later locked checks remain necessary for a concurrent Close,
	// but an already closed Runtime must preserve ErrRuntimeClosed and its quota
	// counters exactly.
	r.mu.RLock()
	closed := r.closing || r.closed
	r.mu.RUnlock()
	if closed {
		return nil, ErrRuntimeClosed
	}
	if err := r.outputLimitError(); err != nil {
		return nil, err
	}
	// Runtime-aware compilation has already charged this immutable Program to
	// the shared family account. Standalone or foreign-runtime Programs are
	// admitted on every load so precompilation cannot bypass the source budget;
	// do not mutate Program here because callers may load it concurrently.
	if program.sourceAccount != r.resources {
		if err := r.reserveResource(resourceSourceBytes, uint64(len(program.source.Data))); err != nil {
			return nil, err
		}
	}
	ctx = withExecutionMeter(ctx, r)

	script := &Script{
		runtime:           r,
		program:           program,
		active:            true,
		debug:             r.debugFlags,
		functions:         make(map[string]Callable),
		removedFuncs:      make(map[string]struct{}),
		imports:           make(map[string]string),
		sharedEnvironment: r.scriptLoaderSharedEnvironment,
	}
	r.mu.Lock()
	if r.closing || r.closed {
		r.mu.Unlock()
		return nil, ErrRuntimeClosed
	}
	r.nextScript++
	script.id = r.nextScript
	r.mu.Unlock()
	globals, err := r.createGlobalScope(ctx, script.id)
	if err != nil {
		return nil, err
	}
	script.globals = globals
	initializeScriptExecution(script)
	executionCtx, release, err := script.acquireExecution(ctx)
	if err != nil {
		return nil, err
	}
	defer release()

	// The script owns an execution lease before publication. Close therefore
	// either closes admission first, or snapshots a fully admitted script whose
	// load observer and top-level body it must wait for.
	r.mu.Lock()
	if r.closing || r.closed {
		r.mu.Unlock()
		script.mu.Lock()
		script.active = false
		script.mu.Unlock()
		script.executionCancel()
		return nil, errors.Join(ErrRuntimeClosed, release())
	}
	if r.scripts == nil {
		r.scripts = make(map[ScriptID]*Script)
	}
	r.scripts[script.id] = script
	r.mu.Unlock()
	bindPortableScriptInstanceRunScript(executionCtx, r.scriptLoaderInstance, script)

	if err := r.installInitialGlobals(executionCtx, script); err != nil {
		executionErr := release()
		cleanupErr := r.unload(context.Background(), script)
		return nil, errors.Join(err, executionErr, cleanupErr)
	}
	if err := script.setGlobalAt(executionCtx, "$__SCRIPT__", String(program.source.Name), Span{}); err != nil {
		executionErr := release()
		cleanupErr := r.unload(context.Background(), script)
		return nil, errors.Join(err, executionErr, cleanupErr)
	}
	if err := script.setGlobalAt(executionCtx, "$__SCRIPT_NAME__", String(program.source.Name), Span{}); err != nil {
		executionErr := release()
		cleanupErr := r.unload(context.Background(), script)
		return nil, errors.Join(err, executionErr, cleanupErr)
	}
	if r.taintMode {
		tainted := make([]Value, len(arguments))
		for index, argument := range arguments {
			tainted[index] = r.Taint(argument)
		}
		arguments = tainted
	}
	argumentArray, err := newRuntimeArray(r, arguments...)
	if err != nil {
		executionErr := release()
		cleanupErr := r.unload(context.Background(), script)
		return nil, errors.Join(err, executionErr, cleanupErr)
	}
	if err := script.setGlobalAt(executionCtx, "@ARGV", ArrayValue(argumentArray), Span{}); err != nil {
		executionErr := release()
		cleanupErr := r.unload(context.Background(), script)
		return nil, errors.Join(err, executionErr, cleanupErr)
	}
	if err := r.notifyScriptLoaded(executionCtx, script); err != nil {
		executionErr := release()
		cleanupErr := r.unload(context.Background(), script)
		return nil, errors.Join(err, executionErr, cleanupErr)
	}

	main := &scriptClosure{script: script, function: program.function, captured: script.globals}
	// Sleep exposes launcher arguments through @ARGV. They are not subroutine
	// arguments to the script's top-level block, so @_ remains empty and $1...
	// remain null until an actual function or binding is invoked.
	result, err := main.invokeFresh(executionCtx, nil)
	if err != nil {
		executionErr := release()
		cleanupErr := r.unload(context.Background(), script)
		return nil, errors.Join(err, executionErr, cleanupErr)
	}
	script.mu.Lock()
	script.result = result
	script.mu.Unlock()
	executionErr := release()
	if !script.Active() {
		executionErr = errors.Join(ErrScriptUnloaded, executionErr)
	}
	if executionErr != nil {
		return nil, executionErr
	}
	return script, nil
}

// Execute runs a Program as a one-shot script and unloads all registrations
// before returning. Arguments use the same @ARGV contract as Load. Use Load
// when callbacks must remain active.
func (r *Runtime) Execute(ctx context.Context, program *Program, arguments ...Value) (Value, error) {
	script, err := r.Load(ctx, program, arguments...)
	if err != nil {
		return Null(), err
	}
	value := script.Result()
	if unloadErr := script.Unload(ctx); unloadErr != nil {
		return value, unloadErr
	}
	return value, nil
}

// Eval compiles and executes one named source string in a runtime-owned
// persistent session. Globals and declarations remain available to later Eval
// calls, which is useful for REPLs. Each call replaces @ARGV with arguments;
// the top-level @_ and positional variables remain empty. Close unloads the
// session.
func (r *Runtime) Eval(ctx context.Context, name, code string, arguments ...Value) (Value, error) {
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	r.mu.RLock()
	closed := r.closing || r.closed
	r.mu.RUnlock()
	if closed {
		return Null(), ErrRuntimeClosed
	}
	if err := r.outputLimitError(); err != nil {
		return Null(), err
	}
	program, err := r.CompileString(name, code)
	if err != nil {
		return Null(), err
	}
	ctx = withExecutionMeter(ctx, r)
	r.evalMu.Lock()
	defer r.evalMu.Unlock()

	r.mu.RLock()
	script := r.evalScript
	closed = r.closing || r.closed
	r.mu.RUnlock()
	if closed {
		return Null(), ErrRuntimeClosed
	}
	created := false
	var release func() error
	if script == nil || !script.Active() {
		script = &Script{
			runtime: r, program: program,
			active: true, debug: r.debugFlags,
			functions: make(map[string]Callable), removedFuncs: make(map[string]struct{}), imports: make(map[string]string),
			sharedEnvironment: r.scriptLoaderSharedEnvironment,
		}
		r.mu.Lock()
		if r.closing || r.closed {
			r.mu.Unlock()
			return Null(), ErrRuntimeClosed
		}
		r.nextScript++
		script.id = r.nextScript
		r.mu.Unlock()
		globals, scopeErr := r.createGlobalScope(ctx, script.id)
		if scopeErr != nil {
			return Null(), scopeErr
		}
		script.globals = globals
		initializeScriptExecution(script)
		var acquireErr error
		ctx, release, acquireErr = script.acquireExecution(ctx)
		if acquireErr != nil {
			return Null(), acquireErr
		}
		defer release()
		r.mu.Lock()
		if r.closing || r.closed {
			r.mu.Unlock()
			script.mu.Lock()
			script.active = false
			script.mu.Unlock()
			script.executionCancel()
			return Null(), errors.Join(ErrRuntimeClosed, release())
		}
		r.scripts[script.id] = script
		r.evalScript = script
		r.mu.Unlock()
		bindPortableScriptInstanceRunScript(ctx, r.scriptLoaderInstance, script)
		if installErr := r.installInitialGlobals(ctx, script); installErr != nil {
			executionErr := release()
			cleanupErr := r.unload(context.Background(), script)
			return Null(), errors.Join(installErr, executionErr, cleanupErr)
		}
		created = true
	} else {
		// Existing sessions acquire after lookup. A concurrent Close may have
		// retired the session since the registry snapshot; admission rejects it
		// before any Eval-owned global is changed.
		ctx, release, err = script.acquireExecution(ctx)
		if err != nil {
			return Null(), err
		}
		defer release()
	}
	script.mu.Lock()
	script.program = program
	script.mu.Unlock()
	if err := script.setGlobalAt(ctx, "$__SCRIPT__", String(name), Span{}); err != nil {
		executionErr := release()
		if created {
			return Null(), errors.Join(err, executionErr, r.unload(context.Background(), script))
		}
		return Null(), errors.Join(err, executionErr)
	}
	if err := script.setGlobalAt(ctx, "$__SCRIPT_NAME__", String(name), Span{}); err != nil {
		executionErr := release()
		if created {
			return Null(), errors.Join(err, executionErr, r.unload(context.Background(), script))
		}
		return Null(), errors.Join(err, executionErr)
	}
	if r.taintMode {
		tainted := make([]Value, len(arguments))
		for index, argument := range arguments {
			tainted[index] = r.Taint(argument)
		}
		arguments = tainted
	}
	argumentArray, err := newRuntimeArray(r, arguments...)
	if err != nil {
		executionErr := release()
		if created {
			return Null(), errors.Join(err, executionErr, r.unload(context.Background(), script))
		}
		return Null(), errors.Join(err, executionErr)
	}
	if err := script.setGlobalAt(ctx, "@ARGV", ArrayValue(argumentArray), Span{}); err != nil {
		executionErr := release()
		if created {
			return Null(), errors.Join(err, executionErr, r.unload(context.Background(), script))
		}
		return Null(), errors.Join(err, executionErr)
	}
	if created {
		if err := r.notifyScriptLoaded(ctx, script); err != nil {
			executionErr := release()
			cleanupErr := r.unload(context.Background(), script)
			return Null(), errors.Join(err, executionErr, cleanupErr)
		}
	}
	closure := &scriptClosure{script: script, function: program.function, captured: script.globals}
	value, err := closure.invokeFresh(ctx, nil)
	if err != nil {
		executionErr := release()
		// Eval ordinarily retains its persistent session after a failed snippet.
		// A newly observed session is the exception: pair ScriptLoaded with a
		// rollback notification because it never completed its first top-level run.
		if created && r.lifecycle != nil {
			cleanupErr := r.unload(context.Background(), script)
			return Null(), errors.Join(err, executionErr, cleanupErr)
		}
		return Null(), errors.Join(err, executionErr)
	}
	script.mu.Lock()
	script.result = value
	script.mu.Unlock()
	executionErr := release()
	if !script.Active() {
		executionErr = errors.Join(ErrScriptUnloaded, executionErr)
	}
	return value, executionErr
}

// Close permanently closes script admission, cancels every active execution,
// and waits for all cancellable scripts and runtime-owned tasks to become
// quiescent. An asynchronous read already blocked in an uncancelable borrowed
// reader instead contributes ErrReadCancellationUnsupported; its callback is
// revoked, but its actual worker remains incomplete until Read returns. See
// WithStdin.
// It is safe to call more than once. When called reentrantly by Host or an
// observer running on this Runtime, Close requests teardown and returns nil;
// the enclosing execution observes cancellation and performs the final release.
func (r *Runtime) Close(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// Any nested execution or lifecycle cleanup is conservatively non-waiting,
	// including one owned by another Runtime. Two independent runtimes may call
	// Close on each other from concurrent Hosts/observers; waiting in both calls
	// would keep the executions or finalizers required by each Close alive.
	reentrant := hasActiveExecutionToken(ctx) || hasActiveLifecycleToken(ctx)
	cleanupCtx := ctx
	releaseCleanupContext := func() {}
	if hasActiveExecutionToken(ctx) {
		cleanupCtx, releaseCleanupContext = detachExecutionLeaseCancellationLease(ctx)
	}
	cleanupContextTransferred := false
	defer func() {
		if !cleanupContextTransferred {
			releaseCleanupContext()
		}
	}()
	executionDeferred, lifecycleDeferred := runtimeScriptActivity(ctx, r)
	var recipientScript *Script
	var deferredRecipient *scriptExecutionToken
	if len(executionDeferred) == 1 && len(lifecycleDeferred) == 0 {
		for script, token := range executionDeferred {
			if token != nil {
				recipientScript = script
				deferredRecipient = token
			}
		}
	}
	if runtimeContext, releaseRuntimeContext, detached := detachRuntimeCancellationLease(cleanupCtx, r); detached {
		// Retain the runtime-specific source before dropping the general
		// selection; nested runtimes can otherwise make these different bridges.
		releaseCleanupContext()
		cleanupCtx = runtimeContext
		releaseCleanupContext = releaseRuntimeContext
	}

	r.mu.Lock()
	start := !r.closing
	var executionCancel context.CancelFunc
	if start {
		r.closing = true
		r.closeDone = make(chan struct{})
		executionCancel = r.executionCancel
		if r.executionDone == nil {
			r.executionDone = make(chan struct{})
		}
		if r.executions == 0 && !channelClosed(r.executionDone) {
			close(r.executionDone)
		}
	}
	done := r.closeDone
	executionDone := r.executionDone
	var scripts []*Script
	if start {
		scripts = make([]*Script, 0, len(r.scripts))
		present := make(map[*Script]struct{}, len(r.scripts))
		for _, script := range r.scripts {
			if script.forkParent == nil {
				scripts = append(scripts, script)
				present[script] = struct{}{}
			}
		}
		// An Unregistered or ScriptUnloaded observer runs after finishUnload
		// removes its script from the public registry. If that observer is the
		// first caller of Close, the script is still an in-progress teardown and
		// must remain part of Close's terminal wait even though it was absent
		// from the registry snapshot.
		for script := range executionDeferred {
			if script != nil {
				if _, exists := present[script]; !exists {
					scripts = append(scripts, script)
					present[script] = struct{}{}
				}
			}
		}
		for script := range lifecycleDeferred {
			if script != nil {
				if _, exists := present[script]; !exists {
					scripts = append(scripts, script)
					present[script] = struct{}{}
				}
			}
		}
	}
	r.mu.Unlock()
	if executionCancel != nil {
		executionCancel()
	}

	if start {
		sort.Slice(scripts, func(left, right int) bool { return scripts[left].id < scripts[right].id })
		workerCtx := withoutScriptLifecycleTokens(cleanupCtx)
		workerCtx, releaseCloseContext := withRuntimeCloseContext(workerCtx, r)
		// Cancellation is part of Close's synchronous request boundary. In
		// particular, a Host that calls Close reentrantly must not return to a VM
		// that can commit the rest of its current expression while the worker is
		// merely waiting to be scheduled.
		for index := len(scripts) - 1; index >= 0; index-- {
			var recipient *scriptExecutionToken
			if scripts[index] == recipientScript {
				recipient = deferredRecipient
			}
			r.requestUnload(workerCtx, scripts[index], recipient)
		}
		go func() {
			defer releaseCleanupContext()
			defer releaseCloseContext()
			r.finishClose(workerCtx, scripts, executionDeferred, lifecycleDeferred, executionDone)
		}()
		cleanupContextTransferred = true
	}
	if reentrant {
		return nil
	}
	if done == nil {
		return nil
	}

	select {
	case <-done:
		r.mu.Lock()
		err := r.consumeCloseErrorLocked()
		r.mu.Unlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// requestUnload closes admission and cancels execution without waiting or
// running teardown callbacks. It is used by reentrant Runtime.Close before it
// returns to importer code; the Close worker subsequently drives each request
// through unload and waits for finalization.
func (r *Runtime) requestUnload(ctx context.Context, script *Script, recipient *scriptExecutionToken) {
	if r == nil || script == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var cancel context.CancelFunc
	var uiResources []aggressorUIResource
	script.mu.Lock()
	if !script.unloadRequested {
		script.active = false
		script.unloadRequested = true
		script.unloadContext = ctx
		if recipient != nil && recipient.active.Load() && recipient.script == script {
			script.unloadRecipient = recipient
			script.unloadRecipientState = unloadRecipientReserved
			script.unloadRecipientDone = make(chan struct{})
		}
		cancel = script.executionCancel
		r.mu.Lock()
		for _, binding := range script.bindings {
			r.removeRuntimeBindingLocked(binding)
		}
		r.mu.Unlock()
		if r.aggressorCommands != nil {
			// Command-help registrations become unavailable at the same unload
			// admission boundary as executable bindings. Waiting until finalization
			// would advertise help for commands whose aliases are already revoked
			// while a callback or child runtime drains.
			r.aggressorCommands.removeScript(script.id)
		}
		if r.aggressorBeaconTechniques != nil {
			// Technique callbacks are executable capabilities. Revoke their
			// metadata and lookup path as soon as unload closes admission.
			r.aggressorBeaconTechniques.removeScript(script.id)
		}
		if len(script.aggressorUIResources) != 0 {
			uiResources = make([]aggressorUIResource, 0, len(script.aggressorUIResources))
			for resource := range script.aggressorUIResources {
				uiResources = append(uiResources, resource)
			}
			script.aggressorUIResources = nil
		}
	}
	script.mu.Unlock()
	// Do not hold the Script lock while closing provider-facing responders.
	// Responder completion also removes itself from this registry, and importer
	// code may observe Done and immediately attempt another script operation.
	for _, resource := range uiResources {
		resource.revokeAggressorUI()
	}
	if cancel != nil {
		cancel()
	}
}

func (r *Runtime) finishClose(
	ctx context.Context,
	scripts []*Script,
	executionDeferred map[*Script]*scriptExecutionToken,
	lifecycleDeferred map[*Script]struct{},
	executionDone <-chan struct{},
) {
	var result error
	var socketTasks []*sleepSocketTask
	if r.socketState != nil {
		// Stop accepting runtime-owned socket work before waiting for scripts;
		// an active script may itself be blocked on one of these tasks.
		socketTasks = r.socketState.shutdown()
	}
	r.mu.Lock()
	readTasks := make([]*sleepReadTask, 0, len(r.readTasks))
	for task := range r.readTasks {
		readTasks = append(readTasks, task)
	}
	r.readTasks = nil
	processes := make([]*processObject, 0, len(r.processes))
	for process := range r.processes {
		processes = append(processes, process)
	}
	r.processes = nil
	r.mu.Unlock()
	joinableReadTasks, incompleteReadCancellation := revokeSleepReadTasks(readTasks)
	if incompleteReadCancellation {
		result = errors.Join(result, ErrReadCancellationUnsupported)
	}
	for _, process := range processes {
		if err := process.close(); err != nil {
			result = errors.Join(result, err)
		}
	}
	for _, task := range joinableReadTasks {
		if err := task.join(ctx); err != nil {
			remaining, waitExpired := splitContextWaitError(ctx, err)
			if remaining != nil {
				result = errors.Join(result, remaining)
			}
			if waitExpired {
				if terminalErr := task.join(context.Background()); terminalErr != nil {
					terminalErr, _ = splitContextWaitError(ctx, terminalErr)
					result = errors.Join(result, terminalErr)
				}
			}
		}
	}
	for _, process := range processes {
		if err := process.join(ctx); err != nil {
			remaining, waitExpired := splitContextWaitError(ctx, err)
			if remaining != nil {
				result = errors.Join(result, remaining)
			}
			if waitExpired {
				if terminalErr := process.join(context.Background()); terminalErr != nil {
					terminalErr, _ = splitContextWaitError(ctx, terminalErr)
					result = errors.Join(result, terminalErr)
				}
			}
		}
	}
	for index := len(scripts) - 1; index >= 0; index-- {
		script := scripts[index]
		if _, deferred := lifecycleDeferred[script]; deferred {
			// Do not become an error-consuming waiter for the script whose Host or
			// observer initiated Close. Its enclosing execution/unload entry owns
			// the final release and deterministically receives cleanup failures.
			script.mu.Lock()
			done := script.unloadDone
			script.mu.Unlock()
			if done != nil {
				<-done
			}
			continue
		}
		if _, deferred := executionDeferred[script]; deferred {
			if err := script.waitUnloaded(context.Background()); err != nil {
				err, _ = splitContextWaitError(ctx, err)
				result = errors.Join(result, err)
			}
			continue
		}
		unloadErr := r.unload(ctx, script)
		remaining, waitExpired := splitContextWaitError(ctx, unloadErr)
		if remaining != nil {
			result = errors.Join(result, remaining)
		}
		// A caller deadline may stop unload's wait, but Close itself remains a
		// terminal operation. Keep the worker alive until the admitted execution
		// releases and teardown has actually completed.
		script.mu.Lock()
		complete := channelClosed(script.unloadDone)
		script.mu.Unlock()
		if waitExpired || !complete {
			if err := script.waitUnloaded(context.Background()); err != nil {
				err, _ = splitContextWaitError(ctx, err)
				result = errors.Join(result, err)
			}
		}
	}

	for _, task := range socketTasks {
		if err := task.join(ctx); err != nil {
			remaining, waitExpired := splitContextWaitError(ctx, err)
			if remaining != nil {
				result = errors.Join(result, remaining)
			}
			// Context-aware callers may stop waiting; terminal shutdown still waits
			// in this private worker so no runtime-owned task survives closed=true.
			if waitExpired {
				if terminalErr := task.join(context.Background()); terminalErr != nil {
					terminalErr, _ = splitContextWaitError(ctx, terminalErr)
					result = errors.Join(result, terminalErr)
				}
			}
		}
	}
	if executionDone != nil {
		<-executionDone
	}
	if r.regexCache != nil {
		r.regexCache.clear()
	}

	r.mu.Lock()
	r.closeErr = result
	r.closed = true
	if !channelClosed(r.closeDone) {
		close(r.closeDone)
	}
	r.mu.Unlock()
}

func contextWaitError(ctx context.Context, err error) bool {
	_, waitExpired := splitContextWaitError(ctx, err)
	return waitExpired
}

// splitContextWaitError distinguishes the caller's expired wait from cleanup
// failures delivered at the same boundary. In particular, an observer may
// return errors.Join(ctx.Err(), sentinel): the context branch only says that
// the original caller stopped waiting, while sentinel must survive for the
// terminal Close or Unload waiter. A boolean errors.Is check loses that branch
// because cleanup errors are consumed exactly once.
func splitContextWaitError(ctx context.Context, err error) (error, bool) {
	if ctx == nil || err == nil || ctx.Err() == nil {
		return err, false
	}
	remaining, removed := removeErrorBranch(err, ctx.Err())
	if !removed {
		return err, false
	}
	return remaining, true
}

func removeErrorBranch(err error, target error) (error, bool) {
	if err == nil || target == nil {
		return err, false
	}
	if multi, ok := err.(interface{ Unwrap() []error }); ok {
		children := multi.Unwrap()
		remaining := make([]error, 0, len(children))
		removed := false
		for _, child := range children {
			childRemaining, childRemoved := removeErrorBranch(child, target)
			removed = removed || childRemoved
			if childRemaining != nil {
				remaining = append(remaining, childRemaining)
			}
		}
		if !removed {
			return err, false
		}
		return errors.Join(remaining...), true
	}
	if single, ok := err.(interface{ Unwrap() error }); ok {
		remaining, removed := removeErrorBranch(single.Unwrap(), target)
		if removed {
			// A single-child wrapper contains no independent error branch. If its
			// child was a join, retain the non-context branches without the wrapper
			// whose message may otherwise misleadingly describe cancellation.
			return remaining, true
		}
	}
	if errors.Is(err, target) {
		return nil, true
	}
	return err, false
}

func (r *Runtime) consumeCloseErrorLocked() error {
	if r == nil || r.closeErrDelivered {
		return nil
	}
	r.closeErrDelivered = true
	return r.closeErr
}

// Scripts returns loaded scripts ordered by ScriptID.
func (r *Runtime) Scripts() []*Script {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	result := make([]*Script, 0, len(r.scripts))
	for _, script := range r.scripts {
		// Fork instances are owned by their parent rather than independently
		// loaded through Runtime.Load. They remain in the internal registry so
		// child native calls can resolve ScriptID, while Close reaches them by
		// recursively unloading each visible parent.
		if script.forkParent != nil {
			continue
		}
		result = append(result, script)
	}
	r.mu.RUnlock()
	sort.Slice(result, func(i, j int) bool { return result[i].id < result[j].id })
	return result
}

// Bindings returns active registrations matching kind and name. An empty name
// returns every registration of the requested kind in registration order.
func (r *Runtime) Bindings(kind BindingKind, name string) []Binding {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	byName := r.bindings[kind]
	if name != "" {
		entries := byName[name]
		result := make([]Binding, len(entries))
		for index, binding := range entries {
			result[index] = cloneBinding(binding)
		}
		return result
	}
	entries := r.bindingOrder[kind]
	result := make([]Binding, len(entries))
	for index, binding := range entries {
		result[index] = cloneBinding(binding)
	}
	return result
}

// dispatchBindingSnapshot captures exact and wildcard event registrations
// under one read lock. Besides producing a deterministic dispatch generation,
// this prevents an exact callback from registering a wildcard listener that
// observes the event already in progress.
func (r *Runtime) dispatchBindingSnapshot(kind BindingKind, name string) (exact, wildcard []Binding) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	copyBindings := func(bindings []Binding) []Binding {
		result := make([]Binding, len(bindings))
		for index, binding := range bindings {
			result[index] = cloneBinding(binding)
		}
		return result
	}
	if name == "" {
		exact = copyBindings(r.bindingOrder[kind])
	} else {
		exact = copyBindings(r.bindings[kind][name])
	}
	if kind == BindingEvent && name != "*" {
		wildcard = copyBindings(r.bindings[kind]["*"])
	}
	return exact, wildcard
}

// claimDispatchBindings filters a dispatch snapshot to registrations this
// dispatch may invoke. Persistent bindings always remain selected. A one-shot
// is selected only when this dispatch atomically removes it from both indexes;
// a concurrent dispatch or unload that already removed it wins the claim.
func (r *Runtime) claimDispatchBindings(bindings []Binding) (selected, claimed []Binding) {
	selected = make([]Binding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.Lifetime != BindingOnce {
			selected = append(selected, binding)
			continue
		}
		owner := r.script(binding.Script)
		if owner == nil || !owner.removeBindingIfPresent(binding) {
			continue
		}
		selected = append(selected, binding)
		claimed = append(claimed, binding)
	}
	return selected, claimed
}

// notifyConsumedBindings reports one-shot retirement after every selected
// one-shot has left the authoritative registries and before event callbacks
// begin. Observer failures do not resurrect a consumed listener or suppress
// its callback; Dispatch returns them alongside any callback failure.
func (r *Runtime) notifyConsumedBindings(ctx context.Context, bindings []Binding) error {
	if r.observer == nil {
		return nil
	}
	var result error
	for _, binding := range bindings {
		if err := r.observer.Unregistered(ctx, cloneBinding(binding)); err != nil {
			result = errors.Join(result, preserveNativeBoundaryError(ctx, err))
		}
	}
	return result
}

// invokeDispatchedBinding applies Sleep's named-closure event ABI. Event
// callbacks receive the concrete event name through the synthetic $0 message;
// non-event callbacks retain their existing unnamed invocation behavior.
func (r *Runtime) invokeDispatchedBinding(
	ctx context.Context,
	kind BindingKind,
	eventName string,
	binding Binding,
	arguments []Value,
) (Value, error) {
	invocationContext, release, err := r.prepareBindingInvocation(ctx, binding, arguments)
	if err != nil {
		return Null(), err
	}
	defer release()
	if kind == BindingEvent {
		if callback, ok := binding.Callback.(namedBindingCallable); ok {
			value, invokeErr := callback.invokeNamed(invocationContext, eventName, arguments...)
			return value, joinExecutionError(invokeErr, release)
		}
	}
	value, invokeErr := binding.Callback.Invoke(invocationContext, arguments...)
	return value, joinExecutionError(invokeErr, release)
}

// invokeRegisteredBinding is the shared direct-invocation path for
// InvokeBinding and InvokeBindingByID. A one-shot registration must not become
// repeatable merely because an importer selected it directly instead of using
// DispatchEvent. claimed is false only when another dispatch or unload won the
// one-shot claim; callers retain their public API's existing unsupported-error
// shape for that race.
func (r *Runtime) invokeRegisteredBinding(
	ctx context.Context,
	binding Binding,
	arguments []Value,
) (value Value, err error, claimed bool) {
	if ctx == nil {
		ctx = context.Background()
	}
	// Direct binding invocation is a public evaluator entry. Give a top-level
	// by-name or by-ID call one fresh callback-tree budget, while preserving the
	// meter carried by an active evaluator during synchronous reentry.
	ctx = withExecutionMeter(ctx, r)
	var lifecycleErr error
	if binding.Lifetime == BindingOnce {
		selected, consumed := r.claimDispatchBindings([]Binding{binding})
		if len(selected) == 0 {
			return Null(), nil, false
		}
		lifecycleErr = r.notifyConsumedBindings(ctx, consumed)
	}
	value, invokeErr := r.invokeDispatchedBinding(ctx, binding.Kind, binding.Name, binding, arguments)
	return value, errors.Join(invokeErr, lifecycleErr), true
}

// Dispatch invokes every matching registration in load order. Event
// one-shots are all atomically consumed before any selected callback runs.
// Wildcard handlers receive the concrete event name as their first positional
// argument, while every event closure receives it separately as Sleep's $0.
func (r *Runtime) Dispatch(ctx context.Context, kind BindingKind, name string, arguments ...Value) ([]Value, error) {
	if r == nil {
		return nil, errors.New("opfor: runtime is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	// A dispatch and every callback it reaches form one evaluator tree. Install
	// one top-level budget before taking the binding snapshot so exact, wildcard,
	// and synchronously reentrant handlers all share it. withExecutionMeter keeps
	// an inherited meter instead of creating a nested budget.
	ctx = withExecutionMeter(ctx, r)
	exact, wildcard := r.dispatchBindingSnapshot(kind, name)
	selectedExact, claimedExact := r.claimDispatchBindings(exact)
	selectedWildcard, claimedWildcard := r.claimDispatchBindings(wildcard)
	claimed := append(claimedExact, claimedWildcard...)
	lifecycleErr := r.notifyConsumedBindings(ctx, claimed)

	results := make([]Value, 0, len(selectedExact)+len(selectedWildcard))
	var callbackErr error
	invoke := func(bindings []Binding, callbackArguments []Value) {
		for _, binding := range bindings {
			value, err := r.invokeDispatchedBinding(ctx, kind, name, binding, callbackArguments)
			if err != nil {
				callbackErr = errors.Join(callbackErr, err)
				continue
			}
			results = append(results, value)
		}
	}
	invoke(selectedExact, arguments)
	if len(selectedWildcard) != 0 {
		callbackArguments := append([]Value{String(name)}, arguments...)
		invoke(selectedWildcard, callbackArguments)
	}
	return results, errors.Join(callbackErr, lifecycleErr)
}

// DispatchEvent invokes on <name> handlers and then on * handlers.
func (r *Runtime) DispatchEvent(ctx context.Context, name string, arguments ...Value) ([]Value, error) {
	return r.Dispatch(ctx, BindingEvent, name, arguments...)
}

// DispatchPopupHook invokes every exact popup <name> registration in load
// order. Aggressor popup declarations are additive: each matching script layer
// contributes items to the same menu composition. Use InvokeBinding with
// BindingPopup when intentionally selecting only the newest layer.
//
// A name without active registrations is a successful no-op and returns an
// empty result slice, matching DispatchEvent's dispatch behavior.
func (r *Runtime) DispatchPopupHook(ctx context.Context, name string, arguments ...Value) ([]Value, error) {
	return r.Dispatch(ctx, BindingPopup, name, arguments...)
}

// ConsoleInvocation describes one user-entered command line dispatched to a
// command, alias, or ssh_alias binding. RawInput is the complete line as typed,
// including Name.
//
// ParsedArguments optionally supplies importer-parsed positional arguments,
// excluding Name. Nil preserves RawInput parsing, including quote and command
// name validation and the legacy treatment of empty RawInput as Name with no
// arguments. A non-nil slice is used exactly as supplied; in particular, an
// empty slice means no arguments, whitespace, empty strings, and literal double
// quotes are preserved without reparsing RawInput, and RawInput is passed to $0
// byte-for-byte even when empty.
//
// Command callbacks receive the unmodified RawInput in $0 and parsed arguments
// in $1 onward. Alias and ssh_alias callbacks additionally receive SessionID in
// $1, with parsed arguments beginning at $2. In both cases, @_ contains only
// the positional values and does not contain $0.
type ConsoleInvocation struct {
	Kind            BindingKind
	Name            string
	RawInput        string
	ParsedArguments []string
	SessionID       Value
}

// InvokeConsole invokes the most recently registered command, alias, or
// ssh_alias callback using Aggressor's console argument contract. When
// ParsedArguments is nil, arguments are separated by ASCII whitespace; double
// quotes group whitespace into one argument and are removed from the
// positional value. Otherwise, ParsedArguments is used exactly and RawInput is
// passed byte-for-byte as Sleep's separate $0 closure message.
func (r *Runtime) InvokeConsole(ctx context.Context, invocation ConsoleInvocation) (Value, error) {
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	switch invocation.Kind {
	case BindingCommand, BindingAlias, BindingSSHAlias:
	default:
		return Null(), fmt.Errorf("opfor: binding kind %q is not a console command or alias", invocation.Kind)
	}
	if invocation.Name == "" {
		return Null(), errors.New("opfor: console binding name is empty")
	}

	bindings := r.Bindings(invocation.Kind, invocation.Name)
	if len(bindings) == 0 {
		return Null(), &UnsupportedError{Operation: string(invocation.Kind), Name: invocation.Name}
	}

	rawInput := invocation.RawInput
	parsedArguments := invocation.ParsedArguments
	if parsedArguments == nil {
		if rawInput == "" {
			rawInput = invocation.Name
		}
		tokens, err := splitConsoleInput(rawInput)
		if err != nil {
			return Null(), err
		}
		if len(tokens) == 0 || tokens[0] != invocation.Name {
			actual := ""
			if len(tokens) != 0 {
				actual = tokens[0]
			}
			return Null(), fmt.Errorf("opfor: console input command %q does not match binding %q", actual, invocation.Name)
		}
		parsedArguments = tokens[1:]
	}

	arguments := make([]Value, 0, len(parsedArguments)+1)
	if invocation.Kind == BindingAlias || invocation.Kind == BindingSSHAlias {
		arguments = append(arguments, invocation.SessionID)
	}
	for _, token := range parsedArguments {
		arguments = append(arguments, String(token))
	}

	binding := bindings[len(bindings)-1]
	closure, ok := binding.Callback.(namedBindingCallable)
	if !ok {
		// Console bindings need Sleep's named $0 message in addition to their
		// positional values. Retain a defensive error rather than silently
		// exposing the raw message as an ordinary positional value.
		return Null(), fmt.Errorf("opfor: %s binding %q does not expose Sleep closure invocation", invocation.Kind, invocation.Name)
	}
	invocationContext, release, err := r.prepareBindingInvocation(ctx, binding, arguments)
	if err != nil {
		return Null(), err
	}
	defer release()
	// InvokeConsole is a public evaluator entry even though named closure
	// invocation deliberately bypasses scriptClosure.Invoke to supply $0.
	// Reuse an inherited meter for synchronous reentry; otherwise create the
	// invocation's fresh top-level budget here.
	invocationContext = withExecutionMeter(invocationContext, r)
	value, err := closure.invokeNamed(invocationContext, rawInput, arguments...)
	return value, joinExecutionError(err, release)
}

// InvokeBinding invokes the most recently registered matching callback.
func (r *Runtime) InvokeBinding(ctx context.Context, kind BindingKind, name string, arguments ...Value) (Value, error) {
	bindings := r.Bindings(kind, name)
	if len(bindings) == 0 {
		return Null(), &UnsupportedError{Operation: string(kind), Name: name}
	}
	binding := bindings[len(bindings)-1]
	value, err, claimed := r.invokeRegisteredBinding(ctx, binding, arguments)
	if !claimed {
		return Null(), &UnsupportedError{Operation: string(kind), Name: name}
	}
	return value, err
}

func splitConsoleInput(input string) ([]string, error) {
	arguments := make([]string, 0, 4)
	var argument strings.Builder
	quoted := false
	started := false

	flush := func() {
		if !started {
			return
		}
		arguments = append(arguments, argument.String())
		argument.Reset()
		started = false
	}

	for index := 0; index < len(input); index++ {
		character := input[index]
		switch {
		case character == '"':
			quoted = !quoted
			started = true
		case !quoted && isConsoleWhitespace(character):
			flush()
		default:
			argument.WriteByte(character)
			started = true
		}
	}
	if quoted {
		return nil, errors.New("opfor: console input has an unterminated double quote")
	}
	flush()
	return arguments, nil
}

func isConsoleWhitespace(character byte) bool {
	switch character {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

func (r *Runtime) unload(ctx context.Context, script *Script) error {
	if r == nil || script == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if contextUnloadsScript(ctx, script) {
		// A teardown observer may defensively unload its own script. The active
		// finalizer already owns that operation, so waiting here would deadlock.
		return nil
	}
	token, reentrant := classifyUnloadContext(ctx, r, script)
	executionReentrant := contextOwnsRuntimeExecution(ctx, r)
	cleanupCtx := ctx
	releaseCleanupContext := func() {}
	if executionReentrant {
		cleanupCtx, releaseCleanupContext = detachExecutionLeaseCancellationLease(ctx)
	}
	cleanupContextTransferred := false
	defer func() {
		if !cleanupContextTransferred {
			releaseCleanupContext()
		}
	}()

	var cancel context.CancelFunc
	var uiResources []aggressorUIResource
	var finalize bool
	script.mu.Lock()
	if script.unloadDone == nil {
		script.unloadDone = make(chan struct{})
	}
	if script.unloadRequested && channelClosed(script.unloadDone) {
		script.mu.Unlock()
		if reentrant {
			return nil
		}
		return script.consumeUnloadErrorForWaiter(ctx)
	}
	if !script.unloadRequested {
		script.active = false
		script.unloadRequested = true
		script.unloadContext = cleanupCtx
		script.unloadContextRelease = releaseCleanupContext
		cleanupContextTransferred = true
		if token != nil && token.active.Load() {
			script.unloadRecipient = token
			script.unloadRecipientState = unloadRecipientReserved
			script.unloadRecipientDone = make(chan struct{})
		}
		cancel = script.executionCancel
		// Stop publishing callbacks at the same Script.mu -> Runtime.mu
		// boundary used by registerBinding. Observer teardown remains deferred
		// until every admitted execution has left.
		r.mu.Lock()
		for _, binding := range script.bindings {
			r.removeRuntimeBindingLocked(binding)
		}
		r.mu.Unlock()
		if r.aggressorCommands != nil {
			// Revoke command-help registrations at the same admission boundary
			// as executable bindings. A draining callback must not leave metadata
			// visible for an alias that can no longer be invoked.
			r.aggressorCommands.removeScript(script.id)
		}
		if r.aggressorBeaconTechniques != nil {
			// Match binding and callback revocation at unload admission, before
			// any already-admitted script execution has finished draining.
			r.aggressorBeaconTechniques.removeScript(script.id)
		}
		if len(script.aggressorUIResources) != 0 {
			uiResources = make([]aggressorUIResource, 0, len(script.aggressorUIResources))
			for resource := range script.aggressorUIResources {
				uiResources = append(uiResources, resource)
			}
			script.aggressorUIResources = nil
		}
	}
	if script.executions == 0 && !script.unloadFinalizing {
		script.unloadFinalizing = true
		finalize = true
	}
	if !reentrant {
		script.unloadWaiters++
	}
	done := script.unloadDone
	script.mu.Unlock()

	for _, resource := range uiResources {
		resource.revokeAggressorUI()
	}
	if cancel != nil {
		cancel()
	}
	if finalize {
		go r.finishUnload(cleanupCtx, script)
	}
	if reentrant {
		return nil
	}

	select {
	case <-done:
		script.mu.Lock()
		if script.unloadWaiters > 0 {
			script.unloadWaiters--
		}
		script.mu.Unlock()
		return script.consumeUnloadErrorForWaiter(ctx)
	case <-ctx.Done():
		script.mu.Lock()
		if script.unloadWaiters > 0 {
			script.unloadWaiters--
		}
		script.mu.Unlock()
		return ctx.Err()
	}
}

// finishUnload runs once, after the script has stopped admitting work and its
// last execution lease has left. Lock ordering is Script.mu then Runtime.mu;
// binding publication follows the same order so a registration cannot appear
// after this snapshot and leak from the runtime registry.
func (r *Runtime) finishUnload(ctx context.Context, script *Script) {
	if r == nil || script == nil {
		return
	}
	script.mu.Lock()
	releaseUnloadCaller := script.unloadContextRelease
	script.unloadContextRelease = nil
	script.mu.Unlock()
	if releaseUnloadCaller != nil {
		defer releaseUnloadCaller()
	}
	ctx, releaseUnloadContext := withScriptUnloadContext(ctx, script)
	defer releaseUnloadContext()

	script.mu.Lock()
	registrations := append([]Binding(nil), script.bindings...)
	tasks := make([]*forkTask, 0, len(script.forkTasks))
	for task := range script.forkTasks {
		tasks = append(tasks, task)
	}
	script.forkTasks = nil
	socketTasks := make([]*sleepSocketTask, 0, len(script.socketTasks))
	for task := range script.socketTasks {
		socketTasks = append(socketTasks, task)
	}
	script.socketTasks = nil
	readTasks := make([]*sleepReadTask, 0, len(script.readTasks))
	for task := range script.readTasks {
		readTasks = append(readTasks, task)
	}
	script.readTasks = nil
	processes := make([]*processObject, 0, len(script.processes))
	for process := range script.processes {
		processes = append(processes, process)
	}
	script.processes = nil
	loaders := make([]*portableScriptLoader, 0, len(script.scriptLoaders))
	for loader := range script.scriptLoaders {
		loaders = append(loaders, loader)
	}
	// Never close a child runtime while holding Script.mu. A child close may
	// unregister bindings through an importer observer and may recursively
	// unload another source-backed ScriptLoader.
	script.scriptLoaders = nil
	loadableUses := append([]scriptLoadableUse(nil), script.loadableUses...)
	script.loadables = nil
	script.loadableUses = nil
	script.bindings = nil
	script.functions = make(map[string]Callable)
	script.removedFuncs = make(map[string]struct{})
	script.functionOrder = nil
	script.mu.Unlock()

	var result error
	for index := len(loadableUses) - 1; index >= 0; index-- {
		bridge := loadableUses[index].bridge
		if isNilInterface(bridge) {
			continue
		}
		if err := bridge.ScriptUnloaded(ctx, script); err != nil {
			result = errors.Join(result, preserveNativeBoundaryError(ctx, err))
		}
	}
	for _, task := range tasks {
		task.cancelAndClose()
	}
	for _, task := range socketTasks {
		task.cancelAndClose()
	}
	joinableReadTasks, incompleteReadCancellation := revokeSleepReadTasks(readTasks)
	if incompleteReadCancellation {
		result = errors.Join(result, ErrReadCancellationUnsupported)
	}
	for _, loader := range loaders {
		closeErr := loader.close(ctx)
		remaining, waitExpired := splitContextWaitError(ctx, closeErr)
		if remaining != nil {
			result = errors.Join(result, remaining)
		}
		if waitExpired {
			if terminalErr := loader.close(context.Background()); terminalErr != nil {
				terminalErr, _ = splitContextWaitError(ctx, terminalErr)
				result = errors.Join(result, terminalErr)
			}
		}
	}
	for _, process := range processes {
		if err := process.close(); err != nil {
			result = errors.Join(result, err)
		}
	}
	for _, process := range processes {
		if err := process.join(ctx); err != nil {
			remaining, waitExpired := splitContextWaitError(ctx, err)
			if remaining != nil {
				result = errors.Join(result, remaining)
			}
			if waitExpired {
				if terminalErr := process.join(context.Background()); terminalErr != nil {
					terminalErr, _ = splitContextWaitError(ctx, terminalErr)
					result = errors.Join(result, terminalErr)
				}
			}
		}
	}
	for _, task := range socketTasks {
		if err := task.join(ctx); err != nil {
			remaining, waitExpired := splitContextWaitError(ctx, err)
			if remaining != nil {
				result = errors.Join(result, remaining)
			}
			if waitExpired {
				if terminalErr := task.join(context.Background()); terminalErr != nil {
					terminalErr, _ = splitContextWaitError(ctx, terminalErr)
					result = errors.Join(result, terminalErr)
				}
			}
		}
	}
	for _, task := range joinableReadTasks {
		if err := task.join(ctx); err != nil {
			remaining, waitExpired := splitContextWaitError(ctx, err)
			if remaining != nil {
				result = errors.Join(result, remaining)
			}
			if waitExpired {
				if terminalErr := task.join(context.Background()); terminalErr != nil {
					terminalErr, _ = splitContextWaitError(ctx, terminalErr)
					result = errors.Join(result, terminalErr)
				}
			}
		}
	}
	for _, task := range tasks {
		if err := task.join(ctx); err != nil {
			remaining, waitExpired := splitContextWaitError(ctx, err)
			if remaining != nil {
				result = errors.Join(result, remaining)
			}
			if waitExpired {
				if terminalErr := task.join(context.Background()); terminalErr != nil {
					terminalErr, _ = splitContextWaitError(ctx, terminalErr)
					result = errors.Join(result, terminalErr)
				}
			}
		}
		unloadErr := r.unload(ctx, task.child)
		remaining, waitExpired := splitContextWaitError(ctx, unloadErr)
		if remaining != nil {
			result = errors.Join(result, remaining)
		}
		task.child.mu.Lock()
		complete := channelClosed(task.child.unloadDone)
		task.child.mu.Unlock()
		if waitExpired || !complete {
			if err := task.child.waitUnloaded(context.Background()); err != nil {
				err, _ = splitContextWaitError(ctx, err)
				result = errors.Join(result, err)
			}
		}
	}
	r.mu.Lock()
	delete(r.scripts, script.id)
	_, notifyLifecycle := r.lifecycleScripts[script.id]
	delete(r.lifecycleScripts, script.id)
	if r.evalScript == script {
		r.evalScript = nil
	}
	for _, binding := range registrations {
		r.removeRuntimeBindingLocked(binding)
	}
	r.mu.Unlock()

	if r.observer != nil {
		for index := len(registrations) - 1; index >= 0; index-- {
			if err := r.observer.Unregistered(ctx, cloneBinding(registrations[index])); err != nil {
				result = errors.Join(result, preserveNativeBoundaryError(ctx, err))
			}
		}
	}
	if notifyLifecycle && r.lifecycle != nil {
		if err := r.lifecycle.ScriptUnloaded(ctx, script); err != nil {
			result = errors.Join(result, fmt.Errorf("opfor: script %d unload observer: %w", script.id, err))
		}
	}

	script.mu.Lock()
	script.unloadErr = result
	if !channelClosed(script.unloadDone) {
		close(script.unloadDone)
	}
	script.mu.Unlock()
}

func (s *Script) registerBinding(ctx context.Context, binding Binding, callback Callable) (resultErr error) {
	if s == nil || s.runtime == nil {
		return ErrScriptUnloaded
	}
	generation := scriptGenerationFromContext(ctx, s)
	if generation == nil {
		generation = s.currentScriptGeneration()
	}
	executionCtx, release, err := s.acquireGenerationExecution(ctx, generation)
	if err != nil {
		return err
	}
	ctx = executionCtx
	defer func() { resultErr = joinExecutionError(resultErr, release) }()
	// Declaration-form `when name { ... }` reaches this common path through
	// OpBind. Keep the language semantic authoritative here so every compiler
	// and environment-registration path gets the same one-shot lifetime. Only
	// apply non-default lifetimes: importer-created persistent bindings and
	// explicitly one-shot programmatic bindings otherwise remain untouched.
	spec, knownEnvironment := envspec.LookupFold(binding.Keyword)
	if knownEnvironment && spec.Lifetime == envspec.Once && binding.Kind == BindingKind(spec.Binding) {
		binding.Lifetime = BindingOnce
	}
	binding.Script = s.id
	binding.Callback = bindScriptGenerationCallable(s, generation, callback)
	if predicate, ok := binding.Predicate.(*scriptPredicateEvaluator); ok && predicate != nil {
		predicate.generation = generation
	}
	binding.Parent = cloneBindingInvocation(currentBindingInvocation(ctx))

	// Lock order is Script.mu then Runtime.mu. Keeping both locks from the
	// active check through publication makes registration atomic with unload's
	// revocation boundary.
	s.mu.Lock()
	if !s.generationAdmissibleLocked(generation) {
		s.mu.Unlock()
		return ErrScriptUnloaded
	}
	s.nextBinding++
	binding.ID = s.nextBinding
	s.bindings = append(s.bindings, binding)
	if binding.Kind == BindingSub || binding.Kind == BindingInline {
		s.functions[binding.Name] = callback
		s.functionOrder = append(s.functionOrder, binding.Name)
		delete(s.removedFuncs, binding.Name)
	}
	r := s.runtime
	r.mu.Lock()
	if r.bindings == nil {
		r.bindings = make(map[BindingKind]map[string][]Binding)
	}
	if r.bindings[binding.Kind] == nil {
		r.bindings[binding.Kind] = make(map[string][]Binding)
	}
	r.bindings[binding.Kind][binding.Name] = append(r.bindings[binding.Kind][binding.Name], binding)
	if r.bindingOrder == nil {
		r.bindingOrder = make(map[BindingKind][]Binding)
	}
	r.bindingOrder[binding.Kind] = append(r.bindingOrder[binding.Kind], binding)
	r.mu.Unlock()
	s.mu.Unlock()

	if r.observer != nil {
		err := preserveNativeBoundaryError(ctx, r.observer.Registered(ctx, cloneBinding(binding)))
		err = joinExecutionContextError(ctx, err)
		if err != nil {
			// Registration failure is transactional from the script's point of
			// view while the script remains active. If unload won concurrently,
			// retain the private snapshot so finishUnload emits the matching
			// Unregistered notification after Registered returns.
			s.removeBindingIfPresent(binding)
			return err
		}
	}
	if (binding.Kind == BindingSub || binding.Kind == BindingInline) && s.sharedEnvironment != nil {
		var publishErr error
		if binding.Kind == BindingInline {
			closure, _ := callback.(*scriptClosure)
			publishErr = s.sharedEnvironment.publishInline(r, binding.Name, closure)
		} else {
			publishErr = s.sharedEnvironment.publish(r, binding.Name, callback)
		}
		if publishErr != nil {
			s.removeBinding(binding)
			if r.observer != nil {
				if observerErr := r.observer.Unregistered(ctx, cloneBinding(binding)); observerErr != nil {
					publishErr = errors.Join(publishErr, preserveNativeBoundaryError(ctx, observerErr))
				}
			}
			return publishErr
		}
	}
	return nil
}

// removeRuntimeBindingLocked removes binding from the runtime registry. The
// caller must hold Runtime.mu; unload callers additionally hold Script.mu so
// this operation is atomic with registerBinding publication.
func (r *Runtime) removeRuntimeBindingLocked(binding Binding) {
	if r == nil {
		return
	}
	byName := r.bindings[binding.Kind]
	entries := byName[binding.Name]
	for index := len(entries) - 1; index >= 0; index-- {
		if sameRuntimeBinding(entries[index], binding) {
			entries = deleteBindingAt(entries, index)
			break
		}
	}
	if len(entries) == 0 {
		delete(byName, binding.Name)
	} else {
		byName[binding.Name] = entries
	}

	ordered := r.bindingOrder[binding.Kind]
	for index := len(ordered) - 1; index >= 0; index-- {
		if sameRuntimeBinding(ordered[index], binding) {
			ordered = deleteBindingAt(ordered, index)
			break
		}
	}
	if len(ordered) == 0 {
		delete(r.bindingOrder, binding.Kind)
	} else {
		r.bindingOrder[binding.Kind] = ordered
	}
}

func sameRuntimeBinding(left, right Binding) bool {
	return left.Script == right.Script && left.ID == right.ID
}

func deleteBindingAt(bindings []Binding, index int) []Binding {
	copy(bindings[index:], bindings[index+1:])
	bindings[len(bindings)-1] = Binding{}
	return bindings[:len(bindings)-1]
}

func (s *Script) removeBinding(binding Binding) {
	s.mu.Lock()
	for index := len(s.bindings) - 1; index >= 0; index-- {
		if s.bindings[index].ID == binding.ID {
			s.bindings = deleteBindingAt(s.bindings, index)
			break
		}
	}
	if binding.Kind == BindingSub || binding.Kind == BindingInline {
		delete(s.functions, binding.Name)
	}
	s.mu.Unlock()

	r := s.runtime
	r.mu.Lock()
	r.removeRuntimeBindingLocked(binding)
	r.mu.Unlock()
}

// removeBindingIfPresent atomically removes one still-active script binding
// from both indexes. The boolean lets callers emit exactly one lifecycle
// notification even when alias_clear races another clear or script unload.
// Inactive scripts retain their private binding snapshot for finishUnload,
// which remains responsible for the corresponding Unregistered notification.
func (s *Script) removeBindingIfPresent(binding Binding) bool {
	if s == nil || s.runtime == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.active {
		return false
	}
	index := -1
	for candidate := len(s.bindings) - 1; candidate >= 0; candidate-- {
		if s.bindings[candidate].ID == binding.ID && sameRuntimeBinding(s.bindings[candidate], binding) {
			index = candidate
			break
		}
	}
	if index < 0 {
		return false
	}
	s.bindings = deleteBindingAt(s.bindings, index)
	if binding.Kind == BindingSub || binding.Kind == BindingInline {
		delete(s.functions, binding.Name)
	}
	r := s.runtime
	r.mu.Lock()
	r.removeRuntimeBindingLocked(binding)
	r.mu.Unlock()
	return true
}

func bindingKind(keyword string) BindingKind {
	keyword = strings.ToLower(keyword)
	if spec, ok := envspec.Lookup(keyword); ok {
		return BindingKind(spec.Binding)
	}
	return BindingKind(keyword)
}

func knownBindingEnvironment(keyword string) bool {
	_, ok := envspec.Lookup(strings.ToLower(keyword))
	return ok
}

func (s *Script) resolveFunction(name string) Callable {
	name = strings.TrimPrefix(name, "&")
	s.mu.RLock()
	shared := s.sharedEnvironment
	local := s.functions[name]
	s.mu.RUnlock()
	if shared != nil {
		return shared.resolve(name, s)
	}
	return local
}

// RegisterFunction installs or replaces a Go-native function in this Script's
// private environment. It is primarily used by LoadableBridge.ScriptLoaded;
// the function shadows runtime defaults and is revoked automatically on
// unload. Calls preserve Invocation name, source span, Script ID, and
// pass-by-name Argument references. A function installed from a portable
// ScriptLoader execution belongs to that exact execution generation and is
// revoked by logical unload even though the Script itself remains active.
func (s *Script) RegisterFunction(name string, function NativeFunc) error {
	if s == nil || s.runtime == nil {
		return ErrScriptUnloaded
	}
	normalized, err := normalizeFunctionName(name)
	if err != nil {
		return err
	}
	if function == nil {
		return fmt.Errorf("opfor: function %q is nil", name)
	}
	s.mu.Lock()
	generation := s.generation
	if !s.generationAdmissibleLocked(generation) {
		s.mu.Unlock()
		return ErrScriptUnloaded
	}
	previous, hadPrevious := s.functions[normalized]
	shared := s.sharedEnvironment
	previousShared, hadPreviousShared := Null(), false
	if shared != nil {
		previousShared, hadPreviousShared = shared.functionEntry(normalized)
	}
	callable := &scriptNativeCallable{
		owner:             s,
		generation:        generation,
		name:              normalized,
		function:          function,
		previous:          previous,
		hadPrevious:       hadPrevious,
		previousShared:    previousShared,
		hadPreviousShared: hadPreviousShared,
	}
	_, wasRemoved := s.removedFuncs[normalized]
	previousOrderLength := len(s.functionOrder)
	s.functions[normalized] = callable
	s.functionOrder = append(s.functionOrder, normalized)
	delete(s.removedFuncs, normalized)
	s.mu.Unlock()
	if shared != nil {
		if err := shared.publish(s.runtime, normalized, callable); err != nil {
			s.mu.Lock()
			if hadPrevious {
				s.functions[normalized] = previous
			} else {
				delete(s.functions, normalized)
			}
			if previousOrderLength <= len(s.functionOrder) {
				s.functionOrder = s.functionOrder[:previousOrderLength]
			}
			if wasRemoved {
				s.removedFuncs[normalized] = struct{}{}
			}
			s.mu.Unlock()
			return err
		}
	}
	return nil
}

type scriptNativeCallable struct {
	owner             *Script
	generation        *scriptGeneration
	name              string
	function          NativeFunc
	previous          Callable
	hadPrevious       bool
	previousShared    Value
	hadPreviousShared bool
}

func (callable *scriptNativeCallable) String() string {
	if callable == nil {
		return "&"
	}
	return "&" + callable.name
}

func (callable *scriptNativeCallable) Invoke(ctx context.Context, values ...Value) (Value, error) {
	arguments := make([]Argument, len(values))
	for index, value := range values {
		arguments[index] = Argument{Value: value}
	}
	return callable.invokeArgumentsAt(ctx, Span{}, arguments)
}

func (callable *scriptNativeCallable) invokeArgumentsAt(
	ctx context.Context,
	span Span,
	arguments []Argument,
) (result Value, resultErr error) {
	if callable == nil || callable.owner == nil || callable.function == nil {
		return Null(), ErrScriptUnloaded
	}
	executionCtx, release, err := callable.owner.acquireGenerationExecution(ctx, callable.generation)
	if err != nil {
		return Null(), err
	}
	defer func() { resultErr = joinExecutionError(resultErr, release) }()
	result, resultErr = callable.function(executionCtx, Invocation{
		Runtime:    callable.owner.runtime,
		Script:     callable.owner.id,
		Name:       callable.name,
		Arguments:  arguments,
		Span:       span,
		generation: callable.generation,
	})
	resultErr = preserveNativeBoundaryError(executionCtx, resultErr)
	resultErr = joinExecutionContextError(executionCtx, resultErr)
	return result, resultErr
}

func (s *Script) setFunction(name string, callable Callable) error {
	name = strings.TrimPrefix(name, "&")
	s.mu.Lock()
	previous, hadPrevious := s.functions[name]
	_, wasRemoved := s.removedFuncs[name]
	previousOrderLength := len(s.functionOrder)
	if callable == nil {
		delete(s.functions, name)
		if s.removedFuncs == nil {
			s.removedFuncs = make(map[string]struct{})
		}
		s.removedFuncs[name] = struct{}{}
	} else {
		s.functions[name] = callable
		s.functionOrder = append(s.functionOrder, name)
		delete(s.removedFuncs, name)
	}
	shared := s.sharedEnvironment
	s.mu.Unlock()
	if shared != nil {
		if err := shared.publish(s.runtime, name, callable); err != nil {
			s.mu.Lock()
			if hadPrevious {
				s.functions[name] = previous
			} else {
				delete(s.functions, name)
			}
			if previousOrderLength <= len(s.functionOrder) {
				s.functionOrder = s.functionOrder[:previousOrderLength]
			}
			if wasRemoved {
				if s.removedFuncs == nil {
					s.removedFuncs = make(map[string]struct{})
				}
				s.removedFuncs[name] = struct{}{}
			} else {
				delete(s.removedFuncs, name)
			}
			s.mu.Unlock()
			return err
		}
	}
	return nil
}

func (s *Script) functionWasRemoved(name string) bool {
	name = strings.TrimPrefix(name, "&")
	s.mu.RLock()
	shared := s.sharedEnvironment
	_, removed := s.removedFuncs[name]
	s.mu.RUnlock()
	if shared != nil {
		return shared.removed(name)
	}
	return removed
}

func (s *Script) setStackTrace(frames []string) {
	if s == nil {
		return
	}
	s.mu.Lock()
	s.stackTrace = append(s.stackTrace[:0], frames...)
	s.mu.Unlock()
}

func (s *Script) getStackTrace() []string {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	frames := append([]string(nil), s.stackTrace...)
	s.stackTrace = s.stackTrace[:0]
	s.mu.Unlock()
	return frames
}

func (s *Script) addImport(target string) {
	if strings.HasSuffix(target, ".*") {
		packageName := strings.TrimSuffix(target, ".*")
		s.mu.Lock()
		for _, imported := range s.importPackages {
			if imported == packageName {
				s.mu.Unlock()
				return
			}
		}
		s.importPackages = append(s.importPackages, packageName)
		s.mu.Unlock()
		return
	}
	short := target
	if index := strings.LastIndex(short, "."); index >= 0 {
		short = short[index+1:]
	}
	s.mu.Lock()
	s.imports[short] = target
	s.mu.Unlock()
}

func (s *Script) resolveClass(name string) string {
	s.mu.RLock()
	resolved := s.imports[name]
	packages := append([]string(nil), s.importPackages...)
	s.mu.RUnlock()
	if resolved != "" {
		return resolved
	}
	// Sleep seeds java.lang.*, java.util.*, and sleep.runtime.* before script
	// imports. Preserve those known defaults when an unrelated wildcard import
	// is also active instead of manufacturing (for example) com.eric.Class.
	if portableDefaultClasses[name] != "" {
		return name
	}
	if len(packages) != 0 && !strings.Contains(name, ".") {
		// Sleep's ImportManager probes every wildcard package until a class is
		// found. Pure-Go class hints and the inert, archive-verified fixture
		// adapter provide authoritative subsets. Prefer their matching package
		// instead of always manufacturing a name from the first wildcard import.
		if s.runtime != nil {
			state := s.runtime.portableFixtureState()
			for _, packageName := range packages {
				candidate := packageName + "." + name
				if _, known := portableImportedClasses[candidate]; known {
					return candidate
				}
				if state.allows(s.id, candidate) {
					return candidate
				}
			}
		}
		return packages[0] + "." + name
	}
	return name
}

func (s *Script) importedClass(original, resolved string) bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for short, target := range s.imports {
		if original == short || original == target || resolved == target {
			return true
		}
	}
	for _, packageName := range s.importPackages {
		if resolved == packageName+"."+original || strings.HasPrefix(resolved, packageName+".") {
			return true
		}
	}
	return false
}

// scriptClosure is a compiled closure plus its captured variable frame. The
// mutex serializes a suspended Sleep coroutine without serializing unrelated
// callbacks from the same script.
type scriptClosure struct {
	script   *Script
	function *bytecode.Function
	captured *scope
	state    *scope
	thisHash *Hash
	id       uint64
	inline   bool

	mu        sync.Mutex
	stateInit sync.Mutex
	stateErr  error
	suspended []*fiber
}

func (c *scriptClosure) String() string {
	if c == nil || c.function == nil {
		return "&closure"
	}
	if c.id == 0 {
		return fmt.Sprintf("&%s", c.function.Name)
	}
	start, end := 0, 0
	source := c.function.Span.Source
	for _, instruction := range c.function.Instructions {
		if instruction.Op == bytecode.OpEnd {
			continue
		}
		line := instruction.Span.Start.Line
		if line <= 0 {
			continue
		}
		if start == 0 || line < start {
			start = line
		}
		if line > end {
			end = line
		}
	}
	hasInstruction := start != 0
	if !hasInstruction {
		start, end = c.function.Span.Start.Line, c.function.Span.Start.Line
	}
	if hasInstruction && start == end && c.function.Span.End.Line > end+1 {
		end = c.function.Span.End.Line - 1
	}
	start = sleepDisplayLineNumber(source, start)
	end = sleepDisplayLineNumber(source, end)
	displaySource := sleepSourceDisplayName(source)
	location := fmt.Sprintf("%s:%d", displaySource, start)
	if end > start {
		location = fmt.Sprintf("%s:%d-%d", displaySource, start, end)
	}
	return fmt.Sprintf("&closure[%s]#%d", location, c.id)
}

func (s *Script) newClosure(function *bytecode.Function, captured *scope) *scriptClosure {
	s.mu.Lock()
	s.nextClosure++
	id := s.nextClosure
	s.mu.Unlock()
	return &scriptClosure{script: s, function: function, captured: captured, id: id}
}

func (s *Script) newInline(function *bytecode.Function, captured *scope) *scriptClosure {
	return &scriptClosure{script: s, function: function, captured: captured, inline: true}
}

type localScopeLeakError struct {
	count   int
	closure *scriptClosure
}

func (e *localScopeLeakError) Error() string {
	if e == nil {
		return "unaccounted local stack frame(s)"
	}
	return fmt.Sprintf("%d unaccounted local stack frame(s) in %s (perhaps you forgot to &popl?)", e.count, e.closure)
}

func (c *scriptClosure) Invoke(ctx context.Context, arguments ...Value) (Value, error) {
	if c != nil && c.script != nil {
		ctx = withExecutionMeter(ctx, c.script.runtime)
	}
	return c.invoke(ctx, nil, arguments...)
}

// invokeNamed supplies Sleep's separate named closure message. Console
// bindings use it for the exact command line exposed as $0.
func (c *scriptClosure) invokeNamed(ctx context.Context, name string, arguments ...Value) (Value, error) {
	return c.invoke(ctx, []Argument{{Name: "$0", Value: String(name)}}, arguments...)
}

func (c *scriptClosure) invoke(ctx context.Context, named []Argument, arguments ...Value) (Value, error) {
	bound := make([]Argument, 0, len(arguments)+len(named))
	for _, value := range arguments {
		bound = append(bound, Argument{Value: value})
	}
	bound = append(bound, named...)
	return c.invokeArguments(ctx, bound)
}

func (c *scriptClosure) invokeArguments(ctx context.Context, arguments []Argument) (result Value, resultErr error) {
	if c == nil || c.script == nil {
		return Null(), ErrScriptUnloaded
	}
	var release func() error
	var err error
	if !canReuseClosureExecution(ctx, c.script) {
		ctx, release, err = c.script.acquireExecution(ctx)
		if err != nil {
			return Null(), err
		}
	}
	defer func() { resultErr = joinExecutionError(resultErr, release) }()
	caller := currentFiber(ctx)
	profileFrame := caller.beginClosureProfileCall(c, arguments)
	c.mu.Lock()
	fiber := c.popSuspendedLocked()
	c.mu.Unlock()
	if fiber != nil {
		if err := fiber.resetArgumentsAt(ctx, arguments, c.function.Span); err != nil {
			caller.finishProfileCall(profileFrame, err)
			return Null(), err
		}
		result, _, err = c.runFiber(ctx, fiber)
		caller.finishProfileCall(profileFrame, err)
		return result, err
	}
	result, err = c.invokeFreshArguments(ctx, arguments)
	caller.finishProfileCall(profileFrame, err)
	return result, err
}

func (c *scriptClosure) invokeFresh(ctx context.Context, named []Argument, arguments ...Value) (result Value, resultErr error) {
	if c == nil || c.script == nil {
		return Null(), ErrScriptUnloaded
	}
	var release func() error
	if !canReuseClosureExecution(ctx, c.script) {
		var err error
		ctx, release, err = c.script.acquireExecution(ctx)
		if err != nil {
			return Null(), err
		}
	}
	defer func() { resultErr = joinExecutionError(resultErr, release) }()
	bound := make([]Argument, 0, len(arguments)+len(named))
	for _, value := range arguments {
		bound = append(bound, Argument{Value: value})
	}
	bound = append(bound, named...)
	return c.invokeFreshArguments(ctx, bound)
}

func (c *scriptClosure) invokeFreshArguments(ctx context.Context, arguments []Argument) (Value, error) {
	span := Span{}
	if c != nil && c.function != nil {
		span = c.function.Span
	}
	if err := c.ensureStateAt(ctx, span); err != nil {
		return Null(), err
	}
	c.mu.Lock()
	state := c.state
	c.mu.Unlock()
	frame, err := state.localChildAt(ctx, span)
	if err != nil {
		return Null(), err
	}
	fiber, err := newFiberAt(ctx, c, frame, arguments, span)
	if err != nil {
		return Null(), err
	}
	result, suspended, err := c.runFiber(ctx, fiber)
	if err != nil {
		return result, err
	}
	if !suspended && len(fiber.locals) != 0 {
		err = &localScopeLeakError{count: len(fiber.locals), closure: c}
	}
	return result, err
}

func (c *scriptClosure) runFiber(ctx context.Context, initial *fiber) (Value, bool, error) {
	if initial == nil {
		return Null(), false, errors.New("opfor: invalid execution fiber")
	}
	contexts := make([]*fiber, 0, 1+len(initial.continuationTail))
	contexts = append(contexts, initial)
	contexts = append(contexts, initial.continuationTail...)
	initial.continuationTail = nil

	// ScriptEnvironment.evaluateOldContext resumes the contexts saved by one
	// closure invocation in FIFO order. Dynamic contexts created during this
	// resume belong to the *next* saved group and are deliberately not consumed
	// by this loop.
	collector := &continuationCollector{}
	result := Null()
	for index, current := range contexts {
		if current == nil {
			return Null(), false, errors.New("opfor: invalid suspended execution context")
		}
		current.continuation = collector
		if current.serializedReturn {
			// A yielded inline call inside expr's synthetic `return (...)`
			// leaves a Context whose resume Step is Return, but Java does not
			// serialize the evaluator frame that held the call result. The
			// preceding saved Block has already reconstructed that result; this
			// marker performs the durable Return without replaying the call.
			current.serializedReturn = false
			current.flow = inlineFlowReturn
			c.pushSuspendedContextsWithState(collector.contexts, current)
			return result, len(collector.contexts) != 0, nil
		}
		var yielded bool
		var err error
		result, yielded, err = current.run(dynamicSourceResumeContext(ctx, current))
		if err != nil {
			if current.serializedMoreHandlers && index+1 < len(contexts) {
				next := contexts[index+1]
				if next != nil && next.catch(err) {
					next.adoptContinuationState(current)
					continue
				}
			}
			var control *loopControl
			if errors.As(err, &control) {
				// ClosureCallRequest and top-level SleepUtils.runCode clear an
				// unmatched break/continue request after the active Block ends.
				// Dynamic contexts already saved by this invocation remain owned
				// by the closure, just as they do for an explicit return.
				c.pushSuspendedContextsWithState(collector.contexts, current)
				return Null(), len(collector.contexts) != 0, nil
			}
			var transfer *callCCTransfer
			if errors.As(err, &transfer) {
				if transfer.source != c || transfer.fiber != current || transfer.target == nil {
					return Null(), false, err
				}
				// A directly resumed callcc is ordinary continuation transfer:
				// retain the current context followed by the untouched old tail,
				// then invoke the target with the owning closure as $1.
				collector.append(current)
				collector.append(contexts[index+1:]...)
				c.pushSuspendedContextsWithState(collector.contexts, current)
				result, err = transfer.target.invoke(
					ctx,
					[]Argument{{Name: "$0", Value: String("CALLCC")}},
					FunctionValue(c),
				)
				if transfer.standaloneTrace && current.callTraceEmissionEnabled() {
					current.writeCallTrace(formatCallCCTrace(transfer.source, transfer.target), result, err, transfer.span)
				}
				return result, c.isSuspended(current), err
			}
			// A throw or fatal evaluation error discards the untouched old tail.
			// Contexts produced by nested dynamic evaluation before that flow
			// change remain owned by the closure, as in SleepEnvironment.saveContext.
			c.pushSuspendedContextsWithState(collector.contexts, current)
			return result, len(collector.contexts) != 0, err
		}
		if yielded {
			collector.append(current)
			collector.append(contexts[index+1:]...)
			c.pushSuspendedContextsWithState(collector.contexts, current)
			return result, true, nil
		}
		if current.flow == inlineFlowReturn {
			// An explicit return terminates evaluateOldContext and discards its
			// untouched tail. Any new dynamic contexts already collected survive.
			c.pushSuspendedContextsWithState(collector.contexts, current)
			return result, len(collector.contexts) != 0, nil
		}
		if index+1 < len(contexts) {
			contexts[index+1].adoptContinuationState(current)
		}
	}
	if len(contexts) != 0 {
		c.pushSuspendedContextsWithState(collector.contexts, contexts[len(contexts)-1])
	}
	return result, len(collector.contexts) != 0, nil
}

func (c *scriptClosure) pushSuspended(suspended *fiber) {
	c.pushSuspendedContexts([]*fiber{suspended})
}

func (c *scriptClosure) pushSuspendedContextsWithState(contexts []*fiber, state *fiber) {
	for _, suspended := range contexts {
		if suspended != nil && state != nil {
			suspended.adoptContinuationState(state)
		}
	}
	c.pushSuspendedContexts(contexts)
}

func (c *scriptClosure) pushSuspendedContexts(contexts []*fiber) {
	if c == nil || len(contexts) == 0 || contexts[0] == nil {
		return
	}
	head := contexts[0]
	head.continuationTail = append(head.continuationTail[:0], contexts[1:]...)
	c.mu.Lock()
	c.suspended = append(c.suspended, head)
	c.mu.Unlock()
}

func (c *scriptClosure) popSuspendedLocked() *fiber {
	if c == nil || len(c.suspended) == 0 {
		return nil
	}
	last := len(c.suspended) - 1
	fiber := c.suspended[last]
	c.suspended[last] = nil
	c.suspended = c.suspended[:last]
	return fiber
}

func (c *scriptClosure) isSuspended(fiber *fiber) bool {
	if c == nil || fiber == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for index := len(c.suspended) - 1; index >= 0; index-- {
		head := c.suspended[index]
		if head == fiber {
			return true
		}
		for _, current := range head.continuationTail {
			if current == fiber {
				return true
			}
		}
	}
	return false
}

func (c *scriptClosure) variableCell(name string) *Cell {
	cell, _ := c.variableCellAt(context.Background(), name, Span{})
	return cell
}

func (c *scriptClosure) variableCellAt(ctx context.Context, name string, span Span) (*Cell, error) {
	if c == nil {
		return nil, nil
	}
	if err := c.ensureStateAt(ctx, span); err != nil {
		return nil, err
	}
	c.mu.Lock()
	state := c.state
	c.mu.Unlock()
	return state.localAt(ctx, name, span)
}
