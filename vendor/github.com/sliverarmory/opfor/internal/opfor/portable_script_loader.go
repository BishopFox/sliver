package opfor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

// portableScriptLoader is the safe, pure-Go subset of Sleep's ScriptLoader.
// It compiles source into OPFOR bytecode and never attempts to inspect or run
// JVM classes or JAR bytecode.
type portableScriptLoader struct {
	runtime *Runtime
	owner   *Script

	mu            sync.Mutex
	loadedScripts *portableJavaCollection
	scriptsByKey  *portableJavaMap
	// loaded is a typed ownership mirror retained for internal lifecycle tests.
	// Public registry behavior is authoritative in loadedScripts because the
	// returned LinkedList permits arbitrary direct mutation.
	loaded             []*portableScriptInstance
	instances          map[*portableScriptInstance]struct{}
	disableConversions bool
	charset            string
	charsetSet         bool
	globalCache        bool
	closed             bool
	closeDone          chan struct{}
	closeErr           error
	errRead            bool
}

func (loader *portableScriptLoader) String() string { return "<sleep.runtime.ScriptLoader>" }

type portableScriptInstance struct {
	loader  *portableScriptLoader
	name    string
	program *Program
	env     *portableScriptEnvironment
	shared  *portableScriptSharedEnvironment

	mu             sync.Mutex
	watchers       []Callable
	loaded         bool
	debug          int32
	loadTimeMillis int64
	sourceFiles    map[string]struct{}

	runMu        sync.Mutex
	stateMu      sync.Mutex
	closing      bool
	activeCancel context.CancelFunc
	closeDone    chan struct{}
	closeErr     error
	errRead      bool
	result       Value
	runErr       error
	child        *Runtime
	childScript  *Script
	childOwner   atomic.Pointer[Script]
	warnings     *portableScriptWarningWriter
}

func (instance *portableScriptInstance) String() string { return "<sleep.runtime.ScriptInstance>" }

type portableScriptEnvironment struct {
	instance *portableScriptInstance

	mu               sync.RWMutex
	table            *portableJavaMap
	environmentStack *portableJavaCollection
	console          *sleepIOHandle
	pendingError     Value
}

// portableScriptInstanceRunToken identifies the ScriptInstance whose runMu is
// held by the current synchronous execution. A ScriptLoader child may call a
// closure owned by its parent Runtime, so the current fiber's Runtime is not a
// reliable ownership signal. The token follows the context through that
// synchronous caller chain, is revoked before runMu is released, and is hidden
// from contexts detached for asynchronous work.
type portableScriptInstanceRunToken struct {
	instance *portableScriptInstance
	parent   *portableScriptInstanceRunToken
	// caller is the cancellation source which preceded this method's private
	// run/evaluate context. Runtime-owned asynchronous work may outlive the
	// synchronous ScriptInstance method while still honoring the importer.
	caller        context.Context
	releaseCaller func()
	script        atomic.Pointer[Script]
	active        atomic.Bool
}

type portableScriptInstanceRunContextKey struct{}

func withPortableScriptInstanceRunOwner(
	ctx context.Context,
	instance *portableScriptInstance,
	caller context.Context,
	releaseCaller func(),
) (context.Context, func()) {
	if ctx == nil {
		ctx = context.Background()
	}
	if caller == nil {
		caller = context.Background()
	}
	parent, _ := ctx.Value(portableScriptInstanceRunContextKey{}).(*portableScriptInstanceRunToken)
	token := &portableScriptInstanceRunToken{
		instance:      instance,
		parent:        parent,
		caller:        caller,
		releaseCaller: idempotentContextRelease(releaseCaller),
	}
	if instance != nil {
		token.script.Store(instance.childOwner.Load())
	}
	token.active.Store(true)
	return context.WithValue(ctx, portableScriptInstanceRunContextKey{}, token), func() {
		token.active.Store(false)
		token.releaseCaller()
	}
}

// bindPortableScriptInstanceRunScript publishes the Script constructed by the
// first Runtime.Load/Eval before its top-level body runs. This lets a
// setEnvironment call made during that body rebind the in-flight Script even
// though portableScriptInstance.childScript is assigned only after Load/Eval
// returns.
func bindPortableScriptInstanceRunScript(ctx context.Context, instance *portableScriptInstance, script *Script) {
	if ctx == nil || instance == nil || script == nil {
		return
	}
	instance.childOwner.CompareAndSwap(nil, script)
	for token, _ := ctx.Value(portableScriptInstanceRunContextKey{}).(*portableScriptInstanceRunToken); token != nil; token = token.parent {
		if token.instance == instance && token.active.Load() {
			token.script.CompareAndSwap(nil, script)
			return
		}
	}
}

func portableScriptInstanceRunOwner(ctx context.Context, instance *portableScriptInstance) (*Script, bool) {
	if ctx == nil || instance == nil {
		return nil, false
	}
	for token, _ := ctx.Value(portableScriptInstanceRunContextKey{}).(*portableScriptInstanceRunToken); token != nil; token = token.parent {
		if token.instance == instance && token.active.Load() {
			return token.script.Load(), true
		}
	}
	return nil, false
}

type portableScriptInstanceRunOwnerMaskedContext struct {
	context.Context
}

func (ctx portableScriptInstanceRunOwnerMaskedContext) Value(key any) any {
	if _, private := key.(portableScriptInstanceRunContextKey); private {
		return (*portableScriptInstanceRunToken)(nil)
	}
	return ctx.Context.Value(key)
}

func (ctx portableScriptInstanceRunOwnerMaskedContext) AfterFunc(function func()) func() bool {
	return context.AfterFunc(ctx.Context, function)
}

func (ctx portableScriptInstanceRunOwnerMaskedContext) retainExecutionCaller() (func(), bool) {
	return retainExecutionCaller(ctx.Context)
}

func withoutPortableScriptInstanceRunOwner(ctx context.Context) context.Context {
	if ctx == nil {
		return context.Background()
	}
	// A typed nil shadows every synchronous owner token in the retained parent
	// context without changing importer-owned values or cancellation. The custom
	// wrapper also forwards AfterFunc so context propagation does not allocate a
	// goroutine merely because the private value is hidden.
	return portableScriptInstanceRunOwnerMaskedContext{Context: ctx}
}

func (environment *portableScriptEnvironment) String() string {
	return "<sleep.runtime.ScriptEnvironment>"
}

// portableCompiledBlock is the pure-Go counterpart of sleep.engine.Block.
// Program is immutable, so a compiled block can safely seed any number of
// independent ScriptInstances without JVM serialization or bytecode loading.
type portableCompiledBlock struct {
	program *Program
}

func (block *portableCompiledBlock) String() string { return "<sleep.engine.Block>" }

// portableScriptVariables exposes only the variable reads that can be
// represented faithfully with OPFOR's Value API. Sleep's mutators require a
// JVM Scalar object, which an ordinary Sleep expression does not supply.
type portableScriptVariables struct {
	instance *portableScriptInstance
}

func (variables *portableScriptVariables) String() string {
	return "<sleep.runtime.ScriptVariables>"
}

// portableScriptInputStream is a comparable view over an IOObject's input.
// Returning the value (rather than a newly allocated pointer) preserves the
// reference identity of repeated IOObject.getInputStream calls.
type portableScriptInputStream struct {
	handle *sleepIOHandle
}

func (stream portableScriptInputStream) String() string { return "<java.io.InputStream>" }

func (stream portableScriptInputStream) Read(data []byte) (int, error) {
	if stream.handle == nil {
		return 0, io.ErrClosedPipe
	}
	return stream.handle.Read(data)
}

func (stream portableScriptInputStream) Close() error {
	if stream.handle == nil {
		return nil
	}
	// Match InputStream.close rather than IOObject.close: only the readable
	// side is detached. Keep the same readMu -> mu order as sleepIOHandle.Read.
	stream.handle.readMu.Lock()
	defer stream.handle.readMu.Unlock()
	stream.handle.mu.Lock()
	defer stream.handle.mu.Unlock()
	if stream.handle.ownRead && stream.handle.readCloser != nil {
		err := stream.handle.readCloser.Close()
		stream.handle.readCloser = nil
		stream.handle.ownRead = false
		return err
	}
	return nil
}

// portableScriptWarning retains the observable ScriptWarning surface passed to
// RuntimeWarningWatcher proxies. The runtime warning writer has already
// rendered Sleep's canonical basename and line-number policy, so String is
// byte-for-byte the text a JVM ScriptWarning.toString() would expose.
type portableScriptWarning struct {
	instance   *portableScriptInstance
	text       string
	message    string
	scriptName string
	nameShort  string
	line       int
	trace      bool
}

func (warning *portableScriptWarning) String() string {
	if warning == nil {
		return ""
	}
	return warning.text
}

func portableScriptObject(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if target, ok := invocation.Target.Object(); ok {
		switch object := target.(type) {
		case *portableScriptLoader:
			if object != nil {
				return object.invoke(ctx, invocation)
			}
		case *portableScriptInstance:
			if object != nil {
				return object.invoke(ctx, invocation)
			}
		case *portableScriptEnvironment:
			if object != nil {
				return object.invoke(ctx, invocation)
			}
		case *portableCompiledBlock:
			if object != nil {
				return object.invoke(invocation)
			}
		case *portableScriptBridge:
			if object != nil {
				return object.invoke(invocation)
			}
		case *portableScriptVariables:
			if object != nil {
				return object.invoke(invocation)
			}
		case portableScriptInputStream:
			return object.invoke(invocation)
		case *sleepIOHandle:
			if object != nil {
				return portableScriptIOObject(object, invocation)
			}
		case *portableScriptWarning:
			if object != nil {
				return object.invoke(invocation)
			}
		}
	}

	class := resolvePortableClassName(invocation.Class)
	switch class {
	case "sleep.runtime.ScriptLoader":
		if invocation.Op != ObjectConstruct {
			return Null(), false, nil
		}
		if len(invocation.Arguments) != 0 {
			return Null(), true, fmt.Errorf("no constructor matching sleep.runtime.ScriptLoader(%s)", portableObjectArgumentList(invocation))
		}
		if invocation.Runtime == nil {
			return Null(), true, errors.New("opfor: ScriptLoader construction requires a runtime")
		}
		loader := &portableScriptLoader{
			runtime:       invocation.Runtime,
			owner:         invocation.Runtime.script(invocation.Script),
			loadedScripts: newPortableJavaCollection("LinkedList", nil),
			scriptsByKey:  newPortableJavaMap("HashMap", nil),
			instances:     make(map[*portableScriptInstance]struct{}),
		}
		if loader.owner != nil {
			loader.owner.mu.Lock()
			if !loader.owner.active {
				loader.owner.mu.Unlock()
				return Null(), true, ErrScriptUnloaded
			}
			if loader.owner.scriptLoaders == nil {
				loader.owner.scriptLoaders = make(map[*portableScriptLoader]struct{})
			}
			loader.owner.scriptLoaders[loader] = struct{}{}
			loader.owner.mu.Unlock()
		}
		return ObjectValue(loader), true, nil

	case "sleep.bridges.io.IOObject":
		if invocation.Op != ObjectInvoke || (invocation.Message != "setConsole" && invocation.Message != "getConsole") {
			return Null(), false, nil
		}
		expected := 2
		if invocation.Message == "getConsole" {
			expected = 1
		}
		if len(invocation.Arguments) != expected {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		environmentObject, ok := invocation.Arg(0).Object()
		if !ok {
			return Null(), true, errors.New("java.lang.IllegalArgumentException: setConsole expected sleep.runtime.ScriptEnvironment")
		}
		environment, ok := environmentObject.(*portableScriptEnvironment)
		if !ok || environment == nil {
			return Null(), true, errors.New("java.lang.IllegalArgumentException: setConsole expected sleep.runtime.ScriptEnvironment")
		}
		if invocation.Message == "getConsole" {
			return ObjectValue(environment.getConsole()), true, nil
		}
		console, ok := ioHandleValue(invocation.Arg(1))
		if !ok {
			return Null(), true, errors.New("java.lang.IllegalArgumentException: setConsole expected sleep.bridges.io.IOObject")
		}
		environment.mu.Lock()
		environment.console = console
		environment.mu.Unlock()
		return Null(), true, nil
	}
	return Null(), false, nil
}

func (loader *portableScriptLoader) invoke(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable("sleep.runtime.ScriptLoader", invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if loader == nil || loader.runtime == nil {
		return Null(), true, errors.New("java.lang.IllegalStateException: ScriptLoader has no runtime")
	}
	switch invocation.Message {
	case "loadScript":
		if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 3 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		value, err := loader.loadScript(ctx, invocation, true)
		return value, true, err

	case "loadScriptNoReference":
		if len(invocation.Arguments) != 3 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		value, err := loader.loadScript(ctx, invocation, false)
		return value, true, err

	case "compileScript":
		if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 2 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		value, err := loader.compileScript(ctx, invocation)
		return value, true, err

	case "touch":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		if loader.runtime.scriptLoaderCache != nil {
			loader.runtime.scriptLoaderCache.touch(invocation.Arg(0).String())
		}
		return Null(), true, nil

	case "setGlobalCache":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		enabled := invocation.Arg(0).Truth()
		if enabled && loader.runtime.scriptLoaderCache == nil {
			return Null(), true, portableScriptLoaderUnsupported("global parsed-Block caches")
		}
		loader.mu.Lock()
		loader.globalCache = enabled
		loader.mu.Unlock()
		return Null(), true, nil

	case "setCharsetConversion":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		loader.mu.Lock()
		loader.disableConversions = !invocation.Arg(0).Truth()
		loader.mu.Unlock()
		return Null(), true, nil

	case "isCharsetConversions":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		loader.mu.Lock()
		enabled := !loader.disableConversions
		loader.mu.Unlock()
		// Sleep's Java bridge boxes a primitive boolean as a numeric scalar, so
		// false renders as "0" rather than the language-level empty false value.
		return Int(boolInt32(enabled)), true, nil

	case "setCharset":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		loader.mu.Lock()
		if invocation.Arg(0).IsNull() {
			loader.charset = ""
			loader.charsetSet = false
		} else {
			loader.charset = invocation.Arg(0).String()
			loader.charsetSet = true
		}
		loader.mu.Unlock()
		return Null(), true, nil

	case "getCharset":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		loader.mu.Lock()
		charset, set := loader.charset, loader.charsetSet
		loader.mu.Unlock()
		if !set {
			// A Java null returned through ObjectUtilities.BuildScalar becomes an
			// empty scalar, not Sleep's distinguished $null container identity.
			return String(""), true, nil
		}
		return String(charset), true, nil

	case "isLoaded":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		_, loaded := loader.scriptByKey(invocation.Arg(0))
		return Int(boolInt32(loaded)), true, nil

	case "getFirstScriptEnvironment":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		value, present := loader.firstLoadedScript()
		if !present {
			return Null(), true, nil
		}
		object, ok := value.Object()
		if !ok {
			return Null(), true, errors.New("java.lang.ClassCastException")
		}
		instance, ok := object.(*portableScriptInstance)
		if !ok || instance == nil {
			return Null(), true, errors.New("java.lang.ClassCastException")
		}
		environment := instance.env
		return ObjectValue(environment), true, nil

	case "getScripts":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		return ObjectValue(loader.loadedScripts), true, nil

	case "getScriptsByKey":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		return ObjectValue(loader.scriptsByKey), true, nil

	case "getScriptsToLoad", "getScriptsToUnload":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		configuredObject, ok := invocation.Arg(0).Object()
		if !ok {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		configured, ok := configuredObject.(*portableJavaCollection)
		if !ok || configured == nil || !configured.isSet() {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		configuredValues, err := configured.snapshotChecked()
		if err != nil {
			return Null(), true, err
		}
		delta, err := loader.scriptDeltaSet(invocation.Runtime, invocation.Message, configuredValues)
		if err != nil {
			return Null(), true, err
		}
		return ObjectValue(delta), true, nil

	case "unloadScript":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), true, nil
		}
		if object, ok := invocation.Arg(0).Object(); ok {
			if instance, ok := object.(*portableScriptInstance); ok && instance != nil {
				return Null(), true, loader.unload(ctx, instance)
			}
		}
		value, present := loader.scriptByKey(invocation.Arg(0))
		if !present || value.IsNull() {
			return Null(), true, errors.New("java.lang.NullPointerException")
		}
		object, ok := value.Object()
		if !ok {
			return Null(), true, errors.New("java.lang.ClassCastException")
		}
		instance, ok := object.(*portableScriptInstance)
		if !ok || instance == nil {
			return Null(), true, errors.New("java.lang.ClassCastException")
		}
		return Null(), true, loader.unload(ctx, instance)
	}
	return Null(), false, nil
}

func (loader *portableScriptLoader) scriptDeltaSet(runtime *Runtime, message string, configured []Value) (*portableJavaCollection, error) {
	loaded := loader.scriptKeys()

	var values []Value
	if message == "getScriptsToLoad" {
		values = make([]Value, 0, len(configured))
		for _, candidate := range configured {
			if !portableJavaContains(loaded, candidate) {
				values = append(values, candidate)
			}
		}
	} else {
		values = make([]Value, 0, len(loaded))
		for _, candidate := range loaded {
			if !portableJavaContains(configured, candidate) {
				values = append(values, candidate)
			}
		}
	}
	if err := reserveCollectionEntries(runtime, len(values)); err != nil {
		return nil, err
	}
	return newPortableJavaCollection("LinkedHashSet", values), nil
}

func (loader *portableScriptLoader) loadScript(ctx context.Context, invocation ObjectInvocation, referenced bool) (Value, error) {
	loader.mu.Lock()
	closed := loader.closed
	loader.mu.Unlock()
	if closed {
		return loader.flagLoadError(invocation, ObjectValue(&portableJavaException{
			class: "java.lang.IllegalStateException", message: "ScriptLoader is closed", text: "java.lang.IllegalStateException: ScriptLoader is closed",
		})), nil
	}
	file, fileOverload := portableJavaFileValue(invocation.Arg(0))
	if !fileOverload && invocation.Arg(0).Kind() != KindString {
		return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), nil
	}
	if fileOverload && (!referenced || len(invocation.Arguments) > 2) {
		return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), nil
	}
	name := invocation.Arg(0).String()
	instanceName := name
	var program *Program
	var modificationPath string
	var shared *portableScriptSharedEnvironment
	var err error
	if fileOverload {
		if len(invocation.Arguments) == 2 && !invocation.Arg(1).IsNull() {
			table, ok := portableScriptEnvironmentTable(invocation.Arg(1))
			if !ok {
				return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), nil
			}
			shared = portableSharedEnvironment(table)
		}
		program, instanceName, modificationPath, err = loader.compilePortableScriptFile(ctx, file)
		if err != nil {
			return loader.flagCompileError(invocation, name, err)
		}
		return loader.registerScriptAtRuntime(invocation, instanceName, program, modificationPath, true, shared)
	}
	switch len(invocation.Arguments) {
	case 1:
		program, modificationPath, err = loader.compileFile(ctx, invocation, name)
		if program != nil {
			instanceName = program.source.Name
		}
	case 2:
		if stream, ok := portableInputStreamValue(invocation.Arg(1)); ok {
			program, err = loader.compilePortableScriptStream(ctx, name, stream)
		} else if invocation.Arg(1).IsNull() {
			program, modificationPath, err = loader.compileFile(ctx, invocation, name)
			if program != nil {
				instanceName = program.source.Name
			}
		} else if table, ok := portableScriptEnvironmentTable(invocation.Arg(1)); ok {
			shared = portableSharedEnvironment(table)
			program, modificationPath, err = loader.compileFile(ctx, invocation, name)
			if program != nil {
				instanceName = program.source.Name
			}
		} else {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), nil
		}
	case 3:
		if !invocation.Arg(2).IsNull() {
			table, ok := portableScriptEnvironmentTable(invocation.Arg(2))
			if !ok {
				return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), nil
			}
			shared = portableSharedEnvironment(table)
		}
		if block, ok := portableCompiledBlockValue(invocation.Arg(1)); ok {
			program = block.program
		} else if referenced && invocation.Arg(1).Kind() == KindString {
			program, err = loader.compileString(name, invocation.Arg(1).String())
		} else if referenced {
			stream, ok := portableInputStreamValue(invocation.Arg(1))
			if !ok {
				return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), nil
			}
			program, err = loader.compilePortableScriptStream(ctx, name, stream)
		} else {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), nil
		}
	}
	if err != nil {
		return loader.flagCompileError(invocation, name, err)
	}
	if program == nil {
		return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), nil
	}
	return loader.registerScriptAtRuntime(invocation, instanceName, program, modificationPath, referenced, shared)
}

func (loader *portableScriptLoader) compileScript(ctx context.Context, invocation ObjectInvocation) (Value, error) {
	loader.mu.Lock()
	closed := loader.closed
	loader.mu.Unlock()
	if closed {
		return loader.flagLoadError(invocation, ObjectValue(&portableJavaException{
			class: "java.lang.IllegalStateException", message: "ScriptLoader is closed", text: "java.lang.IllegalStateException: ScriptLoader is closed",
		})), nil
	}
	file, fileOverload := portableJavaFileValue(invocation.Arg(0))
	if fileOverload && len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), nil
	}
	if !fileOverload && invocation.Arg(0).Kind() != KindString {
		return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), nil
	}
	name := invocation.Arg(0).String()
	var program *Program
	var err error
	if fileOverload {
		program, name, _, err = loader.compilePortableScriptFile(ctx, file)
	} else if len(invocation.Arguments) == 1 {
		program, _, err = loader.compileFile(ctx, invocation, name)
	} else if invocation.Arg(1).Kind() == KindString {
		program, err = loader.compileString(name, invocation.Arg(1).String())
	} else if stream, ok := portableInputStreamValue(invocation.Arg(1)); ok {
		program, err = loader.compilePortableScriptStream(ctx, name, stream)
	} else {
		return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptLoader"), nil
	}
	if err != nil {
		return loader.flagCompileError(invocation, name, err)
	}
	return ObjectValue(&portableCompiledBlock{program: program}), nil
}

// compilePortableScriptFile implements ScriptLoader's java.io.File overloads
// without routing the File through an importer SourceResolver. Sleep names the
// Block with File.getAbsolutePath(), opens the File itself, and feeds that
// stream through the loader's configured charset conversion policy.
func (loader *portableScriptLoader) compilePortableScriptFile(ctx context.Context, file *portableJavaFile) (*Program, string, string, error) {
	if file == nil {
		return nil, "", "", errors.New("java.lang.NullPointerException")
	}
	absolute, err := portableJavaFileAbsoluteValue(file.pathValue())
	if err != nil {
		return nil, "", "", err
	}
	name := absolute.String()
	path := portableJavaFileFilesystemPathValue(file.pathValue())
	stream, err := os.Open(path)
	if err != nil {
		return nil, name, path, fmt.Errorf("java.io.FileNotFoundException: %s", path)
	}
	program, err := loader.compilePortableScriptStream(ctx, name, stream)
	return program, name, path, err
}

func (loader *portableScriptLoader) compileFile(ctx context.Context, invocation ObjectInvocation, name string) (*Program, string, error) {
	request := SourceRequest{
		Script:          invocation.Script,
		IncludingSource: invocation.Span.Source,
		Name:            name,
		Span:            invocation.Span,
	}
	// ScriptLoader's String filename overload opens that exact file. Unlike
	// BasicUtilities.include and ImportManager, it does not route the name
	// through ParserConfig.findJarFile. Exact FileSourceResolver instances use
	// the private direct mode; arbitrary importer resolvers retain ownership of
	// their own lookup semantics.
	resolved, err := loader.runtime.resolveSourceRequestMode(ctx, request, sourceLookupDirect)
	if err != nil {
		return nil, "", err
	}
	source := resolved.source
	if strings.TrimSpace(source.Name) == "" {
		source.Name = name
	}
	decoded, _, decodeErr := loader.decodeSource(source.Data, resolved.reservedBytes)
	if decodeErr != nil {
		return nil, "", decodeErr
	}
	source.Data = decoded
	// The String file-name overload canonicalizes the default filesystem path.
	// Keep the runtime-visible identity slash-separated on every platform so
	// ScriptLoader registry keys, source spans, and importer callbacks do not
	// vary between Unix and Windows. modificationPath remains the native path
	// returned by the resolver and is used independently for filesystem I/O and
	// change tracking. Custom resolver names belong to the importer and remain
	// logical.
	if loader.runtime.defaultFileResolver != nil {
		source.Name = filepath.ToSlash(loader.runtime.defaultFileResolver.resolvePath(name))
	}
	program, err := loader.compileSource(source)
	return program, resolved.modificationPath, err
}

func (loader *portableScriptLoader) compileString(name, code string) (*Program, error) {
	if err := loader.runtime.reserveResource(resourceSourceBytes, uint64(len(code))); err != nil {
		return nil, err
	}
	return loader.compileSource(NewSource(name, []byte(code)))
}

func (loader *portableScriptLoader) compileSource(source Source) (*Program, error) {
	if loader == nil || loader.runtime == nil {
		return nil, errors.New("java.lang.IllegalStateException: ScriptLoader has no runtime")
	}
	loader.mu.Lock()
	enabled := loader.globalCache
	charset := loader.charset
	if !loader.charsetSet {
		charset = ""
	}
	conversion := !loader.disableConversions
	loader.mu.Unlock()
	cache := loader.runtime.scriptLoaderCache
	if !enabled || cache == nil {
		return loader.runtime.compileReservedSource(source)
	}
	return cache.compile(source.Name, charset, conversion, loader.runtime.scriptLoaderEnvironmentFingerprint(), source.Data, func() (*Program, error) {
		return loader.runtime.compileReservedSource(source)
	})
}

func (loader *portableScriptLoader) registerScript(invocation ObjectInvocation, name string, program *Program, modificationPath string, referenced bool, shared *portableScriptSharedEnvironment) Value {
	value, _ := loader.registerScriptAtRuntime(invocation, name, program, modificationPath, referenced, shared)
	return value
}

func (loader *portableScriptLoader) registerScriptAtRuntime(invocation ObjectInvocation, name string, program *Program, modificationPath string, referenced bool, shared *portableScriptSharedEnvironment) (Value, error) {
	environmentTable := newPortableJavaMap("Hashtable", nil)
	if shared != nil {
		environmentTable = shared.table
	} else {
		// ScriptInstance(null) creates a private Hashtable, and ScriptLoader runs
		// the same global bridges against it before returning the instance. Keep
		// an adapter for that private table too so getEnvironment is populated and
		// later script-defined functions remain live in the table.
		shared = portableSharedEnvironment(environmentTable)
	}
	instance := &portableScriptInstance{
		loader: loader, name: name, program: program, loaded: true,
		debug: loader.runtime.debugFlags, result: Null(),
		loadTimeMillis: loader.runtime.clock.Now().UnixMilli(),
		shared:         shared,
	}
	instance.env = &portableScriptEnvironment{
		instance:         instance,
		table:            environmentTable,
		environmentStack: newPortableJavaCollection("Stack", nil),
	}
	instance.associateSourceFile(modificationPath)
	if err := shared.installGlobalBridges(loader, loader.runtime); err != nil {
		return Null(), err
	}
	loader.mu.Lock()
	if loader.closed {
		loader.mu.Unlock()
		return loader.flagLoadError(invocation, ObjectValue(&portableJavaException{
			class: "java.lang.IllegalStateException", message: "ScriptLoader is closed", text: "java.lang.IllegalStateException: ScriptLoader is closed",
		})), nil
	}
	if referenced && name != "<interact mode>" {
		growth := 1 // loadedScripts always appends the new ScriptInstance.
		key := sleepCanonicalString(String(name))
		loader.scriptsByKey.mu.RLock()
		_, exists := loader.scriptsByKey.values[key]
		loader.scriptsByKey.mu.RUnlock()
		if !exists {
			growth++
		}
		if err := reserveCollectionEntries(invocation.Runtime, growth); err != nil {
			loader.mu.Unlock()
			return Null(), err
		}
		loader.addLoadedScript(instance)
		loader.putScriptByKey(String(name), ObjectValue(instance))
		loader.loaded = append(loader.loaded, instance)
	}
	loader.instances[instance] = struct{}{}
	loader.mu.Unlock()
	return ObjectValue(instance), nil
}

func (loader *portableScriptLoader) flagCompileError(invocation ObjectInvocation, name string, err error) (Value, error) {
	if isExecutionResourceError(err) {
		return Null(), err
	}
	var compileError *CompileError
	if errors.As(err, &compileError) {
		source := NewSource(name, nil)
		if compileError != nil && len(compileError.Diagnostics) != 0 {
			// Preserve the source carried by the diagnostics when compilation was
			// performed through a resolver rather than a direct string overload.
			source.Name = compileError.Diagnostics[0].Span.Source
		}
		return loader.flagLoadError(invocation, ObjectValue(newPortableScriptCompileException(source, err))), nil
	}
	return loader.flagLoadError(invocation, sourceErrorValue(err.Error(), err)), nil
}

func (loader *portableScriptLoader) unload(ctx context.Context, instance *portableScriptInstance) error {
	if loader == nil || instance == nil {
		return nil
	}
	loader.removeLoadedScript(instance)
	// Sleep's unloadScript(ScriptInstance) removes the current name key
	// unconditionally. If a caller mutated the live map to another value first,
	// that replacement is removed as well.
	loader.removeScriptByKey(String(instance.name))
	loader.mu.Lock()
	for index, candidate := range loader.loaded {
		if candidate == instance {
			loader.loaded = append(loader.loaded[:index], loader.loaded[index+1:]...)
			break
		}
	}
	loader.mu.Unlock()
	instance.mu.Lock()
	instance.loaded = false
	instance.mu.Unlock()

	childScript, ownedExecution := portableScriptInstanceRunOwner(ctx, instance)
	if !ownedExecution || childScript == nil {
		childScript = instance.childOwner.Load()
	}
	if childScript == nil || childScript.runtime == nil {
		return nil
	}
	return childScript.runtime.retireScriptGeneration(ctx, childScript)
}

func (loader *portableScriptLoader) flagLoadError(invocation ObjectInvocation, value Value) Value {
	if loader == nil || loader.runtime == nil {
		return Null()
	}
	script := loader.runtime.script(invocation.Script)
	if script == nil {
		return Null()
	}
	script.mu.Lock()
	script.lastError = value
	debug := script.debug
	script.mu.Unlock()
	if debug&2 == 2 {
		loader.runtime.writeWarning("checkError(): "+value.String(), invocation.Span)
	}
	return Null()
}

func newPortableScriptCompileException(source Source, err error) *portableCompileException {
	var compileError *CompileError
	if !errors.As(err, &compileError) || compileError == nil {
		return &portableCompileException{summary: "YourCodeSucksException: " + err.Error()}
	}
	summary := sleepSourceErrorMessage("loadScript", &sourceCompileFailure{source: source, cause: err})
	var formatted strings.Builder
	for _, diagnostic := range compileError.Diagnostics {
		if diagnostic.Severity != SeverityError {
			continue
		}
		line := diagnostic.Span.Start.Line
		fmt.Fprintf(&formatted, "Error: %s at line %d\n", diagnostic.Message, line)
		fmt.Fprintf(&formatted, "       %s\n", sourceLine(string(source.Data), line))
	}
	return &portableCompileException{compile: compileError, summary: summary, formatted: formatted.String()}
}

func (instance *portableScriptInstance) invoke(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable("sleep.runtime.ScriptInstance", invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "addWarningWatcher":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		watcher, ok := invocation.Arg(0).Function()
		if !ok || watcher == nil {
			return Null(), true, errors.New("java.lang.IllegalArgumentException: addWarningWatcher expected a Sleep closure")
		}
		instance.mu.Lock()
		instance.watchers = append(instance.watchers, watcher)
		instance.mu.Unlock()
		return Null(), true, nil

	case "removeWarningWatcher":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		watcher, ok := invocation.Arg(0).Function()
		if !ok || watcher == nil {
			return Null(), true, errors.New("java.lang.IllegalArgumentException: removeWarningWatcher expected a Sleep closure")
		}
		instance.mu.Lock()
		for index, candidate := range instance.watchers {
			if samePortableCallable(candidate, watcher) {
				copy(instance.watchers[index:], instance.watchers[index+1:])
				instance.watchers[len(instance.watchers)-1] = nil
				instance.watchers = instance.watchers[:len(instance.watchers)-1]
				break
			}
		}
		instance.mu.Unlock()
		return Null(), true, nil

	case "getScriptEnvironment":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		return ObjectValue(instance.env), true, nil

	case "getName":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		instance.mu.Lock()
		name := instance.name
		instance.mu.Unlock()
		return String(name), true, nil

	case "setName":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		instance.mu.Lock()
		instance.name = invocation.Arg(0).String()
		instance.mu.Unlock()
		return Null(), true, nil

	case "getDebugFlags":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		instance.mu.Lock()
		debug := instance.debug
		instance.mu.Unlock()
		return Int(debug), true, nil

	case "setDebugFlags":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		instance.runMu.Lock()
		instance.mu.Lock()
		instance.debug = invocation.Arg(0).Int32()
		debug := instance.debug
		instance.mu.Unlock()
		if instance.childScript != nil {
			instance.childScript.mu.Lock()
			instance.childScript.debug = debug
			instance.childScript.mu.Unlock()
		}
		instance.runMu.Unlock()
		return Null(), true, nil

	case "isLoaded":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		instance.mu.Lock()
		loaded := instance.loaded
		instance.mu.Unlock()
		return Int(boolInt32(loaded)), true, nil

	case "setUnloaded":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		instance.mu.Lock()
		instance.loaded = false
		instance.mu.Unlock()
		return Null(), true, nil

	case "associateFile":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		file, ok := portableJavaFileValue(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		instance.associateSourceFile(portableJavaFileFilesystemPath(file.path))
		return Null(), true, nil

	case "hasChanged":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		return Int(boolInt32(instance.hasChanged())), true, nil

	case "runScript":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptInstance"), true, nil
		}
		value, err := instance.run(ctx)
		return value, true, err
	}
	return Null(), false, nil
}

// associateSourceFile mirrors ScriptInstance.associateFile: only a path that
// exists at association time becomes evidence. FileSourceResolver supplies an
// absolute path, so later runtime chdir calls cannot retarget an include.
func (instance *portableScriptInstance) associateSourceFile(path string) {
	if instance == nil || path == "" {
		return
	}
	if _, err := os.Stat(path); err != nil {
		return
	}
	instance.mu.Lock()
	if instance.sourceFiles == nil {
		instance.sourceFiles = make(map[string]struct{})
	}
	instance.sourceFiles[path] = struct{}{}
	instance.mu.Unlock()
}

// hasChanged follows Sleep 2.1's intentionally one-sided timestamp check.
// Deletion and replacement with an older timestamp do not count as changes.
func (instance *portableScriptInstance) hasChanged() bool {
	if instance == nil {
		return false
	}
	instance.mu.Lock()
	loadTime := instance.loadTimeMillis
	paths := make([]string, 0, len(instance.sourceFiles))
	for path := range instance.sourceFiles {
		paths = append(paths, path)
	}
	instance.mu.Unlock()
	for _, path := range paths {
		info, err := os.Stat(path)
		if err == nil && info.ModTime().UnixMilli() > loadTime {
			return true
		}
	}
	return false
}

func samePortableCallable(left, right Callable) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	leftValue, rightValue := reflect.ValueOf(left), reflect.ValueOf(right)
	if leftValue.Type() != rightValue.Type() {
		return false
	}
	switch leftValue.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Pointer, reflect.Slice, reflect.UnsafePointer:
		return leftValue.Pointer() == rightValue.Pointer()
	default:
		return leftValue.Comparable() && leftValue.Interface() == rightValue.Interface()
	}
}

func boolInt32(value bool) int32 {
	if value {
		return 1
	}
	return 0
}

func portableCompiledBlockValue(value Value) (*portableCompiledBlock, bool) {
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	block, ok := object.(*portableCompiledBlock)
	return block, ok && block != nil && block.program != nil
}

func portableInputStreamValue(value Value) (io.ReadCloser, bool) {
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	switch stream := object.(type) {
	case portableScriptInputStream:
		if stream.handle != nil {
			return stream, true
		}
	case *portableScriptInputStream:
		if stream != nil && stream.handle != nil {
			return *stream, true
		}
	case io.ReadCloser:
		return stream, stream != nil
	}
	return nil, false
}

func (loader *portableScriptLoader) compilePortableScriptStream(ctx context.Context, name string, stream io.ReadCloser) (*Program, error) {
	if stream == nil {
		return nil, errors.New("java.io.IOException: input stream is null")
	}
	data, reserved, readErr := readReservedSource(ctx, stream, func(amount uint64) error {
		return loader.runtime.reserveResource(resourceSourceBytes, amount)
	})
	closeErr := stream.Close()
	if readErr != nil || closeErr != nil {
		return nil, fmt.Errorf("java.io.IOException: %w", errors.Join(readErr, closeErr))
	}
	data, reserved, err := loader.decodeSource(data, reserved)
	if err != nil {
		return nil, err
	}
	// BufferedReader.readLine strips every line terminator and ScriptLoader
	// prepends one newline per returned line, including an initial empty line.
	normalizedLength := 0
	for start := 0; start < len(data); {
		end := start
		for end < len(data) && data[end] != '\n' && data[end] != '\r' {
			end++
		}
		normalizedLength += 1 + end - start
		if end == len(data) {
			break
		}
		if data[end] == '\r' && end+1 < len(data) && data[end+1] == '\n' {
			end++
		}
		start = end + 1
	}
	if _, err := loader.runtime.reserveSourceLength(normalizedLength, reserved); err != nil {
		return nil, err
	}
	var code strings.Builder
	code.Grow(normalizedLength)
	for start := 0; start < len(data); {
		end := start
		for end < len(data) && data[end] != '\n' && data[end] != '\r' {
			end++
		}
		code.WriteByte('\n')
		code.Write(data[start:end])
		if end == len(data) {
			break
		}
		if data[end] == '\r' && end+1 < len(data) && data[end+1] == '\n' {
			end++
		}
		start = end + 1
	}
	return loader.compileSource(NewSource(name, []byte(code.String())))
}

// decodeSource applies the source-stream character policy configured on this
// ScriptLoader. The stock JVM uses its process-default charset when none is
// selected; OPFOR deliberately uses UTF-8 so an embedded runtime is stable
// across machines. Disabling conversions preserves Sleep's NoConversion
// behavior by mapping every input octet directly to the same-valued UTF-16
// code unit.
func (loader *portableScriptLoader) decodeSource(data []byte, reserved uint64) ([]byte, uint64, error) {
	if loader == nil {
		return append([]byte(nil), data...), reserved, nil
	}
	loader.mu.Lock()
	disabled := loader.disableConversions
	charset, charsetSet := loader.charset, loader.charsetSet
	loader.mu.Unlock()

	if !disabled && !charsetSet {
		// Source.Data is already documented as UTF-8. Preserve exact byte offsets
		// and diagnostics on the overwhelmingly common default path.
		return append([]byte(nil), data...), reserved, nil
	}

	if disabled {
		decodedLength := 0
		for _, octet := range data {
			if octet < 0x80 {
				decodedLength++
			} else {
				decodedLength += 2
			}
		}
		var err error
		reserved, err = loader.runtime.reserveSourceLength(decodedLength, reserved)
		if err != nil {
			return nil, reserved, err
		}
		units := make([]uint16, len(data))
		for index, octet := range data {
			units[index] = uint16(octet)
		}
		return []byte(sleepRenderStringUnits(units, nil)), reserved, nil
	}

	encoding, err := sleepLookupTextCharset(charset)
	if err != nil {
		// Sleep catches UnsupportedEncodingException, prints its stack trace,
		// and then falls back to the process-default reader. Preserve the
		// successful compile and its visible first diagnostic line without
		// fabricating JVM stack frames; OPFOR's deterministic default is UTF-8.
		if loader.runtime != nil && loader.runtime.stderr != nil {
			_, _ = fmt.Fprintf(loader.runtime.stderr, "java.io.UnsupportedEncodingException: %s\n", charset)
			// Sleep ignores an ordinary diagnostic sink failure here, but an
			// OPFOR output-quota violation is a fatal family boundary. Stop before
			// reserving expansion bytes or allocating decoded source.
			if outputErr := loader.runtime.outputLimitError(); outputErr != nil {
				return nil, reserved, outputErr
			}
		}
		encoding = sleepCharsetUTF8
	}
	preflight := &sleepTextDecoder{}
	preflight.reset(encoding)
	decodedLength := preflight.decodedRenderedLength(data, true)
	reserved, err = loader.runtime.reserveSourceLength(decodedLength, reserved)
	if err != nil {
		return nil, reserved, err
	}
	decoder := &sleepTextDecoder{}
	decoder.reset(encoding)
	units := decoder.decode(data, true)
	return []byte(sleepRenderStringUnits(units, nil)), reserved, nil
}

func renderedSourceLength(units []uint16) int {
	length := 0
	for index := 0; index < len(units); index++ {
		unit := units[index]
		switch {
		case unit >= 0xd800 && unit <= 0xdbff && index+1 < len(units) &&
			units[index+1] >= 0xdc00 && units[index+1] <= 0xdfff:
			length += 4
			index++
		case unit < 0x80:
			length++
		case unit < 0x800:
			length += 2
		default:
			length += 3
		}
	}
	return length
}

func portableScriptLoaderUnsupported(surface string) error {
	message := surface + " are not supported by the pure-Go ScriptLoader"
	return &portableJavaException{
		class: "java.lang.UnsupportedOperationException", message: message,
		text: "java.lang.UnsupportedOperationException: " + message,
	}
}

func portableScriptIOObject(handle *sleepIOHandle, invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable("sleep.bridges.io.IOObject", invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke || invocation.Message != "getInputStream" {
		return Null(), false, nil
	}
	if len(invocation.Arguments) != 0 {
		return portableNoMatchingMethod(invocation, "sleep.bridges.io.IOObject"), true, nil
	}
	handle.mu.Lock()
	open := handle.reader != nil
	handle.mu.Unlock()
	if !open {
		return Null(), true, nil
	}
	return ObjectValue(portableScriptInputStream{handle: handle}), true, nil
}

func (stream portableScriptInputStream) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable("java.io.InputStream", invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke || invocation.Message != "close" {
		return Null(), false, nil
	}
	if len(invocation.Arguments) != 0 {
		return portableNoMatchingMethod(invocation, "java.io.InputStream"), true, nil
	}
	return Null(), true, stream.Close()
}

func (block *portableCompiledBlock) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable("sleep.engine.Block", invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke || len(invocation.Arguments) != 0 || block == nil || block.program == nil {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "getSource":
		return String(block.program.source.Name), true, nil
	case "getApproximateLineNumber":
		low, _ := portableBlockLineRange(block.program)
		if low == 0 {
			return Int(-1), true, nil
		}
		return Int(int32(low - 1)), true, nil
	case "getLowLineNumber":
		low, _ := portableBlockLineRange(block.program)
		if low == 0 {
			return Int(1<<31 - 1), true, nil
		}
		return Int(int32(low - 1)), true, nil
	case "getHighLineNumber":
		_, high := portableBlockLineRange(block.program)
		if high == 0 {
			return Int(0), true, nil
		}
		return Int(int32(high - 1)), true, nil
	case "getApproximateLineRange":
		low, high := portableBlockLineRange(block.program)
		if low == 0 {
			return String("2147483647-0"), true, nil
		}
		low--
		high--
		if low == high {
			return String(strconv.Itoa(low)), true, nil
		}
		return String(strconv.Itoa(low) + "-" + strconv.Itoa(high)), true, nil
	case "getSourceLocation":
		low, high := portableBlockLineRange(block.program)
		rangeText := "2147483647-0"
		if low != 0 {
			low--
			high--
			rangeText = strconv.Itoa(low)
			if low != high {
				rangeText += "-" + strconv.Itoa(high)
			}
		}
		return String(filepath.Base(block.program.source.Name) + ":" + rangeText), true, nil
	}
	return Null(), false, nil
}

func portableBlockLineRange(program *Program) (int, int) {
	if program == nil || program.function == nil {
		return 0, 0
	}
	low, high := 0, 0
	for _, instruction := range program.function.Instructions {
		line := instruction.Span.Start.Line
		if line <= 0 {
			continue
		}
		if low == 0 || line < low {
			low = line
		}
		if line > high {
			high = line
		}
	}
	return low, high
}

func (variables *portableScriptVariables) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable("sleep.runtime.ScriptVariables", invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "getScalar":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptVariables"), true, nil
		}
		return variables.instance.scalar(invocation.Arg(0).String()), true, nil
	case "putScalar", "setScalarLevel", "pushClosureLevel", "popClosureLevel", "pushLocalLevel", "popLocalLevel":
		return Null(), true, portableScriptLoaderUnsupported("mutable ScriptVariables operations")
	}
	return Null(), false, nil
}

func (environment *portableScriptEnvironment) invoke(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable("sleep.runtime.ScriptEnvironment", invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "getScriptInstance":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		return ObjectValue(environment.instance), true, nil
	case "getScriptVariables":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		return ObjectValue(&portableScriptVariables{instance: environment.instance}), true, nil
	case "getEnvironment":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		table := environment.environmentTable()
		if table == nil {
			return Null(), true, nil
		}
		return ObjectValue(table), true, nil
	case "getEnvironmentStack":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		return ObjectValue(environment.getEnvironmentStack()), true, nil
	case "getFunction":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		value, err := portableScriptEnvironmentTypedEntry(environment.environmentTable(), invocation.Arg(0), "sleep.interfaces.Function")
		return value, true, err
	case "getBlock":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		key := String("^" + invocation.Arg(0).String())
		value, err := portableScriptEnvironmentTypedEntry(environment.environmentTable(), key, "sleep.engine.Block")
		return value, true, err
	case "getFunctionEnvironment":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		value, err := portableScriptEnvironmentTypedEntry(environment.environmentTable(), invocation.Arg(0), "sleep.interfaces.Environment")
		return value, true, err
	case "getPredicateEnvironment":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		value, err := portableScriptEnvironmentTypedEntry(environment.environmentTable(), invocation.Arg(0), "sleep.interfaces.PredicateEnvironment")
		return value, true, err
	case "getFilterEnvironment":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		value, err := portableScriptEnvironmentTypedEntry(environment.environmentTable(), invocation.Arg(0), "sleep.interfaces.FilterEnvironment")
		return value, true, err
	case "getPredicate":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		value, err := portableScriptEnvironmentTypedEntry(environment.environmentTable(), invocation.Arg(0), "sleep.interfaces.Predicate")
		return value, true, err
	case "getOperator":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		value, err := portableScriptEnvironmentTypedEntry(environment.environmentTable(), invocation.Arg(0), "sleep.interfaces.Operator")
		return value, true, err
	case "getScalar":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		return environment.instance.scalar(invocation.Arg(0).String()), true, nil
	case "putScalar":
		return Null(), true, portableScriptLoaderUnsupported("mutable ScriptEnvironment operations")
	case "setEnvironment":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		var table *portableJavaMap
		if !invocation.Arg(0).IsNull() {
			var ok bool
			table, ok = portableScriptEnvironmentTable(invocation.Arg(0))
			if !ok {
				return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
			}
		}
		environment.setEnvironmentTable(ctx, table)
		return Null(), true, nil
	case "flagError":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		environment.flagError(invocation.Arg(0))
		return Null(), true, nil
	case "checkError":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		return environment.checkError(), true, nil
	case "getCurrentSource":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		return String("unknown"), true, nil
	case "evaluateStatement", "evaluatePredicate", "evaluateExpression", "evaluateParsedLiteral":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.ScriptEnvironment"), true, nil
		}
		code := invocation.Arg(0).String()
		prefix := ""
		suffix := ""
		predicate := false
		switch invocation.Message {
		case "evaluatePredicate":
			prefix = "if ("
			suffix = ") { return 1; } else { return $null; }"
			predicate = true
		case "evaluateExpression":
			prefix = "return ("
			suffix = ");"
		case "evaluateParsedLiteral":
			prefix = "return \""
			suffix = "\";"
		}
		value, err := environment.instance.evaluate(ctx, prefix, code, suffix)
		if err != nil {
			var compileException *portableCompileException
			if errors.As(err, &compileException) {
				environment.flagError(ObjectValue(compileException))
				return Null(), true, nil
			}
			return Null(), true, err
		}
		if predicate {
			return Int(boolInt32(value.Int32() == 1)), true, nil
		}
		return value, true, nil
	}
	return Null(), false, nil
}

func portableScriptEnvironmentEntry(table *portableJavaMap, key Value) (Value, bool) {
	if table == nil {
		return Null(), false
	}
	text := sleepCanonicalString(key)
	table.mu.RLock()
	value, present := table.values[text]
	table.mu.RUnlock()
	return value, present
}

func portableScriptEnvironmentTypedEntry(table *portableJavaMap, key Value, class string) (Value, error) {
	value, present := portableScriptEnvironmentEntry(table, key)
	if !present {
		return Null(), nil
	}
	if portableScriptEnvironmentValueImplements(value, class) {
		return value, nil
	}
	return Null(), &portableJavaException{
		class: "java.lang.ClassCastException",
		text:  "java.lang.ClassCastException",
	}
}

func portableScriptEnvironmentValueImplements(value Value, class string) bool {
	class = resolvePortableClassName(class)
	if class == "sleep.interfaces.Function" {
		if callable, ok := value.Function(); ok && callable != nil {
			return true
		}
	}
	object, ok := value.Object()
	if !ok || object == nil {
		return false
	}
	switch candidate := object.(type) {
	case *portableScriptBridge:
		return candidate.supports(class)
	case *portableCompiledBlock:
		return class == "sleep.engine.Block" && candidate != nil && candidate.program != nil
	default:
		return false
	}
}

func (environment *portableScriptEnvironment) environmentTable() *portableJavaMap {
	if environment == nil {
		return nil
	}
	environment.mu.RLock()
	table := environment.table
	environment.mu.RUnlock()
	return table
}

func (environment *portableScriptEnvironment) getEnvironmentStack() *portableJavaCollection {
	if environment == nil {
		return nil
	}
	environment.mu.Lock()
	if environment.environmentStack == nil {
		// ScriptEnvironment constructs this Stack eagerly. Lazy initialization is
		// retained for package-internal adapter values built directly by tests.
		environment.environmentStack = newPortableJavaCollection("Stack", nil)
	}
	stack := environment.environmentStack
	environment.mu.Unlock()
	return stack
}

// setEnvironmentTable mirrors ScriptEnvironment.setEnvironment's raw field
// replacement. In particular nil is accepted and no bridges are installed in
// a replacement table. OPFOR additionally redirects the already-created child
// interpreter to that exact table so portable environment-table entries are
// resolved from the replacement rather than merely exposed by getEnvironment.
func (environment *portableScriptEnvironment) setEnvironmentTable(ctx context.Context, table *portableJavaMap) {
	if environment == nil {
		return
	}
	instance := environment.instance
	var executingScript *Script
	ownedExecution := false
	if instance != nil {
		executingScript, ownedExecution = portableScriptInstanceRunOwner(ctx, instance)
		if !ownedExecution {
			instance.runMu.Lock()
			defer instance.runMu.Unlock()
		}
	}

	environment.mu.Lock()
	environment.table = table
	environment.mu.Unlock()
	if instance == nil {
		return
	}

	shared := portableSharedEnvironment(table)
	instance.shared = shared
	if instance.child != nil {
		instance.child.scriptLoaderSharedEnvironment = shared
	}
	if instance.childScript != nil {
		instance.childScript.mu.Lock()
		instance.childScript.sharedEnvironment = shared
		instance.childScript.mu.Unlock()
	}
	if executingScript != nil && executingScript != instance.childScript {
		executingScript.mu.Lock()
		executingScript.sharedEnvironment = shared
		executingScript.mu.Unlock()
	}
}

func (instance *portableScriptInstance) scalar(name string) Value {
	if instance == nil {
		return Null()
	}
	instance.runMu.Lock()
	script := instance.childScript
	var value Value
	if script != nil {
		value = script.Get(name)
	}
	instance.runMu.Unlock()
	return value
}

func (environment *portableScriptEnvironment) flagError(value Value) {
	if environment == nil || environment.instance == nil {
		return
	}
	instance := environment.instance
	instance.runMu.Lock()
	if instance.childScript != nil {
		instance.childScript.mu.Lock()
		instance.childScript.lastError = value
		instance.childScript.mu.Unlock()
	} else {
		environment.mu.Lock()
		environment.pendingError = value
		environment.mu.Unlock()
	}
	instance.runMu.Unlock()
}

func (environment *portableScriptEnvironment) checkError() Value {
	if environment == nil || environment.instance == nil {
		return Null()
	}
	instance := environment.instance
	instance.runMu.Lock()
	defer instance.runMu.Unlock()
	if instance.childScript != nil {
		instance.childScript.mu.Lock()
		value := instance.childScript.lastError
		instance.childScript.lastError = Null()
		instance.childScript.mu.Unlock()
		return value
	}
	environment.mu.Lock()
	value := environment.pendingError
	environment.pendingError = Null()
	environment.mu.Unlock()
	return value
}

func (instance *portableScriptInstance) installPendingErrorLocked() {
	if instance == nil || instance.childScript == nil || instance.env == nil {
		return
	}
	instance.env.mu.Lock()
	value := instance.env.pendingError
	instance.env.pendingError = Null()
	instance.env.mu.Unlock()
	if value.IsNull() {
		return
	}
	instance.childScript.mu.Lock()
	instance.childScript.lastError = value
	instance.childScript.mu.Unlock()
}

func (instance *portableScriptInstance) evaluate(ctx context.Context, prefix, code, suffix string) (Value, error) {
	if instance == nil || instance.loader == nil || instance.loader.runtime == nil {
		return Null(), errors.New("java.lang.IllegalStateException: script is not runnable")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	runtimeInstance := instance.loader.runtime
	// Reserve the complete wrapped source before concatenating it. The
	// ScriptEnvironment expression helpers add fixed scaffolding around
	// importer-controlled text, and building that string before admission would
	// let a rejected evaluation allocate beyond MaxSourceBytesPerRuntime.
	total := uint64(len(prefix)) + uint64(len(code)) + uint64(len(suffix))
	if err := runtimeInstance.reserveResource(resourceSourceBytes, total); err != nil {
		return Null(), err
	}
	code = prefix + code + suffix
	program, err := runtimeInstance.compileReservedSource(NewSource("eval", []byte(code)))
	if err != nil {
		if isExecutionResourceError(err) {
			return Null(), err
		}
		return Null(), newPortableScriptCompileException(NewSource("eval", []byte(code)), err)
	}
	instance.runMu.Lock()
	defer instance.runMu.Unlock()
	if instance.loader == nil || instance.loader.runtime == nil {
		return Null(), errors.New("java.lang.IllegalStateException: script is not runnable")
	}
	if instance.env != nil && instance.env.environmentTable() == nil {
		return Null(), portableNullScriptEnvironmentError()
	}
	asyncCaller, releaseAsyncCaller := detachExecutionLeaseCancellationLease(ctx)
	evalContext, cancel := context.WithCancel(ctx)
	evalContext, releaseRunOwner := withPortableScriptInstanceRunOwner(
		evalContext,
		instance,
		asyncCaller,
		releaseAsyncCaller,
	)
	defer releaseRunOwner()
	instance.stateMu.Lock()
	if instance.closing {
		instance.stateMu.Unlock()
		cancel()
		return Null(), errors.New("java.lang.IllegalStateException: script is not runnable")
	}
	instance.activeCancel = cancel
	instance.stateMu.Unlock()
	defer func() {
		cancel()
		instance.stateMu.Lock()
		instance.activeCancel = nil
		instance.stateMu.Unlock()
	}()

	if instance.warnings == nil {
		instance.warnings = &portableScriptWarningWriter{instance: instance}
	}
	instance.warnings.begin(evalContext)
	if instance.child == nil {
		instance.child, err = instance.newChildRuntime(instance.warnings)
		if err != nil {
			return Null(), err
		}
	}
	var value Value
	if instance.childScript == nil {
		value, err = instance.child.Eval(evalContext, "eval", code)
		instance.child.mu.RLock()
		instance.childScript = instance.child.evalScript
		instance.child.mu.RUnlock()
		instance.childOwner.Store(instance.childScript)
		instance.installPendingErrorLocked()
	} else {
		closure := &scriptClosure{script: instance.childScript, function: program.function, captured: instance.childScript.globals}
		value, err = closure.invokeFresh(withExecutionMeter(evalContext, instance.child), nil)
	}
	instance.warnings.flush()
	if err == nil {
		err = instance.warnings.takeErr()
	}
	return value, err
}

func (warning *portableScriptWarning) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable("sleep.error.ScriptWarning", invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke || len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "isDebugTrace":
		return Bool(warning.trace), true, nil
	case "getSource":
		return ObjectValue(warning.instance), true, nil
	case "getMessage":
		return String(warning.message), true, nil
	case "getLineNumber":
		return Int(int32(warning.line)), true, nil
	case "getScriptName":
		return String(warning.scriptName), true, nil
	case "getNameShort":
		return String(warning.nameShort), true, nil
	case "toString":
		return String(warning.String()), true, nil
	}
	return Null(), false, nil
}

func (environment *portableScriptEnvironment) consoleHandle() *sleepIOHandle {
	if environment == nil {
		return nil
	}
	environment.mu.RLock()
	console := environment.console
	environment.mu.RUnlock()
	return console
}

func (environment *portableScriptEnvironment) getConsole() *sleepIOHandle {
	if environment == nil {
		return nil
	}
	environment.mu.Lock()
	defer environment.mu.Unlock()
	if environment.console != nil {
		return environment.console
	}
	if environment.instance == nil || environment.instance.loader == nil || environment.instance.loader.runtime == nil {
		return nil
	}
	parent := environment.instance.loader.runtime
	environment.console = newIOHandle("console", parent.stdin, parent.stdout, false, false, true).withRuntimeOutputAccount(parent.resources)
	return environment.console
}

// portableScriptConsoleRouter keeps IOObject.setConsole live across repeated
// runScript calls. Sleep stores the console in ScriptInstance metadata, so a
// console replacement after the first run affects every later run as well.
type portableScriptConsoleRouter struct {
	instance *portableScriptInstance
}

func (router *portableScriptConsoleRouter) runtimeOutputAccount() *runtimeResourceAccount {
	if router == nil || router.instance == nil || router.instance.loader == nil || router.instance.loader.runtime == nil {
		return nil
	}
	return router.instance.loader.runtime.resources
}

func (router *portableScriptConsoleRouter) Read(data []byte) (int, error) {
	if router == nil || router.instance == nil || router.instance.loader == nil || router.instance.loader.runtime == nil {
		return 0, io.ErrClosedPipe
	}
	if console := router.instance.env.consoleHandle(); console != nil {
		return console.Read(data)
	}
	return router.instance.loader.runtime.stdin.Read(data)
}

func (router *portableScriptConsoleRouter) Write(data []byte) (int, error) {
	if router == nil || router.instance == nil || router.instance.loader == nil || router.instance.loader.runtime == nil {
		return 0, io.ErrClosedPipe
	}
	account := router.instance.loader.runtime.resources
	if console := router.instance.env.consoleHandle(); console != nil {
		return writeRuntimeOutput(account, console, data)
	}
	return writeRuntimeOutput(account, router.instance.loader.runtime.stdout, data)
}

func (instance *portableScriptInstance) run(ctx context.Context) (Value, error) {
	instance.runMu.Lock()
	defer instance.runMu.Unlock()
	if instance.loader == nil || instance.loader.runtime == nil || instance.program == nil {
		instance.runErr = errors.New("java.lang.IllegalStateException: script is not runnable")
		return instance.result, instance.runErr
	}
	if instance.env != nil && instance.env.environmentTable() == nil {
		instance.runErr = portableNullScriptEnvironmentError()
		return instance.result, instance.runErr
	}
	if ctx == nil {
		ctx = context.Background()
	}
	asyncCaller, releaseAsyncCaller := detachExecutionLeaseCancellationLease(ctx)
	runContext, cancel := context.WithCancel(ctx)
	runContext, releaseRunOwner := withPortableScriptInstanceRunOwner(
		runContext,
		instance,
		asyncCaller,
		releaseAsyncCaller,
	)
	defer releaseRunOwner()
	instance.stateMu.Lock()
	if instance.closing {
		instance.stateMu.Unlock()
		cancel()
		instance.runErr = errors.New("java.lang.IllegalStateException: script is not runnable")
		return instance.result, instance.runErr
	}
	instance.activeCancel = cancel
	instance.stateMu.Unlock()
	defer func() {
		cancel()
		instance.stateMu.Lock()
		instance.activeCancel = nil
		instance.stateMu.Unlock()
	}()

	if instance.warnings == nil {
		instance.warnings = &portableScriptWarningWriter{instance: instance}
	}
	instance.warnings.begin(runContext)
	if instance.child == nil {
		child, err := instance.newChildRuntime(instance.warnings)
		if err != nil {
			instance.runErr = err
			return instance.result, err
		}
		instance.child = child
	}
	instance.mu.Lock()
	debug := instance.debug
	instance.mu.Unlock()
	var value Value
	var runErr error
	if instance.childScript == nil {
		executionProgram := *instance.program
		executionProgram.source = instance.program.Source()
		instance.mu.Lock()
		executionProgram.source.Name = instance.name
		instance.mu.Unlock()
		instance.childScript, runErr = instance.child.Load(runContext, &executionProgram)
		if instance.childScript != nil {
			instance.childOwner.Store(instance.childScript)
			instance.childScript.mu.Lock()
			instance.childScript.program = instance.program
			instance.childScript.mu.Unlock()
			instance.installPendingErrorLocked()
			value = instance.childScript.Result()
		}
	} else {
		instance.mu.Lock()
		name := instance.name
		instance.mu.Unlock()
		if err := instance.childScript.setGlobalAt(runContext, "$__SCRIPT__", String(name), Span{}); err != nil {
			instance.runErr = err
			return instance.result, err
		}
		if err := instance.childScript.setGlobalAt(runContext, "$__SCRIPT_NAME__", String(name), Span{}); err != nil {
			instance.runErr = err
			return instance.result, err
		}
		instance.childScript.mu.Lock()
		instance.childScript.program = instance.program
		instance.childScript.mu.Unlock()
		instance.childScript.mu.Lock()
		instance.childScript.debug = debug
		instance.childScript.mu.Unlock()
		main := &scriptClosure{script: instance.childScript, function: instance.program.function, captured: instance.childScript.globals}
		value, runErr = main.invokeFresh(withExecutionMeter(runContext, instance.child), nil)
		if runErr == nil {
			instance.childScript.mu.Lock()
			instance.childScript.result = value
			instance.childScript.mu.Unlock()
		}
	}
	instance.warnings.flush()
	if runErr == nil {
		runErr = instance.warnings.takeErr()
	}
	if instance.childScript != nil {
		instance.childScript.mu.RLock()
		debug = instance.childScript.debug
		instance.childScript.mu.RUnlock()
		instance.mu.Lock()
		instance.debug = debug
		instance.mu.Unlock()
	}
	instance.result = value
	instance.runErr = runErr
	return value, runErr
}

func portableNullScriptEnvironmentError() *portableJavaException {
	return &portableJavaException{
		class: "java.lang.NullPointerException",
		text:  "java.lang.NullPointerException",
	}
}

func (instance *portableScriptInstance) newChildRuntime(warnings io.Writer) (*Runtime, error) {
	parent := instance.loader.runtime
	console := &portableScriptConsoleRouter{instance: instance}
	objectHost := parent.objectHost
	if wrapped, ok := objectHost.(defaultObjectHost); ok {
		objectHost = wrapped.primary
	}
	if objectHost == nil {
		objectHost = unsupportedObjectHost{}
	}
	options := []Option{
		WithStdin(console),
		WithStdout(console),
		WithStderr(warnings),
		WithHost(parent.host),
		WithObjectHost(objectHost),
		withRuntimeResourceAccount(parent.resources),
		WithIncludeCyclePolicy(parent.includeCycles),
		WithClock(parent.clock),
	}
	if parent.scriptLoaderCache != nil {
		options = append(options, WithScriptLoaderCache(parent.scriptLoaderCache))
	}
	options = append(options, parent.scriptLoaderProfileOptions()...)
	options = append(options, WithTaintMode(parent.taintMode))
	if !isNilInterface(parent.variableProvider) {
		options = append(options, WithVariableProvider(parent.variableProvider))
	}
	instance.mu.Lock()
	debug := instance.debug
	instance.mu.Unlock()
	options = append(options, WithDebugFlags(debug))
	if parent.observer != nil {
		options = append(options, WithBindingObserver(parent.observer))
	}
	if len(parent.initialGlobals) != 0 {
		options = append(options, WithInitialGlobals(parent.initialGlobals))
	}
	if parent.lifecycle != nil {
		options = append(options, WithScriptLifecycleObserver(parent.lifecycle))
	}
	if !isNilInterface(parent.loadableProvider) {
		options = append(options, WithLoadableProvider(parent.loadableProvider))
	}
	parent.mu.RLock()
	environments := make(map[string]EnvironmentKind, len(parent.environments))
	for keyword, kind := range parent.environments {
		environments[keyword] = kind
	}
	parent.mu.RUnlock()
	for keyword, kind := range environments {
		options = append(options, WithEnvironment(keyword, kind))
	}
	if parent.defaultFileResolver == nil && parent.resolver != nil {
		options = append(options, WithSourceResolver(parent.resolver))
	}
	child, err := New(options...)
	if err != nil {
		return nil, err
	}
	child.scriptLoaderInstance = instance
	child.scriptLoaderSharedEnvironment = instance.shared
	if parent.defaultFileResolver != nil {
		base := parent.defaultFileResolver.BaseDirectory()
		if child.defaultFileResolver != nil {
			// ScriptLoader children inherit ParserConfig's process-wide
			// sleep.classpath. Copy the detached entries directly so Windows
			// volume separators and deliberate empty entries are not reparsed.
			child.defaultFileResolver.setSleepClasspathEntries(parent.defaultFileResolver.SleepClasspath())
		}
		if base != "" {
			if _, err := child.Invoke(context.Background(), "chdir", String(base)); err != nil {
				_ = child.Close(context.Background())
				return nil, err
			}
		}
	}
	inheritPortableRuntimeExtensions(parent, child)
	return child, nil
}

// inheritPortableRuntimeExtensions copies embedding registrations into a
// ScriptLoader child without replacing child-bound core functions with method
// values bound to the parent Runtime. Added functions and core overrides have
// a distinct code pointer; ordinary core functions have the same code pointer
// in both runtimes and remain bound to the child console and state.
func inheritPortableRuntimeExtensions(parent, child *Runtime) {
	if parent == nil || child == nil {
		return
	}
	parent.mu.RLock()
	functions := make(map[string]NativeFunc, len(parent.functions))
	for name, function := range parent.functions {
		functions[name] = function
	}
	explicitFunctions := make(map[string]struct{}, len(parent.explicitFunctions))
	for name := range parent.explicitFunctions {
		explicitFunctions[name] = struct{}{}
	}
	policies := make(map[string]TaintPolicy, len(parent.taintPolicies))
	for name, policy := range parent.taintPolicies {
		policies[name] = policy
	}
	parent.mu.RUnlock()

	child.mu.Lock()
	for name, function := range functions {
		current := child.functions[name]
		if current == nil || nativeFunctionPointer(current) != nativeFunctionPointer(function) {
			child.functions[name] = function
		}
	}
	for name := range explicitFunctions {
		child.explicitFunctions[name] = struct{}{}
	}
	for name, policy := range policies {
		child.taintPolicies[name] = policy
	}
	child.mu.Unlock()
}

func nativeFunctionPointer(function NativeFunc) uintptr {
	if function == nil {
		return 0
	}
	return reflect.ValueOf(function).Pointer()
}

func (loader *portableScriptLoader) close(ctx context.Context) error {
	if loader == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	loader.mu.Lock()
	start := !loader.closed
	var instances []*portableScriptInstance
	if start {
		loader.closed = true
		loader.closeDone = make(chan struct{})
		instances = make([]*portableScriptInstance, 0, len(loader.instances))
		for instance := range loader.instances {
			instances = append(instances, instance)
		}
		loader.clearRegistries()
		loader.loaded = nil
		loader.instances = nil
	}
	done := loader.closeDone
	loader.mu.Unlock()

	if start {
		go loader.finishClose(ctx, instances)
	}
	if done == nil {
		return nil
	}
	select {
	case <-done:
		loader.mu.Lock()
		err := loader.consumeCloseErrorLocked()
		loader.mu.Unlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (loader *portableScriptLoader) finishClose(ctx context.Context, instances []*portableScriptInstance) {
	var result error
	for _, instance := range instances {
		closeErr := instance.close(ctx)
		remaining, waitExpired := splitContextWaitError(ctx, closeErr)
		if remaining != nil {
			result = errors.Join(result, remaining)
		}
		if waitExpired {
			if terminalErr := instance.close(context.Background()); terminalErr != nil {
				terminalErr, _ = splitContextWaitError(ctx, terminalErr)
				result = errors.Join(result, terminalErr)
			}
		}
	}
	loader.mu.Lock()
	loader.closeErr = result
	if !channelClosed(loader.closeDone) {
		close(loader.closeDone)
	}
	loader.mu.Unlock()
}

func (loader *portableScriptLoader) consumeCloseErrorLocked() error {
	if loader == nil || loader.errRead {
		return nil
	}
	loader.errRead = true
	return loader.closeErr
}

func (instance *portableScriptInstance) close(ctx context.Context) error {
	if instance == nil {
		return nil
	}
	instance.stateMu.Lock()
	start := !instance.closing
	var cancel context.CancelFunc
	if start {
		instance.closing = true
		instance.closeDone = make(chan struct{})
		cancel = instance.activeCancel
	}
	done := instance.closeDone
	instance.stateMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if start {
		go instance.finishClose(ctx)
	}
	if done == nil {
		return nil
	}
	select {
	case <-done:
		instance.stateMu.Lock()
		err := instance.consumeCloseErrorLocked()
		instance.stateMu.Unlock()
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (instance *portableScriptInstance) finishClose(ctx context.Context) {
	// close never holds stateMu while waiting for runMu. run publishes its
	// cancellation function under stateMu only after acquiring runMu.
	instance.runMu.Lock()
	child := instance.child
	instance.child = nil
	instance.childScript = nil
	instance.childOwner.Store(nil)
	instance.mu.Lock()
	instance.loaded = false
	instance.mu.Unlock()
	instance.runMu.Unlock()
	var result error
	if child == nil {
		instance.stateMu.Lock()
		if !channelClosed(instance.closeDone) {
			close(instance.closeDone)
		}
		instance.stateMu.Unlock()
		return
	}
	closeErr := child.Close(ctx)
	remaining, waitExpired := splitContextWaitError(ctx, closeErr)
	result = errors.Join(result, remaining)
	child.mu.RLock()
	complete := child.closed && channelClosed(child.closeDone)
	child.mu.RUnlock()
	// A ScriptLoader is an ownership boundary, not merely another caller of an
	// unrelated Runtime. Close may return early when ctx carries an active
	// parent-runtime unload token (the conservative cross-runtime deadlock
	// policy), but the parent Script must not finish its own unload while the
	// owned child is still running lifecycle observers or other cleanup. Join
	// that already-started terminal Close from a token-free context.
	if waitExpired || !complete {
		terminalErr := child.Close(context.Background())
		terminalErr, _ = splitContextWaitError(ctx, terminalErr)
		result = errors.Join(result, terminalErr)
	}
	instance.stateMu.Lock()
	instance.closeErr = result
	if !channelClosed(instance.closeDone) {
		close(instance.closeDone)
	}
	instance.stateMu.Unlock()
}

func (instance *portableScriptInstance) consumeCloseErrorLocked() error {
	if instance == nil || instance.errRead {
		return nil
	}
	instance.errRead = true
	return instance.closeErr
}

type portableScriptWarningWriter struct {
	ctx      context.Context
	instance *portableScriptInstance

	mu      sync.Mutex
	pending string
	first   error
}

func (writer *portableScriptWarningWriter) begin(ctx context.Context) {
	if writer == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	writer.mu.Lock()
	writer.ctx = ctx
	writer.first = nil
	writer.mu.Unlock()
}

func (writer *portableScriptWarningWriter) Write(data []byte) (int, error) {
	if writer == nil {
		return len(data), nil
	}
	writer.mu.Lock()
	writer.pending += string(data)
	var lines []string
	for {
		index := strings.IndexByte(writer.pending, '\n')
		if index < 0 {
			break
		}
		lines = append(lines, strings.TrimSuffix(writer.pending[:index], "\r"))
		writer.pending = writer.pending[index+1:]
	}
	writer.mu.Unlock()
	for _, line := range lines {
		writer.dispatch(line)
	}
	return len(data), nil
}

func (writer *portableScriptWarningWriter) flush() {
	if writer == nil {
		return
	}
	writer.mu.Lock()
	line := strings.TrimSuffix(writer.pending, "\r")
	writer.pending = ""
	writer.mu.Unlock()
	if line != "" {
		writer.dispatch(line)
	}
}

func (writer *portableScriptWarningWriter) dispatch(line string) {
	if writer == nil || writer.instance == nil || line == "" {
		return
	}
	warning := parsePortableScriptWarning(writer.instance, line)
	writer.mu.Lock()
	ctx := writer.ctx
	writer.mu.Unlock()
	if ctx == nil {
		ctx = context.Background()
	}
	writer.instance.mu.Lock()
	watchers := append([]Callable(nil), writer.instance.watchers...)
	writer.instance.mu.Unlock()
	for _, watcher := range watchers {
		if watcher == nil {
			continue
		}
		if _, err := watcher.Invoke(ctx, ObjectValue(warning)); err != nil {
			writer.mu.Lock()
			if writer.first == nil {
				writer.first = err
			}
			writer.mu.Unlock()
		}
	}
}

func (writer *portableScriptWarningWriter) takeErr() error {
	if writer == nil {
		return nil
	}
	writer.mu.Lock()
	defer writer.mu.Unlock()
	err := writer.first
	writer.first = nil
	return err
}

func parsePortableScriptWarning(instance *portableScriptInstance, line string) *portableScriptWarning {
	warning := &portableScriptWarning{instance: instance, text: line}
	body := line
	switch {
	case strings.HasPrefix(body, "Trace: "):
		warning.trace = true
		body = strings.TrimPrefix(body, "Trace: ")
	case strings.HasPrefix(body, "Warning: "):
		body = strings.TrimPrefix(body, "Warning: ")
	}
	warning.message = body
	if at := strings.LastIndex(body, " at "); at >= 0 {
		location := body[at+4:]
		if colon := strings.LastIndex(location, ":"); colon >= 0 {
			if lineNumber, err := strconv.Atoi(location[colon+1:]); err == nil {
				warning.message = body[:at]
				warning.scriptName = location[:colon]
				warning.nameShort = filepath.Base(filepath.FromSlash(warning.scriptName))
				warning.line = lineNumber
			}
		}
	}
	return warning
}
