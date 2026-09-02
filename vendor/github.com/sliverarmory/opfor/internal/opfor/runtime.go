package opfor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Option configures a Runtime. It is a closed functional-option type: external
// packages should use the exported With* constructors rather than attempting
// to construct custom options against OPFOR's private runtime configuration.
type Option func(*runtimeConfig) error

// RuntimeID is a nonzero, process-local Runtime identity. IDs are allocated
// monotonically and are useful for distinguishing provenance from parent and
// ScriptLoader child runtimes whose Script IDs may overlap. They are not
// stable across process restarts and must not be used as durable identifiers.
type RuntimeID uint64

var runtimeIdentityCounter atomic.Uint64

func nextRuntimeID() RuntimeID {
	for {
		if id := RuntimeID(runtimeIdentityCounter.Add(1)); id != 0 {
			return id
		}
	}
}

// IncludeCyclePolicy controls how include handles a source that is already in
// the active include chain.
type IncludeCyclePolicy uint8

const (
	// IncludeCycleReject stops recursion with ErrIncludeCycle. This is OPFOR's
	// deterministic resource-safe default.
	IncludeCycleReject IncludeCyclePolicy = iota
	// IncludeCycleAllow matches Sleep 2.1, which permits recursive includes
	// until execution is otherwise stopped (for example by an instruction
	// limit, context cancellation, or process resource exhaustion).
	IncludeCycleAllow
)

type runtimeConfig struct {
	stdin             io.Reader
	stdout            io.Writer
	stderr            io.Writer
	host              Host
	objectHost        ObjectHost
	loadableProvider  LoadableProvider
	observer          BindingObserver
	lifecycle         ScriptLifecycleObserver
	variableProvider  VariableProvider
	sourceResolver    SourceResolver
	sleepClasspath    string
	sleepClasspathSet bool
	initialGlobals    map[string]Value
	functions         map[string]NativeFunc
	environments      map[string]EnvironmentKind
	limits            Limits
	resources         *runtimeResourceAccount
	includeCycles     IncludeCyclePolicy
	clock             Clock
	scriptLoaderCache *ScriptLoaderCache
	extensionProfiles []runtimeExtensionProfile
	aggressorConfig
	debugFlags    int32
	taintMode     bool
	taintPolicies map[string]TaintPolicy
}

// Runtime is an isolated OPFOR execution environment. A Runtime owns globals,
// loaded scripts, callbacks, and host registrations. Its registries and value
// containers are safe for concurrent external dispatch. A suspended closure
// resumes at most once; independent callbacks may execute concurrently.
type Runtime struct {
	id RuntimeID

	stdin   io.Reader
	stdout  io.Writer
	stderr  io.Writer
	console *sleepIOHandle

	host             Host
	objectHost       ObjectHost
	loadableProvider LoadableProvider
	observer         BindingObserver
	lifecycle        ScriptLifecycleObserver
	variableProvider VariableProvider
	resolver         SourceResolver
	// defaultFileResolver is non-nil only when New installed OPFOR's default
	// filesystem resolver. Sleep resolves include paths against the script
	// instance's current directory, so chdir keeps this resolver in sync. An
	// importer-supplied resolver remains entirely importer-owned.
	defaultFileResolver *FileSourceResolver
	// scriptLoaderInstance is set only on the private child runtime owned by a
	// portable ScriptInstance. It lets include associate filesystem evidence
	// with that instance without exposing interpreter frames to SourceResolver.
	scriptLoaderInstance *portableScriptInstance
	// scriptLoaderSharedEnvironment is set on a ScriptLoader child to the exact
	// java.util.Hashtable backing its ScriptEnvironment. The table may be
	// caller-supplied and shared or the private table created for a null
	// environment argument.
	scriptLoaderSharedEnvironment *portableScriptSharedEnvironment

	mu             sync.RWMutex
	functions      map[string]NativeFunc
	stockFunctions map[string]NativeFunc
	// explicitFunctions records importer registrations separately from the
	// portable defaults installed into functions. Evaluator intrinsics such as
	// find and setf need this distinction so WithFunction/RegisterFunction can
	// shadow their AST-sensitive fallback implementations.
	explicitFunctions map[string]struct{}
	scripts           map[ScriptID]*Script
	lifecycleScripts  map[ScriptID]struct{}
	bindings          map[BindingKind]map[string][]Binding
	bindingOrder      map[BindingKind][]Binding
	initialGlobals    map[string]Value
	environments      map[string]EnvironmentKind
	nextScript        ScriptID
	nextProxy         uint64
	aggressorState
	closing           bool
	closed            bool
	closeDone         chan struct{}
	closeErr          error
	closeErrDelivered bool
	executionCtx      context.Context
	executionCancel   context.CancelFunc
	executions        uint64
	executionDone     chan struct{}
	printStream       *portableJavaPrintStream
	fixtureObjects    *portableFixtureObjectState
	socketState       *sleepSocketState
	readTasks         map[*sleepReadTask]struct{}
	processes         map[*processObject]struct{}
	regexCache        *sleepRegexCache
	scriptLoaderCache *ScriptLoaderCache
	maxInstructions   uint64
	limits            Limits
	resources         *runtimeResourceAccount
	includeCycles     IncludeCyclePolicy
	clock             Clock
	extensionProfiles []runtimeExtensionProfile
	debugFlags        int32
	taintMode         bool
	taintPolicies     map[string]TaintPolicy

	evalMu     sync.Mutex
	evalScript *Script
}

// ID returns this Runtime's nonzero process-local identity. A nil Runtime has
// ID zero.
func (r *Runtime) ID() RuntimeID {
	if r == nil {
		return 0
	}
	return r.id
}

// New creates an OPFOR runtime. Console output defaults to os.Stdout, warnings
// and diagnostics to os.Stderr, and input to os.Stdin. The portable Sleep
// standard library includes live filesystem and process functions; Host is an
// extension boundary, not a sandbox for those builtins.
func New(options ...Option) (*Runtime, error) {
	config := runtimeConfig{
		stdin:             os.Stdin,
		stdout:            os.Stdout,
		stderr:            os.Stderr,
		host:              unsupportedHost{},
		objectHost:        unsupportedObjectHost{},
		functions:         make(map[string]NativeFunc),
		environments:      make(map[string]EnvironmentKind),
		clock:             systemClock{},
		extensionProfiles: defaultRuntimeExtensionProfiles(),
		aggressorConfig:   defaultAggressorConfig(),
		debugFlags:        1,
		taintPolicies:     make(map[string]TaintPolicy),
	}
	for _, option := range options {
		if option == nil {
			continue
		}
		if err := option(&config); err != nil {
			return nil, err
		}
	}
	var defaultFileResolver *FileSourceResolver
	if config.sourceResolver == nil {
		resolver, err := NewFileSourceResolver("")
		if err != nil {
			return nil, err
		}
		if config.sleepClasspathSet {
			resolver.SetSleepClasspath(config.sleepClasspath)
		}
		config.sourceResolver = resolver
		defaultFileResolver = resolver
	}
	outputMutex := &sync.Mutex{}
	resources := config.resources
	if resources == nil {
		resources = newRuntimeResourceAccount(config.limits)
	}

	runtime := &Runtime{
		id:                  nextRuntimeID(),
		stdin:               config.stdin,
		stdout:              synchronizedWriter{mutex: outputMutex, writer: newRuntimeOutputWriter(resources, config.stdout)},
		stderr:              synchronizedWriter{mutex: outputMutex, writer: newRuntimeOutputWriter(resources, config.stderr)},
		host:                config.host,
		loadableProvider:    config.loadableProvider,
		observer:            config.observer,
		lifecycle:           config.lifecycle,
		variableProvider:    config.variableProvider,
		resolver:            config.sourceResolver,
		defaultFileResolver: defaultFileResolver,
		functions:           make(map[string]NativeFunc, len(config.functions)),
		stockFunctions:      make(map[string]NativeFunc),
		explicitFunctions:   make(map[string]struct{}, len(config.functions)),
		scripts:             make(map[ScriptID]*Script),
		lifecycleScripts:    make(map[ScriptID]struct{}),
		bindings:            make(map[BindingKind]map[string][]Binding),
		bindingOrder:        make(map[BindingKind][]Binding),
		initialGlobals:      cloneInitialGlobals(config.initialGlobals),
		environments:        make(map[string]EnvironmentKind, len(config.environments)),
		maxInstructions:     config.limits.MaxInstructionsPerExecution,
		limits:              config.limits,
		resources:           resources,
		includeCycles:       config.includeCycles,
		clock:               config.clock,
		scriptLoaderCache:   config.scriptLoaderCache,
		extensionProfiles:   cloneRuntimeExtensionProfiles(config.extensionProfiles),
		aggressorState:      newAggressorState(config.aggressorConfig),
		debugFlags:          config.debugFlags,
		taintMode:           config.taintMode,
		taintPolicies:       make(map[string]TaintPolicy, len(config.taintPolicies)),
		regexCache:          newSleepRegexCache(),
	}
	runtime.objectHost = defaultObjectHost{runtime: runtime, primary: config.objectHost}
	runtime.executionCtx, runtime.executionCancel = context.WithCancel(context.Background())
	runtime.executionDone = make(chan struct{})
	for name, function := range config.functions {
		runtime.functions[name] = function
		runtime.explicitFunctions[name] = struct{}{}
	}
	for keyword, kind := range config.environments {
		runtime.environments[keyword] = kind
	}
	for name, policy := range config.taintPolicies {
		runtime.taintPolicies[name] = policy
	}
	runtime.console = newIOHandle("console", runtime.stdin, runtime.stdout, false, false, true).withRuntimeOutputAccount(resources)
	runtime.socketState = newSleepSocketState(runtime)
	runtime.installCoreFunctions()
	runtime.installCoreTaintPolicies()
	return runtime, nil
}

// WithIncludeCyclePolicy selects safe cycle rejection or Sleep-compatible
// recursive includes. Importers that enable IncludeCycleAllow should also set
// an instruction limit or cancelable execution context.
func WithIncludeCyclePolicy(policy IncludeCyclePolicy) Option {
	return func(config *runtimeConfig) error {
		switch policy {
		case IncludeCycleReject, IncludeCycleAllow:
			config.includeCycles = policy
			return nil
		default:
			return fmt.Errorf("opfor: invalid include cycle policy %d", policy)
		}
	}
}

// WithScriptLoaderCache enables the explicit sharing capability used when a
// Sleep script calls ScriptLoader.setGlobalCache(true). The same cache may be
// supplied to multiple runtimes; without this option, enabling the upstream
// process-global cache remains an explicit unsupported operation.
func WithScriptLoaderCache(cache *ScriptLoaderCache) Option {
	return func(config *runtimeConfig) error {
		if cache == nil {
			return errors.New("opfor: ScriptLoader cache is nil")
		}
		config.scriptLoaderCache = cache
		return nil
	}
}

// Clock supplies wall-clock time to portable time/date functions. A configured
// Clock is Runtime-owned, is shared with source-backed ScriptLoader children,
// and may be called concurrently; implementations must provide any required
// synchronization.
type Clock interface {
	Now() time.Time
}

// ClockFunc adapts a function to Clock.
type ClockFunc func() time.Time

// Now calls function and returns its time.
func (function ClockFunc) Now() time.Time { return function() }

type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// WithClock replaces the wall clock used by portable time and date functions,
// including ticks, formatDate, parseDate, dstamp, and tstamp. Timestamp
// formatting uses the location returned by Clock.Now.
func WithClock(clock Clock) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(clock) {
			return errors.New("opfor: clock is nil")
		}
		config.clock = clock
		return nil
	}
}

// WithLimits replaces the complete resource-limit configuration. Zero fields
// are unlimited. Options are applied in order; a later WithInstructionLimit
// changes only MaxInstructionsPerExecution.
func WithLimits(limits Limits) Option {
	return func(config *runtimeConfig) error {
		config.limits = limits
		config.resources = nil
		return nil
	}
}

// WithInstructionLimit bounds the number of VM instructions consumed by one
// top-level execution or callback, including nested Sleep function calls. A
// limit of zero disables the quota. It is compatibility sugar for changing
// Limits.MaxInstructionsPerExecution without modifying other configured
// limits. Context cancellation remains independent.
func WithInstructionLimit(limit uint64) Option {
	return func(config *runtimeConfig) error {
		config.limits.MaxInstructionsPerExecution = limit
		return nil
	}
}

func withRuntimeResourceAccount(account *runtimeResourceAccount) Option {
	return func(config *runtimeConfig) error {
		if account == nil {
			return errors.New("opfor: runtime resource account is nil")
		}
		config.resources = account
		config.limits = account.limits
		return nil
	}
}

// WithDebugFlags sets the initial Sleep debug bitmask for each newly loaded
// script. The reference runtime defaults to 1; scripts may subsequently change
// their own flags with debug(...). Fork children inherit their parent's current
// value rather than this default.
func WithDebugFlags(flags int32) Option {
	return func(config *runtimeConfig) error {
		config.debugFlags = flags
		return nil
	}
}

type synchronizedWriter struct {
	mutex  *sync.Mutex
	writer io.Writer
}

func (writer synchronizedWriter) Write(data []byte) (int, error) {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	return writer.writer.Write(data)
}

func (writer synchronizedWriter) Flush() error {
	writer.mutex.Lock()
	defer writer.mutex.Unlock()
	if flusher, ok := writer.writer.(interface{ Flush() error }); ok {
		return flusher.Flush()
	}
	return nil
}

// WithStdin replaces the input used by the portable console handle and as the
// standard input of commands started by Sleep process functions. OPFOR never
// closes this borrowed reader. A reader may additionally implement
// ReadContext(context.Context, []byte) (int, error) so Close and Script.Unload
// can cancel an in-flight asynchronous Sleep read. Without that method, an
// already-blocked Read cannot be interrupted in pure Go: teardown returns
// ErrReadCancellationUnsupported, and the pending Read may later consume and
// discard one result before its goroutine actually exits.
func WithStdin(reader io.Reader) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(reader) {
			return errors.New("opfor: stdin reader is nil")
		}
		config.stdin = reader
		return nil
	}
}

// WithStdout replaces the default destination used by print, println, and the
// portable console handle. Explicit Sleep I/O handles retain their own writer.
func WithStdout(writer io.Writer) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(writer) {
			return errors.New("opfor: stdout writer is nil")
		}
		config.stdout = writer
		return nil
	}
}

// WithStderr replaces the destination used by warn, compatibility warnings,
// runtime diagnostics, and call tracing.
func WithStderr(writer io.Writer) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(writer) {
			return errors.New("opfor: stderr writer is nil")
		}
		config.stderr = writer
		return nil
	}
}

// WithHost installs the fallback for function calls that are not resolved to
// an importer-defined or portable native function. Aggressor implementations
// normally use this boundary for Cobalt-specific functions and predicates.
func WithHost(host Host) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(host) {
			return errors.New("opfor: host is nil")
		}
		config.host = host
		return nil
	}
}

// WithObjectHost installs the importer boundary for Java-style object syntax.
// The importer receives first refusal; when it returns UnsupportedError, OPFOR
// may handle its small portable java.lang scalar subset.
func WithObjectHost(host ObjectHost) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(host) {
			return errors.New("opfor: object host is nil")
		}
		config.objectHost = host
		return nil
	}
}

// WithBindingObserver installs lifecycle notifications for declarations such
// as events, hooks, commands, aliases, and menus. Runtime registries remain
// authoritative when no observer is configured.
func WithBindingObserver(observer BindingObserver) Option {
	return func(config *runtimeConfig) error {
		if isNilInterface(observer) {
			return errors.New("opfor: binding observer is nil")
		}
		config.observer = observer
		return nil
	}
}

// WithEnvironment registers an importer-defined Sleep environment keyword and
// bridge form. Use Runtime.Compile or Runtime.CompileString so the registration
// is visible while parsing predicate and filter declarations.
func WithEnvironment(keyword string, kind EnvironmentKind) Option {
	return func(config *runtimeConfig) error {
		normalized, err := normalizeEnvironment(keyword, kind)
		if err != nil {
			return err
		}
		if config.environments == nil {
			config.environments = make(map[string]EnvironmentKind)
		}
		config.environments[normalized] = kind
		return nil
	}
}

// WithFunction installs or replaces a Go-native function while constructing a
// Runtime. A leading ampersand is accepted. Importer functions take precedence
// over portable defaults with the same normalized name.
func WithFunction(name string, function NativeFunc) Option {
	return func(config *runtimeConfig) error {
		normalized, err := normalizeFunctionName(name)
		if err != nil {
			return err
		}
		if function == nil {
			return fmt.Errorf("opfor: function %q is nil", name)
		}
		config.functions[normalized] = function
		return nil
	}
}

// RegisterFunction installs or replaces a Go-native function.
func (r *Runtime) RegisterFunction(name string, function NativeFunc) error {
	if r == nil {
		return errors.New("opfor: runtime is nil")
	}
	normalized, err := normalizeFunctionName(name)
	if err != nil {
		return err
	}
	if function == nil {
		return fmt.Errorf("opfor: function %q is nil", name)
	}
	r.mu.Lock()
	r.functions[normalized] = function
	r.explicitFunctions[normalized] = struct{}{}
	r.mu.Unlock()
	return nil
}

func (r *Runtime) hasExplicitFunction(name string) bool {
	if r == nil {
		return false
	}
	name = strings.TrimPrefix(name, "&")
	r.mu.RLock()
	_, exists := r.explicitFunctions[name]
	r.mu.RUnlock()
	return exists
}

func (r *Runtime) hasRegisteredFunction(name string) bool {
	if r == nil {
		return false
	}
	name = strings.TrimPrefix(name, "&")
	r.mu.RLock()
	_, exists := r.functions[name]
	r.mu.RUnlock()
	return exists
}

// DefaultFunctionNames returns the native function names installed by New
// before importer overrides or additions. It derives the names from the same
// registration tables as New without constructing a live Runtime or consulting
// process state. The inventory includes native importer-boundary wrappers as
// well as functions that complete entirely in the portable runtime.
func DefaultFunctionNames() []string {
	runtime := &Runtime{}
	functions := runtime.coreFunctions(ioFunctionsForState(runtime, &ioBuiltinState{runtime: runtime}))
	names := make([]string, 0, len(functions))
	for name := range functions {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// FunctionNames returns the registered native function names in lexical order.
// The snapshot includes portable defaults and any importer overrides or
// additions installed on this runtime.
func (r *Runtime) FunctionNames() []string {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	names := make([]string, 0, len(r.functions))
	for name := range r.functions {
		names = append(names, name)
	}
	r.mu.RUnlock()
	sort.Strings(names)
	return names
}

// Invoke calls a registered native function or delegates the call to Host.
// Script code uses the same resolution path.
func (r *Runtime) Invoke(ctx context.Context, name string, values ...Value) (Value, error) {
	executionCtx, release, err := r.acquireRuntimeExecution(ctx)
	if err != nil {
		return Null(), err
	}
	// Keep importer and native panics transparent while ensuring Runtime.Close
	// never waits on a lease whose callback unwound the stack. The explicit
	// release below remains authoritative for ordinary error composition;
	// release is idempotent, so this deferred safety call is otherwise a no-op.
	defer release()
	arguments := make([]Argument, len(values))
	for index, value := range values {
		arguments[index] = Argument{Value: value}
	}
	value, err := r.invoke(executionCtx, Invocation{Runtime: r, Name: name, Arguments: arguments})
	err = joinExecutionContextError(executionCtx, err)
	return value, errors.Join(err, release())
}

// nativeDispatchState identifies an active stock-native call. The active bit
// makes contexts retained by importer code harmless after their call returns.
type nativeDispatchState struct {
	parent *nativeDispatchState
	active atomic.Bool
}

type nativeDispatchStateContextKey struct{}

// nativeBoundaryError preserves the origin of one of the three errors which
// Runtime.invoke otherwise translates for native container compatibility. It
// crosses nested stock NativeFuncs without making authority sticky for later,
// unrelated errors. Unwrap keeps errors.Is/errors.As transparent.
type nativeBoundaryError struct {
	cause  error
	origin *nativeDispatchState
}

func (err *nativeBoundaryError) Error() string { return err.cause.Error() }
func (err *nativeBoundaryError) Unwrap() error { return err.cause }

func preserveNativeBoundaryError(ctx context.Context, err error) error {
	if err == nil || !isNativeTranslatedError(err) {
		return err
	}
	if ctx == nil {
		return err
	}
	state, _ := ctx.Value(nativeDispatchStateContextKey{}).(*nativeDispatchState)
	if state == nil || !state.active.Load() {
		return err
	}
	if hasNativeBoundaryErrorForDispatch(err, state) {
		return err
	}
	return &nativeBoundaryError{cause: err, origin: state}
}

// hasNativeBoundaryErrorForDispatch reports whether any branch of err was
// created by an importer boundary nested beneath dispatch. Walking every
// errors.Join branch is important: errors.As stops at the first matching
// marker, which may be a stale marker retained from an unrelated invocation.
func hasNativeBoundaryErrorForDispatch(err error, dispatch *nativeDispatchState) bool {
	if err == nil || dispatch == nil || !dispatch.active.Load() {
		return false
	}
	pending := []error{err}
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if current == nil {
			continue
		}
		if preserved, ok := current.(*nativeBoundaryError); ok && nativeDispatchDescendsFrom(preserved.origin, dispatch) {
			return true
		}
		switch wrapped := current.(type) {
		case interface{ Unwrap() []error }:
			pending = append(pending, wrapped.Unwrap()...)
		case interface{ Unwrap() error }:
			pending = append(pending, wrapped.Unwrap())
		}
	}
	return false
}

func nativeDispatchDescendsFrom(origin, ancestor *nativeDispatchState) bool {
	if origin == nil || ancestor == nil {
		return false
	}
	for current := origin; current != nil; current = current.parent {
		if current == ancestor {
			return true
		}
	}
	return false
}

func isNativeTranslatedError(err error) bool {
	return errors.Is(err, ErrUnsafeArrayView) || errors.Is(err, ErrReadOnlyArray) || errors.Is(err, ErrReadOnlyHash)
}

func (r *Runtime) invoke(ctx context.Context, invocation Invocation) (result Value, resultErr error) {
	if r == nil {
		return Null(), errors.New("opfor: runtime is nil")
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	normalized, err := normalizeFunctionName(invocation.Name)
	if err != nil {
		return Null(), err
	}
	invocation.Name = normalized
	invocation.Runtime = r
	if generation := stampInvocationGeneration(ctx, r, &invocation); generation != nil {
		var release func() error
		ctx, release, err = generation.script.acquireGenerationExecution(ctx, generation)
		if err != nil {
			return Null(), err
		}
		defer func() { resultErr = joinExecutionError(resultErr, release) }()
	}
	caller := currentFiber(ctx)
	var profileFrame *profileCallFrame
	if normalized != "@" && normalized != "%" && normalized != "warn" {
		profileFrame = caller.beginProfileCall("&" + normalized)
	}
	defer func() { caller.finishProfileCall(profileFrame, resultErr) }()

	r.mu.RLock()
	function := r.functions[normalized]
	policy := r.taintPolicies[normalized]
	_, explicitFunction := r.explicitFunctions[normalized]
	r.mu.RUnlock()
	if invocation.forcedNative != nil {
		function = invocation.forcedNative
		explicitFunction = false
		if invocation.forcedNativePolicy {
			policy = invocation.forcedTaintPolicy
		}
	}
	if function != nil && !explicitFunction {
		invocation.Arguments = sleepBridgeArguments(invocation.Arguments)
	}
	parentDispatch, _ := ctx.Value(nativeDispatchStateContextKey{}).(*nativeDispatchState)
	if parentDispatch != nil && !parentDispatch.active.Load() {
		parentDispatch = nil
	}
	dispatch := &nativeDispatchState{parent: parentDispatch}
	dispatch.active.Store(true)
	defer dispatch.active.Store(false)
	ctx = context.WithValue(ctx, nativeDispatchStateContextKey{}, dispatch)
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	if err := r.outputLimitError(); err != nil {
		return Null(), err
	}
	call := func() (Value, error) {
		// Taint policy evaluation and another concurrent execution may run
		// between the outer admission check and the actual native/importer
		// dispatch. Recheck at the last safe point before side effects.
		if executionErr := executionContextError(ctx); executionErr != nil {
			return Null(), executionErr
		}
		if outputErr := r.outputLimitError(); outputErr != nil {
			return Null(), outputErr
		}
		if function != nil {
			return function(ctx, invocation)
		}
		value, err := r.host.Call(ctx, invocation)
		return value, preserveNativeBoundaryError(ctx, err)
	}
	value, err := r.applyTaintPolicy(ctx, normalized, policy, invocation.Arguments, invocation.Span, call)
	if hasNativeBoundaryErrorForDispatch(err, dispatch) {
		if dispatch.parent == nil || !dispatch.parent.active.Load() {
			if preserved, ok := err.(*nativeBoundaryError); ok && nativeDispatchDescendsFrom(preserved.origin, dispatch) {
				return value, preserved.cause
			}
		}
		return value, err
	}
	if function != nil {
		if errors.Is(err, ErrUnsafeArrayView) {
			r.writeWarning(unsafeArrayViewWarning, invocation.Span)
			return Null(), nil
		}
		if errors.Is(err, ErrReadOnlyArray) {
			if currentFiber(ctx) != nil {
				return Null(), &uncaughtScriptWarning{err: errors.New(readOnlyArrayWarning)}
			}
			return Null(), err
		}
		if errors.Is(err, ErrReadOnlyHash) {
			if currentFiber(ctx) != nil {
				return Null(), &uncaughtScriptWarning{err: errors.New(readOnlyHashWarning)}
			}
			return Null(), err
		}
		return value, err
	}
	return value, err
}

func normalizeFunctionName(name string) (string, error) {
	name = strings.TrimSpace(name)
	name = strings.TrimPrefix(name, "&")
	if name == "" {
		return "", errors.New("opfor: function name is empty")
	}
	if strings.ContainsAny(name, " \t\r\n(){}[];,:\"'`") {
		return "", fmt.Errorf("opfor: invalid function name %q", name)
	}
	return name, nil
}
