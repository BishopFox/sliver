package opfor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/sliverarmory/opfor/internal/bytecode"
)

// defaultObjectHost gives an importer first refusal on every object request,
// then supplies the small set of java.lang operations that Sleep scripts use
// as scalar helpers. Unknown Java, Swing, and Cobalt objects remain explicit
// UnsupportedErrors.
type defaultObjectHost struct {
	runtime *Runtime
	primary ObjectHost
}

// portableObjectCallbackError marks an error returned through a pure-Go
// callback invoked by a portable Java method. Only a Sleep thrown value models
// a Java exception at that boundary; importer and execution errors remain
// authoritative when control returns through defaultObjectHost.
type portableObjectCallbackError struct {
	cause error
}

func (err *portableObjectCallbackError) Error() string {
	if err == nil || err.cause == nil {
		return ""
	}
	return err.cause.Error()
}

func (err *portableObjectCallbackError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.cause
}

func (h defaultObjectHost) Object(ctx context.Context, invocation ObjectInvocation) (result Value, resultErr error) {
	runtime := h.runtime
	if runtime == nil {
		runtime = invocation.Runtime
	}
	if invocation.Runtime == nil {
		invocation.Runtime = runtime
	}
	if generation := stampObjectInvocationGeneration(ctx, runtime, &invocation); generation != nil {
		var release func() error
		var err error
		ctx, release, err = generation.script.acquireGenerationExecution(ctx, generation)
		if err != nil {
			return Null(), err
		}
		defer func() { resultErr = joinExecutionError(resultErr, release) }()
	}
	caller := currentFiber(ctx)
	profileFrame := caller.beginProfileCall(portableObjectProfileName(invocation))
	defer func() { caller.finishProfileCall(profileFrame, resultErr) }()
	primary := h.primary
	if primary == nil {
		primary = unsupportedObjectHost{}
	}
	value, err := primary.Object(ctx, invocation)
	if err == nil {
		return value, nil
	}
	var unsupported *UnsupportedError
	if !errors.As(err, &unsupported) {
		return Null(), preserveNativeBoundaryError(ctx, err)
	}
	if value, handled, portableErr := portableObject(ctx, invocation); handled {
		if portableErr != nil {
			if executionErr := executionContextError(ctx); executionErr != nil {
				return Null(), executionErr
			}
			var limitErr *LimitError
			if errors.As(portableErr, &limitErr) {
				return Null(), portableErr
			}
			var callbackErr *portableObjectCallbackError
			if errors.As(portableErr, &callbackErr) {
				return Null(), callbackErr.cause
			}
			// Reflection-backed Sleep object calls surface exceptions thrown by
			// Java methods through ScriptEnvironment.flagError. Keep that origin
			// explicit so the evaluator can apply Sleep's soft-error/debug policy
			// without converting importer ObjectHost failures into soft errors.
			return Null(), newPortableJavaException(portableErr)
		}
		return value, nil
	}
	return Null(), err
}

// portableJavaException is the pure-Go counterpart of an exception unwrapped
// from java.lang.reflect.InvocationTargetException. It remains an error at the
// ObjectHost boundary; only script object-expression evaluation translates it
// into Sleep's checkError state.
type portableJavaException struct {
	class   string
	message string
	text    string
	frame   string
	frames  []string
	cause   error
}

func newPortableJavaException(err error) *portableJavaException {
	if err == nil {
		return nil
	}
	var existing *portableJavaException
	if errors.As(err, &existing) {
		return existing
	}
	var thrown *scriptThrow
	if errors.As(err, &thrown) && thrown != nil {
		if object, ok := thrown.value.Object(); ok {
			if exception, ok := object.(*portableJavaException); ok && exception != nil {
				clone := *exception
				clone.frames = append([]string(nil), thrown.frames...)
				clone.cause = err
				return &clone
			}
		}
		message := thrown.value.String()
		return &portableJavaException{
			class:   "java.lang.RuntimeException",
			message: message,
			text:    "java.lang.RuntimeException: " + message,
			frames:  append([]string(nil), thrown.frames...),
			cause:   err,
		}
	}

	text := err.Error()
	class := "java.lang.RuntimeException"
	message := text
	if separator := strings.Index(text, ": "); separator > 0 && strings.HasPrefix(text[:separator], "java.") {
		class = text[:separator]
		message = text[separator+2:]
	} else if strings.HasPrefix(text, "java.") && !strings.ContainsAny(text, " \t\r\n") {
		class = text
		message = ""
	}
	return &portableJavaException{class: class, message: message, text: text, cause: err}
}

func (e *portableJavaException) Error() string {
	if e == nil {
		return ""
	}
	if e.text != "" {
		return e.text
	}
	return e.class
}

func (e *portableJavaException) String() string { return e.Error() }

func (e *portableJavaException) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *portableJavaException) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		return Bool(e.isA(invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke || len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "getMessage", "getLocalizedMessage":
		return String(e.message), true, nil
	case "toString":
		return String(e.Error()), true, nil
	case "getCause":
		return Null(), true, nil
	}
	return Null(), false, nil
}

func (e *portableJavaException) isA(class string) bool {
	if e == nil {
		return false
	}
	class = portableJavaClassName(class)
	exceptionClass := portableJavaClassName(e.class)
	if class == exceptionClass || class == "Object" || class == "Throwable" || class == "Exception" {
		return true
	}
	switch exceptionClass {
	case "ArrayIndexOutOfBoundsException", "StringIndexOutOfBoundsException":
		return class == "IndexOutOfBoundsException" || class == "RuntimeException"
	case "IndexOutOfBoundsException", "NoSuchElementException", "ConcurrentModificationException",
		"IllegalStateException", "NegativeArraySizeException", "NullPointerException",
		"ClassCastException", "UnsupportedOperationException":
		return class == "RuntimeException"
	case "NumberFormatException", "PatternSyntaxException":
		return class == "IllegalArgumentException" || class == "RuntimeException"
	case "IllegalFormatException":
		return class == "IllegalArgumentException" || class == "RuntimeException"
	case "DuplicateFormatFlagsException", "FormatFlagsConversionMismatchException",
		"IllegalFormatArgumentIndexException", "IllegalFormatCodePointException", "IllegalFormatConversionException",
		"IllegalFormatFlagsException", "IllegalFormatPrecisionException",
		"IllegalFormatWidthException", "MissingFormatArgumentException",
		"MissingFormatWidthException", "UnknownFormatConversionException",
		"UnknownFormatFlagsException":
		return class == "IllegalFormatException" || class == "IllegalArgumentException" || class == "RuntimeException"
	case "IllegalArgumentException":
		return class == "RuntimeException"
	case "FileNotFoundException", "UnsupportedEncodingException":
		return class == "IOException"
	case "BindException", "ConnectException", "NoRouteToHostException":
		return class == "SocketException" || class == "IOException"
	case "SocketException", "SocketTimeoutException", "UnknownHostException":
		return class == "IOException"
	default:
		return false
	}
}

// portableSleepStackElement mirrors ScriptInstance.SleepStackElement's
// observable scalar behavior. Sleep returns these objects, rather than plain
// strings, from getStackTrace(); that distinction matters to debug formatting
// because object descriptions are not surrounded with quotes.
type portableSleepStackElement struct {
	text string
}

func (e portableSleepStackElement) String() string { return e.text }

func portableObject(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if value, handled, err := portableScriptObject(ctx, invocation); handled {
		return value, true, err
	}
	if target, ok := invocation.Target.Object(); ok {
		if value, handled, err := portableCompileTarget(target, invocation); handled {
			return value, true, err
		}
		if value, handled, err := portableJavaClassTarget(target, invocation); handled {
			return value, true, err
		}
		if value, handled, err := portableFixtureTarget(target, invocation); handled {
			return value, true, err
		}
		if value, handled, err := portableJavaUtilityTarget(ctx, target, invocation); handled {
			return value, true, err
		}
	}
	if invocation.Op == ObjectInvoke && invocation.Message == "getClass" && len(invocation.Arguments) == 0 {
		if class, ok := portableObjectClass(invocation.Target); ok {
			return ObjectValue(sleepClass(class)), true, nil
		}
	}
	if invocation.Target.Kind() == KindString {
		if value, handled, err := portableStringContext(ctx, invocation); handled {
			return value, true, err
		}
	}
	if target, ok := invocation.Target.Object(); ok {
		if scalar, ok := target.(*serializedSleepScalar); ok && scalar != nil {
			if value, handled, err := scalar.invoke(invocation); handled {
				return value, true, err
			}
		}
		if serialized, ok := target.(*serializedJavaObject); ok && serialized != nil {
			if value, handled, err := serialized.invoke(invocation); handled {
				return value, true, err
			}
		}
		if statistic, ok := target.(*profilerStatistic); ok && statistic != nil {
			if value, handled, err := statistic.invoke(invocation); handled {
				return value, true, err
			}
		}
		if exception, ok := target.(*portableJavaException); ok && exception != nil {
			if value, handled, err := exception.invoke(invocation); handled {
				return value, true, err
			}
		}
		if semaphore, ok := target.(*sleepSemaphore); ok && semaphore != nil {
			if value, handled, err := semaphore.invoke(ctx, invocation); handled {
				return value, true, err
			}
		}
		if tokenizer, ok := target.(*portableStringTokenizer); ok && tokenizer != nil {
			if value, handled, err := tokenizer.invoke(invocation); handled {
				return value, true, err
			}
		}
		if locale, ok := target.(*portableJavaLocale); ok && locale != nil {
			if value, handled, err := locale.invoke(invocation); handled {
				return value, true, err
			}
		}
		if stream, ok := target.(*portableJavaStringStream); ok && stream != nil {
			if value, handled, err := stream.invoke(ctx, invocation); handled {
				return value, true, err
			}
		}
		if stream, ok := target.(*portableJavaRandomStream); ok && stream != nil {
			if value, handled, err := stream.invoke(ctx, invocation); handled {
				return value, true, err
			}
		}
		if iterator, ok := target.(*portableJavaStringStreamIterator); ok && iterator != nil {
			if value, handled, err := iterator.invoke(ctx, invocation); handled {
				return value, true, err
			}
		}
		if iterator, ok := target.(*portableJavaRandomStreamIterator); ok && iterator != nil {
			if value, handled, err := iterator.invoke(ctx, invocation); handled {
				return value, true, err
			}
		}
		if thread, ok := target.(*portableJavaThread); ok && thread != nil {
			if value, handled, err := thread.invoke(invocation); handled {
				return value, true, err
			}
		}
		if random, ok := target.(*portableJavaRandom); ok && random != nil {
			if value, handled, err := random.invoke(invocation); handled {
				return value, true, err
			}
		}
		if uuid, ok := target.(*portableJavaUUID); ok && uuid != nil {
			if value, handled, err := uuid.invoke(invocation); handled {
				return value, true, err
			}
		}
		if file, ok := target.(*portableJavaFile); ok && file != nil {
			if value, handled, err := file.invokeContext(ctx, invocation); handled {
				return value, true, err
			}
		}
		if uri, ok := target.(*portableJavaURI); ok && uri != nil {
			if value, handled, err := uri.invoke(invocation); handled {
				return value, true, err
			}
		}
		if url, ok := target.(*portableJavaURL); ok && url != nil {
			if value, handled, err := url.invoke(invocation); handled {
				return value, true, err
			}
		}
		if path, ok := target.(*portableJavaPath); ok && path != nil {
			if value, handled, err := path.invoke(invocation); handled {
				return value, true, err
			}
		}
		if value, handled, err := portableCollectionTarget(ctx, target, invocation); handled {
			return value, true, err
		}
	}
	if invocation.Op == ObjectTypeCheck {
		if invocation.Target.IsNull() {
			return Bool(false), true, nil
		}
		if actual, ok := portableObjectClass(invocation.Target); ok {
			return Bool(portableJavaAssignable(actual, invocation.Class)), true, nil
		}
	}
	if value, handled, err := portableFixtureClass(invocation); handled {
		return value, true, err
	}
	if value, handled, err := portableJavaFileStatic(ctx, invocation); handled {
		return value, true, err
	}
	if value, handled, err := portableJavaFileConstruct(invocation); handled {
		return value, true, err
	}
	if value, handled, err := portableJavaLocaleClass(invocation); handled {
		return value, true, err
	}
	if value, handled, err := portableJavaRandomGeneratorClass(invocation); handled {
		return value, true, err
	}
	if invocation.Class == "javax.swing.AbstractListModel" && invocation.Op == ObjectConstruct && len(invocation.Arguments) == 0 {
		// The canonical warning fixture intentionally constructs this JDK
		// abstract class. Model only reflection's observable failure; OPFOR does
		// not otherwise provide a Swing implementation.
		return portableObjectWarning(invocation, "unable to instantiate abstract class javax.swing.AbstractListModel"), true, nil
	}
	switch portableJavaClassName(invocation.Class) {
	case "Arrays":
		if value, handled, err := portableJavaArrays(invocation); handled {
			return value, true, err
		}
	case "Array":
		if value, handled, err := portableJavaReflectArray(invocation); handled {
			return value, true, err
		}
	case "String":
		if value, handled, err := portableJavaStringConstruct(invocation); handled {
			return value, true, err
		}
		if value, handled, err := portableJavaStringStaticContext(ctx, invocation); handled {
			return value, true, err
		}
	case "StringBuffer", "StringBuilder":
		if value, handled, err := portableJavaStringBuilderConstruct(invocation); handled {
			return value, true, err
		}
	case "Point":
		if invocation.Op == ObjectConstruct && len(invocation.Arguments) != 1 && len(invocation.Arguments) <= 2 {
			point := &portableJavaPoint{}
			if len(invocation.Arguments) == 2 {
				point.x = sleepInt32(invocation.Arg(0))
				point.y = sleepInt32(invocation.Arg(1))
			}
			return ObjectValue(point), true, nil
		}
	case "MessageDigest":
		if invocation.Op == ObjectInvoke && invocation.Message == "getInstance" && len(invocation.Arguments) == 1 {
			algorithm := invocation.Arg(0).String()
			hash, err := sleepMessageDigest(algorithm)
			if err != nil {
				return Null(), true, fmt.Errorf("java.security.NoSuchAlgorithmException: %s MessageDigest not available", algorithm)
			}
			return ObjectValue(&portableJavaMessageDigest{
				algorithm: algorithm, state: &sleepDigestState{digest: hash},
			}), true, nil
		}
	case "UUID":
		if value, handled, err := portableJavaUUIDClass(invocation); handled {
			return value, true, err
		}
	}
	class := strings.TrimPrefix(invocation.Class, "java.lang.")
	class = strings.TrimPrefix(class, "java.util.")
	switch class {
	case "Math":
		if value, handled, err := portableMath(invocation); handled {
			return value, true, err
		}
	case "Collections":
		if value, handled, err := portableCollections(ctx, invocation); handled {
			return value, true, err
		}
	case "LinkedList", "ArrayList", "HashSet", "LinkedHashSet", "TreeSet", "HashMap", "Hashtable", "LinkedHashMap", "TreeMap":
		if invocation.Op == ObjectConstruct {
			return newPortableCollectionObject(class, invocation)
		}
	case "Random":
		if invocation.Op == ObjectInvoke && invocation.Message == "from" && len(invocation.Arguments) == 1 {
			if invocation.Arg(0).IsNull() {
				return Null(), true, &portableJavaException{
					class: "java.lang.NullPointerException",
					text:  "java.lang.NullPointerException",
					frame: "public static java.util.Random java.util.Random.from(java.util.random.RandomGenerator)",
				}
			}
			if object, ok := invocation.Arg(0).Object(); ok {
				if _, ok := object.(*portableJavaRandom); ok {
					return invocation.Arg(0), true, nil
				}
			}
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
		if invocation.Op == ObjectConstruct {
			if len(invocation.Arguments) > 1 {
				message := fmt.Sprintf("no constructor matching java.util.Random(%s)", portableObjectArgumentList(invocation))
				return portableObjectWarning(invocation, message), true, nil
			}
			seed := portableJavaRandomSeed()
			if len(invocation.Arguments) == 1 {
				seed = sleepInt64(invocation.Arg(0))
			}
			return ObjectValue(newPortableJavaRandom(seed)), true, nil
		}
	case "Thread":
		if invocation.Op == ObjectInvoke && invocation.Message == "currentThread" && len(invocation.Arguments) == 0 {
			return ObjectValue(portableCurrentThread(ctx, invocation)), true, nil
		}
		// java.lang.Thread.yield() is a scheduler hint. A pure-Go runtime has
		// no Java thread to yield, and Go's scheduler already owns goroutine
		// scheduling, so the compatible observable result is a void no-op.
		if invocation.Op == ObjectInvoke && invocation.Message == "yield" && len(invocation.Arguments) == 0 {
			return Null(), true, nil
		}
	case "SleepUtils", "sleep.runtime.SleepUtils":
		if value, handled, err := portableSleepUtils(invocation); handled {
			return value, true, err
		}
	case "System":
		if (invocation.Op == ObjectGet || invocation.Op == ObjectInvoke) && invocation.Message == "out" && len(invocation.Arguments) == 0 {
			return ObjectValue(invocation.Runtime.portableSystemOut()), true, nil
		}
	case "Boolean":
		if invocation.Op == ObjectGet || invocation.Op == ObjectInvoke && len(invocation.Arguments) == 0 {
			switch invocation.Message {
			case "TRUE":
				return Int(1), true, nil
			case "FALSE":
				return Int(0), true, nil
			}
		}
		if invocation.Op == ObjectInvoke && invocation.Message == "toString" {
			if len(invocation.Arguments) != 1 || invocation.Arg(0).Kind() == KindString {
				return portableNoMatchingMethod(invocation, "java.lang.Boolean"), true, nil
			}
			return String(strconv.FormatBool(invocation.Arg(0).Truth())), true, nil
		}
	case "Integer":
		if value, handled, err := portableInteger(invocation, false); handled {
			return value, true, err
		}
	case "Long":
		if value, handled, err := portableInteger(invocation, true); handled {
			return value, true, err
		}
	case "Double":
		if invocation.Op == ObjectConstruct && len(invocation.Arguments) == 1 {
			return ObjectValue(&portableJavaPrimitive{
				class: "java.lang.Double", value: Double(invocation.Arg(0).Float64()),
			}), true, nil
		}
		if invocation.Op == ObjectInvoke && invocation.Message == "valueOf" && len(invocation.Arguments) == 1 {
			return Double(invocation.Arg(0).Float64()), true, nil
		}
	case "Character":
		if invocation.Op == ObjectInvoke {
			if value, handled, err := portableCharacter(invocation); handled {
				return value, true, err
			}
		}
	case "StringTokenizer":
		if invocation.Op == ObjectConstruct && len(invocation.Arguments) >= 1 && len(invocation.Arguments) <= 3 {
			return newPortableStringTokenizer(invocation)
		}
	case "NoSuchElementException":
		if invocation.Op == ObjectConstruct && len(invocation.Arguments) <= 1 {
			message := ""
			if len(invocation.Arguments) == 1 {
				message = invocation.Arg(0).String()
			}
			text := "java.util.NoSuchElementException"
			if message != "" {
				text += ": " + message
			}
			return ObjectValue(&portableJavaException{
				class: "java.util.NoSuchElementException", message: message, text: text,
			}), true, nil
		}
	}
	return portableReflectionFallback(invocation)
}

func portableSleepUtils(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op != ObjectInvoke && invocation.Op != ObjectGet {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "getScalar":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.SleepUtils"), true, nil
		}
		return invocation.Arg(0), true, nil
	case "getHashScalar":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.SleepUtils"), true, nil
		}
		return HashValue(NewHash()), true, nil
	case "getArrayScalar":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.SleepUtils"), true, nil
		}
		return ArrayValue(NewArray()), true, nil
	case "getArrayWrapper":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.SleepUtils"), true, nil
		}
		if object, ok := invocation.Arg(0).Object(); ok {
			if collection, ok := object.(*portableJavaCollection); ok && collection != nil {
				source := collectionWrapperSource(collection)
				if collection.wrapperSource != nil {
					source = collection.wrapperSource
				}
				wrapper, err := newRuntimeCollectionWrapperArray(invocation.Runtime, source)
				if err != nil {
					return Null(), true, err
				}
				return ArrayValue(wrapper), true, nil
			}
		}
		values, ok := portableCollectionValues(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, "sleep.runtime.SleepUtils"), true, nil
		}
		// Retain the prior tolerant handling of an already-Sleep array. Java
		// Collections use the live backend above; this fallback is necessarily a
		// detached read-only conversion because no Collection object exists.
		array, err := newRuntimeReadOnlyArray(invocation.Runtime, values...)
		if err != nil {
			return Null(), true, err
		}
		return ArrayValue(array), true, nil
	case "getHashWrapper":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.SleepUtils"), true, nil
		}
		if object, ok := invocation.Arg(0).Object(); ok {
			if mapping, ok := object.(*portableJavaMap); ok && mapping != nil {
				return HashValue(newRuntimeMapWrapperHash(invocation.Runtime, mapping)), true, nil
			}
		}
		entries, ok, err := portableMapEntriesReserved(invocation.Arg(0), func(count int) error {
			return reserveCollectionEntries(invocation.Runtime, count)
		})
		if !ok {
			return portableNoMatchingMethod(invocation, "sleep.runtime.SleepUtils"), true, nil
		}
		if err != nil {
			return Null(), true, err
		}
		snapshot := NewHash()
		for _, entry := range entries {
			snapshot.SetValue(entry.keyValue, entry.value)
		}
		snapshot.readOnly = true
		return HashValue(snapshot), true, nil
	case "getListFromArray":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.SleepUtils"), true, nil
		}
		values, ok := portableCollectionValues(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, "sleep.runtime.SleepUtils"), true, nil
		}
		if err := reserveCollectionEntries(invocation.Runtime, len(values)); err != nil {
			return Null(), true, err
		}
		return ObjectValue(newPortableJavaCollection("LinkedList", values)), true, nil
	case "getMapFromHash":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "sleep.runtime.SleepUtils"), true, nil
		}
		entries, ok := portableMapEntries(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, "sleep.runtime.SleepUtils"), true, nil
		}
		if err := reserveCollectionEntries(invocation.Runtime, len(entries)); err != nil {
			return Null(), true, err
		}
		return ObjectValue(newPortableJavaMapFromEntries("HashMap", entries)), true, nil
	}
	return Null(), false, nil
}

func portableString(invocation ObjectInvocation) (Value, bool, error) {
	return portableStringContext(context.Background(), invocation)
}

func portableStringContext(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), true, err
	}
	target := sleepStringCoercion(invocation.Target)
	switch invocation.Message {
	case "charAt":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		units := sleepStringUnits(target)
		index := int(sleepInt32(invocation.Arg(0)))
		if index < 0 || index >= len(units) {
			return Null(), true, fmt.Errorf("java.lang.StringIndexOutOfBoundsException: Index %d out of bounds for length %d", index, len(units))
		}
		return sleepUTF16CharacterValue(units[index]), true, nil
	case "codePointAt":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		units := sleepStringUnits(target)
		index := int(sleepInt32(invocation.Arg(0)))
		if index < 0 || index >= len(units) {
			return Null(), true, fmt.Errorf("java.lang.StringIndexOutOfBoundsException: Index %d out of bounds for length %d", index, len(units))
		}
		codePoint, _ := sleepUTF16CodePointAt(units, index)
		return Int(codePoint), true, nil
	case "codePointBefore":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		units := sleepStringUnits(target)
		indexValue := int64(sleepInt32(invocation.Arg(0)))
		if indexValue < 1 || indexValue > int64(len(units)) {
			return Null(), true, fmt.Errorf("java.lang.StringIndexOutOfBoundsException: Index %d out of bounds for length %d", indexValue-1, len(units))
		}
		index := int(indexValue)
		codePoint, _ := sleepUTF16CodePointBefore(units, index)
		return Int(codePoint), true, nil
	case "codePointCount":
		if len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		units := sleepStringUnits(target)
		start := int(sleepInt32(invocation.Arg(0)))
		end := int(sleepInt32(invocation.Arg(1)))
		if start < 0 || start > end || end > len(units) {
			return Null(), true, fmt.Errorf("java.lang.IndexOutOfBoundsException: Range [%d, %d) out of bounds for length %d", start, end, len(units))
		}
		return Int(int32(sleepUTF16CodePointCount(units, start, end))), true, nil
	case "chars":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		return newPortableJavaStringStream(target, portableJavaStringCharsStream), true, nil
	case "codePoints":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		return newPortableJavaStringStream(target, portableJavaStringCodePointsStream), true, nil
	case "compareTo":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		if invocation.Arg(0).IsNull() {
			return Null(), true, errors.New(`java.lang.NullPointerException: Cannot read field "value" because "anotherString" is null`)
		}
		if invocation.Arg(0).Kind() != KindString {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		return Int(int32(sleepStringCompareValues(target, invocation.Arg(0)))), true, nil
	case "compareToIgnoreCase":
		return portableJavaStringCompareToIgnoreCase(ctx, invocation, target)
	case "concat":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		if invocation.Arg(0).IsNull() {
			return Null(), true, errors.New(`java.lang.NullPointerException: Cannot invoke "String.isEmpty()" because "str" is null`)
		}
		if invocation.Arg(0).Kind() != KindString {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		other := invocation.Arg(0)
		if sleepStringLength(other) == 0 {
			return target, true, nil
		}
		if int64(sleepStringLength(target))+int64(sleepStringLength(other)) > math.MaxInt32 {
			return Null(), true, fmt.Errorf("java.lang.OutOfMemoryError: Required length exceeds implementation limit")
		}
		return sleepStringConcat(target, other), true, nil
	case "contentEquals":
		return portableJavaStringContentEquals(ctx, invocation, target)
	case "contains":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		if invocation.Arg(0).IsNull() {
			return Null(), true, errors.New(`java.lang.NullPointerException: Cannot invoke "java.lang.CharSequence.toString()" because "s" is null`)
		}
		needle := invocation.Arg(0)
		if needle.Kind() != KindString {
			object, ok := needle.Object()
			if !ok {
				return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
			}
			buffer, ok := object.(*portableJavaStringBuffer)
			if !ok || buffer == nil {
				return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
			}
			class := portableJavaClassName(buffer.class)
			if class != "StringBuilder" && class != "StringBuffer" {
				return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
			}
			needle = buffer.snapshotValue()
		}
		return portableJavaBooleanValue(sleepStringContains(target, needle)), true, nil
	case "endsWith":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		if invocation.Arg(0).IsNull() {
			return Null(), true, errors.New(`java.lang.NullPointerException: Cannot invoke "String.length()" because "suffix" is null`)
		}
		if invocation.Arg(0).Kind() != KindString {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		needle := sleepStringCoercion(invocation.Arg(0))
		if sleepStringUnitIndex(target, needle, sleepStringLength(target)-sleepStringLength(needle), false) ==
			sleepStringLength(target)-sleepStringLength(needle) {
			return Int(1), true, nil
		}
		return Int(0), true, nil
	case "equals":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		if invocation.Arg(0).Kind() != KindString {
			return Int(0), true, nil
		}
		if sleepStringValuesEqual(target, invocation.Arg(0)) {
			return Int(1), true, nil
		}
		return Int(0), true, nil
	case "equalsIgnoreCase":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		if invocation.Arg(0).Kind() != KindString {
			return Int(0), true, nil
		}
		return portableJavaBooleanValue(sleepStringEqualFold(target, invocation.Arg(0))), true, nil
	case "getBytes":
		if len(invocation.Arguments) > 1 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		if len(invocation.Arguments) == 1 && invocation.Arg(0).IsNull() {
			// Sleep's reflection matcher selects getBytes(Charset) rather than
			// getBytes(String) for a null reference. The former performs an
			// unmessaged Objects.requireNonNull check.
			return Null(), true, errors.New("java.lang.NullPointerException")
		}
		if len(invocation.Arguments) == 1 && invocation.Arg(0).Kind() != KindString {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		charset := sleepCharsetUTF8
		if len(invocation.Arguments) == 1 {
			var err error
			charset, err = portableJavaStringCharset(invocation.Arg(0).String())
			if err != nil {
				return Null(), true, err
			}
		}
		return BinaryString(portableJavaStringEncode(target, charset)), true, nil
	case "getChars":
		return portableJavaStringGetChars(ctx, invocation, target)
	case "hashCode":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		hash, err := portableJavaStringHashCode(ctx, target)
		if err != nil {
			return Null(), true, err
		}
		return Int(hash), true, nil
	case "indexOf", "lastIndexOf":
		if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 2 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		if invocation.Arg(0).IsNull() {
			switch {
			case invocation.Message == "indexOf" && len(invocation.Arguments) == 1:
				return Null(), true, errors.New(`java.lang.NullPointerException: Cannot invoke "String.coder()" because "str" is null`)
			case invocation.Message == "indexOf":
				return Null(), true, errors.New(`java.lang.NullPointerException: Cannot invoke "String.length()" because "tgtStr" is null`)
			case len(invocation.Arguments) == 1:
				return Null(), true, errors.New(`java.lang.NullPointerException: Cannot read field "value" because "tgtStr" is null`)
			default:
				// For this two-argument shape Sleep selects lastIndexOf(int,
				// int), coercing the empty scalar to the NUL code point.
				return Int(-1), true, nil
			}
		}
		units := sleepStringUnits(target)
		from := 0
		if invocation.Message == "lastIndexOf" {
			from = len(units)
		}
		if len(invocation.Arguments) == 2 {
			fromValue, ok := portableJavaStringIntArgument(invocation.Arg(1))
			if !ok {
				return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
			}
			from = int(fromValue)
		}
		if invocation.Arg(0).Kind() == KindString {
			needle := sleepStringUnits(invocation.Arg(0))
			return Int(int32(portableStringUnitIndex(units, needle, from, invocation.Message == "lastIndexOf"))), true, nil
		}
		codePoint, ok := portableJavaStringIntArgument(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		index, err := portableJavaStringCodePointIndex(ctx, units, codePoint, from, invocation.Message == "lastIndexOf")
		return Int(int32(index)), true, err
	case "indent":
		return portableJavaStringIndent(ctx, invocation, target)
	case "length":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		return Int(int32(sleepStringLength(target))), true, nil
	case "matches":
		return portableJavaStringMatches(ctx, invocation, target)
	case "offsetByCodePoints":
		return portableJavaStringOffsetByCodePoints(ctx, invocation, target)
	case "isBlank":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		return portableJavaBooleanValue(sleepStringIsBlank(target)), true, nil
	case "isEmpty":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		return portableJavaBooleanValue(sleepStringLength(target) == 0), true, nil
	case "intern":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		// Sleep converts Java String results straight back into immutable scalar
		// values, so JVM reference identity is not observable. Returning the
		// receiver preserves the same scalar value and OPFOR byte provenance.
		return target, true, nil
	case "repeat":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		count := int(sleepInt32(invocation.Arg(0)))
		if count < 0 {
			return Null(), true, fmt.Errorf("java.lang.IllegalArgumentException: count is negative: %d", count)
		}
		length := sleepStringLength(target)
		if count != 0 && int64(length)*int64(count) > math.MaxInt32 {
			return Null(), true, fmt.Errorf("java.lang.OutOfMemoryError: Required length exceeds implementation limit")
		}
		return sleepStringRepeat(target, count), true, nil
	case "regionMatches":
		return portableJavaStringRegionMatches(ctx, invocation, target)
	case "replace":
		if len(invocation.Arguments) != 2 || invocation.Arg(0).Kind() != KindString || invocation.Arg(1).Kind() != KindString {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		value, err := sleepStringReplaceLiteral(target, invocation.Arg(0), invocation.Arg(1))
		return value, true, err
	case "replaceAll", "replaceFirst":
		return portableJavaStringRegexReplace(ctx, invocation, target, invocation.Message == "replaceFirst")
	case "split":
		return portableJavaStringSplit(ctx, invocation, target)
	case "strip", "stripLeading", "stripTrailing":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		return sleepStringStrip(target, invocation.Message != "stripTrailing", invocation.Message != "stripLeading"), true, nil
	case "stripIndent":
		return portableJavaStringStripIndent(ctx, invocation, target)
	case "substring", "subSequence":
		if invocation.Message == "subSequence" && len(invocation.Arguments) != 2 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 2 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		units := sleepStringUnits(target)
		start := int(invocation.Arg(0).Int32())
		end := len(units)
		if len(invocation.Arguments) == 2 {
			end = int(invocation.Arg(1).Int32())
		}
		if start < 0 || end < start || end > len(units) {
			return Null(), true, fmt.Errorf("java.lang.StringIndexOutOfBoundsException: Range [%d, %d) out of bounds for length %d", start, end, len(units))
		}
		return sleepStringValueSlice(target, start, end), true, nil
	case "startsWith":
		if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 2 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		if invocation.Arg(0).IsNull() {
			return Null(), true, errors.New(`java.lang.NullPointerException: Cannot invoke "String.length()" because "prefix" is null`)
		}
		if invocation.Arg(0).Kind() != KindString {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		units := sleepStringUnits(target)
		prefix := sleepStringUnits(invocation.Arg(0))
		offset := 0
		if len(invocation.Arguments) == 2 {
			offset = int(invocation.Arg(1).Int32())
		}
		if offset < 0 || offset > len(units) || len(prefix) > len(units)-offset {
			return Int(0), true, nil
		}
		return portableJavaBooleanValue(slices.Equal(units[offset:offset+len(prefix)], prefix)), true, nil
	case "toLowerCase":
		return portableJavaStringCase(ctx, invocation, target, false)
	case "trim":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		units := sleepStringUnits(target)
		start, end := 0, len(units)
		for start < end && units[start] <= 0x20 {
			start++
		}
		for end > start && units[end-1] <= 0x20 {
			end--
		}
		return sleepStringValueSlice(target, start, end), true, nil
	case "toUpperCase":
		return portableJavaStringCase(ctx, invocation, target, true)
	case "lines":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		return newPortableJavaStringStream(target, portableJavaStringLinesStream), true, nil
	case "transform":
		return portableJavaStringTransform(ctx, invocation, target)
	case "translateEscapes":
		return portableJavaStringTranslateEscapes(ctx, invocation, target)
	case "formatted":
		return portableJavaStringFormatted(ctx, invocation, target)
	case "toString":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		return invocation.Target, true, nil
	case "toCharArray":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		// Sleep's ObjectUtilities bridge converts a Java char[] return value into
		// an ordinary scalar string before the script observes it. Keep the
		// exact UTF-16 units, but do not leak the intermediate Java-array identity.
		return sleepStringValueFromUnits(sleepStringUnits(target), nil), true, nil
	}
	return Null(), false, nil
}

func portableJavaStringConstruct(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op != ObjectConstruct {
		return Null(), false, nil
	}
	switch len(invocation.Arguments) {
	case 0:
		return String(""), true, nil
	case 1:
		if invocation.Arg(0).Kind() != KindString {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		// Sleep exposes both ordinary Java Strings and byte[] results as scalar
		// string values. The one-argument String(String) constructor therefore
		// retains the exact UTF-16 units and byte provenance of its argument.
		return invocation.Arg(0), true, nil
	case 2:
		if invocation.Arg(0).Kind() != KindString || invocation.Arg(1).Kind() != KindString {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		charset, err := portableJavaStringCharset(invocation.Arg(1).String())
		if err != nil {
			return Null(), true, err
		}
		return portableJavaStringDecode(invocation.Arg(0), charset), true, nil
	default:
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
}

func portableJavaStringStatic(invocation ObjectInvocation) (Value, bool, error) {
	return portableJavaStringStaticContext(context.Background(), invocation)
}

func portableJavaStringCharset(name string) (sleepTextCharset, error) {
	charset, err := sleepLookupTextCharset(name)
	if err != nil {
		return sleepCharsetUTF8, fmt.Errorf("java.io.UnsupportedEncodingException: %s", name)
	}
	return charset, nil
}

func portableJavaStringEncode(value Value, charset sleepTextCharset) []byte {
	var encoder sleepTextEncoder
	encoder.reset(charset)
	encoded := encoder.encodeUnits(sleepStringUnits(value), false)
	return append(encoded, encoder.finish()...)
}

func portableJavaStringDecode(value Value, charset sleepTextCharset) Value {
	var decoder sleepTextDecoder
	decoder.reset(charset)
	units := decoder.decode(sleepStringLowBytes(value), true)
	return sleepStringValueFromUnits(units, nil)
}

func portableStringUnitIndex(haystack, needle []uint16, from int, reverse bool) int {
	if !reverse {
		if from < 0 {
			from = 0
		}
		if len(needle) == 0 {
			return min(from, len(haystack))
		}
		if from > len(haystack) {
			return -1
		}
		for index := from; index+len(needle) <= len(haystack); index++ {
			if slices.Equal(haystack[index:index+len(needle)], needle) {
				return index
			}
		}
		return -1
	}

	if len(needle) == 0 {
		if from < 0 {
			return -1
		}
		return min(from, len(haystack))
	}
	from = min(from, len(haystack)-len(needle))
	for index := from; index >= 0; index-- {
		if slices.Equal(haystack[index:index+len(needle)], needle) {
			return index
		}
	}
	return -1
}

func portableObjectClass(value Value) (string, bool) {
	switch value.Kind() {
	case KindInt:
		return "java.lang.Integer", true
	case KindLong:
		return "java.lang.Long", true
	case KindDouble:
		return "java.lang.Double", true
	case KindString:
		return "java.lang.String", true
	case KindArray:
		return "sleep.engine.types.ListContainer", true
	case KindHash:
		return "sleep.engine.types.HashContainer", true
	case KindFunction:
		return "sleep.bridges.SleepClosure", true
	case KindObject:
		if object, ok := value.Object(); ok {
			if _, ok := object.(*portableCompileException); ok {
				return "sleep.error.YourCodeSucksException", true
			}
			if class, ok := portableFixtureObjectClass(object); ok {
				return class, true
			}
			if class, ok := portableJavaUtilityClass(object); ok {
				return class, true
			}
			switch object.(type) {
			case classReference, sleepClass:
				return "java.lang.Class", true
			case *serializedSleepScalar:
				return sleepScalarDescriptor.Name, true
			case *serializedJavaObject:
				serialized := object.(*serializedJavaObject)
				if serialized != nil && serialized.class != "" {
					return serialized.class, true
				}
			case *profilerStatistic:
				return "sleep.runtime.ScriptInstance$ProfilerStatistic", true
			case *portableJavaException:
				exception := object.(*portableJavaException)
				if exception != nil {
					return exception.class, true
				}
			case *sleepSemaphore:
				return "sleep.bridges.Semaphore", true
			case sleepKeyValue, *sleepKeyValue:
				return "sleep.bridges.KeyValuePair", true
			case *portableStringTokenizer:
				return "java.util.StringTokenizer", true
			case *portableJavaLocale:
				return "java.util.Locale", true
			case *portableJavaStringStream:
				stream := object.(*portableJavaStringStream)
				return stream.className(), true
			case *portableJavaRandomStream:
				stream := object.(*portableJavaRandomStream)
				return stream.className(), true
			case *portableJavaStringStreamIterator:
				iterator := object.(*portableJavaStringStreamIterator)
				return iterator.className(), true
			case *portableJavaRandomStreamIterator:
				iterator := object.(*portableJavaRandomStreamIterator)
				return iterator.className(), true
			case *portableJavaCollection:
				collection := object.(*portableJavaCollection)
				return "java.util." + collection.className(), true
			case *portableJavaMap:
				mapping := object.(*portableJavaMap)
				return "java.util." + mapping.className(), true
			case *portableJavaMapEntry:
				return "java.util.Map$Entry", true
			case *portableJavaReverseComparator:
				return "java.util.Collections$ReverseComparator", true
			case *portableJavaNaturalComparator:
				return "java.util.Comparators$NaturalOrderComparator", true
			case *portableJavaReverseComparator2:
				return "java.util.Collections$ReverseComparator2", true
			case *portableJavaIterator:
				return "java.util.Iterator", true
			case *portableJavaThread:
				return "java.lang.Thread", true
			case *portableJavaRandom:
				return "java.util.Random", true
			case *portableJavaUUID:
				return "java.util.UUID", true
			case *portableJavaFile:
				return portableJavaFileClass, true
			case *portableJavaURI:
				return portableJavaURIClass, true
			case *portableJavaURL:
				return portableJavaURLClass, true
			case *portableJavaPath:
				return portableJavaPathClass(), true
			case *portableScriptLoader:
				return "sleep.runtime.ScriptLoader", true
			case *portableScriptInstance:
				return "sleep.runtime.ScriptInstance", true
			case *portableScriptEnvironment:
				return "sleep.runtime.ScriptEnvironment", true
			case *portableScriptVariables:
				return "sleep.runtime.ScriptVariables", true
			case *portableCompiledBlock:
				return "sleep.engine.Block", true
			case *portableScriptBridge:
				bridge := object.(*portableScriptBridge)
				return bridge.primaryClass(), true
			case portableScriptInputStream, *portableScriptInputStream:
				return "java.io.InputStream", true
			case *portableScriptWarning:
				return "sleep.error.ScriptWarning", true
			case *sleepIOHandle:
				return "sleep.bridges.io.IOObject", true
			}
		}
	}
	return "", false
}

// portableJavaThread is the deterministic subset of java.lang.Thread exposed
// by Sleep's object bridge. The reference CLI executes ordinary scripts on the
// JVM's main thread. BasicIO.fork creates a priority-5 thread in the main group
// whose name is "fork of " plus the runnable block's source location.
type portableJavaThread struct {
	name     string
	priority int
	group    string
}

func (t *portableJavaThread) String() string {
	if t == nil {
		return "Thread[null,0,null]"
	}
	return fmt.Sprintf("Thread[%s,%d,%s]", t.name, t.priority, t.group)
}

func (t *portableJavaThread) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := strings.TrimPrefix(invocation.Class, "java.lang.")
		return Bool(class == "Thread" || class == "Runnable" || class == "Object"), true, nil
	}
	if invocation.Op != ObjectInvoke || len(invocation.Arguments) != 0 {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "toString":
		return String(t.String()), true, nil
	case "getName":
		return String(t.name), true, nil
	case "getPriority":
		return Int(int32(t.priority)), true, nil
	}
	return Null(), false, nil
}

func portableCurrentThread(ctx context.Context, invocation ObjectInvocation) *portableJavaThread {
	thread := &portableJavaThread{name: "main", priority: 5, group: "main"}
	if invocation.Runtime == nil {
		return thread
	}
	script := invocation.Runtime.script(invocation.Script)
	if script == nil || script.forkParent == nil {
		return thread
	}

	// A child goroutine inherits the parent's context, including its current
	// fiber. Stop at the child Script boundary so nested child calls resolve to
	// the fork runnable rather than walking into the parent fiber chain.
	root := currentFiber(ctx)
	for root != nil && root.caller != nil && root.caller.closure != nil &&
		root.closure != nil && root.caller.closure.script == root.closure.script {
		root = root.caller
	}
	if root == nil || root.function == nil {
		return thread
	}
	if location := portableThreadSourceLocation(root.function); location != "" {
		thread.name = "fork of " + location
	}
	return thread
}

func portableThreadSourceLocation(function *bytecode.Function) string {
	if function == nil {
		return ""
	}
	source := ""
	low, high := 0, 0
	for _, instruction := range function.Instructions {
		// OpEnd is an OPFOR sentinel whose span covers the braces around a
		// closure. Sleep's Block.getSourceLocation ranges only executable Steps.
		if instruction.Op == bytecode.OpEnd || instruction.Span.Start.Line <= 0 {
			continue
		}
		if source == "" && instruction.Span.Source != "" {
			source = instruction.Span.Source
		}
		line := instruction.Span.Start.Line
		if low == 0 || line < low {
			low = line
		}
		if line > high {
			high = line
		}
	}
	if source == "" {
		source = function.Span.Source
	}
	if source == "" || low == 0 {
		return ""
	}
	location := filepath.Base(source) + ":" + strconv.Itoa(low)
	if high != low {
		location += "-" + strconv.Itoa(high)
	}
	return location
}

func portableNoMatchingMethod(invocation ObjectInvocation, class string) Value {
	return portableObjectWarning(invocation, fmt.Sprintf("there is no method that matches %s(%s) in %s", invocation.Message, portableObjectArgumentList(invocation), class))
}

func portableObjectArgumentList(invocation ObjectInvocation) string {
	parts := make([]string, len(invocation.Arguments))
	for index, argument := range invocation.Arguments {
		parts[index] = argument.Resolve().Describe()
	}
	return strings.Join(parts, ", ")
}

func portableObjectWarning(invocation ObjectInvocation, message string) Value {
	if invocation.Runtime == nil {
		return Null()
	}
	enabled := true
	if script := invocation.Runtime.script(invocation.Script); script != nil {
		script.mu.RLock()
		enabled = script.debug != 0
		script.mu.RUnlock()
	}
	if enabled {
		invocation.Runtime.writeWarning(message, invocation.Span)
	}
	return Null()
}

// portableReflectionFallback applies ObjectAccess/ObjectNew's direct warning
// policy only to classes for which OPFOR has an honest portable representation.
// Unknown and importer-owned classes remain unsupported at the ObjectHost
// boundary instead of pretending that reflection inspected unavailable JVM
// bytecode.
func portableReflectionFallback(invocation ObjectInvocation) (Value, bool, error) {
	class, ok := portableReflectionClass(invocation)
	if !ok {
		return Null(), false, nil
	}
	switch invocation.Op {
	case ObjectConstruct:
		message := fmt.Sprintf("no constructor matching %s(%s)", class, portableObjectArgumentList(invocation))
		return portableObjectWarning(invocation, message), true, nil
	case ObjectInvoke:
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, class), true, nil
		}
		fallthrough
	case ObjectGet:
		message := fmt.Sprintf("no field/method named %s in %s", invocation.Message, classReference(class).String())
		return portableObjectWarning(invocation, message), true, nil
	default:
		return Null(), false, nil
	}
}

func portableReflectionClass(invocation ObjectInvocation) (string, bool) {
	if !invocation.Target.IsNull() {
		return portableObjectClass(invocation.Target)
	}
	class := resolvePortableClassName(invocation.Class)
	for _, known := range portableDefaultClasses {
		if class == known {
			return class, true
		}
	}
	switch class {
	case "java.awt.Point", "java.io.File", "java.lang.reflect.Array", "java.util.NoSuchElementException":
		return class, true
	}
	if invocation.Runtime != nil && invocation.Runtime.portableFixtureState().allows(invocation.Script, class) {
		return class, true
	}
	return "", false
}

type portableStringTokenizer struct {
	mu           sync.Mutex
	input        []rune
	position     int
	delimiters   map[rune]struct{}
	returnDelims bool
}

func newPortableStringTokenizer(invocation ObjectInvocation) (Value, bool, error) {
	delimiters := " \t\n\r\f"
	if len(invocation.Arguments) > 1 {
		delimiters = sleepStringText(invocation.Arg(1))
	}
	tokenizer := &portableStringTokenizer{
		input:        []rune(sleepStringText(invocation.Arg(0))),
		delimiters:   runeSet(delimiters),
		returnDelims: len(invocation.Arguments) > 2 && invocation.Arg(2).Truth(),
	}
	return ObjectValue(tokenizer), true, nil
}

func (t *portableStringTokenizer) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := strings.TrimPrefix(strings.TrimPrefix(invocation.Class, "java.util."), "java.lang.")
		return Bool(class == "StringTokenizer"), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	switch invocation.Message {
	case "hasMoreTokens", "hasMoreElements":
		return Bool(t.nextPosition(t.position) < len(t.input)), true, nil
	case "nextToken", "nextElement":
		if invocation.Message == "nextToken" && len(invocation.Arguments) != 0 {
			t.delimiters = runeSet(sleepStringText(invocation.Arg(0)))
		}
		value, ok := t.next()
		if !ok {
			return Null(), true, errors.New("java.util.NoSuchElementException")
		}
		return String(value), true, nil
	case "countTokens":
		position := t.position
		count := 0
		for {
			position = t.nextPosition(position)
			if position >= len(t.input) {
				break
			}
			count++
			if t.returnDelims {
				if _, delimiter := t.delimiters[t.input[position]]; delimiter {
					position++
					continue
				}
			}
			for position < len(t.input) {
				if _, delimiter := t.delimiters[t.input[position]]; delimiter {
					break
				}
				position++
			}
		}
		return Int(int32(count)), true, nil
	}
	return Null(), false, nil
}

func (t *portableStringTokenizer) nextPosition(position int) int {
	if t.returnDelims {
		return position
	}
	for position < len(t.input) {
		if _, delimiter := t.delimiters[t.input[position]]; !delimiter {
			break
		}
		position++
	}
	return position
}

func (t *portableStringTokenizer) next() (string, bool) {
	start := t.nextPosition(t.position)
	if start >= len(t.input) {
		t.position = start
		return "", false
	}
	if t.returnDelims {
		if _, delimiter := t.delimiters[t.input[start]]; delimiter {
			t.position = start + 1
			return string(t.input[start]), true
		}
	}
	end := start
	for end < len(t.input) {
		if _, delimiter := t.delimiters[t.input[end]]; delimiter {
			break
		}
		end++
	}
	t.position = end
	return string(t.input[start:end]), true
}

func runeSet(value string) map[rune]struct{} {
	set := make(map[rune]struct{})
	for _, character := range value {
		set[character] = struct{}{}
	}
	return set
}

func portableMath(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectGet || invocation.Op == ObjectInvoke && len(invocation.Arguments) == 0 {
		switch invocation.Message {
		case "PI":
			return Double(math.Pi), true, nil
		case "E":
			return Double(math.E), true, nil
		}
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	a := invocation.Arg(0).Float64()
	switch invocation.Message {
	case "pow":
		return Double(math.Pow(a, invocation.Arg(1).Float64())), true, nil
	case "abs":
		return Double(math.Abs(a)), true, nil
	case "ceil":
		return Double(math.Ceil(a)), true, nil
	case "floor":
		return Double(math.Floor(a)), true, nil
	case "sqrt":
		return Double(math.Sqrt(a)), true, nil
	case "log":
		return Double(math.Log(a)), true, nil
	case "exp":
		return Double(math.Exp(a)), true, nil
	case "sin":
		return Double(math.Sin(a)), true, nil
	case "cos":
		return Double(math.Cos(a)), true, nil
	case "tan":
		return Double(math.Tan(a)), true, nil
	case "round":
		return Long(sleepJavaRound(a)), true, nil
	case "min":
		return Double(math.Min(a, invocation.Arg(1).Float64())), true, nil
	case "max":
		return Double(math.Max(a, invocation.Arg(1).Float64())), true, nil
	}
	return Null(), false, nil
}

func portableInteger(invocation ObjectInvocation, long bool) (Value, bool, error) {
	if invocation.Op != ObjectInvoke && invocation.Op != ObjectConstruct {
		return Null(), false, nil
	}
	name := invocation.Message
	if invocation.Op == ObjectConstruct {
		name = "valueOf"
	}
	switch name {
	case "toBinaryString":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, portableIntegerClass(long)), true, nil
		}
		if long {
			return String(strconv.FormatUint(uint64(invocation.Arg(0).Int64()), 2)), true, nil
		}
		return String(strconv.FormatUint(uint64(uint32(invocation.Arg(0).Int32())), 2)), true, nil
	case "parseInt", "valueOf":
		if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 2 {
			if invocation.Op == ObjectConstruct {
				message := fmt.Sprintf("no constructor matching %s(%s)", portableIntegerClass(long), portableObjectArgumentList(invocation))
				return portableObjectWarning(invocation, message), true, nil
			}
			return portableNoMatchingMethod(invocation, portableIntegerClass(long)), true, nil
		}
		base := 10
		if len(invocation.Arguments) > 1 {
			base = int(invocation.Arg(1).Int32())
		}
		bits := 32
		if long {
			bits = 64
		}
		input := invocation.Arg(0).String()
		parsed, err := strconv.ParseInt(input, base, bits)
		if err != nil {
			message := fmt.Sprintf("For input string: %q", input)
			if base != 10 {
				message += fmt.Sprintf(" under radix %d", base)
			}
			return Null(), true, &portableJavaException{
				class:   "java.lang.NumberFormatException",
				message: message,
				text:    "java.lang.NumberFormatException: " + message,
				frame:   portableIntegerMethodFrame(long, name, len(invocation.Arguments)),
				cause:   err,
			}
		}
		if long {
			if invocation.Op == ObjectConstruct {
				return ObjectValue(&portableJavaPrimitive{class: "java.lang.Long", value: Long(parsed)}), true, nil
			}
			return Long(parsed), true, nil
		}
		if invocation.Op == ObjectConstruct {
			return ObjectValue(&portableJavaPrimitive{class: "java.lang.Integer", value: Int(int32(parsed))}), true, nil
		}
		return Int(int32(parsed)), true, nil
	}
	return Null(), false, nil
}

func portableIntegerClass(long bool) string {
	if long {
		return "java.lang.Long"
	}
	return "java.lang.Integer"
}

func portableIntegerMethodFrame(long bool, name string, arguments int) string {
	class, primitive := "java.lang.Integer", "int"
	if long {
		class, primitive = "java.lang.Long", "long"
	}
	parameters := "java.lang.String"
	if arguments > 1 {
		parameters += ",int"
	}
	if name == "valueOf" {
		primitive = class
	}
	return fmt.Sprintf("public static %s %s.%s(%s) throws java.lang.NumberFormatException", primitive, class, name, parameters)
}

// portableObjectCallTrace returns CallRequest.MethodCallRequest's observable
// trace spelling for portable Java operations. The JVM resolves bare class
// references before formatting these calls.
func portableObjectCallTrace(invocation ObjectInvocation) (string, bool) {
	class := invocation.Class
	if invocation.Op == ObjectConstruct {
		if class == "org.hick.blah.SqueezeBox" || class == "com.eric.Eric" {
			var builder strings.Builder
			builder.WriteString("[new ")
			builder.WriteString(class)
			if len(invocation.Arguments) != 0 {
				builder.WriteString(": ")
				for index, argument := range invocation.Arguments {
					if index != 0 {
						builder.WriteString(", ")
					}
					builder.WriteString(describeTraceValue(argument.Resolve()))
				}
			}
			builder.WriteByte(']')
			return builder.String(), true
		}
		if name := portableJavaClassName(class); name == "StringBuilder" || name == "StringBuffer" {
			class = "java.lang." + name
			var builder strings.Builder
			builder.WriteString("[new ")
			builder.WriteString(class)
			if len(invocation.Arguments) != 0 {
				builder.WriteString(": ")
				for index, argument := range invocation.Arguments {
					if index != 0 {
						builder.WriteString(", ")
					}
					builder.WriteString(describeTraceValue(argument.Resolve()))
				}
			}
			builder.WriteByte(']')
			return builder.String(), true
		}
		if name := portableJavaClassName(class); name == "LinkedList" || name == "ArrayList" ||
			name == "HashSet" || name == "LinkedHashSet" || name == "TreeSet" ||
			name == "HashMap" || name == "Hashtable" || name == "LinkedHashMap" || name == "TreeMap" {
			class = "java.util." + name
			var builder strings.Builder
			builder.WriteString("[new ")
			builder.WriteString(class)
			if len(invocation.Arguments) != 0 {
				builder.WriteString(": ")
				for index, argument := range invocation.Arguments {
					if index != 0 {
						builder.WriteString(", ")
					}
					builder.WriteString(describeTraceValue(argument.Resolve()))
				}
			}
			builder.WriteByte(']')
			return builder.String(), true
		}
		if portableJavaClassName(class) != "NoSuchElementException" || len(invocation.Arguments) > 1 {
			return "", false
		}
		class = "java.util.NoSuchElementException"
		var builder strings.Builder
		builder.WriteString("[new ")
		builder.WriteString(class)
		if len(invocation.Arguments) != 0 {
			builder.WriteString(": ")
			builder.WriteString(describeTraceValue(invocation.Arg(0)))
		}
		builder.WriteByte(']')
		return builder.String(), true
	}
	if invocation.Op != ObjectInvoke {
		return "", false
	}
	if object, ok := invocation.Target.Object(); ok {
		switch object.(type) {
		case *portableSqueezeBox, *portableEric, *portableCompileException, classReference, sleepClass:
			return formatClosureCall(invocation.Target, invocation.Message, invocation.Arguments), true
		}
		if proxy, ok := object.(*portableJavaProxy); ok && proxy != nil {
			return formatClosureCall(invocation.Target, invocation.Message, invocation.Arguments), true
		}
		if stream, ok := object.(*portableJavaPrintStream); ok && stream != nil {
			return formatClosureCall(invocation.Target, invocation.Message, invocation.Arguments), true
		}
		if buffer, ok := object.(*portableJavaStringBuffer); ok && buffer != nil {
			return formatClosureCall(invocation.Target, invocation.Message, invocation.Arguments), true
		}
		if exception, ok := object.(*portableJavaException); ok && exception != nil {
			switch invocation.Message {
			case "getClass", "getMessage", "getLocalizedMessage", "toString", "getCause":
				return formatClosureCall(invocation.Target, invocation.Message, invocation.Arguments), true
			}
		}
	}
	if portableJavaClassName(class) == "Collections" && invocation.Message == "list" && len(invocation.Arguments) == 1 {
		return "[java.util.Collections list: " + describeTraceValue(invocation.Arg(0)) + "]", true
	}
	if portableJavaClassName(class) == "SleepUtils" {
		var builder strings.Builder
		builder.WriteString("[sleep.runtime.SleepUtils ")
		builder.WriteString(invocation.Message)
		if len(invocation.Arguments) != 0 {
			builder.WriteString(": ")
			for index, argument := range invocation.Arguments {
				if index != 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(describeTraceValue(argument.Resolve()))
			}
		}
		builder.WriteByte(']')
		return builder.String(), true
	}
	if portableJavaClassName(class) == "Math" {
		var builder strings.Builder
		builder.WriteString("[java.lang.Math ")
		builder.WriteString(invocation.Message)
		if len(invocation.Arguments) != 0 {
			builder.WriteString(": ")
			for index, argument := range invocation.Arguments {
				if index != 0 {
					builder.WriteString(", ")
				}
				builder.WriteString(describeTraceValue(argument.Resolve()))
			}
		}
		builder.WriteByte(']')
		return builder.String(), true
	}
	class = strings.TrimPrefix(class, "java.lang.")
	if class != "Integer" && class != "Long" {
		return "", false
	}
	if invocation.Message != "parseInt" && invocation.Message != "valueOf" {
		return "", false
	}
	if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 2 {
		return "", false
	}

	var builder strings.Builder
	builder.WriteString("[java.lang.")
	builder.WriteString(class)
	builder.WriteByte(' ')
	builder.WriteString(invocation.Message)
	builder.WriteString(": ")
	for index, argument := range invocation.Arguments {
		if index != 0 {
			builder.WriteString(", ")
		}
		builder.WriteString(describeTraceValue(argument.Resolve()))
	}
	builder.WriteByte(']')
	return builder.String(), true
}

func portableCharacter(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.lang.Character"), true, nil
	}
	argument := invocation.Arg(0)
	var character rune
	if argument.Kind() == KindString {
		units := sleepStringUnits(argument)
		if len(units) != 1 {
			return portableNoMatchingMethod(invocation, "java.lang.Character"), true, nil
		}
		character = rune(units[0])
	} else {
		character = rune(argument.Int32())
	}
	result := false
	switch invocation.Message {
	case "isLetter":
		result = unicode.IsLetter(character)
	case "isDigit":
		result = unicode.IsDigit(character)
	case "isUpperCase":
		result = unicode.IsUpper(character)
	case "isLowerCase":
		result = unicode.IsLower(character)
	default:
		return Null(), false, nil
	}
	if result {
		return Int(1), true, nil
	}
	return Int(0), true, nil
}

func utf8FirstRune(value string) (rune, int) {
	for _, character := range value {
		return character, len(string(character))
	}
	return 0, 0
}
