package opfor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"
)

// stringNumberFunctions returns portable functions implemented by Sleep 2.1's
// BasicStrings, BasicNumbers, RegexBridge, and BasicUtilities bridges. The
// returned map owns its per-script random-number state and should be retained
// for the lifetime of the Runtime.
func (r *Runtime) stringNumberFunctions() map[string]NativeFunc {
	state := &stringNumberBuiltinState{random: make(map[ScriptID]*sleepJavaRandom)}
	return map[string]NativeFunc{
		"strlen":      builtinSleepStrlen,
		"left":        builtinSleepLeft,
		"right":       builtinSleepRight,
		"charAt":      builtinSleepCharAt,
		"byteAt":      builtinSleepCharAt,
		"replaceAt":   builtinSleepReplaceAt,
		"indexOf":     builtinSleepIndexOf,
		"lindexOf":    builtinSleepIndexOf,
		"lastIndexOf": builtinSleepIndexOf,
		"replace":     builtinSleepReplace,
		"split":       builtinSleepSplit,
		"join":        builtinSleepJoin,
		"tr":          builtinSleepTr,
		"asc":         builtinSleepAsc,
		"chr":         builtinSleepChr,
		"reverse":     builtinSleepReverse,
		"abs":         builtinSleepAbs,
		"round":       builtinSleepRound,
		"floor":       builtinSleepFloor,
		"ceil":        builtinSleepCeil,
		"sqrt":        builtinSleepSqrt,
		"log":         builtinSleepLog,
		"sin":         builtinSleepSin,
		"cos":         builtinSleepCos,
		"tan":         builtinSleepTan,
		"rand":        state.rand,
		"srand":       state.srand,
	}
}

func builtinSleepStrlen(_ context.Context, invocation Invocation) (Value, error) {
	value, err := sleepBridgeArgument(invocation, 0)
	if err != nil {
		return Null(), err
	}
	return Int(int32(sleepStringLength(value))), nil
}

func builtinSleepLeft(_ context.Context, invocation Invocation) (Value, error) {
	value, err := sleepBridgeArgument(invocation, 0)
	if err != nil {
		return Null(), err
	}
	count, err := sleepBridgeArgument(invocation, 1)
	if err != nil {
		return Null(), err
	}
	return sleepValueSubstring(invocation.Name, value, 0, int(sleepInt32(count)))
}

func builtinSleepRight(_ context.Context, invocation Invocation) (Value, error) {
	value, err := sleepBridgeArgument(invocation, 0)
	if err != nil {
		return Null(), err
	}
	countValue, err := sleepBridgeArgument(invocation, 1)
	if err != nil {
		return Null(), err
	}
	argument := sleepStringCoercion(value)
	count := sleepInt32(countValue)
	return sleepValueSubstring(invocation.Name, argument, int(-count), sleepStringLength(argument))
}

func sleepValueSubstring(name string, value Value, originalStart, originalEnd int) (Value, error) {
	value = sleepStringCoercion(value)
	length := sleepStringLength(value)
	start := originalStart
	if start < 0 {
		start += length
	}
	end := originalEnd
	if end < 0 {
		end += length
	}
	if end > length {
		end = length
	}

	// BasicStrings.substring returns early for equal normalized indices, even
	// when those equal indices would otherwise be outside the string.
	if start == end {
		return String(""), nil
	}
	if start > end {
		return Null(), sleepBridgeIllegalArgument(fmt.Sprintf(
			"&%s: illegal substring(%s, %d -> %d, %d -> %d) indices",
			builtinName(name), value.Describe(), originalStart, start, originalEnd, end,
		))
	}
	if start < 0 || start > length || end < 0 || end > length {
		return Null(), sleepBridgeInvalidIndex(fmt.Sprintf(
			"Range [%d, %d) out of bounds for length %d", start, end, length,
		))
	}
	return sleepStringValueSlice(value, start, end), nil
}

// sleepByteSubstring is retained for internal callers that start with a Go
// string rather than an existing Sleep Value.
func sleepByteSubstring(name, value string, originalStart, originalEnd int) (Value, error) {
	return sleepValueSubstring(name, String(value), originalStart, originalEnd)
}

func builtinSleepCharAt(_ context.Context, invocation Invocation) (Value, error) {
	value, err := sleepBridgeArgument(invocation, 0)
	if err != nil {
		return Null(), err
	}
	argument := sleepStringCoercion(value)
	length := sleepStringLength(argument)
	index := int(sleepInt32(invocation.Arg(1)))
	if index < 0 {
		index += length
	}
	if index < 0 || index >= length {
		return Null(), sleepBridgeInvalidIndex(fmt.Sprintf("Index %d out of bounds for length %d", index, length))
	}
	dispatchName := builtinName(invocation.Name)
	if invocation.bridgeDispatchName != "" {
		dispatchName = invocation.bridgeDispatchName
	}
	if dispatchName == "byteAt" {
		return Int(int32(sleepStringUnits(argument)[index])), nil
	}
	return sleepStringValueSlice(argument, index, index+1), nil
}

func builtinSleepReplaceAt(_ context.Context, invocation Invocation) (Value, error) {
	value := sleepStringCoercion(invocation.Arg(0))
	replacement := sleepStringCoercion(invocation.Arg(1))
	length := int32(sleepStringLength(value))
	index := sleepInt32(invocation.Arg(2))
	if index < 0 {
		index += length
	}
	count := int32(sleepStringLength(replacement))
	if len(invocation.Arguments) > 3 {
		count = sleepInt32(invocation.Arg(3))
	}
	// AbstractStringBuilder.delete performs this addition as a Java int, then
	// clamps only a high end before checkRangeSIOOBE validates the range. Both
	// the overflow and the clamped endpoint are observable in Sleep's warning.
	end := index + count
	if end > length {
		end = length
	}
	if index < 0 || index > end || index > length {
		return Null(), sleepBridgeInvalidIndex(fmt.Sprintf(
			"Range [%d, %d) out of bounds for length %d", index, end, length,
		))
	}
	return sleepStringConcat(
		sleepStringValueSlice(value, 0, int(index)),
		replacement,
		sleepStringValueSlice(value, int(end), int(length)),
	), nil
}

func builtinSleepIndexOf(_ context.Context, invocation Invocation) (Value, error) {
	valueArgument, err := sleepBridgeArgument(invocation, 0)
	if err != nil {
		return Null(), err
	}
	itemArgument, err := sleepBridgeArgument(invocation, 1)
	if err != nil {
		return Null(), err
	}
	value := sleepStringCoercion(valueArgument)
	item := sleepStringCoercion(itemArgument)
	length := sleepStringLength(value)
	name := builtinName(invocation.Name)
	if name == "lindexOf" || name == "lastIndexOf" {
		start := length
		if len(invocation.Arguments) > 2 {
			start = int(sleepInt32(invocation.Arg(2)))
		}
		if start < 0 {
			start += length
		}
		index := sleepStringUnitIndex(value, item, start, true)
		if index < 0 {
			return Null(), nil
		}
		return Int(int32(index)), nil
	}

	start := 0
	if len(invocation.Arguments) > 2 {
		start = int(sleepInt32(invocation.Arg(2)))
	}
	if start < 0 {
		start += length
	}
	index := sleepStringUnitIndex(value, item, start, false)
	if index < 0 {
		return Null(), nil
	}
	return Int(int32(index)), nil
}

func builtinSleepAsc(_ context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		return Int(0), nil
	}
	argument := sleepStringCoercion(invocation.Arg(0))
	units := sleepStringUnits(argument)
	if len(units) == 0 {
		return Null(), sleepBridgeInvalidIndex("Index 0 out of bounds for length 0")
	}
	return Int(int32(units[0])), nil
}

func builtinSleepChr(_ context.Context, invocation Invocation) (Value, error) {
	unit := uint16(sleepInt32(invocation.Arg(0)))
	return sleepUTF16CharacterValue(unit), nil
}

func builtinSleepSplit(ctx context.Context, invocation Invocation) (Value, error) {
	patternValue, err := sleepBridgeArgument(invocation, 0)
	if err != nil {
		return Null(), err
	}
	textValue, err := sleepBridgeArgument(invocation, 1)
	if err != nil {
		return Null(), err
	}
	patternValue = sleepStringCoercion(patternValue)
	pattern, err := invocation.Runtime.compileSleepRegexBridge(sleepCanonicalString(patternValue), false)
	if err != nil {
		return Null(), err
	}
	textValue = sleepStringCoercion(textValue)
	text := sleepCanonicalString(textValue)
	limit := int32(0)
	if len(invocation.Arguments) > 2 {
		limit = sleepInt32(invocation.Arg(2))
	}
	if sleepStringLength(patternValue) == 0 {
		return portableJavaStringSplitEmptyPattern(ctx, invocation.Runtime, textValue, limit)
	}

	pieces, err := pattern.splitContext(ctx, text, int(limit), func() error {
		return reserveCollectionEntries(invocation.Runtime, 1)
	})
	if err != nil {
		return Null(), fmt.Errorf("&%s: regular expression match: %w", builtinName(invocation.Name), err)
	}
	values := make([]Value, len(pieces))
	for index, piece := range pieces {
		values[index] = sleepStringValueFromCanonical(piece)
	}
	if len(pieces) == 1 && pieces[0] == text {
		values[0] = textValue
	}
	return ArrayValue(NewArray(values...)), nil
}

func builtinSleepJoin(ctx context.Context, invocation Invocation) (Value, error) {
	separatorValue, err := sleepBridgeArgument(invocation, 0)
	if err != nil {
		return Null(), err
	}
	if len(invocation.Arguments) < 2 {
		return String(""), nil
	}
	values, err := iteratorValues(ctx, invocation.Arg(1), invocation.Name)
	if err != nil {
		return Null(), err
	}
	separator := sleepStringCoercion(separatorValue)
	result := String("")
	for index, value := range values {
		if index > 0 {
			result = sleepStringConcat(result, separator)
		}
		result = sleepStringConcat(result, value)
	}
	return result, nil
}

func builtinSleepReplace(ctx context.Context, invocation Invocation) (Value, error) {
	inputValue := sleepStringCoercion(invocation.Arg(0))
	input := sleepCanonicalString(inputValue)
	patternValue := sleepStringCoercion(invocation.Arg(1))
	pattern, err := invocation.Runtime.compileSleepRegexBridge(sleepCanonicalString(patternValue), false)
	if err != nil {
		return Null(), err
	}
	replacement := sleepStringCoercion(invocation.Arg(2))
	limit := int32(-1)
	if len(invocation.Arguments) > 3 {
		limit = sleepInt32(invocation.Arg(3))
	}
	if limit == 0 {
		return inputValue, nil
	}
	if sleepStringLength(patternValue) == 0 {
		value, emptyErr := portableJavaStringReplaceEmptyPatternLimit(ctx, pattern, replacement, inputValue, int(limit))
		if emptyErr != nil {
			return Null(), sleepRegexBridgeReplacementError(emptyErr)
		}
		return value, nil
	}
	matchLimit := -1
	if limit > 0 {
		matchLimit = int(limit)
	}
	matches, err := pattern.FindAllStringSubmatchUTF16IndexContext(ctx, input, matchLimit)
	if err != nil {
		return Null(), fmt.Errorf("&%s: regular expression match: %w", builtinName(invocation.Name), err)
	}
	if len(matches) == 0 {
		return inputValue, nil
	}

	result := newPortableJavaStringBuilder(sleepStringLength(inputValue))
	last := 0
	for _, match := range matches {
		if err := result.appendRange(inputValue, last, match[0]); err != nil {
			return Null(), err
		}
		if err := appendPortableJavaReplacement(ctx, result, pattern, replacement, inputValue, match); err != nil {
			return Null(), sleepRegexBridgeReplacementError(err)
		}
		last = match[1]
	}
	if err := result.appendRange(inputValue, last, sleepStringLength(inputValue)); err != nil {
		return Null(), err
	}
	return result.value(), nil
}

func builtinSleepReverse(ctx context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 0 {
		return ArrayValue(NewArray()), nil
	}
	values, err := iteratorValuesForCollection(ctx, invocation.Runtime, invocation.Arg(0), invocation.Name)
	if err != nil {
		return Null(), err
	}
	for left, right := 0, len(values)-1; left < right; left, right = left+1, right-1 {
		values[left], values[right] = values[right], values[left]
	}
	return ArrayValue(NewArray(values...)), nil
}

func builtinSleepAbs(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Abs(sleepFloat64(invocation.Arg(0)))), nil
}

func builtinSleepRound(_ context.Context, invocation Invocation) (Value, error) {
	if len(invocation.Arguments) == 1 {
		return Long(sleepJavaRound(sleepFloat64(invocation.Arg(0)))), nil
	}
	number := sleepFloat64(invocation.Arg(0))
	places := math.Pow(10, float64(sleepInt32(invocation.Arg(1))))
	return Double(float64(sleepJavaRound(number*places)) / places), nil
}

func sleepJavaRound(value float64) int64 {
	switch {
	case math.IsNaN(value):
		return 0
	case value >= float64(math.MaxInt64):
		return math.MaxInt64
	case value <= float64(math.MinInt64):
		return math.MinInt64
	default:
		return int64(math.Floor(value + 0.5))
	}
}

func builtinSleepFloor(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Floor(sleepFloat64(invocation.Arg(0)))), nil
}

func builtinSleepCeil(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Ceil(sleepFloat64(invocation.Arg(0)))), nil
}

func builtinSleepSqrt(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Sqrt(sleepFloat64(invocation.Arg(0)))), nil
}

func builtinSleepLog(_ context.Context, invocation Invocation) (Value, error) {
	switch len(invocation.Arguments) {
	case 1:
		return Double(math.Log(sleepFloat64(invocation.Arg(0)))), nil
	case 2:
		return Double(math.Log(sleepFloat64(invocation.Arg(0))) /
			math.Log(sleepFloat64(invocation.Arg(1)))), nil
	default:
		return Null(), nil
	}
}

func builtinSleepSin(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Sin(sleepFloat64(invocation.Arg(0)))), nil
}

func builtinSleepCos(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Cos(sleepFloat64(invocation.Arg(0)))), nil
}

func builtinSleepTan(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Tan(sleepFloat64(invocation.Arg(0)))), nil
}

func requireSleepBuiltinArguments(invocation Invocation, count int) error {
	if len(invocation.Arguments) >= count {
		return nil
	}
	return fmt.Errorf("&%s: expected at least %d argument(s), received %d",
		builtinName(invocation.Name), count, len(invocation.Arguments))
}

type stringNumberBuiltinState struct {
	mu      sync.Mutex
	random  map[ScriptID]*sleepJavaRandom
	entropy uint64
}

func (state *stringNumberBuiltinState) srand(_ context.Context, invocation Invocation) (Value, error) {
	state.mu.Lock()
	state.random[invocation.Script] = newSleepJavaRandom(sleepInt64(invocation.Arg(0)))
	state.mu.Unlock()
	return Null(), nil
}

func (state *stringNumberBuiltinState) rand(ctx context.Context, invocation Invocation) (Value, error) {
	state.mu.Lock()
	defer state.mu.Unlock()
	random := state.random[invocation.Script]
	if random == nil {
		state.entropy++
		seed := time.Now().UnixNano() ^ int64(state.entropy*0x9e3779b97f4a7c15)
		random = newSleepJavaRandom(seed)
		state.random[invocation.Script] = random
	}
	if len(invocation.Arguments) == 0 {
		return Double(random.nextDouble()), nil
	}
	argument := invocation.Arg(0)
	if array, ok := argument.Array(); ok {
		if array.Len() == 0 {
			return Null(), sleepBridgeIllegalArgument("bound must be positive")
		}
		index, err := random.nextInt(int32(array.Len()))
		if err != nil {
			return Null(), sleepBridgeIllegalArgument(err.Error())
		}
		value, _, err := array.getForInvocation(ctx, invocation, int(index))
		if err != nil {
			return Null(), err
		}
		return value, nil
	}
	bound := sleepInt32(argument)
	value, err := random.nextInt(bound)
	if err != nil {
		return Null(), sleepBridgeIllegalArgument(err.Error())
	}
	return Int(value), nil
}

// sleepJavaRandom is java.util.Random's 48-bit linear-congruential generator.
// Reproducing it makes srand-controlled Aggressor scripts deterministic across
// the original Java and Go runtimes.
type sleepJavaRandom struct {
	seed uint64
}

const (
	sleepJavaRandomMultiplier = uint64(0x5deece66d)
	sleepJavaRandomAddend     = uint64(0xb)
	sleepJavaRandomMask       = uint64(1<<48 - 1)
)

func newSleepJavaRandom(seed int64) *sleepJavaRandom {
	return &sleepJavaRandom{seed: (uint64(seed) ^ sleepJavaRandomMultiplier) & sleepJavaRandomMask}
}

func (random *sleepJavaRandom) next(bits uint) uint64 {
	random.seed = (random.seed*sleepJavaRandomMultiplier + sleepJavaRandomAddend) & sleepJavaRandomMask
	return random.seed >> (48 - bits)
}

func (random *sleepJavaRandom) nextInt(bound int32) (int32, error) {
	if bound <= 0 {
		return 0, errors.New("bound must be positive")
	}
	if bound&(bound-1) == 0 {
		return int32((int64(bound) * int64(random.next(31))) >> 31), nil
	}
	for {
		bits := int32(random.next(31))
		value := bits % bound
		// The int32 conversion deliberately recreates Java's signed overflow
		// in Random.nextInt's rejection test.
		if int32(bits-value+(bound-1)) >= 0 {
			return value, nil
		}
	}
}

func (random *sleepJavaRandom) nextDouble() float64 {
	numerator := (random.next(26) << 27) + random.next(27)
	return float64(numerator) / float64(uint64(1)<<53)
}
