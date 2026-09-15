package opfor

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// newSleepString preserves the long-standing ability to pass invalid UTF-8
// octets through String while treating valid Go strings as Unicode text. The
// explicit BinaryString constructor is required when a valid UTF-8 byte
// sequence must remain a sequence of individual byte-sized Java chars.
func newSleepString(value string) Value {
	if utf8.ValidString(value) {
		return Value{kind: KindString, data: value}
	}
	units := make([]uint16, 0, len(value))
	raw := make([]bool, 0, len(value))
	for len(value) != 0 {
		character, size := utf8.DecodeRuneInString(value)
		if character == utf8.RuneError && size == 1 {
			units = append(units, uint16(value[0]))
			raw = append(raw, true)
			value = value[1:]
			continue
		}
		if character <= 0xffff {
			units = append(units, uint16(character))
			raw = append(raw, false)
		} else {
			first, second := utf16.EncodeRune(character)
			units = append(units, uint16(first), uint16(second))
			raw = append(raw, false, false)
		}
		value = value[size:]
	}
	return sleepStringValueFromUnits(units, raw)
}

func newSleepBinaryString(data []byte) Value {
	if len(data) == 0 {
		return Value{kind: KindString, data: "", stringUnits: []uint16{}, stringRaw: []bool{}}
	}
	units := make([]uint16, len(data))
	raw := make([]bool, len(data))
	for index, octet := range data {
		units[index] = uint16(octet)
		raw[index] = true
	}
	return Value{kind: KindString, data: string(append([]byte(nil), data...)), stringUnits: units, stringRaw: raw}
}

// sleepStringCoercion retains metadata when value is already a string and
// applies Sleep's ordinary string coercion to every other scalar.
func sleepStringCoercion(value Value) Value {
	if value.Kind() == KindString {
		return value
	}
	return String(value.String())
}

func sleepStringUnits(value Value) []uint16 {
	value = sleepStringCoercion(value)
	if value.stringUnits != nil {
		return append([]uint16(nil), value.stringUnits...)
	}
	return utf16.Encode([]rune(value.String()))
}

func sleepStringRawMask(value Value) []bool {
	value = sleepStringCoercion(value)
	units := sleepStringUnits(value)
	if len(value.stringRaw) == len(units) {
		return append([]bool(nil), value.stringRaw...)
	}
	return make([]bool, len(units))
}

func sleepStringValueFromUnits(units []uint16, raw []bool) Value {
	units = append([]uint16(nil), units...)
	if len(raw) != len(units) {
		raw = make([]bool, len(units))
	} else {
		raw = append([]bool(nil), raw...)
	}
	return Value{
		kind:        KindString,
		data:        sleepRenderStringUnits(units, raw),
		stringUnits: units,
		stringRaw:   raw,
	}
}

func sleepRenderStringUnits(units []uint16, raw []bool) string {
	var output bytes.Buffer
	for index := 0; index < len(units); index++ {
		unit := units[index]
		if index < len(raw) && raw[index] && unit <= 0xff {
			output.WriteByte(byte(unit))
			continue
		}
		if unit >= 0xd800 && unit <= 0xdbff && index+1 < len(units) &&
			units[index+1] >= 0xdc00 && units[index+1] <= 0xdfff &&
			(index >= len(raw) || !raw[index]) && (index+1 >= len(raw) || !raw[index+1]) {
			output.WriteRune(utf16.DecodeRune(rune(unit), rune(units[index+1])))
			index++
			continue
		}
		if unit >= 0xd800 && unit <= 0xdfff {
			// Go runes exclude surrogate code points. WTF-8 is used only as the
			// reversible host spelling of the exact Java code unit.
			output.WriteByte(0xe0 | byte(unit>>12))
			output.WriteByte(0x80 | byte(unit>>6)&0x3f)
			output.WriteByte(0x80 | byte(unit)&0x3f)
			continue
		}
		output.WriteRune(rune(unit))
	}
	return output.String()
}

// sleepWTF8SurrogateAt recognizes the reversible host spelling used for an
// unpaired Java UTF-16 surrogate. Go's UTF-8 decoder deliberately rejects
// these sequences, so consumers that operate on canonical Sleep strings must
// identify them before calling utf8.DecodeRuneInString.
func sleepWTF8SurrogateAt(value string, index int) (uint16, bool) {
	if index < 0 || index+3 > len(value) || value[index] != 0xed {
		return 0, false
	}
	second, third := value[index+1], value[index+2]
	if second < 0xa0 || second > 0xbf || third < 0x80 || third > 0xbf {
		return 0, false
	}
	unit := uint16(value[index]&0x0f)<<12 |
		uint16(second&0x3f)<<6 |
		uint16(third&0x3f)
	return unit, unit >= 0xd800 && unit <= 0xdfff
}

// sleepStringValueFromCanonical reconstructs a Sleep string from the stable
// canonical spelling returned by sleepCanonicalString. In particular, it
// turns each WTF-8 surrogate spelling back into one Java UTF-16 code unit
// instead of the three replacement/error units newSleepString would infer
// from ordinary invalid UTF-8 input.
func sleepStringValueFromCanonical(value string) Value {
	units := make([]uint16, 0, len(value))
	raw := make([]bool, 0, len(value))
	for index := 0; index < len(value); {
		if unit, ok := sleepWTF8SurrogateAt(value, index); ok {
			units = append(units, unit)
			raw = append(raw, false)
			index += 3
			continue
		}
		character, size := utf8.DecodeRuneInString(value[index:])
		if character == utf8.RuneError && size == 1 {
			units = append(units, uint16(value[index]))
			raw = append(raw, true)
			index++
			continue
		}
		if character <= 0xffff {
			units = append(units, uint16(character))
			raw = append(raw, false)
		} else {
			first, second := utf16.EncodeRune(character)
			units = append(units, uint16(first), uint16(second))
			raw = append(raw, false, false)
		}
		index += size
	}
	return sleepStringValueFromUnits(units, raw)
}

func sleepStringLength(value Value) int { return len(sleepStringUnits(value)) }

func sleepStringValueSlice(value Value, start, end int) Value {
	units := sleepStringUnits(value)
	raw := sleepStringRawMask(value)
	return sleepStringValueFromUnits(units[start:end], raw[start:end])
}

func sleepStringConcat(values ...Value) Value {
	// Most Sleep concatenations join ordinary text produced by literals or
	// scalar coercion. Such values have no explicit UTF-16/raw-provenance
	// representation, so their Go string spelling can be joined directly: Go
	// concatenation preserves valid UTF-8 and sleepStringUnits will derive the
	// same Java code units lazily if a later operation needs them. Keep every
	// value with explicit units on the generic path below so binary octets,
	// unpaired surrogates, and provenance masks remain exact.
	allPlainText := true
	totalBytes := 0
	for index := range values {
		values[index] = sleepStringCoercion(values[index])
		if values[index].stringUnits != nil {
			allPlainText = false
		}
		totalBytes += len(values[index].data.(string))
	}
	if allPlainText {
		switch len(values) {
		case 0:
			return Value{kind: KindString, data: ""}
		case 1:
			return Value{kind: KindString, data: values[0].data.(string)}
		case 2:
			return Value{kind: KindString, data: values[0].data.(string) + values[1].data.(string)}
		default:
			var builder strings.Builder
			builder.Grow(totalBytes)
			for _, value := range values {
				builder.WriteString(value.data.(string))
			}
			return Value{kind: KindString, data: builder.String()}
		}
	}

	var units []uint16
	var raw []bool
	for _, value := range values {
		valueUnits := sleepStringUnits(value)
		units = append(units, valueUnits...)
		raw = append(raw, sleepStringRawMask(value)...)
	}
	return sleepStringValueFromUnits(units, raw)
}

func sleepUTF16CodePointAt(units []uint16, index int) (int32, int) {
	first := units[index]
	if first >= 0xd800 && first <= 0xdbff && index+1 < len(units) {
		second := units[index+1]
		if second >= 0xdc00 && second <= 0xdfff {
			return int32(0x10000 + (uint32(first)-0xd800)<<10 + uint32(second) - 0xdc00), 2
		}
	}
	return int32(first), 1
}

func sleepUTF16CodePointBefore(units []uint16, index int) (int32, int) {
	second := units[index-1]
	if second >= 0xdc00 && second <= 0xdfff && index > 1 {
		first := units[index-2]
		if first >= 0xd800 && first <= 0xdbff {
			return int32(0x10000 + (uint32(first)-0xd800)<<10 + uint32(second) - 0xdc00), 2
		}
	}
	return int32(second), 1
}

func sleepUTF16CodePointCount(units []uint16, start, end int) int {
	count := 0
	for index := start; index < end; {
		_, width := sleepUTF16CodePointAt(units[:end], index)
		index += width
		count++
	}
	return count
}

func sleepStringIsBlank(value Value) bool {
	units := sleepStringUnits(value)
	for index := 0; index < len(units); {
		codePoint, width := sleepUTF16CodePointAt(units, index)
		if !sleepJavaIsWhitespace(rune(codePoint)) {
			return false
		}
		index += width
	}
	return true
}

func sleepStringStrip(value Value, leading, trailing bool) Value {
	units := sleepStringUnits(value)
	start, end := 0, len(units)
	if leading {
		for start < end {
			codePoint, width := sleepUTF16CodePointAt(units[:end], start)
			if !sleepJavaIsWhitespace(rune(codePoint)) {
				break
			}
			start += width
		}
	}
	if trailing {
		for end > start {
			codePoint, width := sleepUTF16CodePointBefore(units, end)
			if !sleepJavaIsWhitespace(rune(codePoint)) {
				break
			}
			end -= width
		}
	}
	return sleepStringValueSlice(value, start, end)
}

// sleepStringReplaceLiteral implements String.replace(CharSequence,
// CharSequence). Matching is non-overlapping and proceeds over Java UTF-16
// code units. In particular, an empty target inserts the replacement at all
// length+1 code-unit boundaries, including between a surrogate pair.
func sleepStringReplaceLiteral(value, target, replacement Value) (Value, error) {
	units := sleepStringUnits(value)
	raw := sleepStringRawMask(value)
	targetUnits := sleepStringUnits(target)
	replacementUnits := sleepStringUnits(replacement)
	replacementRaw := sleepStringRawMask(replacement)

	positions := make([]int, 0)
	if len(targetUnits) == 0 {
		positions = make([]int, len(units)+1)
		for index := range positions {
			positions[index] = index
		}
	} else {
		for index := 0; index+len(targetUnits) <= len(units); {
			if equalUTF16Units(units[index:index+len(targetUnits)], targetUnits) {
				positions = append(positions, index)
				index += len(targetUnits)
				continue
			}
			index++
		}
	}
	if len(positions) == 0 {
		return value, nil
	}

	resultLength := int64(len(units)) + int64(len(positions))*int64(len(replacementUnits)-len(targetUnits))
	if resultLength < 0 || resultLength > math.MaxInt32 {
		return Null(), fmt.Errorf("java.lang.OutOfMemoryError: Required length exceeds implementation limit")
	}
	resultUnits := make([]uint16, 0, int(resultLength))
	resultRaw := make([]bool, 0, int(resultLength))
	last := 0
	for _, position := range positions {
		resultUnits = append(resultUnits, units[last:position]...)
		resultRaw = append(resultRaw, raw[last:position]...)
		resultUnits = append(resultUnits, replacementUnits...)
		resultRaw = append(resultRaw, replacementRaw...)
		last = position + len(targetUnits)
	}
	resultUnits = append(resultUnits, units[last:]...)
	resultRaw = append(resultRaw, raw[last:]...)
	return sleepStringValueFromUnits(resultUnits, resultRaw), nil
}

func sleepStringRepeat(value Value, count int) Value {
	if count <= 0 {
		return String("")
	}
	units := sleepStringUnits(value)
	raw := sleepStringRawMask(value)
	resultUnits := make([]uint16, 0, len(units)*count)
	resultRaw := make([]bool, 0, len(raw)*count)
	for range count {
		resultUnits = append(resultUnits, units...)
		resultRaw = append(resultRaw, raw...)
	}
	return sleepStringValueFromUnits(resultUnits, resultRaw)
}

func sleepStringValuesEqual(left, right Value) bool {
	a, b := sleepStringUnits(left), sleepStringUnits(right)
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func sleepStringCompareValues(left, right Value) int {
	a, b := sleepStringUnits(left), sleepStringUnits(right)
	limit := len(a)
	if len(b) < limit {
		limit = len(b)
	}
	for index := 0; index < limit; index++ {
		if a[index] != b[index] {
			// java.lang.String.compareTo returns the arithmetic difference of
			// the first unequal UTF-16 chars, not only its sign.
			return int(a[index]) - int(b[index])
		}
	}
	return len(a) - len(b)
}

func sleepStringUnitIndex(haystack, needle Value, from int, reverse bool) int {
	return sleepUTF16UnitIndex(sleepStringUnits(haystack), sleepStringUnits(needle), from, reverse)
}

func sleepUTF16UnitIndex(haystack, needle []uint16, from int, reverse bool) int {
	if !reverse {
		if from < 0 {
			from = 0
		}
		if len(needle) == 0 {
			if from > len(haystack) {
				return len(haystack)
			}
			return from
		}
		if from >= len(haystack) {
			return -1
		}
		for index := from; index+len(needle) <= len(haystack); index++ {
			if equalUTF16Units(haystack[index:index+len(needle)], needle) {
				return index
			}
		}
		return -1
	}

	if from < 0 {
		return -1
	}
	if len(needle) == 0 {
		if from > len(haystack) {
			return len(haystack)
		}
		return from
	}
	last := len(haystack) - len(needle)
	if last < 0 {
		return -1
	}
	if from < last {
		last = from
	}
	for index := last; index >= 0; index-- {
		if equalUTF16Units(haystack[index:index+len(needle)], needle) {
			return index
		}
	}
	return -1
}

func equalUTF16Units(left, right []uint16) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func sleepStringContains(haystack, needle Value) bool {
	return sleepStringUnitIndex(haystack, needle, 0, false) >= 0
}

func sleepStringReplaceAll(value, oldValue, newValue Value) Value {
	oldUnits := sleepStringUnits(oldValue)
	if len(oldUnits) == 0 {
		return value
	}
	units := sleepStringUnits(value)
	raw := sleepStringRawMask(value)
	newUnits := sleepStringUnits(newValue)
	newRaw := sleepStringRawMask(newValue)
	resultUnits := make([]uint16, 0, len(units))
	resultRaw := make([]bool, 0, len(raw))
	for index := 0; index < len(units); {
		if index+len(oldUnits) <= len(units) && equalUTF16Units(units[index:index+len(oldUnits)], oldUnits) {
			resultUnits = append(resultUnits, newUnits...)
			resultRaw = append(resultRaw, newRaw...)
			index += len(oldUnits)
			continue
		}
		resultUnits = append(resultUnits, units[index])
		resultRaw = append(resultRaw, raw[index])
		index++
	}
	return sleepStringValueFromUnits(resultUnits, resultRaw)
}

func sleepStringText(value Value) string {
	return string(utf16.Decode(sleepStringUnits(value)))
}

type javaStringSimpleCaseMapping struct {
	from rune
	to   rune
}

// sleepStringMapCase implements the locale-independent mapping used by
// String.toUpperCase(Locale.ROOT) and String.toLowerCase(Locale.ROOT). Its
// generated Unicode 17 tables include unconditional one-to-many mappings;
// lowercasing additionally applies SpecialCasing's Final_Sigma context rule.
// Unpaired UTF-16 surrogates remain unchanged, as they do in Java.
func sleepStringMapCase(value Value, upper bool) Value {
	value = sleepStringCoercion(value)
	units := sleepStringUnits(value)
	raw := sleepStringRawMask(value)
	resultUnits := make([]uint16, 0, len(units))
	resultRaw := make([]bool, 0, len(raw))
	changed := false
	for index := 0; index < len(units); {
		codePoint, width := sleepUTF16CodePointAt(units, index)
		mapped, ok := sleepJavaRootCaseMapping(rune(codePoint), upper)
		if !upper && codePoint == 0x03a3 && sleepJavaFinalSigma(units, index, width) {
			mapped, ok = "\u03c2", true
		}
		if !ok {
			resultUnits = append(resultUnits, units[index:index+width]...)
			resultRaw = append(resultRaw, raw[index:index+width]...)
			index += width
			continue
		}
		mappedUnits := utf16.Encode([]rune(mapped))
		if equalUTF16Units(mappedUnits, units[index:index+width]) {
			resultUnits = append(resultUnits, units[index:index+width]...)
			resultRaw = append(resultRaw, raw[index:index+width]...)
		} else {
			changed = true
			resultUnits = append(resultUnits, mappedUnits...)
			resultRaw = append(resultRaw, make([]bool, len(mappedUnits))...)
		}
		index += width
	}
	if !changed {
		return value
	}
	return sleepStringValueFromUnits(resultUnits, resultRaw)
}

func sleepJavaRootCaseMapping(value rune, upper bool) (string, bool) {
	if upper {
		return javaRegexRootUpperMapping(value)
	}
	left, right := 0, len(javaStringRootLowerMappings)
	for left < right {
		middle := left + (right-left)/2
		if javaStringRootLowerMappings[middle].from < value {
			left = middle + 1
		} else {
			right = middle
		}
	}
	if left < len(javaStringRootLowerMappings) && javaStringRootLowerMappings[left].from == value {
		return javaStringRootLowerMappings[left].to, true
	}
	return "", false
}

func sleepJavaSimpleCase(value rune, upper bool) rune {
	mappings := javaStringSimpleLowerMappings
	if upper {
		mappings = javaStringSimpleUpperMappings
	}
	left, right := 0, len(mappings)
	for left < right {
		middle := left + (right-left)/2
		if mappings[middle].from < value {
			left = middle + 1
		} else {
			right = middle
		}
	}
	if left < len(mappings) && mappings[left].from == value {
		return mappings[left].to
	}
	return value
}

func sleepJavaFinalSigma(units []uint16, index, width int) bool {
	precededByCased := false
	for current := index; current > 0; {
		codePoint, codePointWidth := sleepUTF16CodePointBefore(units, current)
		current -= codePointWidth
		if javaRegexRuneInRanges(rune(codePoint), javaStringCaseIgnorableRanges) {
			continue
		}
		precededByCased = javaRegexRuneInRanges(rune(codePoint), javaStringCasedRanges)
		break
	}
	if !precededByCased {
		return false
	}
	for current := index + width; current < len(units); {
		codePoint, codePointWidth := sleepUTF16CodePointAt(units, current)
		if javaRegexRuneInRanges(rune(codePoint), javaStringCaseIgnorableRanges) {
			current += codePointWidth
			continue
		}
		return !javaRegexRuneInRanges(rune(codePoint), javaStringCasedRanges)
	}
	return true
}

// sleepStringEqualFold mirrors String.equalsIgnoreCase's code-point loop:
// direct equality, simple upper-case equality, then simple lower-case equality
// of the upper results. Generated Unicode 17 simple mappings keep this stable
// across Go toolchain versions and cover supplementary pairs such as Deseret.
func sleepStringEqualFold(left, right Value) bool {
	a, b := sleepStringUnits(left), sleepStringUnits(right)
	if len(a) != len(b) {
		return false
	}
	for leftIndex, rightIndex := 0, 0; leftIndex < len(a) && rightIndex < len(b); {
		leftCodePoint, leftWidth := sleepUTF16CodePointAt(a, leftIndex)
		rightCodePoint, rightWidth := sleepUTF16CodePointAt(b, rightIndex)
		if leftCodePoint != rightCodePoint {
			upperLeft := sleepJavaSimpleCase(rune(leftCodePoint), true)
			upperRight := sleepJavaSimpleCase(rune(rightCodePoint), true)
			if upperLeft != upperRight &&
				sleepJavaSimpleCase(upperLeft, false) != sleepJavaSimpleCase(upperRight, false) {
				return false
			}
		}
		leftIndex += leftWidth
		rightIndex += rightWidth
	}
	return true
}

// sleepCanonicalString is a stable Go map-key spelling for Java String
// identity. Binary provenance is intentionally excluded because Java hashes
// and equality observe only UTF-16 code units.
func sleepCanonicalString(value Value) string {
	units := sleepStringUnits(value)
	return sleepRenderStringUnits(units, make([]bool, len(units)))
}

func sleepStringLowBytes(value Value) []byte {
	units := sleepStringUnits(value)
	result := make([]byte, len(units))
	for index, unit := range units {
		result[index] = byte(unit)
	}
	return result
}

func sleepStringAlign(value Value, width int) Value {
	length := sleepStringLength(value)
	amount := width
	right := false
	if amount < 0 {
		amount = -amount
		right = true
	}
	if amount <= length {
		return value
	}
	padding := String(strings.Repeat(" ", amount-length))
	if right {
		return sleepStringConcat(padding, value)
	}
	return sleepStringConcat(value, padding)
}
