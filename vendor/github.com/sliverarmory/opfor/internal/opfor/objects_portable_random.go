package opfor

import (
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

// portableJavaRandom implements the non-stream java.util.Random methods used
// by Sleep/Aggressor scripts. The mutex mirrors Random's thread-safe atomic
// seed transitions while reusing Sleep's already-audited 48-bit generator.
type portableJavaRandom struct {
	mu                   sync.Mutex
	random               *sleepJavaRandom
	nextNextGaussian     float64
	haveNextNextGaussian bool
}

func newPortableJavaRandom(seed int64) *portableJavaRandom {
	return &portableJavaRandom{random: newSleepJavaRandom(seed)}
}

func (random *portableJavaRandom) String() string { return "java.util.Random" }

var portableJavaRandomUniquifier atomic.Uint64

func portableJavaRandomSeed() int64 {
	// OpenJDK's no-argument constructor combines a process-wide seed uniquifier
	// with System.nanoTime. Exact values are intentionally nondeterministic; the
	// observable contract here is independent instances with Java's recurrence.
	const (
		initial    = uint64(8682522807148012)
		multiplier = uint64(1181783497276652981)
	)
	for {
		observed := portableJavaRandomUniquifier.Load()
		current := observed
		if observed == 0 {
			current = initial
		}
		next := current * multiplier
		if portableJavaRandomUniquifier.CompareAndSwap(observed, next) {
			return int64(next) ^ time.Now().UnixNano()
		}
	}
}

func (random *portableJavaRandom) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op == ObjectTypeCheck {
		class := resolvePortableClassName(invocation.Class)
		return Bool(class == "java.util.Random" || class == "java.util.random.RandomGenerator" ||
			class == "java.lang.Object" || class == "java.io.Serializable"), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if random == nil || random.random == nil {
		return Null(), true, fmt.Errorf("java.lang.NullPointerException: java.util.Random state is null")
	}

	random.mu.Lock()
	defer random.mu.Unlock()
	switch invocation.Message {
	case "setSeed":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
		random.random = newSleepJavaRandom(sleepInt64(invocation.Arg(0)))
		random.nextNextGaussian = 0
		random.haveNextNextGaussian = false
		return Null(), true, nil
	case "nextBytes":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
		if invocation.Arg(0).IsNull() {
			return Null(), true, &portableJavaException{
				class:   "java.lang.NullPointerException",
				message: "Cannot read the array length because bytes is null",
				text:    "java.lang.NullPointerException: Cannot read the array length because bytes is null",
				frame:   "public void java.util.Random.nextBytes(byte[])",
			}
		}
		bytes, ok := portableJavaRandomByteArray(invocation.Arg(0))
		if !ok {
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
		for index := 0; index < len(bytes.values); {
			bits := int32(random.random.next(32))
			remaining := min(len(bytes.values)-index, 4)
			for count := 0; count < remaining; count++ {
				bytes.values[index] = Int(int32(int8(bits)))
				index++
				bits >>= 8
			}
		}
		return Null(), true, nil
	case "nextInt":
		switch len(invocation.Arguments) {
		case 0:
			return Int(random.nextInt()), true, nil
		case 1:
			bound := sleepInt32(invocation.Arg(0))
			value, err := random.random.nextInt(bound)
			if err != nil {
				return Null(), true, &portableJavaException{
					class:   "java.lang.IllegalArgumentException",
					message: "bound must be positive",
					text:    "java.lang.IllegalArgumentException: bound must be positive",
					frame:   "public int java.util.Random.nextInt(int)",
					cause:   err,
				}
			}
			return Int(value), true, nil
		case 2:
			origin := sleepInt32(invocation.Arg(0))
			bound := sleepInt32(invocation.Arg(1))
			if origin >= bound {
				return Null(), true, portableJavaRandomArgumentException(
					"bound must be greater than origin",
					"public default int java.util.random.RandomGenerator.nextInt(int,int)",
				)
			}
			return Int(random.boundedNextInt(origin, bound)), true, nil
		default:
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
	case "nextLong":
		switch len(invocation.Arguments) {
		case 0:
			return Long(random.nextLong()), true, nil
		case 1:
			bound := sleepInt64(invocation.Arg(0))
			if bound <= 0 {
				return Null(), true, portableJavaRandomArgumentException(
					"bound must be positive",
					"public default long java.util.random.RandomGenerator.nextLong(long)",
				)
			}
			return Long(random.boundedNextLong(0, bound)), true, nil
		case 2:
			origin := sleepInt64(invocation.Arg(0))
			bound := sleepInt64(invocation.Arg(1))
			if origin >= bound {
				return Null(), true, portableJavaRandomArgumentException(
					"bound must be greater than origin",
					"public default long java.util.random.RandomGenerator.nextLong(long,long)",
				)
			}
			return Long(random.boundedNextLong(origin, bound)), true, nil
		default:
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
	case "nextBoolean":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
		return portableJavaBooleanValue(random.random.next(1) != 0), true, nil
	case "nextFloat":
		switch len(invocation.Arguments) {
		case 0:
			return Double(float64(random.nextFloat())), true, nil
		case 1:
			bound := float32(sleepFloat64(invocation.Arg(0)))
			if !(0 < bound && bound < float32(math.Inf(1))) {
				return Null(), true, portableJavaRandomArgumentException(
					"bound must be finite and positive",
					"public default float java.util.random.RandomGenerator.nextFloat(float)",
				)
			}
			return Double(float64(random.boundedNextFloat(0, bound))), true, nil
		case 2:
			origin := float32(sleepFloat64(invocation.Arg(0)))
			bound := float32(sleepFloat64(invocation.Arg(1)))
			if !(float32(math.Inf(-1)) < origin && origin < bound && bound < float32(math.Inf(1))) {
				return Null(), true, portableJavaRandomArgumentException(
					"bound must be greater than origin",
					"public default float java.util.random.RandomGenerator.nextFloat(float,float)",
				)
			}
			return Double(float64(random.boundedNextFloat(origin, bound))), true, nil
		default:
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
	case "nextDouble":
		switch len(invocation.Arguments) {
		case 0:
			return Double(random.random.nextDouble()), true, nil
		case 1:
			bound := sleepFloat64(invocation.Arg(0))
			if !(0 < bound && bound < math.Inf(1)) {
				return Null(), true, portableJavaRandomArgumentException(
					"bound must be finite and positive",
					"public default double java.util.random.RandomGenerator.nextDouble(double)",
				)
			}
			return Double(random.boundedNextDouble(0, bound)), true, nil
		case 2:
			origin := sleepFloat64(invocation.Arg(0))
			bound := sleepFloat64(invocation.Arg(1))
			if !(math.Inf(-1) < origin && origin < bound && bound < math.Inf(1)) {
				return Null(), true, portableJavaRandomArgumentException(
					"bound must be greater than origin",
					"public default double java.util.random.RandomGenerator.nextDouble(double,double)",
				)
			}
			return Double(random.boundedNextDouble(origin, bound)), true, nil
		default:
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
	case "nextGaussian":
		switch len(invocation.Arguments) {
		case 0:
			if random.haveNextNextGaussian {
				random.haveNextNextGaussian = false
				return Double(random.nextNextGaussian), true, nil
			}
			var first, second, radius float64
			for {
				first = 2*random.random.nextDouble() - 1
				second = 2*random.random.nextDouble() - 1
				radius = first*first + second*second
				if radius < 1 && radius != 0 {
					break
				}
			}
			multiplier := math.Sqrt(-2 * math.Log(radius) / radius)
			random.nextNextGaussian = second * multiplier
			random.haveNextNextGaussian = true
			return Double(first * multiplier), true, nil
		case 2:
			mean := sleepFloat64(invocation.Arg(0))
			standardDeviation := sleepFloat64(invocation.Arg(1))
			if standardDeviation < 0 {
				return Null(), true, portableJavaRandomArgumentException(
					"standard deviation must be non-negative",
					"public default double java.util.random.RandomGenerator.nextGaussian(double,double)",
				)
			}
			// The explicit conversion is a Go-spec rounding point. Without it,
			// an implementation may contract the multiply and add into an FMA,
			// while Java evaluates this inherited default as two strict-double
			// operations.
			product := float64(standardDeviation * random.nextGeneratorGaussian())
			return Double(mean + product), true, nil
		default:
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
	case "nextExponential":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
		return Double(random.nextExponential()), true, nil
	case "isDeprecated":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, "java.util.Random"), true, nil
		}
		return portableJavaBooleanValue(false), true, nil
	case "ints", "longs", "doubles":
		return portableJavaRandomStreamForInvocation(random, invocation)
	}
	return Null(), false, nil
}

func portableJavaRandomArgumentException(message, frame string) *portableJavaException {
	return &portableJavaException{
		class:   "java.lang.IllegalArgumentException",
		message: message,
		text:    "java.lang.IllegalArgumentException: " + message,
		frame:   frame,
	}
}

func (random *portableJavaRandom) nextInt() int32 {
	return int32(random.random.next(32))
}

func (random *portableJavaRandom) nextLong() int64 {
	high := int64(random.nextInt())
	low := int64(random.nextInt())
	return (high << 32) + low
}

func (random *portableJavaRandom) nextFloat() float32 {
	return float32(random.random.next(24)) / float32(uint32(1)<<24)
}

// boundedNextInt reproduces RandomGenerator's two-argument default. It is
// intentionally separate from Random.nextInt(int), whose historical
// high-bit algorithm is specified by Random itself and remains in
// sleepJavaRandom.nextInt.
func (random *portableJavaRandom) boundedNextInt(origin, bound int32) int32 {
	r := random.nextInt()
	n := bound - origin
	m := n - 1
	if n&m == 0 {
		return (r & m) + origin
	}
	if n > 0 {
		for {
			u := int32(uint32(r) >> 1)
			r = u % n
			if u+m-r >= 0 {
				return r + origin
			}
			r = random.nextInt()
		}
	}
	for r < origin || r >= bound {
		r = random.nextInt()
	}
	return r
}

// boundedNextLong reproduces both RandomGenerator long defaults. Callers
// validate the public range before this helper consumes generator state.
func (random *portableJavaRandom) boundedNextLong(origin, bound int64) int64 {
	r := random.nextLong()
	n := bound - origin
	m := n - 1
	if n&m == 0 {
		return (r & m) + origin
	}
	if n > 0 {
		for {
			u := int64(uint64(r) >> 1)
			r = u % n
			if u+m-r >= 0 {
				return r + origin
			}
			r = random.nextLong()
		}
	}
	for r < origin || r >= bound {
		r = random.nextLong()
	}
	return r
}

func (random *portableJavaRandom) boundedNextFloat(origin, bound float32) float32 {
	r := random.nextFloat()
	delta := bound - origin
	if delta < float32(math.Inf(1)) {
		// Explicit conversions preserve Java's mandatory intermediate float
		// rounding and prevent Go from contracting the multiply/add to an FMA.
		scaled := float32(r * delta)
		r = float32(scaled + origin)
	} else {
		halfOrigin := float32(0.5) * origin
		halfBound := float32(float32(0.5) * bound)
		halfRange := float32(halfBound - halfOrigin)
		scaled := float32(r * halfRange)
		translated := float32(scaled + halfOrigin)
		r = float32(translated * 2)
	}
	if r >= bound {
		return math.Nextafter32(bound, float32(math.Inf(-1)))
	}
	return r
}

func (random *portableJavaRandom) boundedNextDouble(origin, bound float64) float64 {
	r := random.random.nextDouble()
	delta := bound - origin
	if delta < math.Inf(1) {
		// See boundedNextFloat: Java rounds the product before addition.
		scaled := float64(r * delta)
		r = float64(scaled + origin)
	} else {
		halfOrigin := 0.5 * origin
		halfBound := float64(0.5 * bound)
		halfRange := float64(halfBound - halfOrigin)
		scaled := float64(r * halfRange)
		translated := float64(scaled + halfOrigin)
		r = float64(translated * 2)
	}
	if r >= bound {
		return math.Nextafter(bound, math.Inf(-1))
	}
	return r
}

func portableJavaRandomByteArray(value Value) (*portableJavaArray, bool) {
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	array, ok := object.(*portableJavaArray)
	if !ok || array == nil || array.typeInfo.name != "byte" || len(array.dimensions) != 1 {
		return nil, false
	}
	return array, true
}
