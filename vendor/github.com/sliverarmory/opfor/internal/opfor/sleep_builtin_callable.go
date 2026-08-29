package opfor

import (
	"context"
	"errors"
	"strings"
)

// sleepBuiltinFunctionFamily identifies one Function object installed by
// Sleep 2.1's stock bridges. Most objects are installed under several keys;
// single-registration objects are included only when Function.evaluate's
// called-key argument is observably significant. It is deliberately about
// source object identity, not about Go helpers which happen to share an
// implementation.
type sleepBuiltinFunctionFamily uint8

const (
	sleepBuiltinFamilyNumbers sleepBuiltinFunctionFamily = iota + 1
	sleepBuiltinFamilyUtilities
	sleepBuiltinFamilyArray
	sleepBuiltinFamilyHash
	sleepBuiltinFamilyMap
	sleepBuiltinFamilyCast
	sleepBuiltinFamilyUse
	sleepBuiltinFamilyEval
	sleepBuiltinFamilySync
	sleepBuiltinFamilyStringCharAt
	sleepBuiltinFamilyStringSubstring
	sleepBuiltinFamilyStringIndexOf
	sleepBuiltinFamilyStringSorters
	sleepBuiltinFamilyStringLeft
	sleepBuiltinFamilyStringRight
	sleepBuiltinFamilyIO
	sleepBuiltinFamilySocket
	sleepBuiltinFamilyConsume
	sleepBuiltinFamilyPrintln
	sleepBuiltinFamilyFileSystem
	sleepBuiltinFamilyFileListing
)

// sleepBuiltinFamilyForFunction is pinned to the objects installed by
// BasicNumbers, BasicStrings, BasicUtilities, BasicIO, and FileSystemBridge at
// Cobalt-Strike/sleep@60ac3ff9dacc3e7b5a6c58be201c5830afbda398. Names whose
// bridges are already evaluator-owned (scope, lambda/function, and regex
// matcher families) remain in intrinsicFunctionCallable.
func sleepBuiltinFamilyForFunction(name string) (sleepBuiltinFunctionFamily, bool) {
	switch strings.TrimPrefix(name, "&") {
	case "abs", "acos", "asin", "atan", "atan2", "ceil", "cos", "log", "round",
		"sin", "sqrt", "tan", "radians", "degrees", "exp", "floor", "sum",
		"double", "int", "uint", "long", "parseNumber", "formatNumber", "rand", "srand", "not":
		return sleepBuiltinFamilyNumbers, true
	case "concat", "keys", "size", "push", "pop", "add", "flatten", "clear", "splice",
		"subarray", "sublist", "setRemovalPolicy", "setMissPolicy", "untaint", "taint",
		"putAll", "addAll", "removeAll", "retainAll", "pushl", "popl", "search", "reduce",
		"values", "remove", "setField", "typeOf", "newInstance", "scalar", "exit", "watch",
		"debug", "warn", "profile", "getStackTrace", "checkError", "invoke", "inline":
		return sleepBuiltinFamilyUtilities, true
	case "array", "@":
		return sleepBuiltinFamilyArray, true
	case "hash", "ohash", "ohasha", "%":
		return sleepBuiltinFamilyHash, true
	case "map", "filter":
		return sleepBuiltinFamilyMap, true
	case "cast", "casti":
		return sleepBuiltinFamilyCast, true
	case "use", "include":
		return sleepBuiltinFamilyUse, true
	case "eval", "expr":
		return sleepBuiltinFamilyEval, true
	case "semaphore", "acquire", "release":
		return sleepBuiltinFamilySync, true
	case "charAt", "byteAt":
		return sleepBuiltinFamilyStringCharAt, true
	case "substr", "mid":
		return sleepBuiltinFamilyStringSubstring, true
	case "indexOf", "lindexOf":
		return sleepBuiltinFamilyStringIndexOf, true
	case "sorta", "sortn", "sortd":
		return sleepBuiltinFamilyStringSorters, true
	case "left":
		return sleepBuiltinFamilyStringLeft, true
	case "right":
		return sleepBuiltinFamilyStringRight, true
	case "allocate", "readc", "readObject", "writeObject", "readAsObject", "writeAsObject",
		"sizeof", "wait", "setEncoding", "checksum", "digest":
		return sleepBuiltinFamilyIO, true
	case "connect", "listen":
		return sleepBuiltinFamilySocket, true
	case "consume", "skip":
		return sleepBuiltinFamilyConsume, true
	case "println", "printf":
		return sleepBuiltinFamilyPrintln, true
	case "createNewFile", "deleteFile", "chdir", "cwd", "getCurrentDirectory", "mkdir",
		"rename", "setLastModified", "setReadOnly":
		return sleepBuiltinFamilyFileSystem, true
	case "ls", "listRoots":
		return sleepBuiltinFamilyFileListing, true
	default:
		return 0, false
	}
}

func sleepBuiltinFamilyBridgeName(family sleepBuiltinFunctionFamily) string {
	switch family {
	case sleepBuiltinFamilyNumbers:
		return "BasicNumbers"
	case sleepBuiltinFamilyUtilities:
		return "BasicUtilities"
	case sleepBuiltinFamilyArray:
		return "BasicUtilities.array"
	case sleepBuiltinFamilyHash:
		return "BasicUtilities.hash"
	case sleepBuiltinFamilyMap:
		return "BasicUtilities.map"
	case sleepBuiltinFamilyCast:
		return "BasicUtilities.cast"
	case sleepBuiltinFamilyUse:
		return "BasicUtilities.use"
	case sleepBuiltinFamilyEval:
		return "BasicUtilities.eval"
	case sleepBuiltinFamilySync:
		return "BasicUtilities.sync"
	case sleepBuiltinFamilyStringCharAt:
		return "BasicStrings.charAt"
	case sleepBuiltinFamilyStringSubstring:
		return "BasicStrings.substr"
	case sleepBuiltinFamilyStringIndexOf:
		return "BasicStrings.indexOf"
	case sleepBuiltinFamilyStringSorters:
		return "BasicStrings.sorters"
	case sleepBuiltinFamilyStringLeft:
		return "BasicStrings.left"
	case sleepBuiltinFamilyStringRight:
		return "BasicStrings.right"
	case sleepBuiltinFamilyIO:
		return "BasicIO"
	case sleepBuiltinFamilySocket:
		return "BasicIO.socket"
	case sleepBuiltinFamilyConsume:
		return "BasicIO.consume"
	case sleepBuiltinFamilyPrintln:
		return "BasicIO.println"
	case sleepBuiltinFamilyFileSystem:
		return "FileSystemBridge"
	case sleepBuiltinFamilyFileListing:
		return "FileSystemBridge.listFiles"
	default:
		return ""
	}
}

func sleepBuiltinFamilyMatchesBridge(family sleepBuiltinFunctionFamily, bridge string) bool {
	want := sleepBuiltinFamilyBridgeName(family)
	if bridge == want {
		return true
	}
	// BasicNumbers is stored through one Sanitizer wrapper when Sleep taint mode
	// is enabled. The wrapper still forwards the called key to the same object.
	return family == sleepBuiltinFamilyNumbers && bridge == "Sanitizer(BasicNumbers)"
}

// sleepBuiltinFunctionCallable preserves the identity of a stock Sleep bridge
// object across function()/setf() aliases. CallRequest supplies the called
// environment key to Function.evaluate; the bridge does not blindly retain
// the key from which its handle was first obtained.
type sleepBuiltinFunctionCallable struct {
	origin  string
	family  sleepBuiltinFunctionFamily
	runtime *Runtime
	script  ScriptID
	span    Span
}

// appliesTaintPolicy reports that this callable models the source bridge's
// Tainter/Sanitizer/Sensitive wrapper itself. invokeNamed must not apply its
// generic closure permeability a second time after the wrapper policy ran.
func (*sleepBuiltinFunctionCallable) appliesTaintPolicy() bool { return true }

func newSleepBuiltinFunctionCallable(
	name string,
	family sleepBuiltinFunctionFamily,
	runtime *Runtime,
	script ScriptID,
	span Span,
) *sleepBuiltinFunctionCallable {
	return &sleepBuiltinFunctionCallable{
		origin: strings.TrimPrefix(name, "&"), family: family,
		runtime: runtime, script: script, span: span,
	}
}

func (callable *sleepBuiltinFunctionCallable) String() string {
	if callable == nil {
		return "&"
	}
	return "&" + callable.origin
}

func (callable *sleepBuiltinFunctionCallable) Invoke(ctx context.Context, values ...Value) (Value, error) {
	arguments := make([]Argument, len(values))
	for index, value := range values {
		arguments[index] = Argument{Value: value}
	}
	return callable.invokeOrigin(ctx, callable.span, arguments)
}

func (callable *sleepBuiltinFunctionCallable) invokeArguments(ctx context.Context, arguments []Argument) (Value, error) {
	return callable.invokeOrigin(ctx, callable.span, arguments)
}

func (callable *sleepBuiltinFunctionCallable) invokeNamedArgumentsAt(
	ctx context.Context,
	calledName string,
	span Span,
	arguments []Argument,
) (Value, error) {
	if callable == nil {
		return Null(), ErrInvalidCallable
	}
	calledName = strings.TrimPrefix(calledName, "&")
	dispatchName, dispatch := sleepBuiltinFamilyDispatch(callable.family, calledName)
	switch dispatch {
	case sleepBuiltinDispatchNull:
		return Null(), nil
	case sleepBuiltinDispatchSortIdentity:
		return callable.invokeSortIdentity(ctx, calledName, arguments)
	case sleepBuiltinDispatchNative:
		return callable.invokeAt(ctx, calledName, dispatchName, span, arguments)
	default:
		return Null(), ErrInvalidCallable
	}
}

type sleepBuiltinDispatch uint8

const (
	sleepBuiltinDispatchNull sleepBuiltinDispatch = iota + 1
	sleepBuiltinDispatchNative
	sleepBuiltinDispatchSortIdentity
)

func sleepBuiltinFamilyDispatch(family sleepBuiltinFunctionFamily, calledName string) (string, sleepBuiltinDispatch) {
	if calledFamily, ok := sleepBuiltinFamilyForFunction(calledName); ok && calledFamily == family {
		return calledName, sleepBuiltinDispatchNative
	}
	switch family {
	case sleepBuiltinFamilyArray:
		return "array", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyHash:
		return "hash", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyMap:
		return "filter", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyCast:
		return "cast", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyUse:
		return "include", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyEval:
		return "expr", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyStringCharAt:
		return "byteAt", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyStringSubstring:
		return "substr", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyStringIndexOf:
		return "indexOf", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyStringSorters:
		return "", sleepBuiltinDispatchSortIdentity
	case sleepBuiltinFamilySocket:
		return "connect", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyConsume:
		return "consume", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyPrintln:
		return "println", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyFileListing:
		return "ls", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyStringLeft:
		return "left", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyStringRight:
		return "right", sleepBuiltinDispatchNative
	case sleepBuiltinFamilyNumbers, sleepBuiltinFamilyUtilities, sleepBuiltinFamilySync,
		sleepBuiltinFamilyIO, sleepBuiltinFamilyFileSystem:
		return "", sleepBuiltinDispatchNull
	default:
		return "", 0
	}
}

func (callable *sleepBuiltinFunctionCallable) invokeOrigin(ctx context.Context, span Span, arguments []Argument) (Value, error) {
	if callable == nil || callable.origin == "" {
		return Null(), ErrInvalidCallable
	}
	return callable.invokeNamedArgumentsAt(ctx, callable.origin, span, arguments)
}

func (callable *sleepBuiltinFunctionCallable) activeRuntime(ctx context.Context) (*Runtime, ScriptID, error) {
	if callable == nil {
		return nil, 0, ErrInvalidCallable
	}
	runtime, script := callable.runtime, callable.script
	if fiber := currentFiber(ctx); fiber != nil && fiber.closure != nil && fiber.closure.script != nil {
		runtime = fiber.closure.script.runtime
		script = fiber.closure.script.id
	}
	if runtime == nil {
		return nil, 0, errors.New("opfor: Sleep bridge function requires an active runtime")
	}
	return runtime, script, nil
}

func (callable *sleepBuiltinFunctionCallable) invokeAt(
	ctx context.Context,
	calledName string,
	dispatchName string,
	span Span,
	arguments []Argument,
) (Value, error) {
	runtime, script, err := callable.activeRuntime(ctx)
	if err != nil {
		return Null(), err
	}

	// Four members of BasicUtilities are evaluator-sensitive in OPFOR, but the
	// pinned source stores them on the same `this` bridge as its ordinary native
	// functions. Route them without inventing a JVM bridge object.
	if callable.family == sleepBuiltinFamilyUtilities && intrinsicUtilitiesFunction(dispatchName) {
		runtime.mu.RLock()
		policy := runtime.taintPolicies[callable.origin]
		runtime.mu.RUnlock()
		return runtime.applyTaintPolicy(ctx, calledName, policy, arguments, span, func() (Value, error) {
			return newIntrinsicFunctionCallable(dispatchName).invokeNamedArgumentsAt(ctx, dispatchName, span, arguments)
		})
	}

	runtime.mu.RLock()
	function := runtime.stockFunctions[dispatchName]
	policy := runtime.taintPolicies[callable.origin]
	runtime.mu.RUnlock()
	if function == nil {
		// A source family must never fall through to Host merely because its Go
		// implementation is unavailable. Stock names are gated before a handle is
		// created; this branch covers only an internally inconsistent inventory.
		return Null(), nil
	}
	return runtime.invoke(ctx, Invocation{
		Runtime: runtime, Script: script, Name: calledName, Span: span, Arguments: arguments,
		forcedNative: function, forcedTaintPolicy: policy, forcedNativePolicy: true,
		bridgeDispatchName: dispatchName,
	})
}

func intrinsicUtilitiesFunction(name string) bool {
	switch strings.TrimPrefix(name, "&") {
	case "checkError", "getStackTrace", "invoke", "inline":
		return true
	default:
		return false
	}
}

func (callable *sleepBuiltinFunctionCallable) invokeSortIdentity(
	ctx context.Context,
	calledName string,
	arguments []Argument,
) (Value, error) {
	runtime, script, err := callable.activeRuntime(ctx)
	if err != nil {
		return Null(), err
	}
	if len(arguments) == 0 {
		return runtime.applyTaintPolicy(ctx, calledName, runtime.taintPolicy(callable.origin), arguments, callable.span, func() (Value, error) {
			return ArrayValue(NewArray()), nil
		})
	}
	invocation := Invocation{Runtime: runtime, Script: script, Name: calledName, Arguments: sleepBridgeArguments(arguments)}
	return runtime.applyTaintPolicy(ctx, calledName, runtime.taintPolicy(callable.origin), invocation.Arguments, callable.span, func() (Value, error) {
		array, err := invocationWorkableArray(ctx, invocation, 0)
		if err != nil {
			return Null(), err
		}
		return ArrayValue(array), nil
	})
}

func (r *Runtime) hasStockFunction(name string) bool {
	if r == nil {
		return false
	}
	name = strings.TrimPrefix(name, "&")
	r.mu.RLock()
	_, exists := r.stockFunctions[name]
	r.mu.RUnlock()
	return exists
}
