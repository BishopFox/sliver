package opfor

import (
	"context"
	"errors"
	"sort"
	"strings"

	"github.com/sliverarmory/opfor/internal/envspec"
)

// portableScriptSharedEnvironment is the pure-Go state carried by a
// ScriptInstance's exact java.util.Hashtable. It may be a caller-supplied table
// shared by several instances or the private table created for a null
// environment. Sleep stores functions, predicates, operators, environment
// keywords, and an (isloaded) marker in that table. OPFOR's parser and evaluator
// own the executable predicate/operator implementations; this adapter exposes
// bounded portable bridge objects for their Java introspection surface while
// retaining the table's visible mutable identity.
type portableScriptSharedEnvironment struct {
	table *portableJavaMap
	// plannedKeys is used only by installGlobalBridges' key-only dry run. It
	// records canonical keys without materializing portable map entries or
	// bridge values proportional to importer function/environment registries.
	plannedKeys map[string]struct{}
	// known records names installed by a global bridge or a loaded script. It
	// distinguishes deletion from an importer-owned function which was never
	// represented in the Hashtable and may still resolve through Host.
	known map[string]struct{}
}

func portableScriptEnvironmentTable(value Value) (*portableJavaMap, bool) {
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	table, ok := object.(*portableJavaMap)
	return table, ok && table != nil && table.className() == "Hashtable"
}

func portableSharedEnvironment(table *portableJavaMap) *portableScriptSharedEnvironment {
	if table == nil || table.className() != "Hashtable" {
		return nil
	}
	table.mu.Lock()
	shared := table.scriptEnvironment
	if shared == nil {
		shared = &portableScriptSharedEnvironment{table: table, known: make(map[string]struct{})}
		table.scriptEnvironment = shared
	}
	table.mu.Unlock()
	return shared
}

// installGlobalBridges applies the observable part of ScriptLoader's
// inProcessScript marker contract. A matching marker preserves every caller
// mutation. A missing or foreign marker reinstalls this loader's global
// function bridge entries exactly once and replaces the marker.
func (shared *portableScriptSharedEnvironment) installGlobalBridges(loader *portableScriptLoader, runtime *Runtime) error {
	if shared == nil || shared.table == nil || loader == nil || runtime == nil {
		return nil
	}
	table := shared.table
	table.mu.RLock()
	if marker, present := table.values["(isloaded)"]; present {
		if object, ok := marker.Object(); ok && object == loader {
			table.mu.RUnlock()
			return nil
		}
	}
	table.mu.RUnlock()

	runtime.mu.RLock()
	names := make([]string, 0, len(runtime.functions))
	for name := range runtime.functions {
		names = append(names, name)
	}
	environments := make(map[string]EnvironmentKind, len(runtime.environments))
	for keyword, kind := range runtime.environments {
		environments[keyword] = kind
	}
	taintMode := runtime.taintMode
	runtime.mu.RUnlock()
	sort.Strings(names)

	table.mu.Lock()
	if marker, present := table.values["(isloaded)"]; present {
		if object, ok := marker.Object(); ok && object == loader {
			table.mu.Unlock()
			return nil
		}
	}

	// Preflight the exact canonical key set before constructing the populated
	// portable plan. The key-only map is transient Go accounting metadata; its
	// size is bounded by importer registries and it contains no collection
	// entries or proportional bridge objects.
	keyPlan := &portableScriptSharedEnvironment{
		table:       newPortableJavaMap("Hashtable", nil),
		plannedKeys: make(map[string]struct{}),
		known:       make(map[string]struct{}),
	}
	for _, name := range names {
		name = strings.TrimPrefix(name, "&")
		keyPlan.putLocked(String("&"+name), Null())
	}
	keyPlan.installDefaultIntrospectionBridgesLocked(nil, taintMode)
	for keyword := range environments {
		keyPlan.putLocked(String(keyword), Null())
	}
	keyPlan.putLocked(String("(isloaded)"), Null())

	growth := 0
	for key := range keyPlan.plannedKeys {
		if _, present := table.values[key]; !present {
			growth++
		}
	}
	if err := reserveCollectionEntries(runtime, growth); err != nil {
		table.mu.Unlock()
		return err
	}

	// Build the exact replacement set against a private table after reservation.
	// Seeding an importer-owned &function preserves the bridge's special identity
	// rule; every other global entry is unconditionally replaced by installation.
	plannedTable := newPortableJavaMap("Hashtable", nil)
	planned := &portableScriptSharedEnvironment{table: plannedTable, known: make(map[string]struct{})}
	if function, present := table.values["&function"]; present {
		planned.putLocked(String("&function"), function)
	}
	for _, name := range names {
		name = strings.TrimPrefix(name, "&")
		planned.putLocked(String("&"+name), FunctionValue(&portableSharedRuntimeCallable{name: name}))
		planned.known[name] = struct{}{}
	}
	planned.installDefaultIntrospectionBridgesLocked(environments, taintMode)
	planned.putLocked(String("(isloaded)"), ObjectValue(loader))

	if shared.known == nil {
		shared.known = make(map[string]struct{})
	}
	for name := range planned.known {
		shared.known[name] = struct{}{}
	}
	for _, key := range plannedTable.keys {
		shared.putLocked(plannedTable.keyValues[key], plannedTable.values[key])
	}
	table.mu.Unlock()
	return nil
}

// installDefaultIntrospectionBridgesLocked materializes the non-function
// entries installed by Sleep 2.1's default global bridges. These objects model
// only the interfaces OPFOR can implement truthfully; arbitrary JVM bridge
// classes and their java.util.Stack ABIs remain importer-owned.
func (shared *portableScriptSharedEnvironment) installDefaultIntrospectionBridgesLocked(environments map[string]EnvironmentKind, taintMode bool) {
	if shared == nil || shared.table == nil {
		return
	}
	putAliases := func(value Value, keys ...string) {
		for _, key := range keys {
			shared.putLocked(String(key), value)
		}
	}

	// BasicNumbers implements Function, Predicate, and Operator. With Sleep's
	// default (untainted) mode the same bridge object is stored for every entry.
	numbers := ObjectValue(newPortableScriptBridge("BasicNumbers", portableBridgeFunction|portableBridgePredicate|portableBridgeOperator))
	numberOperators := numbers
	if taintMode {
		// Sleep's Sanitizer wrapper implements Function and Operator but not
		// Predicate. Predicate keys still point at the underlying BasicNumbers.
		numberOperators = ObjectValue(newPortableScriptBridge("Sanitizer(BasicNumbers)", portableBridgeFunction|portableBridgeOperator))
	}
	// BasicNumbers publishes its ordinary functions and arithmetic operators
	// through the same (possibly taint-mode Sanitizer-wrapped) bridge object.
	// Preserve that identity because Sleep's `is` operator can observe it through
	// ScriptEnvironment's typed getters.
	for _, key := range []string{
		"&abs", "&acos", "&asin", "&atan", "&atan2", "&ceil", "&cos",
		"&log", "&round", "&sin", "&sqrt", "&tan", "&radians",
		"&degrees", "&exp", "&floor", "&sum", "&double", "&int",
		"&uint", "&long", "&parseNumber", "&formatNumber", "&rand", "&srand",
	} {
		shared.putLocked(String(key), numberOperators)
	}
	for _, key := range []string{"+", "-", "/", "*", "**", "% ", "<<", ">>", "&", "|", "^", "&not"} {
		shared.putLocked(String(key), numberOperators)
	}
	for _, key := range []string{"==", "!=", "<=", ">=", "<", ">", "is"} {
		shared.putLocked(String(key), numbers)
	}

	stringsBridge := ObjectValue(newPortableScriptBridge("BasicStrings", portableBridgePredicate))
	// BasicStrings creates distinct helper objects for most functions, but these
	// source-defined aliases share one helper instance in each group.
	putAliases(ObjectValue(newPortableScriptBridge("BasicStrings.charAt", portableBridgeFunction)), "&charAt", "&byteAt")
	putAliases(ObjectValue(newPortableScriptBridge("BasicStrings.substr", portableBridgeFunction)), "&substr", "&mid")
	putAliases(ObjectValue(newPortableScriptBridge("BasicStrings.indexOf", portableBridgeFunction)), "&indexOf", "&lindexOf")
	putAliases(ObjectValue(newPortableScriptBridge("BasicStrings.sorters", portableBridgeFunction)), "&sorta", "&sortn", "&sortd")
	for _, key := range []string{"eq", "ne", "lt", "gt", "-isletter", "-isnumber", "-isupper", "-islower", "isin"} {
		shared.putLocked(String(key), stringsBridge)
	}
	shared.putLocked(String("iswm"), ObjectValue(newPortableScriptBridge("BasicStrings.iswm", portableBridgePredicate)))
	for _, key := range []string{".", "x", "cmp", "<=>"} {
		shared.putLocked(String(key), ObjectValue(newPortableScriptBridge("BasicStrings."+key, portableBridgeOperator)))
	}

	utilities := ObjectValue(newPortableScriptBridge("BasicUtilities", portableBridgeFunction|portableBridgePredicate))
	putAliases(utilities,
		"&concat", "&keys", "&size", "&push", "&pop", "&add", "&flatten",
		"&clear", "&splice", "&subarray", "&sublist", "&setRemovalPolicy",
		"&setMissPolicy", "&putAll", "&addAll", "&removeAll", "&retainAll",
		"&pushl", "&popl", "&search", "&reduce", "&values", "&remove",
		"&setField", "&typeOf", "&newInstance", "&scalar", "&exit", "&watch",
		"&debug", "&warn", "&profile", "&getStackTrace", "&checkError", "&invoke",
		"&inline",
	)
	putAliases(ObjectValue(newPortableScriptBridge("BasicUtilities.array", portableBridgeFunction)), "&array", "&@")
	putAliases(ObjectValue(newPortableScriptBridge("BasicUtilities.hash", portableBridgeFunction)), "&hash", "&ohash", "&ohasha", "&%")
	putAliases(ObjectValue(newPortableScriptBridge("BasicUtilities.map", portableBridgeFunction)), "&map", "&filter")
	putAliases(ObjectValue(newPortableScriptBridge("BasicUtilities.cast", portableBridgeFunction)), "&cast", "&casti")
	putAliases(ObjectValue(newPortableScriptBridge("BasicUtilities.scope", portableBridgeFunction)), "&local", "&this", "&global")
	putAliases(ObjectValue(newPortableScriptBridge("BasicUtilities.sync", portableBridgeFunction)), "&semaphore", "&acquire", "&release")
	lambda := ObjectValue(newPortableScriptBridge("BasicUtilities.lambda", portableBridgeFunction))
	putAliases(lambda, "&lambda", "&let")
	if !taintMode {
		// Sensitive/Tainter/Sanitizer are identity-preserving when taint mode is
		// disabled. With tainting enabled Sleep creates distinct wrappers for these
		// entries, so retain their separately materialized callables in that mode.
		putAliases(utilities, "&untaint", "&taint")
		putAliases(lambda, "&compile_closure")
		putAliases(ObjectValue(newPortableScriptBridge("BasicUtilities.eval", portableBridgeFunction)), "&eval", "&expr")
		putAliases(ObjectValue(newPortableScriptBridge("BasicUtilities.use", portableBridgeFunction)), "&use", "&include")
	}
	for _, key := range []string{"-istrue", "-isarray", "-ishash", "-isfunction", "-istainted", "isa", "in", "=~"} {
		shared.putLocked(String(key), utilities)
	}
	shared.putLocked(String("=>"), ObjectValue(newPortableScriptBridge("BasicUtilities.=>", portableBridgeOperator)))
	// BasicUtilities installs this compiler special form as the exact same
	// Function object as &function.
	function, present := shared.table.values["&function"]
	if !present {
		function = FunctionValue(&portableSharedRuntimeCallable{name: "function"})
		shared.putLocked(String("&function"), function)
		shared.known["function"] = struct{}{}
	}
	shared.putLocked(String("function"), function)
	if !taintMode {
		shared.putLocked(String("&setf"), function)
	}

	shared.putLocked(String("__EXEC__"), FunctionValue(&portableSharedRuntimeCallable{name: "exec"}))
	shared.putLocked(String("-eof"), ObjectValue(newPortableScriptBridge("BasicIO.-eof", portableBridgePredicate)))
	basicIO := ObjectValue(newPortableScriptBridge("BasicIO", portableBridgeFunction))
	putAliases(basicIO,
		"&allocate", "&writeObject", "&writeAsObject", "&sizeof", "&wait",
		"&setEncoding", "&checksum", "&digest",
	)
	putAliases(ObjectValue(newPortableScriptBridge("BasicIO.consume", portableBridgeFunction)), "&consume", "&skip")
	putAliases(ObjectValue(newPortableScriptBridge("BasicIO.println", portableBridgeFunction)), "&println", "&printf")
	if !taintMode {
		putAliases(basicIO, "__EXEC__", "&readc", "&readObject", "&readAsObject")
		putAliases(ObjectValue(newPortableScriptBridge("BasicIO.socket", portableBridgeFunction)), "&connect", "&listen")
	}
	fileSystem := ObjectValue(newPortableScriptBridge("FileSystemBridge", portableBridgeFunction|portableBridgePredicate))
	// FileSystemBridge installs these functions and predicates as the same
	// object. Its filename/listing helper classes are intentionally left as their
	// separate function values below, except for the source-defined ls/listRoots
	// alias pair.
	for _, key := range []string{
		"&createNewFile", "&deleteFile", "&chdir", "&cwd",
		"&getCurrentDirectory", "&mkdir", "&rename", "&setLastModified",
		"&setReadOnly",
	} {
		shared.putLocked(String(key), fileSystem)
	}
	for _, key := range []string{"-exists", "-canread", "-canwrite", "-isDir", "-isFile", "-isHidden"} {
		shared.putLocked(String(key), fileSystem)
	}
	fileListing := ObjectValue(newPortableScriptBridge("FileSystemBridge.listFiles", portableBridgeFunction))
	shared.putLocked(String("&ls"), fileListing)
	shared.putLocked(String("&listRoots"), fileListing)

	defaultEnvironment := ObjectValue(newPortableScriptBridge("DefaultEnvironment", portableBridgeEnvironment))
	for _, spec := range envspec.Builtins() {
		shared.putLocked(String(spec.Keyword), defaultEnvironment)
	}

	regex := ObjectValue(newPortableScriptBridge("RegexBridge.isMatch", portableBridgeFunction|portableBridgePredicate))
	shared.putLocked(String("ismatch"), regex)
	shared.putLocked(String("hasmatch"), regex)
	shared.putLocked(String("&matched"), regex)

	keywords := make([]string, 0, len(environments))
	for keyword := range environments {
		keywords = append(keywords, keyword)
	}
	sort.Strings(keywords)
	for _, keyword := range keywords {
		interfaces := portableBridgeEnvironment
		switch environments[keyword] {
		case EnvironmentFilter:
			interfaces = portableBridgeFilterEnvironment
		case EnvironmentPredicate:
			interfaces = portableBridgePredicateEnvironment
		}
		shared.putLocked(String(keyword), ObjectValue(newPortableScriptBridge("importer."+keyword, interfaces)))
	}
}

type portableScriptBridgeInterfaces uint8

const (
	portableBridgeFunction portableScriptBridgeInterfaces = 1 << iota
	portableBridgePredicate
	portableBridgeOperator
	portableBridgeEnvironment
	portableBridgePredicateEnvironment
	portableBridgeFilterEnvironment
)

// portableScriptBridge is a bounded pure-Go representation of a bridge entry
// that OPFOR itself implements. It deliberately does not manufacture JVM
// Scalar, Stack, Check, or Loadable objects; its surface is limited to
// identity, interface checks, and ScriptEnvironment's typed getters.
type portableScriptBridge struct {
	name       string
	interfaces portableScriptBridgeInterfaces
}

func newPortableScriptBridge(name string, interfaces portableScriptBridgeInterfaces) *portableScriptBridge {
	return &portableScriptBridge{name: name, interfaces: interfaces}
}

func (bridge *portableScriptBridge) String() string {
	if bridge == nil || bridge.name == "" {
		return "<sleep bridge>"
	}
	return "<sleep bridge " + bridge.name + ">"
}

func (bridge *portableScriptBridge) supports(class string) bool {
	if bridge == nil {
		return false
	}
	class = resolvePortableClassName(class)
	switch class {
	case "java.lang.Object":
		return true
	case "sleep.interfaces.Function":
		return bridge.interfaces&portableBridgeFunction != 0
	case "sleep.interfaces.Predicate":
		return bridge.interfaces&portableBridgePredicate != 0
	case "sleep.interfaces.Operator":
		return bridge.interfaces&portableBridgeOperator != 0
	case "sleep.interfaces.Environment":
		return bridge.interfaces&portableBridgeEnvironment != 0
	case "sleep.interfaces.PredicateEnvironment":
		return bridge.interfaces&portableBridgePredicateEnvironment != 0
	case "sleep.interfaces.FilterEnvironment":
		return bridge.interfaces&portableBridgeFilterEnvironment != 0
	default:
		return false
	}
}

func (bridge *portableScriptBridge) primaryClass() string {
	if bridge == nil {
		return "java.lang.Object"
	}
	for _, candidate := range []struct {
		flag  portableScriptBridgeInterfaces
		class string
	}{
		{portableBridgePredicate, "sleep.interfaces.Predicate"},
		{portableBridgeOperator, "sleep.interfaces.Operator"},
		{portableBridgeEnvironment, "sleep.interfaces.Environment"},
		{portableBridgePredicateEnvironment, "sleep.interfaces.PredicateEnvironment"},
		{portableBridgeFilterEnvironment, "sleep.interfaces.FilterEnvironment"},
		{portableBridgeFunction, "sleep.interfaces.Function"},
	} {
		if bridge.interfaces&candidate.flag != 0 {
			return candidate.class
		}
	}
	return "java.lang.Object"
}

func (bridge *portableScriptBridge) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		return Bool(bridge.supports(invocation.Class)), true, nil
	}
	if invocation.Op == ObjectInvoke && invocation.Message == "toString" && len(invocation.Arguments) == 0 {
		return String(bridge.String()), true, nil
	}
	return Null(), false, nil
}

func (shared *portableScriptSharedEnvironment) publish(runtime *Runtime, name string, callable Callable) error {
	if shared == nil || shared.table == nil {
		return nil
	}
	name = strings.TrimPrefix(name, "&")
	table := shared.table
	table.mu.Lock()
	if shared.known == nil {
		shared.known = make(map[string]struct{})
	}
	if callable == nil {
		shared.removeLocked("&" + name)
	} else {
		if _, present := table.values["&"+name]; !present {
			if err := reserveCollectionEntries(runtime, 1); err != nil {
				table.mu.Unlock()
				return err
			}
		}
		shared.putLocked(String("&"+name), FunctionValue(callable))
	}
	shared.known[name] = struct{}{}
	table.mu.Unlock()
	return nil
}

// functionEntry snapshots the exact value currently published for one
// function name. Script-scoped native bridges retain this value so logical
// ScriptLoader unload can reveal the entry they shadowed, including the
// object-valued default Sleep bridges which are not representable as Callable.
func (shared *portableScriptSharedEnvironment) functionEntry(name string) (Value, bool) {
	if shared == nil || shared.table == nil {
		return Null(), false
	}
	name = strings.TrimPrefix(name, "&")
	shared.table.mu.RLock()
	value, present := shared.table.values["&"+name]
	shared.table.mu.RUnlock()
	return value, present
}

// restoreFunctionIfCurrent replaces expected only when it is still the
// published entry. Raw Sleep closures remain runnable across logical unload
// and may legitimately replace a function while generation cleanup drains;
// the identity check prevents cleanup from overwriting that later mutation.
func (shared *portableScriptSharedEnvironment) restoreFunctionIfCurrent(
	name string,
	expected Callable,
	replacement Value,
	hasReplacement bool,
) {
	if shared == nil || shared.table == nil || expected == nil {
		return
	}
	name = strings.TrimPrefix(name, "&")
	table := shared.table
	table.mu.Lock()
	current, present := table.values["&"+name]
	callable, callableOK := current.Function()
	if !present || !callableOK || !samePortableCallable(callable, expected) {
		table.mu.Unlock()
		return
	}
	if hasReplacement {
		shared.putLocked(String("&"+name), replacement)
	} else {
		shared.removeLocked("&" + name)
	}
	if shared.known == nil {
		shared.known = make(map[string]struct{})
	}
	shared.known[name] = struct{}{}
	table.mu.Unlock()
}

func (shared *portableScriptSharedEnvironment) publishInline(runtime *Runtime, name string, closure *scriptClosure) error {
	if shared == nil || shared.table == nil {
		return nil
	}
	name = strings.TrimPrefix(name, "&")
	table := shared.table
	table.mu.Lock()
	if shared.known == nil {
		shared.known = make(map[string]struct{})
	}
	if closure == nil || closure.function == nil {
		shared.removeLocked("^&" + name)
	} else {
		if _, present := table.values["^&"+name]; !present {
			if err := reserveCollectionEntries(runtime, 1); err != nil {
				table.mu.Unlock()
				return err
			}
		}
		source := Source{Name: closure.function.Span.Source}
		var sourceAccount *runtimeResourceAccount
		if closure.script != nil && closure.script.program != nil {
			source = closure.script.program.source
			sourceAccount = closure.script.program.sourceAccount
		}
		program := &Program{source: source, function: closure.function, sourceAccount: sourceAccount}
		shared.putLocked(String("^&"+name), ObjectValue(&portableCompiledBlock{program: program}))
	}
	shared.known[name] = struct{}{}
	table.mu.Unlock()
	return nil
}

func (shared *portableScriptSharedEnvironment) resolve(name string, caller *Script) Callable {
	if shared == nil || shared.table == nil {
		return nil
	}
	name = strings.TrimPrefix(name, "&")
	table := shared.table
	table.mu.RLock()
	blockValue, blockPresent := table.values["^&"+name]
	value, present := table.values["&"+name]
	table.mu.RUnlock()
	// sleep.engine.atoms.Call asks getFunction before getBlock, so an ordinary
	// function wins when both &name and ^&name exist. A wrong-typed &name also
	// prevents fallback because Java's getFunction cast throws first.
	if present {
		callable, ok := value.Function()
		if ok {
			if _, runtimeMarker := callable.(*portableSharedRuntimeCallable); runtimeMarker && caller != nil && caller.runtime != nil {
				if family, familyOK := sleepBuiltinFamilyForFunction(name); familyOK &&
					caller.runtime.hasStockFunction(name) && !caller.runtime.hasExplicitFunction(name) {
					return newSleepBuiltinFunctionCallable(name, family, caller.runtime, caller.id, Span{})
				}
			}
			return callable
		}
		if object, objectOK := value.Object(); objectOK {
			if bridge, bridgeOK := object.(*portableScriptBridge); bridgeOK && bridge.supports("sleep.interfaces.Function") {
				if family, familyOK := sleepBuiltinFamilyForFunction(name); familyOK &&
					sleepBuiltinFamilyMatchesBridge(family, bridge.name) && caller != nil && caller.runtime != nil &&
					(caller.runtime.hasStockFunction(name) ||
						(family == sleepBuiltinFamilyUtilities && intrinsicUtilitiesFunction(name))) &&
					!caller.runtime.hasExplicitFunction(name) {
					return newSleepBuiltinFunctionCallable(name, family, caller.runtime, caller.id, Span{})
				}
				return &portableSharedRuntimeCallable{name: name}
			}
		}
		return nil
	}
	if blockPresent {
		if block, ok := portableCompiledBlockValue(blockValue); ok && caller != nil {
			return caller.newInline(block.program.function, caller.globals)
		}
	}
	return nil
}

func (shared *portableScriptSharedEnvironment) removed(name string) bool {
	if shared == nil || shared.table == nil {
		return false
	}
	name = strings.TrimPrefix(name, "&")
	table := shared.table
	table.mu.RLock()
	_, known := shared.known[name]
	function, functionPresent := table.values["&"+name]
	block, blockPresent := table.values["^&"+name]
	table.mu.RUnlock()
	if !known {
		return false
	}
	if functionPresent {
		if _, callable := function.Function(); callable {
			return false
		}
		if object, objectOK := function.Object(); objectOK {
			if bridge, bridgeOK := object.(*portableScriptBridge); bridgeOK && bridge.supports("sleep.interfaces.Function") {
				return false
			}
		}
		return true
	}
	if blockPresent {
		_, valid := portableCompiledBlockValue(block)
		return !valid
	}
	return true
}

func (shared *portableScriptSharedEnvironment) putLocked(key, value Value) {
	text, keyValue := sleepHashKey(key)
	if shared.plannedKeys != nil {
		shared.plannedKeys[text] = struct{}{}
		return
	}
	table := shared.table
	if table == nil {
		return
	}
	if _, present := table.values[text]; !present {
		table.keys = append(table.keys, text)
		table.keyValues[text] = keyValue
		table.entries[text] = &portableJavaMapEntry{
			mapping: table, key: text, keyValue: keyValue, value: value,
		}
		table.mod++
	} else if entry := table.entries[text]; entry != nil {
		entry.mu.Lock()
		entry.value = value
		entry.mu.Unlock()
	}
	table.values[text] = value
}

func (shared *portableScriptSharedEnvironment) removeLocked(key string) {
	_, _ = shared.table.removeKeyLocked(key)
}

// portableSharedRuntimeCallable makes the functions installed by OPFOR's
// global bridge set visible in the shared Hashtable without binding them to a
// parent Runtime or Script. Named calls retain pass-by-name arguments through
// invokeArgumentsAt; closure-style invoke receives resolved Values, matching
// Sleep's ordinary Callable boundary.
type portableSharedRuntimeCallable struct {
	name string
}

type portableArgumentCallable interface {
	Callable
	invokeArgumentsAt(context.Context, Span, []Argument) (Value, error)
}

func (callable *portableSharedRuntimeCallable) String() string {
	if callable == nil {
		return "&"
	}
	return "&" + callable.name
}

func (callable *portableSharedRuntimeCallable) Invoke(ctx context.Context, values ...Value) (Value, error) {
	arguments := make([]Argument, len(values))
	for index, value := range values {
		arguments[index] = Argument{Value: value}
	}
	return callable.invokeArgumentsAt(ctx, Span{}, arguments)
}

func (callable *portableSharedRuntimeCallable) invokeArgumentsAt(ctx context.Context, span Span, arguments []Argument) (Value, error) {
	if callable == nil || callable.name == "" {
		return Null(), errors.New("opfor: invalid shared environment function")
	}
	fiber := currentFiber(ctx)
	if fiber == nil || fiber.closure == nil || fiber.closure.script == nil || fiber.closure.script.runtime == nil {
		return Null(), errors.New("opfor: shared environment function requires an active script")
	}
	script := fiber.closure.script
	return script.runtime.invoke(ctx, Invocation{
		Runtime: script.runtime, Script: script.id, Name: callable.name,
		Span: span, Arguments: arguments,
	})
}
