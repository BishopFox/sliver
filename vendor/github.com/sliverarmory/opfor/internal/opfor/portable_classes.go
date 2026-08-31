package opfor

import "strings"

// Sleep's parser starts with java.lang.*, java.util.*, and sleep.runtime.* in
// its ImportManager. Pure Go cannot ask a JVM class loader to resolve a token,
// so this table covers the portable classes and interfaces the runtime models.
var portableDefaultClasses = map[string]string{
	"ArrayList":                  "java.util.ArrayList",
	"Appendable":                 "java.lang.Appendable",
	"Arrays":                     "java.util.Arrays",
	"AutoCloseable":              "java.lang.AutoCloseable",
	"Boolean":                    "java.lang.Boolean",
	"Character":                  "java.lang.Character",
	"CharSequence":               "java.lang.CharSequence",
	"Class":                      "java.lang.Class",
	"Cloneable":                  "java.lang.Cloneable",
	"Collection":                 "java.util.Collection",
	"Collections":                "java.util.Collections",
	"Comparable":                 "java.lang.Comparable",
	"Deque":                      "java.util.Deque",
	"Double":                     "java.lang.Double",
	"Enumeration":                "java.util.Enumeration",
	"Exception":                  "java.lang.Exception",
	"HashMap":                    "java.util.HashMap",
	"HashSet":                    "java.util.HashSet",
	"Hashtable":                  "java.util.Hashtable",
	"Integer":                    "java.lang.Integer",
	"Iterable":                   "java.lang.Iterable",
	"Iterator":                   "java.util.Iterator",
	"LinkedHashMap":              "java.util.LinkedHashMap",
	"LinkedHashSet":              "java.util.LinkedHashSet",
	"LinkedList":                 "java.util.LinkedList",
	"List":                       "java.util.List",
	"ListIterator":               "java.util.ListIterator",
	"Long":                       "java.lang.Long",
	"Locale":                     "java.util.Locale",
	"Map":                        "java.util.Map",
	"Math":                       "java.lang.Math",
	"MessageDigest":              "java.security.MessageDigest",
	"NavigableMap":               "java.util.NavigableMap",
	"NavigableSet":               "java.util.NavigableSet",
	"Number":                     "java.lang.Number",
	"Object":                     "java.lang.Object",
	"PrimitiveIterator":          "java.util.PrimitiveIterator",
	"PrimitiveIterator$OfDouble": "java.util.PrimitiveIterator$OfDouble",
	"PrimitiveIterator$OfInt":    "java.util.PrimitiveIterator$OfInt",
	"PrimitiveIterator$OfLong":   "java.util.PrimitiveIterator$OfLong",
	"Queue":                      "java.util.Queue",
	"Random":                     "java.util.Random",
	"Runnable":                   "java.lang.Runnable",
	"RuntimeException":           "java.lang.RuntimeException",
	"ScriptEnvironment":          "sleep.runtime.ScriptEnvironment",
	"ScriptInstance":             "sleep.runtime.ScriptInstance",
	"ScriptLoader":               "sleep.runtime.ScriptLoader",
	"ScriptVariables":            "sleep.runtime.ScriptVariables",
	"ScriptWarning":              "sleep.error.ScriptWarning",
	"Set":                        "java.util.Set",
	"SleepUtils":                 "sleep.runtime.SleepUtils",
	"SortedMap":                  "java.util.SortedMap",
	"SortedSet":                  "java.util.SortedSet",
	"String":                     "java.lang.String",
	"StringBuffer":               "java.lang.StringBuffer",
	"StringBuilder":              "java.lang.StringBuilder",
	"StringTokenizer":            "java.util.StringTokenizer",
	"System":                     "java.lang.System",
	"Thread":                     "java.lang.Thread",
	"Throwable":                  "java.lang.Throwable",
	"TreeMap":                    "java.util.TreeMap",
	"TreeSet":                    "java.util.TreeSet",
	"UUID":                       "java.util.UUID",
}

// portableImportedClasses records classes whose package is needed to reproduce
// Sleep ImportManager wildcard resolution without a JVM class loader. Some
// entries have a bounded portable implementation; the ObjectHost remains
// responsible for every operation outside that implementation.
//
// JLabel is exercised by the pinned official mouse.cna example, which imports
// java.awt.*, javax.swing.*, and javax.swing.event.* in that order. Selecting
// the first wildcard blindly manufactures java.awt.JLabel, a class which does
// not exist, instead of the source's javax.swing.JLabel. CommandBuilder is the
// analogous case in official oneliner.cna: common.* precedes beacon.*, while
// the instantiated class is beacon.CommandBuilder.
var portableImportedClasses = map[string]struct{}{
	"beacon.CommandBuilder":            {},
	"java.io.File":                     {},
	"java.io.FileFilter":               {},
	"java.io.FilenameFilter":           {},
	"java.net.URI":                     {},
	"java.net.URL":                     {},
	"java.lang.reflect.Proxy":          {},
	"java.nio.file.Path":               {},
	"java.util.function.Function":      {},
	"java.util.stream.BaseStream":      {},
	"java.util.stream.DoubleStream":    {},
	"java.util.stream.IntStream":       {},
	"java.util.stream.LongStream":      {},
	"java.util.stream.Stream":          {},
	"java.util.random.RandomGenerator": {},
	"javax.swing.JLabel":               {},
}

var portableJavaInterfaces = map[string]struct{}{
	"java.io.FileFilter":                    {},
	"java.io.FilenameFilter":                {},
	"java.io.Serializable":                  {},
	"java.lang.Appendable":                  {},
	"java.lang.AutoCloseable":               {},
	"java.lang.CharSequence":                {},
	"java.lang.Cloneable":                   {},
	"java.lang.Comparable":                  {},
	"java.lang.Iterable":                    {},
	"java.lang.Runnable":                    {},
	"java.nio.file.Path":                    {},
	"java.nio.file.Watchable":               {},
	"java.util.Collection":                  {},
	"java.util.Deque":                       {},
	"java.util.Enumeration":                 {},
	"java.util.Iterator":                    {},
	"java.util.List":                        {},
	"java.util.ListIterator":                {},
	"java.util.Map":                         {},
	"java.util.NavigableMap":                {},
	"java.util.NavigableSet":                {},
	"java.util.Queue":                       {},
	"java.util.PrimitiveIterator":           {},
	"java.util.PrimitiveIterator$OfDouble":  {},
	"java.util.PrimitiveIterator$OfInt":     {},
	"java.util.PrimitiveIterator$OfLong":    {},
	"java.util.Set":                         {},
	"java.util.SortedMap":                   {},
	"java.util.SortedSet":                   {},
	"java.util.random.RandomGenerator":      {},
	"java.util.function.Function":           {},
	"java.util.stream.BaseStream":           {},
	"java.util.stream.DoubleStream":         {},
	"java.util.stream.IntStream":            {},
	"java.util.stream.LongStream":           {},
	"java.util.stream.Stream":               {},
	"sleep.interfaces.Environment":          {},
	"sleep.interfaces.FilterEnvironment":    {},
	"sleep.interfaces.Function":             {},
	"sleep.interfaces.Loadable":             {},
	"sleep.interfaces.Operator":             {},
	"sleep.interfaces.Predicate":            {},
	"sleep.interfaces.PredicateEnvironment": {},
	"sleep.interfaces.Variable":             {},
}

func resolvePortableClassName(name string) string {
	if resolved := portableDefaultClasses[name]; resolved != "" {
		return resolved
	}
	return name
}

func (class classReference) String() string {
	name := string(class)
	prefix := "class "
	if _, ok := portableJavaInterfaces[name]; ok {
		prefix = "interface "
	}
	return prefix + name
}

type portableJavaMethod string

func (method portableJavaMethod) String() string { return string(method) }

var portableMapMethods = []string{
	"public abstract java.lang.Object java.util.Map.remove(java.lang.Object)",
	"public abstract java.lang.Object java.util.Map.get(java.lang.Object)",
	"public abstract java.lang.Object java.util.Map.put(java.lang.Object,java.lang.Object)",
	"public abstract boolean java.util.Map.equals(java.lang.Object)",
	"public abstract java.util.Collection java.util.Map.values()",
	"public abstract int java.util.Map.hashCode()",
	"public abstract void java.util.Map.clear()",
	"public abstract boolean java.util.Map.isEmpty()",
	"public abstract int java.util.Map.size()",
	"public abstract java.util.Set java.util.Map.entrySet()",
	"public abstract void java.util.Map.putAll(java.util.Map)",
	"public abstract java.util.Set java.util.Map.keySet()",
	"public abstract boolean java.util.Map.containsValue(java.lang.Object)",
	"public abstract boolean java.util.Map.containsKey(java.lang.Object)",
}

func portableJavaClassTarget(target any, invocation ObjectInvocation) (Value, bool, error) {
	var class string
	switch reference := target.(type) {
	case classReference:
		class = string(reference)
	case sleepClass:
		class = string(reference)
	default:
		return Null(), false, nil
	}
	class = resolvePortableClassName(class)
	if invocation.Op == ObjectTypeCheck {
		targetClass := resolvePortableClassName(invocation.Class)
		return Bool(targetClass == "java.lang.Class" || targetClass == "java.lang.Object"), true, nil
	}
	if invocation.Op != ObjectInvoke && invocation.Op != ObjectGet || len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "getName":
		return String(class), true, nil
	case "isInterface":
		_, isInterface := portableJavaInterfaces[class]
		return Bool(isInterface), true, nil
	case "toString":
		return String(classReference(class).String()), true, nil
	case "getMethods":
		if class != "java.util.Map" {
			return Null(), false, nil
		}
		methods := make([]Value, len(portableMapMethods))
		for index, method := range portableMapMethods {
			methods[index] = ObjectValue(portableJavaMethod(method))
		}
		array, err := newRuntimeArray(invocation.Runtime, methods...)
		if err != nil {
			return Null(), true, err
		}
		return ArrayValue(array), true, nil
	}
	return Null(), false, nil
}

func portableClassOperand(value Value) (string, bool) {
	// BridgeUtilities.getClass returns its nil default for Sleep's empty scalar;
	// BasicUtilities then reports a simple false predicate without a cast.
	if value.IsNull() {
		return "", true
	}
	object, ok := value.Object()
	if !ok {
		return "", false
	}
	switch class := object.(type) {
	case classReference:
		return string(class), true
	case sleepClass:
		return string(class), true
	default:
		return "", false
	}
}

func portableJavaAssignable(actual, target string) bool {
	actual = resolvePortableClassName(actual)
	target = resolvePortableClassName(target)
	if actual == target || target == "java.lang.Object" {
		return true
	}
	switch actual {
	case "java.lang.Byte", "java.lang.Short", "java.lang.Integer", "java.lang.Long", "java.lang.Float", "java.lang.Double":
		return target == "java.lang.Number" || target == "java.lang.Comparable" || target == "java.io.Serializable"
	case "java.lang.Boolean", "java.lang.Character":
		return target == "java.lang.Comparable" || target == "java.io.Serializable"
	case "java.lang.String":
		return target == "java.lang.CharSequence" || target == "java.lang.Comparable" || target == "java.io.Serializable"
	case "java.lang.StringBuilder", "java.lang.StringBuffer":
		return target == "java.lang.Appendable" || target == "java.lang.CharSequence" || target == "java.lang.Comparable" || target == "java.io.Serializable"
	case "java.lang.Class":
		return target == "java.lang.reflect.AnnotatedElement" || target == "java.lang.reflect.GenericDeclaration" || target == "java.lang.reflect.Type" || target == "java.io.Serializable"
	case "java.io.File":
		return target == "java.lang.Comparable" || target == "java.io.Serializable"
	case "java.net.URI":
		return target == "java.lang.Comparable" || target == "java.io.Serializable"
	case "java.net.URL":
		return target == "java.io.Serializable"
	case "sun.nio.fs.UnixPath", "sun.nio.fs.WindowsPath":
		return target == "java.nio.file.Path" || target == "java.nio.file.Watchable" ||
			target == "java.lang.Comparable" || target == "java.lang.Iterable"
	case "java.util.ArrayList":
		return target == "java.util.List" || target == "java.util.Collection" || target == "java.lang.Iterable" || target == "java.io.Serializable"
	case "java.util.LinkedList":
		return target == "java.util.List" || target == "java.util.Deque" || target == "java.util.Queue" || target == "java.util.Collection" || target == "java.lang.Iterable" || target == "java.io.Serializable"
	case "java.util.HashSet", "java.util.LinkedHashSet", "java.util.TreeSet":
		return target == "java.util.Set" || target == "java.util.Collection" || target == "java.lang.Iterable" || target == "java.io.Serializable"
	case "java.util.HashMap", "java.util.Hashtable", "java.util.LinkedHashMap", "java.util.TreeMap":
		return target == "java.util.Map" || target == "java.io.Serializable"
	case "java.util.Random":
		return target == "java.io.Serializable" || target == "java.util.random.RandomGenerator"
	case "java.util.UUID":
		return target == "java.lang.Comparable" || target == "java.io.Serializable"
	case "java.util.Locale":
		return target == "java.lang.Cloneable" || target == "java.io.Serializable"
	case "java.util.stream.ReferencePipeline$Head":
		return target == "java.util.stream.Stream" || target == "java.util.stream.BaseStream" || target == "java.lang.AutoCloseable"
	case "java.util.stream.IntPipeline$Head":
		return target == "java.util.stream.IntStream" || target == "java.util.stream.BaseStream" || target == "java.lang.AutoCloseable"
	case "java.util.stream.LongPipeline$Head":
		return target == "java.util.stream.LongStream" || target == "java.util.stream.BaseStream" || target == "java.lang.AutoCloseable"
	case "java.util.stream.DoublePipeline$Head":
		return target == "java.util.stream.DoubleStream" || target == "java.util.stream.BaseStream" || target == "java.lang.AutoCloseable"
	case "java.util.Spliterators$1Adapter":
		return target == "java.util.Iterator"
	case "java.util.Spliterators$2Adapter":
		return target == "java.util.PrimitiveIterator$OfInt" || target == "java.util.PrimitiveIterator" || target == "java.util.Iterator"
	case "java.util.Spliterators$3Adapter":
		return target == "java.util.PrimitiveIterator$OfLong" || target == "java.util.PrimitiveIterator" || target == "java.util.Iterator"
	case "java.util.Spliterators$4Adapter":
		return target == "java.util.PrimitiveIterator$OfDouble" || target == "java.util.PrimitiveIterator" || target == "java.util.Iterator"
	case "sleep.runtime.ScriptInstance":
		return target == "java.lang.Runnable" || target == "java.io.Serializable"
	case "sleep.bridges.SleepClosure":
		return target == "sleep.interfaces.Function" || target == "java.lang.Runnable" || target == "java.io.Serializable"
	case "sleep.engine.Block", "sleep.runtime.ScriptEnvironment", "sleep.runtime.ScriptVariables":
		return target == "java.io.Serializable"
	case "java.io.InputStream":
		return target == "java.io.Closeable" || target == "java.lang.AutoCloseable"
	default:
		if strings.HasPrefix(actual, "[") {
			return target == "java.lang.Cloneable" || target == "java.io.Serializable"
		}
		return false
	}
}

type portableInvalidClassCastError struct {
	actual string
}

func (e *portableInvalidClassCastError) Error() string {
	actual := "java.lang.Object"
	if e != nil && e.actual != "" {
		actual = e.actual
	}
	return "attempted an invalid cast: " + actual + " cannot be cast to java.lang.Class"
}

func newPortableInvalidClassCast(value Value) error {
	class, ok := portableObjectClass(value)
	if !ok {
		class = "java.lang.Object"
	}
	return &portableInvalidClassCastError{actual: class}
}
