package bofloader

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
	"unicode/utf8"
	"unsafe"
)

const (
	maxCallbackData   = 16 << 20
	maxFormatString   = 64 << 10
	maxFormattedData  = 1 << 20
	maxPrintfArgument = 10
	maxPrintfField    = 64 << 10
)

type beaconDataParser struct {
	original uintptr
	buffer   uintptr
	length   int32
	size     int32
}

type beaconFormat struct {
	original uintptr
	buffer   uintptr
	length   int32
	size     int32
}

var (
	callbackOnce    sync.Once
	callbackSymbols map[string]uintptr
	callbackInitErr error
)

var builtinBeaconCallbackNames = map[string]struct{}{
	"BeaconDataParse":         {},
	"BeaconDataInt":           {},
	"BeaconDataShort":         {},
	"BeaconDataLength":        {},
	"BeaconDataExtract":       {},
	"BeaconDataExtractOrNull": {},
	"BeaconFormatAlloc":       {},
	"BeaconFormatReset":       {},
	"BeaconFormatFree":        {},
	"BeaconFormatAppend":      {},
	"BeaconFormatPrintf":      {},
	"BeaconFormatToString":    {},
	"BeaconFormatInt":         {},
	"BeaconPrintf":            {},
	"BeaconOutput":            {},
	"toWideChar":              {},
}

func resolveBeaconCallback(symbol string) (uintptr, bool, error) {
	name := normalizeImportedSymbol(symbol)
	if _, ok := builtinBeaconCallbackNames[name]; !ok {
		return 0, false, nil
	}
	callbackOnce.Do(func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				callbackInitErr = fmt.Errorf("register Beacon callbacks: %v", recovered)
			}
		}()
		callbackSymbols = platformCallbacks()
	})
	if callbackInitErr != nil {
		return 0, false, callbackInitErr
	}
	address, ok := callbackSymbols[name]
	return address, ok, nil
}

func normalizeImportedSymbol(symbol string) string {
	name := strings.TrimSpace(symbol)
	if strings.HasPrefix(name, "__imp_") {
		name = strings.TrimPrefix(name, "__imp_")
		name = strings.TrimPrefix(name, "_")
	}
	name = trimStdcallSuffix(name)
	if strings.HasPrefix(name, "_") && (strings.HasPrefix(name[1:], "Beacon") || name[1:] == "toWideChar") {
		name = name[1:]
	}
	return name
}

func trimStdcallSuffix(name string) string {
	separator := strings.LastIndexByte(name, '@')
	if separator <= 0 || separator == len(name)-1 {
		return name
	}
	for _, digit := range name[separator+1:] {
		if digit < '0' || digit > '9' {
			return name
		}
	}
	return name[:separator]
}

func callbackPanic(name string) {
	if recovered := recover(); recovered != nil {
		callbackError(name, "panic: %v", recovered)
	}
}

func callbackError(name, format string, arguments ...any) {
	context := activeExecution.Load()
	if context == nil {
		return
	}
	context.addError(fmt.Errorf("%s: %s", name, fmt.Sprintf(format, arguments...)))
}

//go:nocheckptr
func beaconDataParse(parserAddress, bufferAddress, sizeValue uintptr) (result uintptr) {
	defer callbackPanic("BeaconDataParse")
	if parserAddress == 0 {
		callbackError("BeaconDataParse", "nil parser")
		return 0
	}
	parser := (*beaconDataParser)(unsafe.Pointer(parserAddress))
	*parser = beaconDataParser{}
	size := int64(int32(sizeValue))
	if size == 0 && bufferAddress == 0 {
		return 0
	}
	if bufferAddress == 0 || size < 4 || size > maxCallbackData {
		callbackError("BeaconDataParse", "invalid buffer %#x or size %d", bufferAddress, size)
		return 0
	}
	parser.original = bufferAddress
	parser.buffer = bufferAddress + 4
	parser.length = int32(size - 4)
	parser.size = int32(size - 4)
	return 0
}

func beaconDataInt(parserAddress uintptr) (result uintptr) {
	defer callbackPanic("BeaconDataInt")
	parser, ok := checkedDataParser("BeaconDataInt", parserAddress, 4)
	if !ok {
		return 0
	}
	value := binary.LittleEndian.Uint32(pointerBytes(parser.buffer, 4))
	parser.buffer += 4
	parser.length -= 4
	return uintptr(value)
}

func beaconDataShort(parserAddress uintptr) (result uintptr) {
	defer callbackPanic("BeaconDataShort")
	parser, ok := checkedDataParser("BeaconDataShort", parserAddress, 2)
	if !ok {
		return 0
	}
	value := binary.LittleEndian.Uint16(pointerBytes(parser.buffer, 2))
	parser.buffer += 2
	parser.length -= 2
	return uintptr(value)
}

//go:nocheckptr
func beaconDataLength(parserAddress uintptr) (result uintptr) {
	defer callbackPanic("BeaconDataLength")
	if parserAddress == 0 {
		callbackError("BeaconDataLength", "nil parser")
		return 0
	}
	parser := (*beaconDataParser)(unsafe.Pointer(parserAddress))
	if parser.length < 0 || parser.length > parser.size || parser.size < 0 || parser.size > maxCallbackData {
		callbackError("BeaconDataLength", "invalid parser state")
		return 0
	}
	return uintptr(uint32(parser.length))
}

func beaconDataExtract(parserAddress, sizeAddress uintptr) (result uintptr) {
	defer callbackPanic("BeaconDataExtract")
	result, length, ok := beaconDataExtractField("BeaconDataExtract", parserAddress)
	if !ok {
		return 0
	}
	if sizeAddress != 0 {
		writeInt32(sizeAddress, length)
	}
	return result
}

func beaconDataExtractOrNull(parserAddress, sizeAddress uintptr) (result uintptr) {
	defer callbackPanic("BeaconDataExtractOrNull")
	result, length, ok := beaconDataExtractField("BeaconDataExtractOrNull", parserAddress)
	if !ok {
		return 0
	}
	if sizeAddress != 0 {
		writeInt32(sizeAddress, length)
	}
	if result == 0 || length <= 0 || pointerBytes(result, 1)[0] == 0 {
		return 0
	}
	return result
}

func beaconDataExtractField(name string, parserAddress uintptr) (result uintptr, length int32, ok bool) {
	parser, ok := checkedDataParser(name, parserAddress, 4)
	if !ok {
		return 0, 0, false
	}
	fieldLength := binary.LittleEndian.Uint32(pointerBytes(parser.buffer, 4))
	if fieldLength > uint32(parser.length-4) {
		callbackError(name, "field length %d exceeds %d remaining bytes", fieldLength, parser.length-4)
		return 0, 0, false
	}
	result = parser.buffer + 4
	parser.buffer = result + uintptr(fieldLength)
	parser.length -= int32(fieldLength) + 4
	return result, int32(fieldLength), true
}

//go:nocheckptr
func checkedDataParser(name string, parserAddress uintptr, needed int32) (*beaconDataParser, bool) {
	if parserAddress == 0 {
		callbackError(name, "nil parser")
		return nil, false
	}
	parser := (*beaconDataParser)(unsafe.Pointer(parserAddress))
	if parser.original == 0 || parser.buffer == 0 || parser.size < 0 || parser.size > maxCallbackData || parser.length < needed || parser.length > parser.size {
		callbackError(name, "invalid parser state or insufficient data")
		return nil, false
	}
	consumed := int64(parser.size - parser.length)
	if consumed < 0 || parser.buffer != parser.original+4+uintptr(consumed) {
		callbackError(name, "inconsistent parser cursor")
		return nil, false
	}
	return parser, true
}

//go:nocheckptr
func beaconFormatAlloc(formatAddress, sizeValue uintptr) (result uintptr) {
	defer callbackPanic("BeaconFormatAlloc")
	if formatAddress == 0 {
		callbackError("BeaconFormatAlloc", "nil format")
		return 0
	}
	format := (*beaconFormat)(unsafe.Pointer(formatAddress))
	*format = beaconFormat{}
	size := int64(int32(sizeValue))
	context := activeExecution.Load()
	if size <= 0 || size > maxFormatAllocation || context == nil {
		callbackError("BeaconFormatAlloc", "invalid size %d or no active execution", size)
		return 0
	}
	address, _, err := context.allocate(int(size))
	if err != nil {
		callbackError("BeaconFormatAlloc", "%v", err)
		return 0
	}
	format.original = address
	format.buffer = address
	format.size = int32(size)
	return 0
}

func beaconFormatReset(formatAddress uintptr) (result uintptr) {
	defer callbackPanic("BeaconFormatReset")
	format, data, ok := checkedFormat("BeaconFormatReset", formatAddress)
	if !ok {
		return 0
	}
	clear(data)
	format.buffer = format.original
	format.length = 0
	return 0
}

//go:nocheckptr
func beaconFormatFree(formatAddress uintptr) (result uintptr) {
	defer callbackPanic("BeaconFormatFree")
	if formatAddress == 0 {
		return 0
	}
	format := (*beaconFormat)(unsafe.Pointer(formatAddress))
	if context := activeExecution.Load(); context != nil {
		context.release(format.original)
	}
	*format = beaconFormat{}
	return 0
}

func beaconFormatAppend(formatAddress, textAddress, lengthValue uintptr) (result uintptr) {
	defer callbackPanic("BeaconFormatAppend")
	length := int64(int32(lengthValue))
	if length == 0 {
		return 0
	}
	if textAddress == 0 || length < 0 || length > maxCallbackData {
		callbackError("BeaconFormatAppend", "invalid text %#x or length %d", textAddress, length)
		return 0
	}
	format, data, ok := checkedFormat("BeaconFormatAppend", formatAddress)
	if !ok {
		return 0
	}
	if int64(format.length)+length > int64(len(data)) {
		callbackError("BeaconFormatAppend", "append of %d bytes exceeds %d-byte buffer", length, len(data))
		return 0
	}
	copy(data[int(format.length):], pointerBytes(textAddress, int(length)))
	format.length += int32(length)
	format.buffer = format.original + uintptr(format.length)
	return 0
}

func beaconFormatPrintf(formatAddress, formatStringAddress, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) (result uintptr) {
	defer callbackPanic("BeaconFormatPrintf")
	format, data, ok := checkedFormat("BeaconFormatPrintf", formatAddress)
	if !ok {
		return 0
	}
	formatted, err := formatPrintf(formatStringAddress, [maxPrintfArgument]uintptr{a0, a1, a2, a3, a4, a5, a6, a7, a8, a9})
	if err != nil {
		callbackError("BeaconFormatPrintf", "%v", err)
	}
	if len(formatted) > len(data)-int(format.length) {
		callbackError("BeaconFormatPrintf", "formatted data of %d bytes exceeds remaining capacity %d", len(formatted), len(data)-int(format.length))
		return 0
	}
	copy(data[int(format.length):], formatted)
	format.length += int32(len(formatted))
	format.buffer = format.original + uintptr(format.length)
	return 0
}

func beaconFormatToString(formatAddress, sizeAddress uintptr) (result uintptr) {
	defer callbackPanic("BeaconFormatToString")
	format, _, ok := checkedFormat("BeaconFormatToString", formatAddress)
	if !ok {
		return 0
	}
	if sizeAddress != 0 {
		writeInt32(sizeAddress, format.length)
	}
	return format.original
}

func beaconFormatInt(formatAddress, value uintptr) (result uintptr) {
	defer callbackPanic("BeaconFormatInt")
	format, data, ok := checkedFormat("BeaconFormatInt", formatAddress)
	if !ok {
		return 0
	}
	if len(data)-int(format.length) < 4 {
		callbackError("BeaconFormatInt", "four-byte append exceeds remaining capacity")
		return 0
	}
	binary.BigEndian.PutUint32(data[int(format.length):], uint32(value))
	format.length += 4
	format.buffer = format.original + uintptr(format.length)
	return 0
}

//go:nocheckptr
func checkedFormat(name string, formatAddress uintptr) (*beaconFormat, []byte, bool) {
	if formatAddress == 0 {
		callbackError(name, "nil format")
		return nil, nil, false
	}
	format := (*beaconFormat)(unsafe.Pointer(formatAddress))
	context := activeExecution.Load()
	data, ok := context.allocation(format.original)
	if !ok || format.size != int32(len(data)) || format.length < 0 || int(format.length) > len(data) || format.buffer != format.original+uintptr(format.length) {
		callbackError(name, "invalid or foreign format buffer")
		return nil, nil, false
	}
	return format, data, true
}

func beaconOutput(typeValue, dataAddress, lengthValue uintptr) (result uintptr) {
	defer callbackPanic("BeaconOutput")
	length := int64(int32(lengthValue))
	if length == 0 {
		return 0
	}
	if dataAddress == 0 || length < 0 || length > maxCallbackData {
		callbackError("BeaconOutput", "invalid data %#x or length %d", dataAddress, length)
		return 0
	}
	if context := activeExecution.Load(); context != nil {
		context.appendOutput(int(int32(typeValue)), pointerBytes(dataAddress, int(length)))
	}
	return 0
}

func beaconPrintf(typeValue, formatAddress, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) (result uintptr) {
	defer callbackPanic("BeaconPrintf")
	formatted, err := formatPrintf(formatAddress, [maxPrintfArgument]uintptr{a0, a1, a2, a3, a4, a5, a6, a7, a8, a9})
	if err != nil {
		callbackError("BeaconPrintf", "%v", err)
	}
	if context := activeExecution.Load(); context != nil {
		context.appendOutput(int(int32(typeValue)), []byte(formatted))
	}
	return 0
}

type printfArguments struct {
	values          [maxPrintfArgument]uintptr
	index           int
	wordSize        int
	alignDoubleWord bool
}

func (arguments *printfArguments) next(bits int) (uint64, error) {
	if arguments.wordSize == 4 && bits == 64 && arguments.alignDoubleWord && arguments.index&1 != 0 {
		// AAPCS32 aligns double-word variadic arguments to an even core
		// register or stack slot. BeaconPrintf's two fixed arguments end at
		// r1, so the relative variadic slot index has the same parity as the
		// underlying register/stack slot.
		arguments.index++
	}
	if arguments.index >= len(arguments.values) {
		return 0, errors.New("format string requires more than 10 machine-word arguments")
	}
	low := uint64(arguments.values[arguments.index])
	arguments.index++
	if arguments.wordSize == 4 && bits == 64 {
		if arguments.index >= len(arguments.values) {
			return 0, errors.New("64-bit format argument is missing its high word")
		}
		high := uint64(arguments.values[arguments.index])
		arguments.index++
		return low | high<<32, nil
	}
	return low, nil
}

func formatPrintf(address uintptr, values [maxPrintfArgument]uintptr) (string, error) {
	if address == 0 {
		return "", errors.New("nil format string")
	}
	format, err := readCString(address, maxFormatString)
	if err != nil {
		return "", err
	}
	arguments := printfArguments{
		values:          values,
		wordSize:        pointerSize(),
		alignDoubleWord: runtime.GOARCH == "arm",
	}
	var output strings.Builder
	var formatErrors []error

	for index := 0; index < len(format); {
		if format[index] != '%' {
			start := index
			for index < len(format) && format[index] != '%' {
				index++
			}
			if err := appendBounded(&output, format[start:index]); err != nil {
				return output.String(), err
			}
			continue
		}
		index++
		if index < len(format) && format[index] == '%' {
			if err := appendBounded(&output, "%"); err != nil {
				return output.String(), err
			}
			index++
			continue
		}

		flagsStart := index
		for index < len(format) && strings.ContainsRune("-+ #0'", rune(format[index])) {
			index++
		}
		flags := format[flagsStart:index]
		var width int
		if index < len(format) && format[index] == '*' {
			value, argumentErr := arguments.next(32)
			if argumentErr != nil {
				formatErrors = append(formatErrors, argumentErr)
				value = 0
			}
			width = int(int32(value))
			if width < 0 {
				flags += "-"
				width = -width
			}
			index++
		} else {
			width, index = parseDecimalField(format, index)
		}
		precision := -1
		if index < len(format) && format[index] == '.' {
			index++
			if index < len(format) && format[index] == '*' {
				value, argumentErr := arguments.next(32)
				if argumentErr != nil {
					formatErrors = append(formatErrors, argumentErr)
					value = 0
				}
				precision = int(int32(value))
				if precision < 0 {
					precision = -1
				}
				index++
			} else {
				precision, index = parseDecimalField(format, index)
				if precision < 0 {
					precision = 0
				}
			}
		}
		if width > maxPrintfField || precision > maxPrintfField {
			formatErrors = append(formatErrors, fmt.Errorf("printf field width or precision exceeds %d", maxPrintfField))
			width = minPositive(width, maxPrintfField)
			precision = minPositive(precision, maxPrintfField)
		}

		length := ""
		for _, candidate := range []string{"I64", "I32", "hh", "ll", "h", "l", "j", "z", "t", "L", "w"} {
			if strings.HasPrefix(format[index:], candidate) {
				length = candidate
				index += len(candidate)
				break
			}
		}
		if index >= len(format) {
			formatErrors = append(formatErrors, errors.New("unterminated printf conversion"))
			break
		}
		conversion := format[index]
		index++

		formatted, conversionErr := formatConversion(conversion, length, flags, width, precision, &arguments)
		if conversionErr != nil {
			formatErrors = append(formatErrors, conversionErr)
		}
		if err := appendBounded(&output, formatted); err != nil {
			formatErrors = append(formatErrors, err)
			break
		}
	}
	return output.String(), errors.Join(formatErrors...)
}

func formatConversion(conversion byte, length, flags string, width, precision int, arguments *printfArguments) (string, error) {
	switch conversion {
	case 's', 'S':
		value, err := arguments.next(pointerSize() * 8)
		if err != nil {
			return "", err
		}
		text := "(null)"
		if value != 0 {
			wide := conversion == 'S' || length == "l" || length == "w"
			if precision >= 0 && wide {
				text, err = readWideCStringAtMost(uintptr(value), precision)
			} else if wide {
				text, err = readWideCString(uintptr(value), maxFormatString)
			} else if precision >= 0 {
				text = readCStringAtMost(uintptr(value), precision)
			} else {
				text, err = readCString(uintptr(value), maxFormatString)
			}
			if err != nil {
				return "<invalid-string>", err
			}
		}
		if precision >= 0 && len(text) > precision {
			text = text[:precision]
		}
		return padString(text, flags, width, ' '), nil

	case 'c', 'C':
		value, err := arguments.next(32)
		if err != nil {
			return "", err
		}
		return padString(string(rune(uint32(value))), flags, width, ' '), nil

	case 'd', 'i', 'u', 'o', 'x', 'X':
		bits := integerBits(length)
		value, err := arguments.next(bits)
		if err != nil {
			return "", err
		}
		value = truncateInteger(value, bits)
		goConversion := conversion
		var argument any
		if conversion == 'd' || conversion == 'i' {
			goConversion = 'd'
			argument = signedInteger(value, bits)
		} else {
			if conversion == 'u' {
				goConversion = 'd'
			}
			argument = value
		}
		return fmt.Sprintf(buildGoFormat(flags, width, precision, goConversion), argument), nil

	case 'p':
		value, err := arguments.next(pointerSize() * 8)
		if err != nil {
			return "", err
		}
		text := "0x" + strconv.FormatUint(value, 16)
		pad := byte(' ')
		if strings.Contains(flags, "0") && !strings.Contains(flags, "-") {
			pad = '0'
		}
		return padString(text, flags, width, pad), nil

	case 'f', 'F', 'e', 'E', 'g', 'G', 'a', 'A':
		bits, err := arguments.next(64)
		if err != nil {
			return "", err
		}
		if pointerSize() != 4 && runtime.GOOS != "windows" {
			return "<unsupported-float>", fmt.Errorf("%%%c is unsupported by the fixed uintptr callback on %s/%s", conversion, runtime.GOOS, runtime.GOARCH)
		}
		return fmt.Sprintf(buildGoFormat(flags, width, precision, conversion), math.Float64frombits(bits)), nil

	case 'n':
		_, err := arguments.next(pointerSize() * 8)
		if err != nil {
			return "", err
		}
		return "", errors.New("%n is disabled")

	default:
		return "%" + string(conversion), fmt.Errorf("unsupported printf conversion %%%c", conversion)
	}
}

func integerBits(length string) int {
	switch length {
	case "hh":
		return 8
	case "h":
		return 16
	case "ll", "j", "I64":
		return 64
	case "l":
		if runtime.GOOS != "windows" && pointerSize() == 8 {
			return 64
		}
		return 32
	case "z", "t":
		return pointerSize() * 8
	default:
		return 32
	}
}

func truncateInteger(value uint64, bits int) uint64 {
	if bits >= 64 {
		return value
	}
	return value & (uint64(1)<<bits - 1)
}

func signedInteger(value uint64, bits int) int64 {
	if bits >= 64 {
		return int64(value)
	}
	shift := 64 - bits
	return int64(value<<shift) >> shift
}

func buildGoFormat(flags string, width, precision int, conversion byte) string {
	var spec strings.Builder
	spec.WriteByte('%')
	for _, flag := range flags {
		if strings.ContainsRune("-+ #0", flag) {
			spec.WriteRune(flag)
		}
	}
	if width >= 0 {
		spec.WriteString(strconv.Itoa(width))
	}
	if precision >= 0 {
		spec.WriteByte('.')
		spec.WriteString(strconv.Itoa(precision))
	}
	spec.WriteByte(conversion)
	return spec.String()
}

func parseDecimalField(value string, start int) (int, int) {
	index := start
	parsed := 0
	for index < len(value) && value[index] >= '0' && value[index] <= '9' {
		if parsed <= maxPrintfField {
			parsed = parsed*10 + int(value[index]-'0')
		}
		index++
	}
	if index == start {
		return -1, start
	}
	return parsed, index
}

func minPositive(value, maximum int) int {
	if value < 0 {
		return value
	}
	if value > maximum {
		return maximum
	}
	return value
}

func padString(value, flags string, width int, padding byte) string {
	if width <= len(value) {
		return value
	}
	pad := strings.Repeat(string(padding), width-len(value))
	if strings.Contains(flags, "-") {
		return value + pad
	}
	if padding == '0' && strings.HasPrefix(value, "0x") {
		return "0x" + pad + value[2:]
	}
	return pad + value
}

func appendBounded(builder *strings.Builder, value string) error {
	remaining := maxFormattedData - builder.Len()
	if remaining <= 0 {
		return fmt.Errorf("formatted output exceeds %d bytes", maxFormattedData)
	}
	if len(value) > remaining {
		builder.WriteString(value[:remaining])
		return fmt.Errorf("formatted output exceeds %d bytes", maxFormattedData)
	}
	builder.WriteString(value)
	return nil
}

//go:nocheckptr
func readCString(address uintptr, limit int) (string, error) {
	if address == 0 {
		return "", errors.New("nil C string")
	}
	for length := 0; length < limit; length++ {
		if *(*byte)(unsafe.Pointer(address + uintptr(length))) == 0 {
			return string(pointerBytes(address, length)), nil
		}
	}
	return string(pointerBytes(address, limit)), fmt.Errorf("C string exceeds %d bytes", limit)
}

//go:nocheckptr
func readCStringAtMost(address uintptr, limit int) string {
	if address == 0 || limit <= 0 {
		return ""
	}
	for length := 0; length < limit; length++ {
		if *(*byte)(unsafe.Pointer(address + uintptr(length))) == 0 {
			return string(pointerBytes(address, length))
		}
	}
	return string(pointerBytes(address, limit))
}

//go:nocheckptr
func readWideCString(address uintptr, limit int) (string, error) {
	if address == 0 {
		return "", errors.New("nil wide C string")
	}
	if runtime.GOOS == "windows" {
		units := make([]uint16, 0, minPositive(limit, maxFormatString))
		for index := 0; index < limit; index++ {
			unit := *(*uint16)(unsafe.Pointer(address + uintptr(index*2)))
			if unit == 0 {
				return string(utf16.Decode(units)), nil
			}
			units = append(units, unit)
		}
		return string(utf16.Decode(units)), fmt.Errorf("wide C string exceeds %d code units", limit)
	}

	runes := make([]rune, 0, minPositive(limit, maxFormatString))
	for index := 0; index < limit; index++ {
		value := *(*uint32)(unsafe.Pointer(address + uintptr(index*4)))
		if value == 0 {
			return string(runes), nil
		}
		runes = append(runes, rune(value))
	}
	return string(runes), fmt.Errorf("wide C string exceeds %d code units", limit)
}

func readWideCStringAtMost(address uintptr, byteLimit int) (string, error) {
	if address == 0 || byteLimit <= 0 {
		return "", nil
	}
	text, err := readWideCString(address, minPositive(byteLimit, maxFormatString))
	if err != nil && !strings.Contains(err.Error(), "exceeds") {
		return "", err
	}
	if len(text) > byteLimit {
		end := byteLimit
		for end > 0 && !utf8.RuneStart(text[end]) {
			end--
		}
		text = text[:end]
	}
	return text, nil
}

func byteSliceAddress(data []byte) uintptr {
	if len(data) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(unsafe.SliceData(data)))
}

//go:nocheckptr
func pointerBytes(address uintptr, length int) []byte {
	if length == 0 {
		return nil
	}
	return unsafe.Slice((*byte)(unsafe.Pointer(address)), length)
}

func writeInt32(address uintptr, value int32) {
	binary.LittleEndian.PutUint32(pointerBytes(address, 4), uint32(value))
}
