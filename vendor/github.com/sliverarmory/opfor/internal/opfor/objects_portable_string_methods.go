package opfor

import (
	"context"
	"errors"
	"fmt"
	"math"
)

// The method algorithms in this file are pinned to OpenJDK 17u commit
// 352633b5cef98ef3de7e562751222c38d76bb319: String.java supplies the public
// delegation and UTF-16 range rules, Pattern.java supplies split's limit and
// zero-width behavior, and Matcher.java supplies replacement parsing.

const portableJavaStringNativeLoopChunk = 32 * 1024

func portableJavaStringIntArgument(value Value) (int32, bool) {
	if value.IsNull() {
		return 0, true
	}
	switch value.Kind() {
	case KindInt, KindLong, KindDouble:
		return sleepInt32(value), true
	case KindObject:
		primitive, ok := value.data.(*portableJavaPrimitive)
		if ok && primitive != nil && primitive.className() == "java.lang.Integer" {
			return sleepInt32(primitive.sleepValue()), true
		}
	}
	return 0, false
}

func portableJavaStringCodePointIndex(
	ctx context.Context,
	units []uint16,
	codePoint int32,
	from int,
	reverse bool,
) (int, error) {
	if codePoint < 0 || codePoint > 0x10ffff {
		return -1, executionContextError(ctx)
	}
	width := 1
	first, second := uint16(codePoint), uint16(0)
	if codePoint >= 0x10000 {
		width = 2
		value := codePoint - 0x10000
		first = uint16(0xd800 + value>>10)
		second = uint16(0xdc00 + value&0x3ff)
	}
	if len(units) < width {
		return -1, executionContextError(ctx)
	}
	if !reverse {
		if from < 0 {
			from = 0
		}
		last := len(units) - width
		if from > last {
			return -1, executionContextError(ctx)
		}
		for index := from; index <= last; index++ {
			if err := portableJavaStringLoopCheck(ctx, index-from); err != nil {
				return -1, err
			}
			if units[index] == first && (width == 1 || units[index+1] == second) {
				return index, nil
			}
		}
		return -1, executionContextError(ctx)
	}

	if from > len(units)-width {
		from = len(units) - width
	}
	for index, iteration := from, 0; index >= 0; index, iteration = index-1, iteration+1 {
		if err := portableJavaStringLoopCheck(ctx, iteration); err != nil {
			return -1, err
		}
		if units[index] == first && (width == 1 || units[index+1] == second) {
			return index, nil
		}
	}
	return -1, executionContextError(ctx)
}

func portableJavaStringCompareToIgnoreCase(ctx context.Context, invocation ObjectInvocation, target Value) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	other := invocation.Arg(0)
	if other.IsNull() {
		return Null(), true, errors.New(`java.lang.NullPointerException: Cannot read field "value" because "s2" is null`)
	}
	if other.Kind() != KindString {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	comparison, err := sleepStringCompareIgnoreCase(ctx, target, other)
	if err != nil {
		return Null(), true, err
	}
	return Int(int32(comparison)), true, nil
}

func sleepStringCompareIgnoreCase(ctx context.Context, left, right Value) (int, error) {
	leftUnits, rightUnits := sleepStringUnits(left), sleepStringUnits(right)
	leftLatin1, err := sleepUTF16CanEncodeLatin1(ctx, leftUnits)
	if err != nil {
		return 0, err
	}
	rightLatin1, err := sleepUTF16CanEncodeLatin1(ctx, rightUnits)
	if err != nil {
		return 0, err
	}
	if leftLatin1 || rightLatin1 {
		return sleepUTF16CompareIgnoreCaseUnits(ctx, leftUnits, 0, len(leftUnits), rightUnits, 0, len(rightUnits))
	}
	return sleepUTF16CompareIgnoreCase(ctx, leftUnits, 0, len(leftUnits), rightUnits, 0, len(rightUnits))
}

func sleepJavaCodePointCompareIgnoreCase(left, right int32) int {
	if left == right {
		return 0
	}
	leftUpper := sleepJavaSimpleCase(rune(left), true)
	rightUpper := sleepJavaSimpleCase(rune(right), true)
	if leftUpper == rightUpper {
		return 0
	}
	leftLower := sleepJavaSimpleCase(leftUpper, false)
	rightLower := sleepJavaSimpleCase(rightUpper, false)
	return int(leftLower) - int(rightLower)
}

func sleepUTF16CanEncodeLatin1(ctx context.Context, units []uint16) (bool, error) {
	for index, unit := range units {
		if err := portableJavaStringLoopCheck(ctx, index); err != nil {
			return false, err
		}
		if unit > 0xff {
			return false, nil
		}
	}
	if err := executionContextError(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func sleepUTF16CompareIgnoreCaseUnits(
	ctx context.Context,
	left []uint16,
	leftStart, leftLength int,
	right []uint16,
	rightStart, rightLength int,
) (int, error) {
	limit := leftLength
	if rightLength < limit {
		limit = rightLength
	}
	for index := 0; index < limit; index++ {
		if err := portableJavaStringLoopCheck(ctx, index); err != nil {
			return 0, err
		}
		leftCodePoint := int32(left[leftStart+index])
		rightCodePoint := int32(right[rightStart+index])
		if leftCodePoint == rightCodePoint {
			continue
		}
		if difference := sleepJavaCodePointCompareIgnoreCase(leftCodePoint, rightCodePoint); difference != 0 {
			return difference, nil
		}
	}
	if err := executionContextError(ctx); err != nil {
		return 0, err
	}
	return leftLength - rightLength, nil
}

func sleepUTF16CompareIgnoreCase(
	ctx context.Context,
	left []uint16,
	leftStart, leftLength int,
	right []uint16,
	rightStart, rightLength int,
) (int, error) {
	leftEnd, rightEnd := leftStart+leftLength, rightStart+rightLength
	iterations := 0
	for leftIndex, rightIndex := leftStart, rightStart; leftIndex < leftEnd && rightIndex < rightEnd; leftIndex, rightIndex = leftIndex+1, rightIndex+1 {
		if err := portableJavaStringLoopCheck(ctx, iterations); err != nil {
			return 0, err
		}
		iterations++
		leftCodePoint, rightCodePoint := int32(left[leftIndex]), int32(right[rightIndex])
		if leftCodePoint == rightCodePoint || sleepJavaCodePointCompareIgnoreCase(leftCodePoint, rightCodePoint) == 0 {
			continue
		}

		var skip bool
		leftCodePoint, skip = sleepUTF16CodePointIncluding(left, leftCodePoint, leftIndex, leftStart, leftEnd)
		if skip {
			leftIndex++
		}
		rightCodePoint, skip = sleepUTF16CodePointIncluding(right, rightCodePoint, rightIndex, rightStart, rightEnd)
		if skip {
			rightIndex++
		}
		if difference := sleepJavaCodePointCompareIgnoreCase(leftCodePoint, rightCodePoint); difference != 0 {
			return difference, nil
		}
	}
	if err := executionContextError(ctx); err != nil {
		return 0, err
	}
	return leftLength - rightLength, nil
}

// sleepUTF16CodePointIncluding mirrors OpenJDK StringUTF16.codePointIncluding.
// skipNext reports the negated high-surrogate-pair sentinel used by the Java
// implementation to advance over the low surrogate in its enclosing loop.
func sleepUTF16CodePointIncluding(units []uint16, codePoint int32, index, start, end int) (combined int32, skipNext bool) {
	unit := uint16(codePoint)
	if unit < 0xd800 || unit > 0xdfff {
		return codePoint, false
	}
	if unit >= 0xdc00 {
		if index > start {
			previous := units[index-1]
			if previous >= 0xd800 && previous <= 0xdbff {
				combined, _ = sleepUTF16CodePointAt(units, index-1)
				return combined, false
			}
		}
		return codePoint, false
	}
	if index+1 < end {
		next := units[index+1]
		if next >= 0xdc00 && next <= 0xdfff {
			combined, _ = sleepUTF16CodePointAt(units[:end], index)
			return combined, true
		}
	}
	return codePoint, false
}

func sleepStringValuesEqualContext(ctx context.Context, left, right Value) (bool, error) {
	leftUnits, rightUnits := sleepStringUnits(left), sleepStringUnits(right)
	if len(leftUnits) != len(rightUnits) {
		if err := executionContextError(ctx); err != nil {
			return false, err
		}
		return false, nil
	}
	for index := range leftUnits {
		if err := portableJavaStringLoopCheck(ctx, index); err != nil {
			return false, err
		}
		if leftUnits[index] != rightUnits[index] {
			return false, nil
		}
	}
	if err := executionContextError(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func portableJavaStringLoopCheck(ctx context.Context, iteration int) error {
	if iteration%portableJavaStringNativeLoopChunk != 0 {
		return nil
	}
	if err := executionContextError(ctx); err != nil {
		return err
	}
	return consumeInstruction(ctx)
}

func portableJavaStringCancellationCheck(ctx context.Context, iteration int) error {
	if iteration%portableJavaStringNativeLoopChunk != 0 {
		return nil
	}
	return executionContextError(ctx)
}

func portableJavaStringContentEquals(ctx context.Context, invocation ObjectInvocation, target Value) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	other := invocation.Arg(0)
	if other.IsNull() {
		return Null(), true, errors.New("java.lang.NullPointerException")
	}
	if other.Kind() == KindString {
		equal, err := sleepStringValuesEqualContext(ctx, target, other)
		if err != nil {
			return Null(), true, err
		}
		return portableJavaBooleanValue(equal), true, nil
	}
	object, ok := other.Object()
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
	equal, err := sleepStringValuesEqualContext(ctx, target, buffer.snapshotValue())
	if err != nil {
		return Null(), true, err
	}
	return portableJavaBooleanValue(equal), true, nil
}

func portableJavaStringRegionMatches(ctx context.Context, invocation ObjectInvocation, target Value) (Value, bool, error) {
	if len(invocation.Arguments) != 4 && len(invocation.Arguments) != 5 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	ignoreCase := false
	base := 0
	if len(invocation.Arguments) == 5 {
		ignoreCase = invocation.Arg(0).Truth()
		base = 1
	}
	other := invocation.Arg(base + 1)
	if !other.IsNull() && other.Kind() != KindString {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}

	targetOffset := int64(sleepInt32(invocation.Arg(base)))
	otherOffset := int64(sleepInt32(invocation.Arg(base + 2)))
	length := int64(sleepInt32(invocation.Arg(base + 3)))
	if other.IsNull() {
		// The four-argument overload dereferences other before checking ranges.
		// The five-argument overload checks both offsets and the receiver range
		// before it reaches other.length(), regardless of ignoreCase's value.
		if len(invocation.Arguments) == 5 && ignoreCase && (otherOffset < 0 || targetOffset < 0 ||
			targetOffset > int64(sleepStringLength(target))-length) {
			return portableJavaBooleanValue(false), true, nil
		}
		return Null(), true, errors.New("java.lang.NullPointerException")
	}

	targetUnits, otherUnits := sleepStringUnits(target), sleepStringUnits(other)
	if otherOffset < 0 || targetOffset < 0 ||
		targetOffset > int64(len(targetUnits))-length ||
		otherOffset > int64(len(otherUnits))-length {
		return portableJavaBooleanValue(false), true, nil
	}
	if length <= 0 {
		if err := executionContextError(ctx); err != nil {
			return Null(), true, err
		}
		return portableJavaBooleanValue(true), true, nil
	}
	startLeft, startRight, count := int(targetOffset), int(otherOffset), int(length)
	if !ignoreCase {
		equal, err := sleepUTF16RegionEqual(ctx, targetUnits, startLeft, otherUnits, startRight, count)
		if err != nil {
			return Null(), true, err
		}
		return portableJavaBooleanValue(equal), true, nil
	}
	equal, err := sleepUTF16RegionEqualIgnoreCase(ctx, targetUnits, startLeft, otherUnits, startRight, count)
	if err != nil {
		return Null(), true, err
	}
	return portableJavaBooleanValue(equal), true, nil
}

func sleepUTF16RegionEqual(ctx context.Context, left []uint16, leftStart int, right []uint16, rightStart, count int) (bool, error) {
	for index := 0; index < count; index++ {
		if err := portableJavaStringLoopCheck(ctx, index); err != nil {
			return false, err
		}
		if left[leftStart+index] != right[rightStart+index] {
			return false, nil
		}
	}
	if err := executionContextError(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func sleepUTF16RegionEqualIgnoreCase(ctx context.Context, left []uint16, leftStart int, right []uint16, rightStart, count int) (bool, error) {
	leftLatin1, err := sleepUTF16CanEncodeLatin1(ctx, left)
	if err != nil {
		return false, err
	}
	rightLatin1, err := sleepUTF16CanEncodeLatin1(ctx, right)
	if err != nil {
		return false, err
	}
	if leftLatin1 || rightLatin1 {
		comparison, err := sleepUTF16CompareIgnoreCaseUnits(ctx, left, leftStart, count, right, rightStart, count)
		if err != nil {
			return false, err
		}
		return comparison == 0, nil
	}
	comparison, err := sleepUTF16CompareIgnoreCase(ctx, left, leftStart, count, right, rightStart, count)
	if err != nil {
		return false, err
	}
	return comparison == 0, nil
}

func portableJavaStringOffsetByCodePoints(ctx context.Context, invocation ObjectInvocation, target Value) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	units := sleepStringUnits(target)
	index := int64(sleepInt32(invocation.Arg(0)))
	offset := int64(sleepInt32(invocation.Arg(1)))
	if index < 0 || index > int64(len(units)) {
		return Null(), true, errors.New("java.lang.IndexOutOfBoundsException")
	}
	current := int(index)
	if offset >= 0 {
		for step := int64(0); step < offset; step++ {
			if err := portableJavaStringLoopCheck(ctx, int(step)); err != nil {
				return Null(), true, err
			}
			if current >= len(units) {
				return Null(), true, errors.New("java.lang.IndexOutOfBoundsException")
			}
			_, width := sleepUTF16CodePointAt(units, current)
			current += width
		}
	} else {
		for step := int64(0); step > offset; step-- {
			if err := portableJavaStringLoopCheck(ctx, int(-step)); err != nil {
				return Null(), true, err
			}
			if current <= 0 {
				return Null(), true, errors.New("java.lang.IndexOutOfBoundsException")
			}
			_, width := sleepUTF16CodePointBefore(units, current)
			current -= width
		}
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), true, err
	}
	return Int(int32(current)), true, nil
}

func portableJavaStringGetChars(ctx context.Context, invocation ObjectInvocation, target Value) (Value, bool, error) {
	if len(invocation.Arguments) != 4 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	destinationValue := invocation.Arg(2)
	var destination *portableJavaArray
	if !destinationValue.IsNull() {
		object, ok := destinationValue.Object()
		if !ok {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		destination, ok = object.(*portableJavaArray)
		if !ok || destination == nil {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		destination.mu.RLock()
		isCharacters := destination.typeInfo.name == "char" && len(destination.dimensions) == 1
		destination.mu.RUnlock()
		if !isCharacters {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
	}

	units := sleepStringUnits(target)
	sourceBegin := int(sleepInt32(invocation.Arg(0)))
	sourceEnd := int(sleepInt32(invocation.Arg(1)))
	if sourceBegin < 0 || sourceBegin > sourceEnd || sourceEnd > len(units) {
		return Null(), true, fmt.Errorf("java.lang.StringIndexOutOfBoundsException: begin %d, end %d, length %d", sourceBegin, sourceEnd, len(units))
	}
	if destination == nil {
		return Null(), true, errors.New("java.lang.NullPointerException")
	}
	destinationBegin := int(sleepInt32(invocation.Arg(3)))
	count := sourceEnd - sourceBegin
	destination.mu.RLock()
	destinationLength := len(destination.values)
	destination.mu.RUnlock()
	if destinationBegin < 0 || destinationBegin > destinationLength-count {
		return Null(), true, fmt.Errorf("java.lang.StringIndexOutOfBoundsException: offset %d, count %d, length %d", destinationBegin, count, destinationLength)
	}
	characters := make([]Value, count)
	for index, unit := range units[sourceBegin:sourceEnd] {
		if err := portableJavaStringLoopCheck(ctx, index); err != nil {
			return Null(), true, err
		}
		characters[index] = sleepUTF16CharacterValue(unit)
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), true, err
	}
	destination.mu.Lock()
	defer destination.mu.Unlock()
	destinationLength = len(destination.values)
	if destinationBegin < 0 || destinationBegin > destinationLength-count {
		return Null(), true, fmt.Errorf("java.lang.StringIndexOutOfBoundsException: offset %d, count %d, length %d", destinationBegin, count, destinationLength)
	}
	copy(destination.values[destinationBegin:destinationBegin+count], characters)
	return Null(), true, nil
}

func portableJavaStringMatches(ctx context.Context, invocation ObjectInvocation, target Value) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	regex := invocation.Arg(0)
	if regex.IsNull() {
		return Null(), true, errors.New("java.lang.NullPointerException")
	}
	if regex.Kind() != KindString {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	pattern, err := portableJavaStringPattern(ctx, invocation.Runtime, regex, true)
	if err != nil {
		return Null(), true, err
	}
	match, err := pattern.FindStringSubmatchIndexContext(ctx, sleepCanonicalString(target))
	if err != nil {
		return Null(), true, portableJavaStringRegexMatchError(err)
	}
	return portableJavaBooleanValue(match != nil), true, nil
}

func portableJavaStringRegexReplace(ctx context.Context, invocation ObjectInvocation, target Value, first bool) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	regex, replacement := invocation.Arg(0), invocation.Arg(1)
	if !regex.IsNull() && regex.Kind() != KindString || !replacement.IsNull() && replacement.Kind() != KindString {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	if regex.IsNull() {
		return Null(), true, errors.New("java.lang.NullPointerException")
	}
	pattern, err := portableJavaStringPattern(ctx, invocation.Runtime, regex, false)
	if err != nil {
		return Null(), true, err
	}
	if first && replacement.IsNull() {
		return Null(), true, errors.New("java.lang.NullPointerException: replacement")
	}
	if sleepStringLength(regex) == 0 {
		if replacement.IsNull() {
			return Null(), true, errors.New("java.lang.NullPointerException")
		}
		value, err := portableJavaStringReplaceEmptyPattern(ctx, pattern, replacement, target, first)
		return value, true, err
	}

	input := sleepCanonicalString(target)
	limit := -1
	if first {
		limit = 1
	}
	matches, err := pattern.FindAllStringSubmatchUTF16IndexContext(ctx, input, limit)
	if err != nil {
		return Null(), true, portableJavaStringRegexMatchError(err)
	}
	if len(matches) == 0 {
		return target, true, nil
	}
	if replacement.IsNull() {
		return Null(), true, errors.New("java.lang.NullPointerException")
	}

	builder := newPortableJavaStringBuilder(sleepStringLength(target))
	last := 0
	for matchIndex, match := range matches {
		if err := portableJavaStringCancellationCheck(ctx, matchIndex); err != nil {
			return Null(), true, err
		}
		if err := builder.appendRange(target, last, match[0]); err != nil {
			return Null(), true, err
		}
		if err := appendPortableJavaReplacement(ctx, builder, pattern, replacement, target, match); err != nil {
			return Null(), true, err
		}
		last = match[1]
	}
	if err := builder.appendRange(target, last, sleepStringLength(target)); err != nil {
		return Null(), true, err
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), true, err
	}
	return builder.value(), true, nil
}

func portableJavaStringReplaceEmptyPattern(
	ctx context.Context,
	pattern *sleepRegex,
	replacement Value,
	target Value,
	first bool,
) (Value, error) {
	limit := -1
	if first {
		limit = 1
	}
	return portableJavaStringReplaceEmptyPatternLimit(ctx, pattern, replacement, target, limit)
}

func portableJavaStringReplaceEmptyPatternLimit(
	ctx context.Context,
	pattern *sleepRegex,
	replacement Value,
	target Value,
	limit int,
) (Value, error) {
	length := sleepStringLength(target)
	positions := length + 1
	if limit >= 0 && limit < positions {
		positions = limit
	}
	builder := newPortableJavaStringBuilder(length)
	last := 0
	for position := 0; position < positions; position++ {
		if err := executionContextError(ctx); err != nil {
			return Null(), err
		}
		if err := consumeInstruction(ctx); err != nil {
			return Null(), err
		}
		if err := builder.appendRange(target, last, position); err != nil {
			return Null(), err
		}
		if err := appendPortableJavaReplacement(
			ctx, builder, pattern, replacement, target, []int{position, position},
		); err != nil {
			return Null(), err
		}
		last = position
	}
	if err := builder.appendRange(target, last, length); err != nil {
		return Null(), err
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return builder.value(), nil
}

type portableJavaStringSplitSlice struct {
	start int
	end   int
}

func portableJavaStringSplitArray(runtime *Runtime, target Value, pieces []portableJavaStringSplitSlice) (Value, error) {
	for range pieces {
		if err := reserveCollectionEntries(runtime, 1); err != nil {
			return Null(), err
		}
	}
	values := make([]Value, len(pieces))
	for index, piece := range pieces {
		values[index] = sleepStringValueSlice(target, piece.start, piece.end)
	}
	return ArrayValue(NewArray(values...)), nil
}

func portableJavaStringSplit(ctx context.Context, invocation ObjectInvocation, target Value) (Value, bool, error) {
	if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 2 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	regex := invocation.Arg(0)
	if regex.IsNull() {
		return Null(), true, errors.New("java.lang.NullPointerException")
	}
	if regex.Kind() != KindString {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	limit := int32(0)
	if len(invocation.Arguments) == 2 {
		limit = sleepInt32(invocation.Arg(1))
	}
	pattern, err := portableJavaStringPattern(ctx, invocation.Runtime, regex, false)
	if err != nil {
		return Null(), true, err
	}
	if sleepStringLength(regex) == 0 {
		value, err := portableJavaStringSplitEmptyPattern(ctx, invocation.Runtime, target, limit)
		return value, true, err
	}
	input := sleepCanonicalString(target)
	matches, err := pattern.FindAllStringSubmatchUTF16IndexContext(ctx, input, -1)
	if err != nil {
		return Null(), true, portableJavaStringRegexMatchError(err)
	}
	pieces := make([]portableJavaStringSplitSlice, 0, len(matches)+1)
	index := 0
	matchLimited := limit > 0
	for matchIndex, match := range matches {
		if err := portableJavaStringCancellationCheck(ctx, matchIndex); err != nil {
			return Null(), true, err
		}
		start, end := match[0], match[1]
		if !matchLimited || len(pieces) < int(limit)-1 {
			if index == 0 && index == start && start == end {
				continue
			}
			pieces = append(pieces, portableJavaStringSplitSlice{start: index, end: start})
			index = end
		} else if len(pieces) == int(limit)-1 {
			pieces = append(pieces, portableJavaStringSplitSlice{start: index, end: sleepStringLength(target)})
			index = end
		}
	}
	if index == 0 {
		array, err := newRuntimeArray(invocation.Runtime, target)
		if err != nil {
			return Null(), true, err
		}
		return ArrayValue(array), true, nil
	}
	if !matchLimited || len(pieces) < int(limit) {
		pieces = append(pieces, portableJavaStringSplitSlice{start: index, end: sleepStringLength(target)})
	}
	if limit == 0 {
		for len(pieces) != 0 && pieces[len(pieces)-1].start == pieces[len(pieces)-1].end {
			pieces = pieces[:len(pieces)-1]
		}
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), true, err
	}
	value, err := portableJavaStringSplitArray(invocation.Runtime, target, pieces)
	return value, true, err
}

func portableJavaStringSplitEmptyPattern(ctx context.Context, runtime *Runtime, target Value, limit int32) (Value, error) {
	length := sleepStringLength(target)
	pieces := make([]portableJavaStringSplitSlice, 0, length+1)
	index := 0
	matchLimited := limit > 0
	for position := 0; position <= length; position++ {
		if err := executionContextError(ctx); err != nil {
			return Null(), err
		}
		if err := consumeInstruction(ctx); err != nil {
			return Null(), err
		}
		if !matchLimited || len(pieces) < int(limit)-1 {
			if index == 0 && position == 0 {
				continue
			}
			pieces = append(pieces, portableJavaStringSplitSlice{start: index, end: position})
			index = position
		} else if len(pieces) == int(limit)-1 {
			pieces = append(pieces, portableJavaStringSplitSlice{start: index, end: length})
			index = position
		}
	}
	if index == 0 {
		array, err := newRuntimeArray(runtime, target)
		if err != nil {
			return Null(), err
		}
		return ArrayValue(array), nil
	}
	if !matchLimited || len(pieces) < int(limit) {
		pieces = append(pieces, portableJavaStringSplitSlice{start: index, end: length})
	}
	if limit == 0 {
		for len(pieces) != 0 && pieces[len(pieces)-1].start == pieces[len(pieces)-1].end {
			pieces = pieces[:len(pieces)-1]
		}
	}
	if err := executionContextError(ctx); err != nil {
		return Null(), err
	}
	return portableJavaStringSplitArray(runtime, target, pieces)
}

func portableJavaStringPattern(ctx context.Context, runtime *Runtime, regex Value, whole bool) (*sleepRegex, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := executionContextError(ctx); err != nil {
		return nil, err
	}
	var pattern *sleepRegex
	var err error
	if runtime != nil {
		pattern, err = runtime.compileSleepRegexCached(sleepCanonicalString(regex), whole)
	} else {
		pattern, err = compileSleepRegex(sleepCanonicalString(regex), whole)
	}
	if err != nil {
		return nil, fmt.Errorf("java.util.regex.PatternSyntaxException: %v", err)
	}
	if err := executionContextError(ctx); err != nil {
		return nil, err
	}
	return pattern, nil
}

func portableJavaStringRegexMatchError(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("opfor: regular expression match: %w", err)
}

type portableJavaRegexUnitMap map[int]int

func newPortableJavaRegexUnitMap(ctx context.Context, input string) (portableJavaRegexUnitMap, error) {
	text := newSleepRegexText(input)
	result := make(portableJavaRegexUnitMap, len(text.byteOffsets))
	units := 0
	for index, byteOffset := range text.byteOffsets {
		if err := portableJavaStringLoopCheck(ctx, index); err != nil {
			return nil, err
		}
		result[byteOffset] = units
		if index < len(text.runes) {
			units++
			if text.runes[index] > 0xffff {
				units++
			}
		}
	}
	if err := executionContextError(ctx); err != nil {
		return nil, err
	}
	return result, nil
}

func (mapping portableJavaRegexUnitMap) offset(byteOffset int) (int, bool) {
	if byteOffset < 0 {
		return -1, true
	}
	value, ok := mapping[byteOffset]
	return value, ok
}

func (mapping portableJavaRegexUnitMap) match(match []int) ([]int, error) {
	result := make([]int, len(match))
	for index, byteOffset := range match {
		unitOffset, ok := mapping.offset(byteOffset)
		if !ok {
			return nil, errors.New("opfor: regular-expression capture did not align to a UTF-16 boundary")
		}
		result[index] = unitOffset
	}
	return result, nil
}

type portableJavaStringBuilder struct {
	units []uint16
	raw   []bool
}

func newPortableJavaStringBuilder(capacity int) *portableJavaStringBuilder {
	if capacity < 0 {
		capacity = 0
	}
	return &portableJavaStringBuilder{
		units: make([]uint16, 0, capacity),
		raw:   make([]bool, 0, capacity),
	}
}

func (builder *portableJavaStringBuilder) appendRange(value Value, start, end int) error {
	units := sleepStringUnits(value)
	if start < 0 || end < start || end > len(units) {
		return errors.New("opfor: invalid UTF-16 append range")
	}
	return builder.append(units[start:end], sleepStringRawMask(value)[start:end])
}

func (builder *portableJavaStringBuilder) append(units []uint16, raw []bool) error {
	if int64(len(builder.units))+int64(len(units)) > math.MaxInt32 {
		return errors.New("java.lang.OutOfMemoryError: Required length exceeds implementation limit")
	}
	builder.units = append(builder.units, units...)
	if len(raw) == len(units) {
		builder.raw = append(builder.raw, raw...)
	} else {
		builder.raw = append(builder.raw, make([]bool, len(units))...)
	}
	return nil
}

func (builder *portableJavaStringBuilder) value() Value {
	return sleepStringValueFromUnits(builder.units, builder.raw)
}

func appendPortableJavaReplacement(
	ctx context.Context,
	builder *portableJavaStringBuilder,
	pattern *sleepRegex,
	replacement Value,
	input Value,
	match []int,
) error {
	replacementUnits := sleepStringUnits(replacement)
	replacementRaw := sleepStringRawMask(replacement)
	for cursor := 0; cursor < len(replacementUnits); {
		if err := portableJavaStringLoopCheck(ctx, cursor); err != nil {
			return err
		}
		switch replacementUnits[cursor] {
		case '\\':
			cursor++
			if cursor == len(replacementUnits) {
				return errors.New("java.lang.IllegalArgumentException: character to be escaped is missing")
			}
			if err := builder.append(replacementUnits[cursor:cursor+1], replacementRaw[cursor:cursor+1]); err != nil {
				return err
			}
			cursor++
		case '$':
			cursor++
			if cursor == len(replacementUnits) {
				return errors.New("java.lang.IllegalArgumentException: Illegal group reference: group index is missing")
			}
			group := -1
			if replacementUnits[cursor] == '{' {
				cursor++
				nameStart := cursor
				for cursor < len(replacementUnits) && portableJavaReplacementNameCharacter(replacementUnits[cursor]) {
					if err := portableJavaStringLoopCheck(ctx, cursor); err != nil {
						return err
					}
					cursor++
				}
				if cursor == nameStart {
					return errors.New("java.lang.IllegalArgumentException: named capturing group has 0 length name")
				}
				if cursor == len(replacementUnits) || replacementUnits[cursor] != '}' {
					return errors.New("java.lang.IllegalArgumentException: named capturing group is missing trailing '}'")
				}
				nameUnits := replacementUnits[nameStart:cursor]
				if nameUnits[0] >= '0' && nameUnits[0] <= '9' {
					return fmt.Errorf("java.lang.IllegalArgumentException: capturing group name {%s} starts with digit character", portableJavaASCIIUnits(nameUnits))
				}
				name := portableJavaASCIIUnits(nameUnits)
				group = pattern.SubexpIndex(name)
				if group < 0 {
					return fmt.Errorf("java.lang.IllegalArgumentException: No group with name {%s}", name)
				}
				cursor++
			} else {
				if replacementUnits[cursor] < '0' || replacementUnits[cursor] > '9' {
					return errors.New("java.lang.IllegalArgumentException: Illegal group reference")
				}
				group = int(replacementUnits[cursor] - '0')
				cursor++
				for cursor < len(replacementUnits) && replacementUnits[cursor] >= '0' && replacementUnits[cursor] <= '9' {
					if err := portableJavaStringLoopCheck(ctx, cursor); err != nil {
						return err
					}
					digit := int(replacementUnits[cursor] - '0')
					if group > (pattern.NumSubexp()-digit)/10 {
						break
					}
					candidate := group*10 + digit
					if candidate > pattern.NumSubexp() {
						break
					}
					group = candidate
					cursor++
				}
				if group > pattern.NumSubexp() {
					return fmt.Errorf("java.lang.IndexOutOfBoundsException: No group %d", group)
				}
			}
			groupOffset := group * 2
			if groupOffset+1 < len(match) && match[groupOffset] >= 0 {
				if err := builder.appendRange(input, match[groupOffset], match[groupOffset+1]); err != nil {
					return err
				}
			}
		default:
			if err := builder.append(replacementUnits[cursor:cursor+1], replacementRaw[cursor:cursor+1]); err != nil {
				return err
			}
			cursor++
		}
	}
	return nil
}

func portableJavaReplacementNameCharacter(unit uint16) bool {
	return unit >= 'a' && unit <= 'z' || unit >= 'A' && unit <= 'Z' || unit >= '0' && unit <= '9'
}

func portableJavaASCIIUnits(units []uint16) string {
	result := make([]byte, len(units))
	for index, unit := range units {
		result[index] = byte(unit)
	}
	return string(result)
}
