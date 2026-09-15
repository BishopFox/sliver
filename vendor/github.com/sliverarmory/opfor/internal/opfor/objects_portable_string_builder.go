package opfor

import (
	"errors"
	"fmt"
)

const (
	portableStringBuilderDefaultCapacity = 16
	portableStringBuilderMaximumLength   = int(^uint32(0) >> 1)
)

// portableJavaStringBuilderConstruct models StringBuilder and StringBuffer as
// mutable Java UTF-16 sequences. Capacity is logical rather than a Go slice
// allocation so a script cannot reserve gigabytes merely by constructing an
// otherwise empty builder.
func portableJavaStringBuilderConstruct(invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op != ObjectConstruct {
		return Null(), false, nil
	}
	class := "java.lang." + portableJavaClassName(invocation.Class)
	if len(invocation.Arguments) > 1 {
		return portableStringBuilderNoConstructor(invocation, class), true, nil
	}

	buffer := &portableJavaStringBuffer{class: class, capacity: portableStringBuilderDefaultCapacity}
	if len(invocation.Arguments) == 0 {
		return ObjectValue(buffer), true, nil
	}

	argument := invocation.Arg(0)
	if capacity, ok := portableStringBuilderCapacityArgument(argument); ok {
		if capacity < 0 {
			return Null(), true, fmt.Errorf("java.lang.NegativeArraySizeException: %d", capacity)
		}
		buffer.capacity = capacity
		return ObjectValue(buffer), true, nil
	}
	if argument.IsNull() {
		return Null(), true, errors.New("java.lang.NullPointerException")
	}
	initial, array, ok := portableStringBuilderSequence(argument, false)
	if !ok || array {
		return portableStringBuilderNoConstructor(invocation, class), true, nil
	}
	buffer.units = sleepStringUnits(initial)
	buffer.raw = sleepStringRawMask(initial)
	buffer.capacity = portableStringBuilderDefaultCapacity + len(buffer.units)
	return ObjectValue(buffer), true, nil
}

func portableStringBuilderCapacityArgument(value Value) (int, bool) {
	if value.Kind() == KindInt {
		return int(value.Int32()), true
	}
	object, ok := value.Object()
	if !ok {
		return 0, false
	}
	primitive, ok := object.(*portableJavaPrimitive)
	if !ok || primitive == nil || primitive.className() != "java.lang.Integer" {
		return 0, false
	}
	return int(primitive.sleepValue().Int32()), true
}

func (b *portableJavaStringBuffer) invoke(invocation ObjectInvocation) (Value, bool, error) {
	if b == nil {
		return Null(), false, nil
	}
	if invocation.Op == ObjectTypeCheck {
		return Bool(portableJavaAssignable(b.class, invocation.Class)), true, nil
	}
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}

	switch invocation.Message {
	case "append":
		return b.append(invocation)
	case "insert":
		return b.insert(invocation)
	case "delete":
		return b.delete(invocation)
	case "deleteCharAt":
		return b.deleteCharAt(invocation)
	case "replace":
		return b.replace(invocation)
	case "reverse":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, b.class), true, nil
		}
		b.reverse()
		return invocation.Target, true, nil
	case "length":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, b.class), true, nil
		}
		b.mu.RLock()
		length := len(b.units)
		b.mu.RUnlock()
		return Int(int32(length)), true, nil
	case "isEmpty":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, b.class), true, nil
		}
		b.mu.RLock()
		empty := len(b.units) == 0
		b.mu.RUnlock()
		return portableJavaBooleanValue(empty), true, nil
	case "capacity":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, b.class), true, nil
		}
		b.mu.RLock()
		capacity := b.capacity
		b.mu.RUnlock()
		return Int(int32(capacity)), true, nil
	case "ensureCapacity":
		if len(invocation.Arguments) != 1 {
			return portableNoMatchingMethod(invocation, b.class), true, nil
		}
		minimum := int(sleepInt32(invocation.Arg(0)))
		b.mu.Lock()
		b.ensureCapacityLocked(minimum)
		b.mu.Unlock()
		return Null(), true, nil
	case "trimToSize":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, b.class), true, nil
		}
		b.mu.Lock()
		b.capacity = len(b.units)
		b.mu.Unlock()
		return Null(), true, nil
	case "setLength":
		return b.setLength(invocation)
	case "charAt":
		return b.charAt(invocation)
	case "setCharAt":
		return b.setCharAt(invocation)
	case "substring", "subSequence":
		return b.substring(invocation)
	case "indexOf", "lastIndexOf":
		return b.indexOf(invocation)
	case "compareTo":
		return b.compareTo(invocation)
	case "toString":
		if len(invocation.Arguments) != 0 {
			return portableNoMatchingMethod(invocation, b.class), true, nil
		}
		return b.snapshotValue(), true, nil
	}
	return Null(), false, nil
}

func (b *portableJavaStringBuffer) append(invocation ObjectInvocation) (Value, bool, error) {
	var value Value
	switch len(invocation.Arguments) {
	case 1:
		value = portableStringBuilderAppendValue(invocation.Arg(0))
	case 3:
		sequence, array, ok := portableStringBuilderSequence(invocation.Arg(0), true)
		if !ok {
			return portableNoMatchingMethod(invocation, b.class), true, nil
		}
		start := int(sleepInt32(invocation.Arg(1)))
		end := int(sleepInt32(invocation.Arg(2)))
		if array {
			end += start
		}
		length := sleepStringLength(sequence)
		if start < 0 || end < start || end > length {
			return Null(), true, portableStringBuilderRangeError(start, end, length)
		}
		value = sleepStringValueSlice(sequence, start, end)
	default:
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	if err := b.insertValue(b.length(), value); err != nil {
		return Null(), true, err
	}
	return invocation.Target, true, nil
}

func (b *portableJavaStringBuffer) insert(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 && len(invocation.Arguments) != 4 {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	offset := int(sleepInt32(invocation.Arg(0)))
	value := Null()
	if len(invocation.Arguments) == 2 {
		value = portableStringBuilderAppendValue(invocation.Arg(1))
	} else {
		sequence, array, ok := portableStringBuilderSequence(invocation.Arg(1), true)
		if !ok {
			return portableNoMatchingMethod(invocation, b.class), true, nil
		}
		start := int(sleepInt32(invocation.Arg(2)))
		end := int(sleepInt32(invocation.Arg(3)))
		if array {
			end += start
		}
		length := sleepStringLength(sequence)
		if start < 0 || end < start || end > length {
			return Null(), true, portableStringBuilderRangeError(start, end, length)
		}
		value = sleepStringValueSlice(sequence, start, end)
	}
	if err := b.insertValue(offset, value); err != nil {
		return Null(), true, err
	}
	return invocation.Target, true, nil
}

func (b *portableJavaStringBuffer) insertValue(offset int, value Value) error {
	units := sleepStringUnits(value)
	raw := sleepStringRawMask(value)
	b.mu.Lock()
	defer b.mu.Unlock()
	if offset < 0 || offset > len(b.units) {
		return portableStringBuilderIndexError(offset, len(b.units))
	}
	if len(units) > portableStringBuilderMaximumLength-len(b.units) {
		return errors.New("java.lang.OutOfMemoryError: Required length exceeds implementation limit")
	}
	newLength := len(b.units) + len(units)
	b.ensureCapacityLocked(newLength)
	newUnits := make([]uint16, 0, newLength)
	newUnits = append(newUnits, b.units[:offset]...)
	newUnits = append(newUnits, units...)
	newUnits = append(newUnits, b.units[offset:]...)
	newRaw := make([]bool, 0, newLength)
	newRaw = append(newRaw, b.raw[:offset]...)
	newRaw = append(newRaw, raw...)
	newRaw = append(newRaw, b.raw[offset:]...)
	b.units, b.raw = newUnits, newRaw
	return nil
}

func (b *portableJavaStringBuffer) delete(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	start := int(sleepInt32(invocation.Arg(0)))
	end := int(sleepInt32(invocation.Arg(1)))
	b.mu.Lock()
	defer b.mu.Unlock()
	length := len(b.units)
	if end > length {
		end = length
	}
	if start < 0 || start > length || start > end {
		return Null(), true, portableStringBuilderRangeError(start, end, length)
	}
	b.units = append(b.units[:start], b.units[end:]...)
	b.raw = append(b.raw[:start], b.raw[end:]...)
	return invocation.Target, true, nil
}

func (b *portableJavaStringBuffer) deleteCharAt(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	index := int(sleepInt32(invocation.Arg(0)))
	b.mu.Lock()
	defer b.mu.Unlock()
	if index < 0 || index >= len(b.units) {
		return Null(), true, portableStringBuilderIndexError(index, len(b.units))
	}
	b.units = append(b.units[:index], b.units[index+1:]...)
	b.raw = append(b.raw[:index], b.raw[index+1:]...)
	return invocation.Target, true, nil
}

func (b *portableJavaStringBuffer) replace(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 3 {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	if invocation.Arg(2).IsNull() {
		return Null(), true, errors.New("java.lang.NullPointerException")
	}
	start := int(sleepInt32(invocation.Arg(0)))
	end := int(sleepInt32(invocation.Arg(1)))
	// ObjectUtilities treats a non-string Sleep scalar as a fallback match for
	// a Java String parameter and marshals it through Scalar.toString().
	replacement := sleepStringCoercion(invocation.Arg(2))
	replacementUnits := sleepStringUnits(replacement)
	replacementRaw := sleepStringRawMask(replacement)
	b.mu.Lock()
	defer b.mu.Unlock()
	length := len(b.units)
	if end > length {
		end = length
	}
	if start < 0 || start > length || start > end {
		return Null(), true, portableStringBuilderRangeError(start, end, length)
	}
	retained := length - (end - start)
	if len(replacementUnits) > portableStringBuilderMaximumLength-retained {
		return Null(), true, errors.New("java.lang.OutOfMemoryError: Required length exceeds implementation limit")
	}
	newLength := retained + len(replacementUnits)
	b.ensureCapacityLocked(newLength)
	units := make([]uint16, 0, newLength)
	units = append(units, b.units[:start]...)
	units = append(units, replacementUnits...)
	units = append(units, b.units[end:]...)
	raw := make([]bool, 0, newLength)
	raw = append(raw, b.raw[:start]...)
	raw = append(raw, replacementRaw...)
	raw = append(raw, b.raw[end:]...)
	b.units, b.raw = units, raw
	return invocation.Target, true, nil
}

func (b *portableJavaStringBuffer) reverse() {
	b.mu.Lock()
	defer b.mu.Unlock()
	for left, right := 0, len(b.units)-1; left < right; left, right = left+1, right-1 {
		b.units[left], b.units[right] = b.units[right], b.units[left]
		b.raw[left], b.raw[right] = b.raw[right], b.raw[left]
	}
	// AbstractStringBuilder.reverse preserves every valid surrogate pair. The
	// second pass also mirrors Java's observable behavior of turning a reversed
	// low/high pair into a valid high/low pair.
	for index := 0; index+1 < len(b.units); index++ {
		if isLowUTF16Surrogate(b.units[index]) && isHighUTF16Surrogate(b.units[index+1]) {
			b.units[index], b.units[index+1] = b.units[index+1], b.units[index]
			b.raw[index], b.raw[index+1] = b.raw[index+1], b.raw[index]
			index++
		}
	}
}

func (b *portableJavaStringBuffer) setLength(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	newLength := int(sleepInt32(invocation.Arg(0)))
	if newLength < 0 {
		return Null(), true, portableStringBuilderIndexError(newLength, b.length())
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	if newLength <= len(b.units) {
		b.units = b.units[:newLength]
		b.raw = b.raw[:newLength]
		return Null(), true, nil
	}
	b.ensureCapacityLocked(newLength)
	b.units = append(b.units, make([]uint16, newLength-len(b.units))...)
	b.raw = append(b.raw, make([]bool, newLength-len(b.raw))...)
	return Null(), true, nil
}

func (b *portableJavaStringBuffer) charAt(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	index := int(sleepInt32(invocation.Arg(0)))
	b.mu.RLock()
	defer b.mu.RUnlock()
	if index < 0 || index >= len(b.units) {
		return Null(), true, portableStringBuilderIndexError(index, len(b.units))
	}
	return sleepUTF16CharacterValue(b.units[index]), true, nil
}

func (b *portableJavaStringBuffer) setCharAt(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	character, ok, err := portableStringBuilderCharacter(invocation.Arg(1))
	if err != nil {
		return Null(), true, err
	}
	if !ok {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	index := int(sleepInt32(invocation.Arg(0)))
	b.mu.Lock()
	defer b.mu.Unlock()
	if index < 0 || index >= len(b.units) {
		return Null(), true, portableStringBuilderIndexError(index, len(b.units))
	}
	b.units[index] = character
	b.raw[index] = false
	return Null(), true, nil
}

func (b *portableJavaStringBuffer) substring(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 2 {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	start := int(sleepInt32(invocation.Arg(0)))
	b.mu.RLock()
	defer b.mu.RUnlock()
	end := len(b.units)
	if len(invocation.Arguments) == 2 {
		end = int(sleepInt32(invocation.Arg(1)))
	}
	if start < 0 || end < start || end > len(b.units) {
		return Null(), true, portableStringBuilderRangeError(start, end, len(b.units))
	}
	return sleepStringValueFromUnits(b.units[start:end], b.raw[start:end]), true, nil
}

func (b *portableJavaStringBuffer) indexOf(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) < 1 || len(invocation.Arguments) > 2 {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	if invocation.Arg(0).IsNull() {
		return Null(), true, errors.New("java.lang.NullPointerException")
	}
	// Unlike String.indexOf, AbstractStringBuilder exposes only the String
	// overload, so Sleep's reflection bridge accepts other non-null scalars as
	// fallback String matches instead of selecting a code-point overload.
	needle := sleepStringUnits(sleepStringCoercion(invocation.Arg(0)))
	b.mu.RLock()
	defer b.mu.RUnlock()
	from := 0
	reverse := invocation.Message == "lastIndexOf"
	if reverse {
		from = len(b.units)
	}
	if len(invocation.Arguments) == 2 {
		from = int(sleepInt32(invocation.Arg(1)))
	}
	return Int(int32(sleepUTF16UnitIndex(b.units, needle, from, reverse))), true, nil
}

func (b *portableJavaStringBuffer) compareTo(invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	if invocation.Arg(0).IsNull() {
		return Null(), true, errors.New("java.lang.NullPointerException")
	}
	object, ok := invocation.Arg(0).Object()
	if !ok {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	other, ok := object.(*portableJavaStringBuffer)
	if !ok || other == nil || other.class != b.class {
		return portableNoMatchingMethod(invocation, b.class), true, nil
	}
	return Int(int32(sleepStringCompareValues(b.snapshotValue(), other.snapshotValue()))), true, nil
}

func (b *portableJavaStringBuffer) snapshotValue() Value {
	if b == nil {
		return String("null")
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return sleepStringValueFromUnits(b.units, b.raw)
}

func (b *portableJavaStringBuffer) length() int {
	if b == nil {
		return 0
	}
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.units)
}

func (b *portableJavaStringBuffer) ensureCapacityLocked(minimum int) {
	if minimum <= b.capacity || minimum <= 0 {
		return
	}
	grown := int64(b.capacity)*2 + 2
	if grown > int64(portableStringBuilderMaximumLength) {
		grown = int64(portableStringBuilderMaximumLength)
	}
	if int64(minimum) > grown {
		grown = int64(minimum)
	}
	b.capacity = int(grown)
}

// portableStringBuilderSequence returns a stable scalar snapshot and reports
// whether the source was a Java char[], whose three/four-argument overloads
// interpret their final integer as a length rather than an end index.
func portableStringBuilderSequence(value Value, nullLiteral bool) (Value, bool, bool) {
	if value.IsNull() {
		if nullLiteral {
			return String("null"), false, true
		}
		return Null(), false, false
	}
	if value.Kind() == KindString {
		return value, false, true
	}
	object, ok := value.Object()
	if !ok {
		return Null(), false, false
	}
	if buffer, ok := object.(*portableJavaStringBuffer); ok && buffer != nil {
		return buffer.snapshotValue(), false, true
	}
	if array, ok := object.(*portableJavaArray); ok && array != nil {
		typeInfo, dimensions, values := array.snapshot()
		if typeInfo.name == "char" && len(dimensions) == 1 {
			converted, _ := portableJavaArraySnapshotToSleepValue(nil, typeInfo, dimensions, values)
			return converted, true, true
		}
	}
	return Null(), false, false
}

func portableStringBuilderAppendValue(value Value) Value {
	if sequence, _, ok := portableStringBuilderSequence(value, true); ok {
		return sequence
	}
	return String(portableJavaValueString(value))
}

func portableStringBuilderCharacter(value Value) (uint16, bool, error) {
	if value.IsNull() {
		// ObjectUtilities accepts the empty scalar for any signature, then its
		// primitive-char marshaller calls charAt(0) on the empty spelling.
		return 0, false, errors.New("java.lang.StringIndexOutOfBoundsException: String index out of range: 0")
	}
	if object, ok := value.Object(); ok {
		primitive, ok := object.(*portableJavaPrimitive)
		if !ok || primitive == nil || primitive.className() != "java.lang.Character" {
			return 0, false, nil
		}
		value = primitive.sleepValue()
	}
	units := sleepStringUnits(value)
	if len(units) == 0 || value.Kind() == KindString && len(units) != 1 {
		return 0, false, nil
	}
	return units[0], true, nil
}

func portableStringBuilderIndexError(index, length int) error {
	return fmt.Errorf("java.lang.StringIndexOutOfBoundsException: index %d, length %d", index, length)
}

func portableStringBuilderRangeError(start, end, length int) error {
	return fmt.Errorf("java.lang.StringIndexOutOfBoundsException: start %d, end %d, length %d", start, end, length)
}

func portableStringBuilderNoConstructor(invocation ObjectInvocation, class string) Value {
	return portableObjectWarning(invocation, fmt.Sprintf("no constructor matching %s(%s)", class, portableObjectArgumentList(invocation)))
}

func isHighUTF16Surrogate(unit uint16) bool { return unit >= 0xd800 && unit <= 0xdbff }

func isLowUTF16Surrogate(unit uint16) bool { return unit >= 0xdc00 && unit <= 0xdfff }
