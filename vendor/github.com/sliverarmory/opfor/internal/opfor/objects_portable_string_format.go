package opfor

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	goruntime "runtime"
	"strconv"
	"strings"
)

// This file models the non-stream String APIs added after the first portable
// String tranche. String and Formatter behavior is pinned to OpenJDK 17u
// commit 352633b5cef98ef3de7e562751222c38d76bb319. Method selection follows
// Sleep 2.1's reflection bridge: a Java varargs parameter is supplied as one
// Sleep array scalar or one typed portable Java array, rather than by appending
// arbitrary method arguments.

func portableJavaStringStaticContext(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if invocation.Op != ObjectInvoke {
		return Null(), false, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	switch invocation.Message {
	case "valueOf", "copyValueOf":
		return portableJavaStringStaticConversion(ctx, invocation)
	case "join":
		return portableJavaStringJoin(ctx, invocation)
	case "format":
		return portableJavaStringFormatStatic(ctx, invocation)
	default:
		return Null(), false, nil
	}
}

func portableJavaStringStaticConversion(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 1 && len(invocation.Arguments) != 3 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	argument := invocation.Arg(0)
	if len(invocation.Arguments) == 3 {
		if argument.IsNull() {
			return Null(), true, errors.New(`java.lang.NullPointerException: Cannot read the array length because "value" is null`)
		}
		characters, ok, err := portableJavaStringCharacterArray(invocation.Runtime, argument)
		if err != nil {
			return Null(), true, err
		}
		if !ok {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		offset := int64(sleepInt32(invocation.Arg(1)))
		count := int64(sleepInt32(invocation.Arg(2)))
		length := int64(sleepStringLength(characters))
		if offset < 0 || count < 0 || offset > length-count {
			return Null(), true, fmt.Errorf(
				"java.lang.StringIndexOutOfBoundsException: Range [%d, %d + %d) out of bounds for length %d",
				offset, offset, count, length,
			)
		}
		result, err := portableJavaStringCopyRange(ctx, characters, int(offset), int(offset+count), false)
		return result, true, err
	}

	if characters, ok, err := portableJavaStringCharacterArray(invocation.Runtime, argument); err != nil {
		return Null(), true, err
	} else if ok {
		result, err := portableJavaStringCopyRange(ctx, characters, 0, sleepStringLength(characters), false)
		return result, true, err
	}
	if invocation.Message == "copyValueOf" {
		if argument.IsNull() {
			return Null(), true, errors.New(`java.lang.NullPointerException: Cannot read the array length because "value" is null`)
		}
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	if argument.IsNull() {
		return String("null"), true, nil
	}
	switch argument.Kind() {
	case KindString:
		// String.valueOf(Object) returns the same immutable String. Retaining the
		// scalar also retains OPFOR's optional raw-byte provenance.
		return argument, true, nil
	case KindInt, KindLong, KindDouble, KindObject:
		return String(argument.String()), true, nil
	default:
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
}

func portableJavaStringCharacterArray(runtime *Runtime, value Value) (Value, bool, error) {
	object, ok := value.Object()
	if !ok {
		return Null(), false, nil
	}
	array, ok := object.(*portableJavaArray)
	if !ok || array == nil {
		return Null(), false, nil
	}
	typeInfo, dimensions, values := array.snapshot()
	if typeInfo.name != "char" || len(dimensions) != 1 {
		return Null(), false, nil
	}
	converted, err := portableJavaArraySnapshotToSleepValue(runtime, typeInfo, dimensions, values)
	return converted, true, err
}

func portableJavaStringHashCode(ctx context.Context, value Value) (int32, error) {
	var hash int32
	for index, unit := range sleepStringUnits(value) {
		if err := portableJavaStringLoopCheck(ctx, index); err != nil {
			return 0, err
		}
		hash = 31*hash + int32(unit)
	}
	if err := executionContextError(ctx); err != nil {
		return 0, err
	}
	return hash, nil
}

func portableJavaStringCopyRange(ctx context.Context, value Value, start, end int, preserveRaw bool) (Value, error) {
	units := sleepStringUnits(value)
	raw := sleepStringRawMask(value)
	if !preserveRaw {
		raw = nil
	}
	builder := newPortableJavaStringBuilder(end - start)
	if err := portableJavaStringAppendRangeChecked(ctx, builder, units, raw, start, end); err != nil {
		return Null(), err
	}
	return builder.value(), nil
}

func portableJavaStringAppendValueChecked(ctx context.Context, builder *portableJavaStringBuilder, value Value) error {
	return portableJavaStringAppendRangeChecked(
		ctx, builder, sleepStringUnits(value), sleepStringRawMask(value), 0, sleepStringLength(value),
	)
}

func portableJavaStringAppendRangeChecked(
	ctx context.Context,
	builder *portableJavaStringBuilder,
	units []uint16,
	raw []bool,
	start, end int,
) error {
	if start < 0 || end < start || end > len(units) {
		return errors.New("opfor: invalid UTF-16 append range")
	}
	for cursor := start; cursor < end; {
		if err := portableJavaStringLoopCheck(ctx, cursor-start); err != nil {
			return err
		}
		next := min(cursor+portableJavaStringNativeLoopChunk, end)
		var rawChunk []bool
		if len(raw) == len(units) {
			rawChunk = raw[cursor:next]
		}
		if err := builder.append(units[cursor:next], rawChunk); err != nil {
			return err
		}
		cursor = next
	}
	return executionContextError(ctx)
}

func portableJavaStringJoin(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	delimiterValue := invocation.Arg(0)
	if delimiterValue.IsNull() {
		return Null(), true, errors.New(`java.lang.NullPointerException: Cannot invoke "java.lang.CharSequence.toString()" because "delimiter" is null`)
	}
	delimiter, ok := portableJavaStringCharSequence(delimiterValue)
	if !ok {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}

	elements, iterable, matched, err := portableJavaStringJoinElements(invocation.Arg(1))
	if err != nil {
		return Null(), true, err
	}
	if !matched {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	builder := newPortableJavaStringBuilder(0)
	for index, element := range elements {
		if err := portableJavaStringLoopCheck(ctx, index); err != nil {
			return Null(), true, err
		}
		if index != 0 {
			if err := portableJavaStringAppendValueChecked(ctx, builder, delimiter); err != nil {
				return Null(), true, err
			}
		}
		if element.IsNull() {
			element = String("null")
		} else if sequence, sequenceOK := portableJavaStringCharSequence(element); sequenceOK {
			element = sequence
		} else if iterable {
			class, _ := portableObjectClass(element)
			if class == "" {
				class = "java.lang.Object"
			}
			return Null(), true, fmt.Errorf(
				"java.lang.ClassCastException: class %s cannot be cast to class java.lang.CharSequence", class,
			)
		} else {
			return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
		}
		if err := portableJavaStringAppendValueChecked(ctx, builder, element); err != nil {
			return Null(), true, err
		}
	}
	return builder.value(), true, nil
}

func portableJavaStringCharSequence(value Value) (Value, bool) {
	if value.Kind() == KindString {
		return value, true
	}
	object, ok := value.Object()
	if !ok {
		return Null(), false
	}
	buffer, ok := object.(*portableJavaStringBuffer)
	if !ok || buffer == nil {
		return Null(), false
	}
	class := portableJavaClassName(buffer.class)
	if class != "StringBuilder" && class != "StringBuffer" {
		return Null(), false
	}
	return buffer.snapshotValue(), true
}

func portableJavaStringJoinElements(value Value) (values []Value, iterable, matched bool, err error) {
	if value.IsNull() {
		return nil, false, true, errors.New("java.lang.NullPointerException")
	}
	if array, ok := value.Array(); ok && array != nil {
		values = array.Values()
		if len(values) == 0 {
			return nil, false, false, nil
		}
		return values, false, true, nil
	}
	object, ok := value.Object()
	if !ok {
		return nil, false, false, nil
	}
	if array, ok := object.(*portableJavaArray); ok && array != nil {
		typeInfo, dimensions, elements := array.snapshot()
		if len(dimensions) != 1 || typeInfo.primitive ||
			!portableJavaAssignable(typeInfo.name, "java.lang.CharSequence") {
			return nil, false, false, nil
		}
		return elements, false, true, nil
	}
	if collection, ok := object.(*portableJavaCollection); ok && collection != nil {
		values, err := collection.snapshotChecked()
		return values, true, true, err
	}
	return nil, false, false, nil
}

type portableJavaFormatArgument struct {
	value Value
	class string
}

func portableJavaStringFormatStatic(ctx context.Context, invocation ObjectInvocation) (Value, bool, error) {
	if len(invocation.Arguments) != 2 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	format := invocation.Arg(0)
	if format.IsNull() {
		return Null(), true, errors.New(`java.lang.NullPointerException: Cannot invoke "String.length()" because "s" is null`)
	}
	if format.Kind() != KindString {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	arguments, matched := portableJavaStringFormatArguments(invocation.Arg(1))
	if !matched {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	value, err := portableJavaStringApplyFormat(ctx, format, arguments)
	return value, true, err
}

func portableJavaStringFormatted(ctx context.Context, invocation ObjectInvocation, target Value) (Value, bool, error) {
	if len(invocation.Arguments) != 1 {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	arguments, matched := portableJavaStringFormatArguments(invocation.Arg(0))
	if !matched {
		return portableNoMatchingMethod(invocation, "java.lang.String"), true, nil
	}
	value, err := portableJavaStringApplyFormat(ctx, target, arguments)
	return value, true, err
}

func portableJavaStringFormatArguments(value Value) ([]portableJavaFormatArgument, bool) {
	if value.IsNull() {
		return []portableJavaFormatArgument{{value: Null()}}, true
	}
	if array, ok := value.Array(); ok && array != nil {
		values := array.Values()
		if len(values) == 0 {
			return nil, false
		}
		inferred := inferPortableArrayClass(value)
		if portableArrayType(inferred).primitive || inferred == "java.lang.Object" {
			return nil, false
		}
		arguments := make([]portableJavaFormatArgument, len(values))
		for index, element := range values {
			if element.IsNull() {
				arguments[index] = portableJavaFormatArgument{value: Null()}
				continue
			}
			if inferred == "java.lang.String" {
				if element.Kind() == KindString {
					arguments[index] = portableJavaFormatArgument{value: element, class: inferred}
				} else {
					arguments[index] = portableJavaFormatArgument{value: String(element.String()), class: inferred}
				}
				continue
			}
			actual, _ := portableObjectClass(element)
			if actual == "" || !portableJavaAssignable(actual, inferred) {
				return nil, false
			}
			arguments[index] = portableJavaFormatArgument{value: element, class: actual}
		}
		return arguments, true
	}
	object, ok := value.Object()
	if !ok {
		return nil, false
	}
	array, ok := object.(*portableJavaArray)
	if !ok || array == nil {
		return nil, false
	}
	typeInfo, dimensions, values := array.snapshot()
	if len(dimensions) != 1 || typeInfo.primitive {
		return nil, false
	}
	arguments := make([]portableJavaFormatArgument, len(values))
	for index, element := range values {
		class := ""
		if !element.IsNull() {
			class, _ = portableObjectClass(element)
		}
		arguments[index] = portableJavaFormatArgument{value: element, class: class}
	}
	return arguments, true
}

type portableJavaFormatSpecifier struct {
	original      string
	argument      int
	explicit      bool
	reusePrevious bool
	flags         string
	width         int
	hasWidth      bool
	precision     int
	hasPrecision  bool
	dateTime      bool
	upperDateTime bool
	conversion    byte
}

func portableJavaStringApplyFormat(ctx context.Context, format Value, arguments []portableJavaFormatArgument) (Value, error) {
	units := sleepStringUnits(format)
	raw := sleepStringRawMask(format)
	builder := newPortableJavaStringBuilder(len(units))
	ordinaryArgument, previousArgument := 0, -1
	literalStart := 0
	for cursor := 0; cursor < len(units); {
		if err := portableJavaStringLoopCheck(ctx, cursor); err != nil {
			return Null(), err
		}
		if units[cursor] != '%' {
			cursor++
			continue
		}
		if err := portableJavaStringAppendRangeChecked(ctx, builder, units, raw, literalStart, cursor); err != nil {
			return Null(), err
		}
		specifier, next, err := portableJavaStringParseFormatSpecifier(units, cursor)
		if err != nil {
			return Null(), err
		}
		cursor, literalStart = next, next

		if specifier.conversion == '%' || specifier.conversion == 'n' {
			if err := portableJavaStringRenderNoArgumentFormat(ctx, builder, specifier); err != nil {
				return Null(), err
			}
			continue
		}
		argumentIndex := ordinaryArgument
		switch {
		case specifier.reusePrevious:
			argumentIndex = previousArgument
		case specifier.explicit:
			argumentIndex = specifier.argument
		default:
			ordinaryArgument++
		}
		if argumentIndex < 0 || argumentIndex >= len(arguments) {
			return Null(), fmt.Errorf("java.util.MissingFormatArgumentException: Format specifier '%s'", specifier.original)
		}
		previousArgument = argumentIndex
		if err := portableJavaStringRenderArgument(ctx, builder, specifier, arguments[argumentIndex]); err != nil {
			return Null(), err
		}
	}
	if err := portableJavaStringAppendRangeChecked(ctx, builder, units, raw, literalStart, len(units)); err != nil {
		return Null(), err
	}
	return builder.value(), nil
}

func portableJavaStringParseFormatSpecifier(units []uint16, start int) (portableJavaFormatSpecifier, int, error) {
	specifier := portableJavaFormatSpecifier{width: -1, precision: -1}
	cursor := start + 1
	if cursor >= len(units) {
		return specifier, cursor, errors.New("java.util.UnknownFormatConversionException: Conversion = '%'")
	}

	digitsStart := cursor
	for cursor < len(units) && units[cursor] >= '0' && units[cursor] <= '9' {
		cursor++
	}
	if cursor > digitsStart && cursor < len(units) && units[cursor] == '$' {
		parsed, err := portableJavaStringParseFormatNumber(units[digitsStart:cursor])
		if err != nil || parsed <= 0 {
			return specifier, cursor, fmt.Errorf("java.util.IllegalFormatArgumentIndexException: Illegal format argument index = %d", parsed)
		}
		specifier.argument = parsed - 1
		specifier.explicit = true
		cursor++
	} else {
		cursor = digitsStart
	}

	seenFlags := make(map[uint16]bool)
	for cursor < len(units) && strings.ContainsRune("-#+ 0,(<", rune(units[cursor])) {
		flag := units[cursor]
		if seenFlags[flag] {
			return specifier, cursor, fmt.Errorf("java.util.DuplicateFormatFlagsException: Flags = '%c'", flag)
		}
		seenFlags[flag] = true
		specifier.flags += string(rune(flag))
		if flag == '<' {
			specifier.reusePrevious = true
		}
		cursor++
	}

	widthStart := cursor
	for cursor < len(units) && units[cursor] >= '0' && units[cursor] <= '9' {
		cursor++
	}
	if cursor > widthStart {
		width, err := portableJavaStringParseFormatNumber(units[widthStart:cursor])
		if err != nil {
			return specifier, cursor, errors.New("java.util.IllegalFormatWidthException: -2147483648")
		}
		specifier.width, specifier.hasWidth = width, true
	}
	if cursor < len(units) && units[cursor] == '.' {
		cursor++
		precisionStart := cursor
		for cursor < len(units) && units[cursor] >= '0' && units[cursor] <= '9' {
			cursor++
		}
		if precisionStart == cursor {
			return specifier, cursor, errors.New("java.util.UnknownFormatConversionException: Conversion = '.'")
		}
		precision, err := portableJavaStringParseFormatNumber(units[precisionStart:cursor])
		if err != nil {
			return specifier, cursor, errors.New("java.util.IllegalFormatPrecisionException: -2147483648")
		}
		specifier.precision, specifier.hasPrecision = precision, true
	}
	if cursor < len(units) && (units[cursor] == 't' || units[cursor] == 'T') {
		specifier.dateTime = true
		specifier.upperDateTime = units[cursor] == 'T'
		cursor++
	}
	if cursor >= len(units) {
		return specifier, cursor, errors.New("java.util.UnknownFormatConversionException: Conversion = 't'")
	}
	conversion := units[cursor]
	if conversion > 0x7f {
		return specifier, cursor + 1, fmt.Errorf("java.util.UnknownFormatConversionException: Conversion = '%c'", conversion)
	}
	specifier.conversion = byte(conversion)
	cursor++
	originalUnits := units[start:cursor]
	var original strings.Builder
	for _, unit := range originalUnits {
		original.WriteByte(byte(unit))
	}
	specifier.original = original.String()
	return specifier, cursor, nil
}

func portableJavaStringParseFormatNumber(units []uint16) (int, error) {
	var value int64
	for _, unit := range units {
		value = value*10 + int64(unit-'0')
		if value > math.MaxInt32 {
			return int(int32(value)), errors.New("overflow")
		}
	}
	return int(value), nil
}

func portableJavaStringRenderNoArgumentFormat(
	ctx context.Context,
	builder *portableJavaStringBuilder,
	specifier portableJavaFormatSpecifier,
) error {
	flags := strings.ReplaceAll(specifier.flags, "<", "")
	if specifier.conversion == 'n' {
		if flags != "" {
			return fmt.Errorf("java.util.IllegalFormatFlagsException: Flags = '%s'", flags)
		}
		if specifier.hasWidth {
			return fmt.Errorf("java.util.IllegalFormatWidthException: %d", specifier.width)
		}
		if specifier.hasPrecision {
			return fmt.Errorf("java.util.IllegalFormatPrecisionException: %d", specifier.precision)
		}
		lineSeparator := "\n"
		if goruntime.GOOS == "windows" {
			lineSeparator = "\r\n"
		}
		return portableJavaStringAppendValueChecked(ctx, builder, String(lineSeparator))
	}
	if specifier.hasPrecision {
		return fmt.Errorf("java.util.IllegalFormatPrecisionException: %d", specifier.precision)
	}
	if strings.ContainsAny(flags, "#+ 0,(") {
		return fmt.Errorf("java.util.FormatFlagsConversionMismatchException: Conversion = %%, Flags = %s", flags)
	}
	if strings.Contains(flags, "-") && !specifier.hasWidth {
		return fmt.Errorf("java.util.MissingFormatWidthException: %s", specifier.original)
	}
	return portableJavaStringAppendPadded(ctx, builder, String("%"), specifier.width, specifier.hasWidth, strings.Contains(flags, "-"), false, "")
}

func portableJavaStringRenderArgument(
	ctx context.Context,
	builder *portableJavaStringBuilder,
	specifier portableJavaFormatSpecifier,
	argument portableJavaFormatArgument,
) error {
	if specifier.dateTime {
		conversion := specifier.conversion
		if specifier.upperDateTime && conversion >= 'a' && conversion <= 'z' {
			conversion -= 'a' - 'A'
		}
		return portableJavaStringIllegalFormatConversion(conversion, argument)
	}
	switch specifier.conversion {
	case 'b', 'B', 'h', 'H', 's', 'S':
		return portableJavaStringRenderGeneral(ctx, builder, specifier, argument)
	case 'c', 'C':
		return portableJavaStringRenderCharacter(ctx, builder, specifier, argument)
	case 'd', 'o', 'x', 'X':
		return portableJavaStringRenderInteger(ctx, builder, specifier, argument)
	case 'e', 'E', 'f', 'g', 'G', 'a', 'A':
		return portableJavaStringRenderFloat(ctx, builder, specifier, argument)
	default:
		return fmt.Errorf("java.util.UnknownFormatConversionException: Conversion = '%c'", specifier.conversion)
	}
}

func portableJavaStringRenderGeneral(
	ctx context.Context,
	builder *portableJavaStringBuilder,
	specifier portableJavaFormatSpecifier,
	argument portableJavaFormatArgument,
) error {
	flags := strings.ReplaceAll(specifier.flags, "<", "")
	if strings.Contains(flags, "-") && !specifier.hasWidth {
		return fmt.Errorf("java.util.MissingFormatWidthException: %s", specifier.original)
	}
	for _, flag := range flags {
		if flag != '-' {
			return fmt.Errorf("java.util.FormatFlagsConversionMismatchException: Conversion = %c, Flags = %c", byteLower(specifier.conversion), flag)
		}
	}

	var text Value
	switch byteLower(specifier.conversion) {
	case 's':
		if argument.value.IsNull() {
			text = String("null")
		} else if sequence, ok := portableJavaStringCharSequence(argument.value); ok {
			text = sequence
		} else {
			text = String(portableJavaValueString(argument.value))
		}
	case 'b':
		truth := !argument.value.IsNull()
		if argument.class == "java.lang.Boolean" {
			if object, ok := argument.value.Object(); ok {
				if primitive, ok := object.(*portableJavaPrimitive); ok && primitive != nil {
					truth = primitive.sleepValue().Int32() != 0
				}
			}
		}
		text = String(strconv.FormatBool(truth))
	case 'h':
		if argument.value.IsNull() {
			text = String("null")
		} else {
			text = String(strconv.FormatUint(uint64(uint32(portableJavaValueHash(argument.value))), 16))
		}
	}
	if specifier.conversion >= 'A' && specifier.conversion <= 'Z' {
		text = sleepStringMapCase(text, true)
	}
	if specifier.hasPrecision && sleepStringLength(text) > specifier.precision {
		text = sleepStringValueSlice(text, 0, specifier.precision)
	}
	return portableJavaStringAppendPadded(
		ctx, builder, text, specifier.width, specifier.hasWidth, strings.Contains(flags, "-"), false, "",
	)
}

func portableJavaStringRenderCharacter(
	ctx context.Context,
	builder *portableJavaStringBuilder,
	specifier portableJavaFormatSpecifier,
	argument portableJavaFormatArgument,
) error {
	flags := strings.ReplaceAll(specifier.flags, "<", "")
	if specifier.hasPrecision {
		return fmt.Errorf("java.util.IllegalFormatPrecisionException: %d", specifier.precision)
	}
	if strings.Contains(flags, "-") && !specifier.hasWidth {
		return fmt.Errorf("java.util.MissingFormatWidthException: %s", specifier.original)
	}
	for _, flag := range flags {
		if flag != '-' {
			return fmt.Errorf("java.util.FormatFlagsConversionMismatchException: Conversion = c, Flags = %c", flag)
		}
	}
	if argument.value.IsNull() {
		return portableJavaStringRenderNull(ctx, builder, specifier, false)
	}
	codePoint, ok := portableJavaStringFormatCodePoint(argument)
	if !ok {
		return portableJavaStringIllegalFormatConversion(specifier.conversion, argument)
	}
	if codePoint < 0 || codePoint > 0x10ffff {
		return fmt.Errorf("java.util.IllegalFormatCodePointException: Code point = 0x%x", uint32(codePoint))
	}
	var units []uint16
	if codePoint <= 0xffff {
		units = []uint16{uint16(codePoint)}
	} else {
		codePoint -= 0x10000
		units = []uint16{uint16(0xd800 + codePoint>>10), uint16(0xdc00 + codePoint&0x3ff)}
	}
	text := sleepStringValueFromUnits(units, nil)
	if specifier.conversion == 'C' {
		text = sleepStringMapCase(text, true)
	}
	return portableJavaStringAppendPadded(ctx, builder, text, specifier.width, specifier.hasWidth, strings.Contains(flags, "-"), false, "")
}

func portableJavaStringFormatCodePoint(argument portableJavaFormatArgument) (int64, bool) {
	value := argument.value
	class := argument.class
	if object, ok := value.Object(); ok {
		if primitive, ok := object.(*portableJavaPrimitive); ok && primitive != nil {
			value = primitive.sleepValue()
			class = primitive.className()
		}
	}
	switch class {
	case "java.lang.Byte", "java.lang.Short", "java.lang.Integer":
		return int64(value.Int32()), true
	case "java.lang.Character":
		units := sleepStringUnits(value)
		if len(units) == 0 {
			return 0, true
		}
		return int64(units[0]), true
	default:
		return 0, false
	}
}

func portableJavaStringRenderInteger(
	ctx context.Context,
	builder *portableJavaStringBuilder,
	specifier portableJavaFormatSpecifier,
	argument portableJavaFormatArgument,
) error {
	if specifier.hasPrecision {
		return fmt.Errorf("java.util.IllegalFormatPrecisionException: %d", specifier.precision)
	}
	flags := strings.ReplaceAll(specifier.flags, "<", "")
	if strings.Contains(flags, "-") && !specifier.hasWidth {
		return fmt.Errorf("java.util.MissingFormatWidthException: %s", specifier.original)
	}
	if strings.Contains(flags, "-") && strings.Contains(flags, "0") ||
		strings.Contains(flags, "+") && strings.Contains(flags, " ") ||
		strings.Contains(flags, "+") && strings.Contains(flags, "(") ||
		strings.Contains(flags, " ") && strings.Contains(flags, "(") {
		return fmt.Errorf("java.util.IllegalFormatFlagsException: Flags = '%s'", flags)
	}
	conversion := byteLower(specifier.conversion)
	allowed := "-0("
	if conversion == 'd' {
		allowed += "+ ,"
	} else {
		allowed += "#"
	}
	for _, flag := range flags {
		if !strings.ContainsRune(allowed, flag) {
			return fmt.Errorf("java.util.FormatFlagsConversionMismatchException: Conversion = %c, Flags = %c", conversion, flag)
		}
	}
	if argument.value.IsNull() {
		return portableJavaStringRenderNull(ctx, builder, specifier, false)
	}
	value, bits, ok := portableJavaStringFormatInteger(argument)
	if !ok {
		return portableJavaStringIllegalFormatConversion(specifier.conversion, argument)
	}

	negative := value < 0
	var magnitude uint64
	base := 10
	if conversion == 'd' {
		if negative {
			magnitude = uint64(-(value + 1)) + 1
		} else {
			magnitude = uint64(value)
		}
	} else {
		base = 16
		if conversion == 'o' {
			base = 8
		}
		mask := uint64(math.MaxUint64)
		if bits < 64 {
			mask = uint64(1)<<bits - 1
		}
		magnitude = uint64(value) & mask
		negative = false
	}
	digits := strconv.FormatUint(magnitude, base)
	if conversion == 'd' && strings.Contains(flags, ",") {
		digits = portableJavaStringGroupDecimal(digits)
	}
	if specifier.conversion == 'X' {
		digits = strings.ToUpper(digits)
	}
	prefix, suffix := "", ""
	if conversion != 'd' && strings.Contains(flags, "#") {
		if conversion == 'o' {
			if !strings.HasPrefix(digits, "0") {
				prefix = "0"
			}
		} else if specifier.conversion == 'X' {
			prefix = "0X"
		} else {
			prefix = "0x"
		}
	}
	if conversion == 'd' {
		switch {
		case negative && strings.Contains(flags, "("):
			prefix, suffix = "(", ")"
		case negative:
			prefix = "-"
		case strings.Contains(flags, "+"):
			prefix = "+"
		case strings.Contains(flags, " "):
			prefix = " "
		}
	}
	return portableJavaStringAppendPadded(
		ctx, builder, String(prefix+digits+suffix), specifier.width, specifier.hasWidth,
		strings.Contains(flags, "-"), strings.Contains(flags, "0"), prefix,
	)
}

func portableJavaStringFormatInteger(argument portableJavaFormatArgument) (int64, uint, bool) {
	value, class := argument.value, argument.class
	if object, ok := value.Object(); ok {
		if primitive, ok := object.(*portableJavaPrimitive); ok && primitive != nil {
			value, class = primitive.sleepValue(), primitive.className()
		}
	}
	switch class {
	case "java.lang.Byte":
		return int64(int8(value.Int32())), 8, true
	case "java.lang.Short":
		return int64(int16(value.Int32())), 16, true
	case "java.lang.Integer":
		return int64(value.Int32()), 32, true
	case "java.lang.Long":
		return value.Int64(), 64, true
	default:
		return 0, 0, false
	}
}

func portableJavaStringRenderFloat(
	ctx context.Context,
	builder *portableJavaStringBuilder,
	specifier portableJavaFormatSpecifier,
	argument portableJavaFormatArgument,
) error {
	flags := strings.ReplaceAll(specifier.flags, "<", "")
	if strings.Contains(flags, "-") && !specifier.hasWidth {
		return fmt.Errorf("java.util.MissingFormatWidthException: %s", specifier.original)
	}
	if strings.Contains(flags, "-") && strings.Contains(flags, "0") ||
		strings.Contains(flags, "+") && strings.Contains(flags, " ") ||
		strings.Contains(flags, "+") && strings.Contains(flags, "(") ||
		strings.Contains(flags, " ") && strings.Contains(flags, "(") {
		return fmt.Errorf("java.util.IllegalFormatFlagsException: Flags = '%s'", flags)
	}
	conversion := byteLower(specifier.conversion)
	allowed := "-+ 0("
	if conversion == 'f' {
		allowed += ","
	}
	if conversion == 'a' {
		allowed += "#"
	}
	for _, flag := range flags {
		if !strings.ContainsRune(allowed, flag) {
			return fmt.Errorf("java.util.FormatFlagsConversionMismatchException: Conversion = %c, Flags = %c", conversion, flag)
		}
	}
	precision := 6
	if specifier.hasPrecision {
		precision = specifier.precision
	}
	if err := portableJavaStringAccountFormatWork(ctx, precision); err != nil {
		return err
	}
	if argument.value.IsNull() {
		return portableJavaStringRenderNull(ctx, builder, specifier, true)
	}
	value, ok := portableJavaStringFormatFloat(argument)
	if !ok {
		return portableJavaStringIllegalFormatConversion(specifier.conversion, argument)
	}
	negative := math.Signbit(value)
	absolute := math.Abs(value)
	var digits string
	switch {
	case math.IsNaN(value):
		digits, negative = "NaN", false
	case math.IsInf(value, 0):
		digits = "Infinity"
	case conversion == 'f':
		digits = portableJavaStringFormatFixedHalfUp(absolute, precision)
		if strings.Contains(flags, ",") {
			integer, fraction, _ := strings.Cut(digits, ".")
			digits = portableJavaStringGroupDecimal(integer)
			if fraction != "" || precision > 0 {
				digits += "." + fraction
			}
		}
	case conversion == 'e':
		digits = portableJavaStringFormatScientificHalfUp(absolute, precision, specifier.conversion == 'E')
	case conversion == 'g':
		if precision == 0 {
			precision = 1
		}
		exponent := 0
		if absolute != 0 {
			exponent = int(math.Floor(math.Log10(absolute)))
		}
		if exponent < -4 || exponent >= precision {
			digits = portableJavaStringFormatScientificHalfUp(absolute, precision-1, specifier.conversion == 'G')
		} else {
			fractionDigits := precision - exponent - 1
			if fractionDigits < 0 {
				fractionDigits = 0
			}
			digits = portableJavaStringFormatFixedHalfUp(absolute, fractionDigits)
		}
	case conversion == 'a':
		hexPrecision := -1
		if specifier.hasPrecision {
			hexPrecision = specifier.precision
			if hexPrecision == 0 {
				hexPrecision = 1
			}
		}
		digits = portableJavaStringFormatHexFloat(absolute, hexPrecision)
	}
	if specifier.conversion >= 'A' && specifier.conversion <= 'Z' {
		digits = strings.ToUpper(digits)
	}
	prefix, suffix := "", ""
	switch {
	case negative && strings.Contains(flags, "("):
		prefix, suffix = "(", ")"
	case negative:
		prefix = "-"
	case strings.Contains(flags, "+"):
		prefix = "+"
	case strings.Contains(flags, " "):
		prefix = " "
	}
	return portableJavaStringAppendPadded(
		ctx, builder, String(prefix+digits+suffix), specifier.width, specifier.hasWidth,
		strings.Contains(flags, "-"), strings.Contains(flags, "0"), prefix,
	)
}

func portableJavaStringFormatFloat(argument portableJavaFormatArgument) (float64, bool) {
	value, class := argument.value, argument.class
	if object, ok := value.Object(); ok {
		if primitive, ok := object.(*portableJavaPrimitive); ok && primitive != nil {
			value, class = primitive.sleepValue(), primitive.className()
		}
	}
	if class != "java.lang.Float" && class != "java.lang.Double" {
		return 0, false
	}
	return value.Float64(), true
}

func portableJavaStringIllegalFormatConversion(conversion byte, argument portableJavaFormatArgument) error {
	class := argument.class
	if argument.value.IsNull() {
		// All ordinary Formatter conversions render null as the literal "null";
		// this helper is reached only after a type-specific non-null check.
		class = "null"
	}
	if class == "" {
		class, _ = portableObjectClass(argument.value)
	}
	if class == "" {
		class = "java.lang.Object"
	}
	return fmt.Errorf("java.util.IllegalFormatConversionException: %c != %s", conversion, class)
}

func portableJavaStringRenderNull(
	ctx context.Context,
	builder *portableJavaStringBuilder,
	specifier portableJavaFormatSpecifier,
	allowPrecision bool,
) error {
	text := String("null")
	if specifier.conversion >= 'A' && specifier.conversion <= 'Z' {
		text = String("NULL")
	}
	if allowPrecision && specifier.hasPrecision && specifier.precision < sleepStringLength(text) {
		text = sleepStringValueSlice(text, 0, specifier.precision)
	}
	flags := strings.ReplaceAll(specifier.flags, "<", "")
	return portableJavaStringAppendPadded(
		ctx, builder, text, specifier.width, specifier.hasWidth, strings.Contains(flags, "-"), false, "",
	)
}

func portableJavaStringAppendPadded(
	ctx context.Context,
	builder *portableJavaStringBuilder,
	value Value,
	width int,
	hasWidth, left, zero bool,
	prefix string,
) error {
	length := sleepStringLength(value)
	padding := 0
	if hasWidth && width > length {
		padding = width - length
	}
	if left {
		if err := portableJavaStringAppendValueChecked(ctx, builder, value); err != nil {
			return err
		}
		return portableJavaStringAppendPadding(ctx, builder, padding, ' ')
	}
	if zero && padding > 0 && prefix != "" && strings.HasPrefix(value.String(), prefix) {
		prefixValue := String(prefix)
		if err := portableJavaStringAppendValueChecked(ctx, builder, prefixValue); err != nil {
			return err
		}
		if err := portableJavaStringAppendPadding(ctx, builder, padding, '0'); err != nil {
			return err
		}
		return portableJavaStringAppendRangeChecked(
			ctx, builder, sleepStringUnits(value), sleepStringRawMask(value), sleepStringLength(prefixValue), length,
		)
	}
	pad := uint16(' ')
	if zero {
		pad = '0'
	}
	if err := portableJavaStringAppendPadding(ctx, builder, padding, pad); err != nil {
		return err
	}
	return portableJavaStringAppendValueChecked(ctx, builder, value)
}

func portableJavaStringAppendPadding(ctx context.Context, builder *portableJavaStringBuilder, count int, unit uint16) error {
	chunk := make([]uint16, min(count, portableJavaStringNativeLoopChunk))
	for index := range chunk {
		chunk[index] = unit
	}
	for written := 0; written < count; {
		if err := portableJavaStringLoopCheck(ctx, written); err != nil {
			return err
		}
		amount := min(len(chunk), count-written)
		if err := builder.append(chunk[:amount], nil); err != nil {
			return err
		}
		written += amount
	}
	return executionContextError(ctx)
}

func portableJavaStringAccountFormatWork(ctx context.Context, amount int) error {
	for accounted := portableJavaStringNativeLoopChunk; accounted <= amount; accounted += portableJavaStringNativeLoopChunk {
		if err := executionContextError(ctx); err != nil {
			return err
		}
		if err := consumeInstruction(ctx); err != nil {
			return err
		}
		if amount-accounted < portableJavaStringNativeLoopChunk {
			break
		}
	}
	return executionContextError(ctx)
}

func portableJavaStringGroupDecimal(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	first := len(digits) % 3
	if first == 0 {
		first = 3
	}
	var builder strings.Builder
	builder.Grow(len(digits) + len(digits)/3)
	builder.WriteString(digits[:first])
	for cursor := first; cursor < len(digits); cursor += 3 {
		builder.WriteByte(',')
		builder.WriteString(digits[cursor : cursor+3])
	}
	return builder.String()
}

func portableJavaStringNormalizeExponent(value string, upper bool) string {
	separator := strings.IndexAny(value, "eE")
	if separator < 0 {
		return value
	}
	mantissa, exponent := value[:separator], value[separator+1:]
	sign := byte('+')
	if strings.HasPrefix(exponent, "-") {
		sign = '-'
		exponent = exponent[1:]
	} else if strings.HasPrefix(exponent, "+") {
		exponent = exponent[1:]
	}
	for len(exponent) < 2 {
		exponent = "0" + exponent
	}
	marker := byte('e')
	if upper {
		marker = 'E'
	}
	return mantissa + string(marker) + string(sign) + exponent
}

func portableJavaStringFormatFixedHalfUp(value float64, precision int) string {
	if precision < 0 {
		precision = 0
	}
	decimal := strconv.FormatFloat(value, 'f', -1, 64)
	rational, ok := new(big.Rat).SetString(decimal)
	if !ok {
		return strconv.FormatFloat(value, 'f', precision, 64)
	}
	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)
	scaledNumerator := new(big.Int).Mul(rational.Num(), scale)
	quotient, remainder := new(big.Int), new(big.Int)
	quotient.QuoRem(scaledNumerator, rational.Denom(), remainder)
	if new(big.Int).Lsh(remainder, 1).Cmp(rational.Denom()) >= 0 {
		quotient.Add(quotient, big.NewInt(1))
	}
	digits := quotient.String()
	if precision == 0 {
		return digits
	}
	if len(digits) <= precision {
		digits = strings.Repeat("0", precision-len(digits)+1) + digits
	}
	separator := len(digits) - precision
	return digits[:separator] + "." + digits[separator:]
}

func portableJavaStringFormatScientificHalfUp(value float64, precision int, upper bool) string {
	exponent := 0
	if value != 0 {
		exponent = int(math.Floor(math.Log10(value)))
	}
	mantissa := value
	if value != 0 {
		mantissa = value / math.Pow10(exponent)
	}
	digits := portableJavaStringFormatFixedHalfUp(mantissa, precision)
	if strings.HasPrefix(digits, "10") {
		exponent++
		digits = portableJavaStringFormatFixedHalfUp(1, precision)
	}
	marker := "e"
	if upper {
		marker = "E"
	}
	sign := "+"
	if exponent < 0 {
		sign = "-"
		exponent = -exponent
	}
	exponentText := strconv.Itoa(exponent)
	if len(exponentText) < 2 {
		exponentText = "0" + exponentText
	}
	return digits + marker + sign + exponentText
}

func portableJavaStringFormatHexFloat(value float64, precision int) string {
	formatted := strconv.FormatFloat(value, 'x', precision, 64)
	separator := strings.LastIndexByte(formatted, 'p')
	if separator < 0 {
		return formatted
	}
	mantissa, exponent := formatted[:separator], formatted[separator+1:]
	if !strings.Contains(mantissa, ".") {
		mantissa += ".0"
	}
	sign := ""
	if strings.HasPrefix(exponent, "-") {
		sign = "-"
		exponent = exponent[1:]
	} else if strings.HasPrefix(exponent, "+") {
		exponent = exponent[1:]
	}
	exponent = strings.TrimLeft(exponent, "0")
	if exponent == "" {
		exponent = "0"
	}
	return mantissa + "p" + sign + exponent
}

func byteLower(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + ('a' - 'A')
	}
	return value
}
