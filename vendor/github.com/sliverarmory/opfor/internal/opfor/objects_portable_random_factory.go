package opfor

// portableJavaRandomGeneratorClass implements the portion of
// RandomGenerator.of(String) whose concrete algorithm already has a faithful
// portable implementation. OpenJDK's catalog is deliberately kept here so an
// unknown algorithm can retain Java's exact IllegalArgumentException while a
// known-but-not-portable algorithm remains available to an ObjectHost instead
// of being falsely reported as absent.
func portableJavaRandomGeneratorClass(invocation ObjectInvocation) (Value, bool, error) {
	if resolvePortableClassName(invocation.Class) != "java.util.random.RandomGenerator" ||
		invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	switch invocation.Message {
	case "of":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util.random.RandomGenerator"), true, nil
		}
		if invocation.Arg(0).IsNull() {
			return Null(), true, &portableJavaException{
				class: "java.lang.NullPointerException",
				text:  "java.lang.NullPointerException",
				frame: "public static java.util.random.RandomGenerator java.util.random.RandomGenerator.of(java.lang.String)",
			}
		}
		switch invocation.Arg(0).Kind() {
		case KindString, KindInt, KindLong, KindDouble:
			// ObjectUtilities converts ordinary scalar values to a Java String.
		default:
			return portableNoMatchingMethod(invocation, "java.util.random.RandomGenerator"), true, nil
		}
		name := invocation.Arg(0).String()
		if name == "Random" {
			return ObjectValue(newPortableJavaRandom(portableJavaRandomSeed())), true, nil
		}
		if _, exists := portableOpenJDKRandomGeneratorAlgorithms[name]; exists {
			// This is a real OpenJDK algorithm, but substituting classic Random
			// would violate its type, recurrence, and state-consumption contract.
			return Null(), false, nil
		}
		message := "No implementation of the random number generator algorithm \"" + name + "\" is available"
		return Null(), true, &portableJavaException{
			class:   "java.lang.IllegalArgumentException",
			message: message,
			text:    "java.lang.IllegalArgumentException: " + message,
			frame:   "public static java.util.random.RandomGenerator java.util.random.RandomGenerator.of(java.lang.String)",
		}
	case "getDefault":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.random.RandomGenerator"), true, nil
		}
		// The pinned JDK requires L32X64MixRandom. Leave first-class handling
		// to an importer until that distinct algorithm is implemented.
		return Null(), false, nil
	default:
		return Null(), false, nil
	}
}

var portableOpenJDKRandomGeneratorAlgorithms = map[string]struct{}{
	"L32X64MixRandom":       {},
	"L64X128MixRandom":      {},
	"L64X128StarStarRandom": {},
	"L64X256MixRandom":      {},
	"L64X1024MixRandom":     {},
	"L128X128MixRandom":     {},
	"L128X256MixRandom":     {},
	"L128X1024MixRandom":    {},
	"Random":                {},
	"SecureRandom":          {},
	"SplittableRandom":      {},
	"Xoroshiro128PlusPlus":  {},
	"Xoshiro256PlusPlus":    {},
}
