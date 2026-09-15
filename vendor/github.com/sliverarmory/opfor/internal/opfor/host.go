package opfor

import (
	"context"
	"errors"
)

// ScriptID uniquely identifies one loaded script within a Runtime.
type ScriptID uint64

// argumentSyntax records how a script argument acquired its public Name.
// Pair expressions and explicit pass-by-name references both look named at
// importer boundaries and both become KeyValuePair objects for stock Sleep
// bridges. Keeping their origins distinct is still necessary because only the
// explicit form grants pass-by-name semantics independent of its value syntax.
type argumentSyntax uint8

const (
	argumentSyntaxNone argumentSyntax = iota
	argumentSyntaxPair
	argumentSyntaxReference
)

// Argument preserves the distinctions between ordinary, named, and
// pass-by-name Sleep arguments. When Reference is non-nil (including an
// ordinary bare variable argument), Resolve observes the referenced cell's
// current value and Set mutates the caller's scalar.
//
// Reference is an intentionally low-level capability for trusted importer
// code. Retaining it after the Host call also retains direct mutation access;
// unlike Invocation.Callback, a raw *Cell is not revoked when its Script
// unloads. Importers which do not need pass-by-name mutation should snapshot
// with Resolve instead of retaining Reference.
type Argument struct {
	Name      string
	Value     Value
	Reference *Cell

	syntax             argumentSyntax
	syntaxName         string
	syntaxCell         *Cell
	bridgeMaterialized bool
	generation         *scriptGeneration
}

// Resolve returns the argument's current value.
func (a Argument) Resolve() Value {
	if a.Reference != nil {
		return a.Reference.Get()
	}
	return a.Value
}

// Set mutates a referenced argument and reports whether mutation was possible.
// Bare variable arguments and explicit pass-by-name arguments are referenced;
// expression temporaries are not. Set has the same trusted-capability lifetime
// as Reference and does not add a Script lifecycle check.
func (a Argument) Set(value Value) bool {
	if a.Reference == nil {
		return false
	}
	a.Reference.Set(value)
	return true
}

// Invocation describes one script-to-host or script-to-native function call.
// Runtime is the trusted originating Runtime and Arguments may contain live
// pass-by-name Cell capabilities as documented by Argument. Retaining either
// after the call therefore retains the corresponding capability; use Values
// and Callback when a detached top-level value snapshot or lifecycle-bound
// function is sufficient. Compound Values retain their ordinary reference
// identity.
type Invocation struct {
	Runtime   *Runtime
	Script    ScriptID
	Name      string
	Arguments []Argument
	Span      Span

	// forcedNative is used only by source-backed stock Sleep bridge handles.
	// Function.evaluate receives the environment key used for the call, but the
	// Function object itself remains authoritative when setf installs it over a
	// different native/importer entry. Keeping the override private prevents host
	// callers from bypassing ordinary Runtime resolution.
	forcedNative       NativeFunc
	forcedTaintPolicy  TaintPolicy
	forcedNativePolicy bool
	bridgeDispatchName string
	generation         *scriptGeneration
}

// Arg returns the zero-based argument value, or $null when index is absent.
func (i Invocation) Arg(index int) Value {
	if index < 0 || index >= len(i.Arguments) {
		return Null()
	}
	return i.Arguments[index].Resolve()
}

// Values snapshots the invocation's current argument values.
func (i Invocation) Values() []Value {
	values := make([]Value, len(i.Arguments))
	for index, argument := range i.Arguments {
		values[index] = argument.Resolve()
	}
	return values
}

// Callback snapshots a function-valued argument as a callable capability tied
// to the invocation's owning script. The returned Callable may be retained and
// invoked after the Host or NativeFunc call returns. It rejects calls after
// the script unloads and passes the caller's context through to script code.
//
// Callback returns ErrInvalidCallable when index is absent or does not contain
// a function, ErrScriptUnloaded when the invocation has no active owner, and
// an error when the Invocation did not originate from a Runtime.
func (i Invocation) Callback(index int) (Callable, error) {
	return i.RetainCallback(i.Arg(index))
}

// RetainCallback snapshots value as a callable capability tied to the
// invocation's owning script. It is useful when an importer has already
// resolved a live reference and needs its observed value and callback identity
// to describe the same instant.
func (i Invocation) RetainCallback(value Value) (Callable, error) {
	return retainInvocationCallback(i.Runtime, i.Script, i.generationToken(), value)
}

// Bindings returns an opaque, generation-bound capability for dispatching the
// registrations visible to this importer invocation. A retained capability is
// revoked by logical ScriptLoader unload even though the underlying Script and
// its raw closures remain active.
func (i Invocation) Bindings() AggressorBindings {
	return aggressorBindingsForInvocation(i)
}

func (i Invocation) generationToken() *scriptGeneration {
	if i.generation != nil {
		return i.generation
	}
	return scriptGenerationForInvocation(nil, i.Runtime, i.Script)
}

func retainInvocationCallback(
	runtime *Runtime,
	script ScriptID,
	generation *scriptGeneration,
	value Value,
) (Callable, error) {
	callable, ok := value.Function()
	if !ok {
		return nil, ErrInvalidCallable
	}
	if runtime == nil {
		return nil, errors.New("opfor: invocation has no originating runtime")
	}

	owner := (*Script)(nil)
	if generation != nil {
		owner = generation.script
	}
	if owner == nil {
		runtime.mu.RLock()
		owner = runtime.scripts[script]
		runtime.mu.RUnlock()
		if owner != nil {
			generation = owner.currentScriptGeneration()
		}
	}
	if owner == nil || owner.runtime != runtime || owner.id != script || !owner.generationAdmissible(generation) {
		return nil, ErrScriptUnloaded
	}
	return &invocationCallback{owner: owner, generation: generation, callable: callable}, nil
}

// invocationCallback is deliberately private so importers cannot detach a
// retained callback from the script lifetime established by Invocation.
type invocationCallback struct {
	owner      *Script
	generation *scriptGeneration
	callable   Callable
}

func (callback *invocationCallback) Invoke(ctx context.Context, arguments ...Value) (Value, error) {
	return callback.invokeNamed(ctx, "", arguments...)
}

// invokeNamed preserves SleepClosure.callClosure's synthetic $0 message for
// bridges whose callbacks have a defined invocation name (for example the
// socket bridge's "&callback"). Ordinary importer-retained callbacks continue
// to use Invoke and therefore have no synthetic message.
func (callback *invocationCallback) invokeNamed(ctx context.Context, name string, arguments ...Value) (result Value, resultErr error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	if callback == nil || callback.owner == nil {
		return Null(), ErrScriptUnloaded
	}
	if callback.callable == nil {
		return Null(), ErrInvalidCallable
	}
	ctx, release, err := callback.owner.acquireGenerationExecution(ctx, callback.generation)
	if err != nil {
		return Null(), err
	}
	defer func() { resultErr = joinExecutionError(resultErr, release) }()
	if name != "" {
		if closure, ok := callback.callable.(*scriptClosure); ok {
			result, resultErr = closure.invoke(ctx, []Argument{{Name: "$0", Value: String(name)}}, arguments...)
		} else {
			result, resultErr = callback.callable.Invoke(ctx, arguments...)
		}
	} else {
		result, resultErr = callback.callable.Invoke(ctx, arguments...)
	}
	resultErr = joinExecutionContextError(ctx, resultErr)
	return result, resultErr
}

// NativeFunc implements a Sleep function in Go. Calls are synchronous, may
// occur concurrently for independent script executions, and receive the
// active execution context. Implementations should observe ctx and must not
// retain it after returning; retain an Invocation callback and invoke it with a
// new caller-owned context for asynchronous work. Returned compound Values are
// shared with script code rather than cloned. Returned errors propagate unless
// a directly returned ErrUnsafeArrayView, ErrReadOnlyArray, or ErrReadOnlyHash
// is translated into Sleep's corresponding container warning behavior.
type NativeFunc func(context.Context, Invocation) (Value, error)

// Host receives calls for Aggressor functions that are not implemented by the
// portable runtime. It is the primary importer boundary for Cobalt, UI,
// data-model, session-tasking, payload, network, and other application-defined
// effects. Installed portable functions, including file and process builtins,
// do not cross this boundary.
//
// Call is synchronous and may run concurrently for independent executions.
// Implementations should observe ctx and must not retain it after returning.
// An Invocation may be retained subject to its documented trusted-capability
// ownership. OPFOR transfers a successful returned Value directly to script
// code, so compound results remain shared with the importer. A returned error
// is authoritative and is not retried through another Host.
type Host interface {
	Call(context.Context, Invocation) (Value, error)
}

// HostFunc adapts a function to Host.
type HostFunc func(context.Context, Invocation) (Value, error)

// Call invokes f with the supplied host invocation.
func (f HostFunc) Call(ctx context.Context, invocation Invocation) (Value, error) {
	if f == nil {
		return Null(), errors.New("opfor: host function is nil")
	}
	return f(ctx, invocation)
}

type unsupportedHost struct{}

func (unsupportedHost) Call(_ context.Context, invocation Invocation) (Value, error) {
	return Null(), &UnsupportedError{Operation: "host function", Name: invocation.Name, Span: invocation.Span}
}

// BindingKind identifies an Aggressor/Sleep environment registration.
type BindingKind string

const (
	// BindingSub identifies a named sub declaration.
	BindingSub BindingKind = "sub"
	// BindingInline identifies a named inline declaration.
	BindingInline BindingKind = "inline"
	// BindingEvent identifies an on event registration.
	BindingEvent BindingKind = "on"
	// BindingCommand identifies a script-console command registration.
	BindingCommand BindingKind = "command"
	// BindingAlias identifies a Beacon console alias registration.
	BindingAlias BindingKind = "alias"
	// BindingSSHAlias identifies an SSH console alias registration.
	BindingSSHAlias BindingKind = "ssh_alias"
	// BindingHook identifies a set hook registration.
	BindingHook BindingKind = "set"
	// BindingPopup identifies a popup hook registration.
	BindingPopup BindingKind = "popup"
	// BindingMenu identifies a menu or menubar registration.
	BindingMenu BindingKind = "menu"
	// BindingItem identifies a menu item registration.
	BindingItem BindingKind = "item"
	// BindingKey identifies a key binding registration.
	BindingKey BindingKind = "bind"
)

// BindingLifetime controls how long an event registration remains active.
// The zero value is persistent so existing importers and bindings retain their
// previous lifecycle unless they explicitly opt in to one-shot behavior.
type BindingLifetime uint8

const (
	// BindingPersistent remains registered until its owning script unloads or
	// another runtime operation explicitly removes it.
	BindingPersistent BindingLifetime = iota
	// BindingOnce is atomically consumed by the first matching event dispatch.
	// Consumption happens before any callbacks selected by that dispatch run.
	BindingOnce
)

// EnvironmentKind selects one of Sleep's three environment bridge ABIs.
// Importer-defined predicate environments must be registered before
// compilation because their parenthesized condition would otherwise parse as
// an ordinary function call followed by a block.
type EnvironmentKind uint8

const (
	// EnvironmentOrdinary evaluates a declaration's selector when it binds.
	EnvironmentOrdinary EnvironmentKind = iota
	// EnvironmentFilter preserves an identifier and one raw, unevaluated
	// parameter for the importer.
	EnvironmentFilter
	// EnvironmentPredicate compiles a parenthesized condition into a reusable
	// predicate evaluator.
	EnvironmentPredicate
)

// BindingSelector retains one declaration selector in source order. Raw is
// the exact source spelling. Evaluated reports whether Value was evaluated by
// Sleep at registration time; filter parameters and predicate conditions are
// deliberately not evaluated while binding.
type BindingSelector struct {
	Raw       string
	Value     Value
	Evaluated bool
	Span      Span
}

// PredicateEvaluator is a script-owned compiled predicate. Arguments become
// the predicate evaluation's $1... values and @_. Evaluators reject calls once
// their owning script unloads.
type PredicateEvaluator interface {
	Evaluate(context.Context, ...Value) (bool, error)
}

// PredicateEvaluatorFunc adapts a function to PredicateEvaluator.
type PredicateEvaluatorFunc func(context.Context, ...Value) (bool, error)

// Evaluate invokes f.
func (f PredicateEvaluatorFunc) Evaluate(ctx context.Context, values ...Value) (bool, error) {
	if f == nil {
		return false, errors.New("opfor: predicate evaluator function is nil")
	}
	return f(ctx, values...)
}

// BindingInvocation describes the parent composition invocation in which a
// nested popup, menu, or item was registered. Parent forms a snapshot chain
// for nested menus; Arguments are copied when the invocation begins.
type BindingInvocation struct {
	BindingID uint64
	Kind      BindingKind
	Keyword   string
	Name      string
	Script    ScriptID
	Arguments []Value
	Parent    *BindingInvocation
}

// Binding describes a script-owned function, event, command, alias, hook, or
// UI registration. Callback becomes invalid when the owning script unloads.
type Binding struct {
	ID          uint64
	Kind        BindingKind
	Keyword     string
	Lifetime    BindingLifetime
	Environment EnvironmentKind
	Name        string
	Script      ScriptID
	Span        Span
	Selectors   []BindingSelector
	Filter      string
	Predicate   PredicateEvaluator
	Parent      *BindingInvocation
	Callback    Callable
}

// BindingObserver optionally receives registration lifecycle notifications.
// Notifications are synchronous but may occur concurrently for independent
// script executions, so implementations must provide their own
// synchronization, should observe ctx, and must not retain ctx after returning.
// A Runtime's internal registries remain authoritative even when no observer
// is configured.
//
// Registered runs after the binding is published. Returning an error removes
// that binding and rejects its declaration; OPFOR does not send a compensating
// Unregistered notification for that rollback. Unregistered runs after the
// authoritative removal. Its error is reported by the operation performing
// teardown but cannot restore the binding.
//
// Binding slices and parent metadata are detached snapshots which may be
// retained or mutated by the observer. Compound Values preserve reference
// identity, while Callback and Predicate remain script-owned capabilities and
// reject invocation after their owning execution generation retires or their
// Script unloads.
type BindingObserver interface {
	Registered(context.Context, Binding) error
	Unregistered(context.Context, Binding) error
}

// ObjectOperation identifies a Sleep bracket-object request.
type ObjectOperation uint8

const (
	// ObjectConstruct requests construction of Class.
	ObjectConstruct ObjectOperation = iota
	// ObjectInvoke requests a method or static method invocation.
	ObjectInvoke
	// ObjectGet requests a property or static field read.
	ObjectGet
	// ObjectSet requests a property or static field write.
	ObjectSet
	// ObjectTypeCheck requests an isa-style type test.
	ObjectTypeCheck
)

// ObjectInvocation describes Java-style syntax delegated to an importer. The
// target is either an opaque object value or a class name, depending on Op.
// Runtime and any Argument.Reference values are trusted capabilities with the
// same retention rules as Invocation; Values and Callback provide the
// corresponding snapshot and lifecycle-bound helpers. Target and compound
// argument Values preserve reference identity.
type ObjectInvocation struct {
	Runtime   *Runtime
	Script    ScriptID
	Op        ObjectOperation
	Class     string
	Target    Value
	Message   string
	Arguments []Argument
	Span      Span
}

// Arg returns the zero-based object-call argument, or $null when absent.
func (i ObjectInvocation) Arg(index int) Value {
	if index < 0 || index >= len(i.Arguments) {
		return Null()
	}
	return i.Arguments[index].Resolve()
}

// Values snapshots the object call's current argument values.
func (i ObjectInvocation) Values() []Value {
	values := make([]Value, len(i.Arguments))
	for index, argument := range i.Arguments {
		values[index] = argument.Resolve()
	}
	return values
}

// Callback snapshots a function-valued object-call argument as a callable
// capability tied to the invocation's owning script. The returned Callable may
// be retained after ObjectHost.Object returns and observes the same context and
// script-lifetime rules as Invocation.Callback.
func (i ObjectInvocation) Callback(index int) (Callable, error) {
	if index < 0 || index >= len(i.Arguments) {
		return nil, ErrInvalidCallable
	}
	argument := i.Arguments[index]
	generation := argument.generation
	if generation == nil {
		generation = i.generationToken()
	}
	return retainInvocationCallback(i.Runtime, i.Script, generation, argument.Resolve())
}

// RetainCallback snapshots value as an owner-bound callable capability. It is
// the object-call counterpart to Invocation.RetainCallback.
func (i ObjectInvocation) RetainCallback(value Value) (Callable, error) {
	return retainInvocationCallback(i.Runtime, i.Script, i.generationToken(), value)
}

// generationToken recovers the hidden token stamped into object arguments at
// the dispatch boundary. Keeping this metadata on Argument preserves
// ObjectInvocation's all-exported field layout for external unkeyed literals.
func (i ObjectInvocation) generationToken() *scriptGeneration {
	for _, argument := range i.Arguments {
		if argument.generation != nil {
			return argument.generation
		}
	}
	return scriptGenerationForInvocation(nil, i.Runtime, i.Script)
}

// ObjectHost implements Java-style construction, method/property dispatch,
// and type checks for pure-Go embeddings. Object calls are synchronous and may
// occur concurrently. Implementations should observe ctx and must not retain
// it after returning. Returned compound Values remain shared with script code.
// A returned UnsupportedError explicitly declines the request and permits
// OPFOR's portable object fallback; every other error is authoritative.
type ObjectHost interface {
	Object(context.Context, ObjectInvocation) (Value, error)
}

// ObjectHostFunc adapts a function to ObjectHost.
type ObjectHostFunc func(context.Context, ObjectInvocation) (Value, error)

// Object invokes f with the supplied object request.
func (f ObjectHostFunc) Object(ctx context.Context, invocation ObjectInvocation) (Value, error) {
	if f == nil {
		return Null(), errors.New("opfor: object host function is nil")
	}
	return f(ctx, invocation)
}

// Iterator adapts an importer-owned opaque object to Sleep's java.util.Iterator
// boundary. Next returns the next scalar and whether one was present. Returning
// present=false exhausts the iterator. The runtime supplies the active
// execution context so blocking implementations remain cancelable.
//
// Values implementing Iterator may be wrapped with ObjectValue and used by
// foreach as well as iterator-consuming builtins such as map, reduce, and sum.
type Iterator interface {
	Next(context.Context) (value Value, present bool, err error)
}

// MutableIterator is an Iterator whose current element can be removed by
// Sleep's zero-argument remove() while a foreach loop is active. Remove must
// follow the same state rules as java.util.Iterator.remove.
type MutableIterator interface {
	Iterator
	Remove(context.Context) error
}

// IteratorFunc adapts a function to a read-only Iterator.
type IteratorFunc func(context.Context) (value Value, present bool, err error)

// Next invokes f.
func (f IteratorFunc) Next(ctx context.Context) (Value, bool, error) {
	if f == nil {
		return Null(), false, errors.New("opfor: iterator function is nil")
	}
	return f(ctx)
}

type unsupportedObjectHost struct{}

func (unsupportedObjectHost) Object(_ context.Context, invocation ObjectInvocation) (Value, error) {
	name := invocation.Message
	if name == "" {
		name = invocation.Class
	}
	return Null(), &UnsupportedError{Operation: "object operation", Name: name, Span: invocation.Span}
}
