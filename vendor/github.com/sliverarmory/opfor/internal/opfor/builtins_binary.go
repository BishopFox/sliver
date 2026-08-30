package opfor

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha3"
	"crypto/sha512"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"
	"hash"
	"hash/adler32"
	"hash/crc32"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"unicode/utf16"
	"unicode/utf8"
)

// sleepBinaryFunctions returns the pure-Go Sleep binary bridge. Binary-
// producing helpers return Values with byte-sized UTF-16 units and explicit
// raw-byte provenance; text encodings are applied only by text I/O.
func sleepBinaryFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"pack":     builtinSleepPack,
		"unpack":   builtinSleepUnpack,
		"sizeof":   builtinSleepSizeof,
		"digest":   builtinSleepDigest,
		"checksum": builtinSleepChecksum,
	}
}

// aggressorBinaryFunctions returns the client-independent Aggressor binary
// helpers documented outside Sleep's stock bridge namespace.
func aggressorBinaryFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"base64_encode": builtinBase64Encode,
		"base64_decode": builtinBase64Decode,
	}
}

func builtinBase64Encode(_ context.Context, invocation Invocation) (Value, error) {
	return String(base64.StdEncoding.EncodeToString(sleepStringLowBytes(invocation.Arg(0)))), nil
}

func builtinBase64Decode(_ context.Context, invocation Invocation) (Value, error) {
	decoded, err := base64.StdEncoding.DecodeString(invocation.Arg(0).String())
	if err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	return BinaryString(decoded), nil
}

type sleepBinaryPattern struct {
	value byte
	count int
	size  int
	order binary.ByteOrder
}

const sleepFormattedDefaultMarkLimit = 1024 * 10

// sleepFormattedReader is the shared input boundary for unpack and BasicIO's
// bread. It deliberately exposes mark/reset and skip instead of adapting a
// handle by first reading all remaining bytes: formatted reads must leave the
// first byte after the pattern untouched.
type sleepFormattedReader interface {
	io.Reader
	Skip(int64) (int64, error)
	Mark(int) error
	Reset() error
}

type sleepByteFormattedReader struct {
	data     []byte
	position int
	mark     int
}

func (reader *sleepByteFormattedReader) Read(destination []byte) (int, error) {
	if len(destination) == 0 {
		return 0, nil
	}
	if reader == nil || reader.position >= len(reader.data) {
		return 0, io.EOF
	}
	amount := copy(destination, reader.data[reader.position:])
	reader.position += amount
	return amount, nil
}

func (reader *sleepByteFormattedReader) Skip(count int64) (int64, error) {
	if reader == nil || count <= 0 || reader.position >= len(reader.data) {
		return 0, nil
	}
	remaining := len(reader.data) - reader.position
	if count > int64(remaining) {
		count = int64(remaining)
	}
	reader.position += int(count)
	return count, nil
}

func (reader *sleepByteFormattedReader) Mark(_ int) error {
	if reader != nil {
		reader.mark = reader.position
	}
	return nil
}

func (reader *sleepByteFormattedReader) Reset() error {
	if reader != nil {
		reader.position = reader.mark
	}
	return nil
}

// sleepHandleFormattedReader is used only while sleepIOHandle.readMu is held.
// Its reads therefore participate in the handle's existing replay and mark
// accounting without taking a second lock or bypassing buffered input.
type sleepHandleFormattedReader struct {
	ctx    context.Context
	handle *sleepIOHandle
	err    error
}

func (reader *sleepHandleFormattedReader) Read(destination []byte) (int, error) {
	if reader == nil || reader.handle == nil {
		return 0, io.EOF
	}
	amount, err := reader.handle.readBinaryLockedContext(reader.ctx, destination)
	if err != nil && (errors.Is(err, ErrResourceLimit) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)) {
		reader.err = err
	}
	return amount, err
}

func (reader *sleepHandleFormattedReader) Skip(count int64) (int64, error) {
	if reader == nil || reader.handle == nil || count <= 0 {
		return 0, nil
	}
	var buffer [sleepIOReadBufferSize]byte
	var skipped int64
	for skipped < count {
		if reader.ctx != nil {
			if err := reader.ctx.Err(); err != nil {
				return skipped, err
			}
		}
		chunk := buffer[:]
		if remaining := count - skipped; remaining < int64(len(chunk)) {
			chunk = chunk[:int(remaining)]
		}
		amount, err := reader.handle.readBinaryLockedContext(reader.ctx, chunk)
		skipped += int64(amount)
		if err != nil {
			if errors.Is(err, ErrResourceLimit) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				reader.err = err
			}
			return skipped, err
		}
		if amount == 0 {
			return skipped, io.ErrNoProgress
		}
	}
	return skipped, nil
}

func (reader *sleepHandleFormattedReader) Mark(limit int) error {
	if reader == nil || reader.handle == nil {
		return io.ErrClosedPipe
	}
	return reader.handle.markInputLocked(limit)
}

func (reader *sleepHandleFormattedReader) Reset() error {
	if reader == nil || reader.handle == nil {
		return io.ErrClosedPipe
	}
	return reader.handle.resetInputLocked()
}

type sleepFormattedWriteFailure uint8

const (
	sleepFormattedWriteOK sleepFormattedWriteFailure = iota
	sleepFormattedWriteStopped
	sleepFormattedWriteOutput
	sleepFormattedWriteHex
	sleepFormattedWriteSerialization
)

func parseSleepBinaryPattern(format string) ([]sleepBinaryPattern, error) {
	patterns := make([]sleepBinaryPattern, 0, len(format))
	current := -1
	digits := ""

	finishCount := func() error {
		if current < 0 || digits == "" {
			return nil
		}
		count, err := strconv.Atoi(digits)
		if err != nil {
			return fmt.Errorf("invalid binary pattern count %q: %w", digits, err)
		}
		patterns[current].count = count
		digits = ""
		return nil
	}

	for index := 0; index < len(format); index++ {
		character := format[index]
		if sleepASCIILetter(character) {
			if err := finishCount(); err != nil {
				return nil, err
			}
			patterns = append(patterns, sleepBinaryPattern{
				value: character,
				count: sleepBinaryDefaultCount(character),
				size:  sleepBinaryPatternSize(character),
				order: binary.BigEndian,
			})
			current = len(patterns) - 1
			continue
		}
		if current < 0 {
			if character >= '0' && character <= '9' {
				return nil, fmt.Errorf("binary pattern count appears before a format character")
			}
			continue
		}
		switch {
		case character == '*':
			patterns[current].count = -1
		case character == '!':
			patterns[current].order = binary.NativeEndian
		case character == '-':
			patterns[current].order = binary.LittleEndian
		case character == '+':
			patterns[current].order = binary.BigEndian
		case character >= '0' && character <= '9':
			digits += string(character)
		}
	}
	if err := finishCount(); err != nil {
		return nil, err
	}
	return patterns, nil
}

func sleepASCIILetter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func sleepBinaryDefaultCount(value byte) int {
	switch value {
	case 'u', 'U', 'z', 'Z':
		return -1
	default:
		return 1
	}
}

func sleepBinaryPatternSize(value byte) int {
	switch value {
	case 'b', 'B', 'C', 'h', 'H', 'x', 'o', 'z', 'Z':
		return 1
	case 'c', 's', 'S', 'u', 'U':
		return 2
	case 'i', 'I', 'f':
		return 4
	case 'd', 'l':
		return 8
	default:
		return 0
	}
}

func builtinSleepSizeof(ctx context.Context, invocation Invocation) (Value, error) {
	format := invocation.Arg(0).String()
	if format == "" && currentFiber(ctx) != nil {
		return Null(), sleepBridgeNullValue()
	}
	patterns, err := parseSleepBinaryPattern(format)
	if err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	var size int32
	for _, pattern := range patterns {
		if pattern.count > 0 {
			size += int32(pattern.count) * int32(pattern.size)
		}
	}
	return Int(size), nil
}

func builtinSleepPack(ctx context.Context, invocation Invocation) (Value, error) {
	format := invocation.Arg(0).String()
	if format == "" && currentFiber(ctx) != nil {
		return Null(), sleepBridgeNullValue()
	}
	patterns, err := parseSleepBinaryPattern(format)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation,
			fmt.Errorf("&%s: %w", builtinName(invocation.Name), err))
	}
	arguments := invocation.Values()
	if len(arguments) > 0 {
		arguments = arguments[1:]
	}

	var output bytes.Buffer
	writer := newRuntimeOutputWriter(runtimeOutputAccountFor(ctx, invocation.Runtime), &output)
	failure, err := sleepWriteFormatted(ctx, patterns, arguments, writer)
	if err != nil {
		if failure == sleepFormattedWriteHex {
			return Null(), sleepIOBridgeWarning(ctx, invocation,
				fmt.Errorf("&%s: %w", builtinName(invocation.Name), err))
		}
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	return BinaryString(output.Bytes()), nil
}

// sleepWriteFormatted is the common WriteFormatted engine used by pack and
// bwrite. Each field is emitted as it is processed so a handle observes the
// same partial-write and independent-object-stream framing as Sleep 2.1.
func sleepWriteFormatted(ctx context.Context, patterns []sleepBinaryPattern, arguments []Value, writer io.Writer) (sleepFormattedWriteFailure, error) {
	if len(arguments) == 1 {
		if array, ok := arguments[0].Array(); ok {
			cells, snapshotErr := array.snapshotCells()
			if snapshotErr != nil {
				return sleepFormattedWriteOutput, snapshotErr
			}
			arguments = valuesFromCells(cells)
			// BasicIO copies the array iterator into a Java Stack and then
			// consumes it with pop(), reversing this special argument form.
			for left, right := 0, len(arguments)-1; left < right; left, right = left+1, right-1 {
				arguments[left], arguments[right] = arguments[right], arguments[left]
			}
		}
	}

	argumentIndex := 0
	for _, pattern := range patterns {
		if err := ctx.Err(); err != nil {
			return sleepFormattedWriteOutput, err
		}
		switch pattern.value {
		case 'z', 'Z', 'u', 'U':
			value := String("")
			if argumentIndex < len(arguments) {
				value = sleepStringCoercion(arguments[argumentIndex])
				argumentIndex++
			}
			if err := sleepWritePackString(ctx, writer, pattern, value); err != nil {
				return sleepFormattedWriteOutput, err
			}
			continue
		case 'h', 'H':
			value := ""
			if argumentIndex < len(arguments) {
				value = arguments[argumentIndex].String()
				argumentIndex++
			}
			if err := validateSleepPackHex(value); err != nil {
				return sleepFormattedWriteHex, err
			}
			if err := sleepWritePackHex(ctx, writer, pattern.value, value); err != nil {
				return sleepFormattedWriteOutput, err
			}
			continue
		}

		if pattern.value == 'x' && pattern.count < 0 && argumentIndex < len(arguments) {
			return sleepFormattedWriteOutput, fmt.Errorf("unbounded x* pattern would not consume its argument")
		}
		for repetition := 0; (pattern.count < 0 || repetition < pattern.count) && argumentIndex < len(arguments); repetition++ {
			if repetition&0x3fff == 0 {
				if err := ctx.Err(); err != nil {
					return sleepFormattedWriteOutput, err
				}
			}
			value := Null()
			if pattern.value != 'x' {
				value = arguments[argumentIndex]
				argumentIndex++
			}
			packed, complete, err := sleepPackScalar(nil, pattern, value)
			if err != nil {
				if pattern.value == 'o' {
					return sleepFormattedWriteSerialization, err
				}
				return sleepFormattedWriteOutput, err
			}
			if !complete {
				return sleepFormattedWriteStopped, nil
			}
			if err := sleepWriteFormattedBytes(ctx, writer, packed); err != nil {
				return sleepFormattedWriteOutput, err
			}
		}
	}
	return sleepFormattedWriteOK, nil
}

func sleepWriteFormattedBytes(ctx context.Context, writer io.Writer, data []byte) error {
	if writer == nil {
		return io.ErrClosedPipe
	}
	for len(data) != 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		written, err := writer.Write(data)
		if written > 0 {
			data = data[written:]
		}
		if err != nil {
			return err
		}
		if written == 0 {
			return io.ErrNoProgress
		}
	}
	return nil
}

func sleepWritePackString(ctx context.Context, writer io.Writer, pattern sleepBinaryPattern, value Value) error {
	wide := pattern.value == 'u' || pattern.value == 'U'
	units := sleepStringUnits(value)
	var storage [sleepIOReadBufferSize]byte
	chunk := storage[:0]
	flush := func() error {
		if len(chunk) == 0 {
			return nil
		}
		if err := sleepWriteFormattedBytes(ctx, writer, chunk); err != nil {
			return err
		}
		chunk = storage[:0]
		return nil
	}
	appendUnit := func(unit uint16) error {
		width := 1
		if wide {
			width = 2
		}
		if len(chunk)+width > cap(chunk) {
			if err := flush(); err != nil {
				return err
			}
		}
		if wide {
			position := len(chunk)
			chunk = chunk[:position+2]
			pattern.order.PutUint16(chunk[position:], unit)
		} else {
			chunk = append(chunk, byte(unit))
		}
		return nil
	}

	for index, unit := range units {
		if index&0x3fff == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		if err := appendUnit(unit); err != nil {
			return err
		}
	}
	if pattern.count > len(units) && (pattern.value == 'Z' || pattern.value == 'U') {
		for index := len(units); index < pattern.count; index++ {
			if index&0x3fff == 0 {
				if err := ctx.Err(); err != nil {
					return err
				}
			}
			if err := appendUnit(0); err != nil {
				return err
			}
		}
	}
	if pattern.value == 'z' || pattern.value == 'Z' && pattern.count == -1 {
		if err := appendUnit(0); err != nil {
			return err
		}
	} else if pattern.value == 'u' || pattern.value == 'U' && pattern.count == -1 {
		if err := appendUnit(0); err != nil {
			return err
		}
	}
	return flush()
}

func validateSleepPackHex(value string) error {
	if len(value)%2 != 0 {
		return fmt.Errorf("can not pack '%s' as hex string, number of characters must be even", value)
	}
	for index := 0; index < len(value); index++ {
		if _, ok := sleepHexDigit(value[index]); !ok {
			return fmt.Errorf("can not pack '%s' as hex string", value)
		}
	}
	return nil
}

func sleepWritePackHex(ctx context.Context, writer io.Writer, order byte, value string) error {
	var storage [sleepIOReadBufferSize]byte
	for position := 0; position < len(value); {
		if err := ctx.Err(); err != nil {
			return err
		}
		pairs := (len(value) - position) / 2
		if pairs > len(storage) {
			pairs = len(storage)
		}
		chunk := storage[:pairs]
		for index := 0; index < pairs; index++ {
			first, _ := sleepHexDigit(value[position+index*2])
			second, _ := sleepHexDigit(value[position+index*2+1])
			if order == 'h' {
				first, second = second, first
			}
			chunk[index] = first<<4 | second
		}
		if err := sleepWriteFormattedBytes(ctx, writer, chunk); err != nil {
			return err
		}
		position += pairs * 2
	}
	return nil
}

// sleepFormattedUTF16Units models String.toCharArray for u/U fields while
// retaining invalid UTF-8 octets as byte-sized code units. Valid UTF-8 text is
// encoded as Java UTF-16, including surrogate pairs for supplementary runes.
func sleepFormattedUTF16Units(value string) []uint16 {
	units := make([]uint16, 0, len(value))
	for len(value) != 0 {
		// A wide formatted read represents an unpaired Java surrogate as its
		// three-byte WTF-8 spelling inside Go's byte-capable string. Recognize
		// that spelling here so unpack -> pack and bread -> bwrite preserve the
		// otherwise unrepresentable UTF-16 code unit.
		if len(value) >= 3 && value[0] == 0xed && value[1] >= 0xa0 && value[1] <= 0xbf && value[2]&0xc0 == 0x80 {
			unit := uint16(value[0]&0x0f)<<12 | uint16(value[1]&0x3f)<<6 | uint16(value[2]&0x3f)
			units = append(units, unit)
			value = value[3:]
			continue
		}
		character, size := utf8.DecodeRuneInString(value)
		if character == utf8.RuneError && size == 1 {
			units = append(units, uint16(value[0]))
			value = value[1:]
			continue
		}
		if character <= 0xffff {
			units = append(units, uint16(character))
		} else {
			first, second := utf16.EncodeRune(character)
			units = append(units, uint16(first), uint16(second))
		}
		value = value[size:]
	}
	return units
}

func sleepHexDigit(value byte) (byte, bool) {
	switch {
	case value >= '0' && value <= '9':
		return value - '0', true
	case value >= 'a' && value <= 'f':
		return value - 'a' + 10, true
	case value >= 'A' && value <= 'F':
		return value - 'A' + 10, true
	default:
		return 0, false
	}
}

func sleepPackScalar(output []byte, pattern sleepBinaryPattern, value Value) ([]byte, bool, error) {
	var scratch [8]byte
	switch pattern.value {
	case 'x':
		return append(output, 0), true, nil
	case 'C':
		units := sleepStringUnits(value)
		if len(units) == 0 {
			return output, false, nil
		}
		return append(output, byte(units[0])), true, nil
	case 'c':
		units := sleepStringUnits(value)
		if len(units) == 0 {
			return output, false, nil
		}
		pattern.order.PutUint16(scratch[:2], units[0])
		return append(output, scratch[:2]...), true, nil
	case 'b', 'B':
		return append(output, byte(sleepInt32(value))), true, nil
	case 's', 'S':
		pattern.order.PutUint16(scratch[:2], uint16(sleepInt32(value)))
		return append(output, scratch[:2]...), true, nil
	case 'i', 'I':
		pattern.order.PutUint32(scratch[:4], uint32(sleepInt64(value)))
		return append(output, scratch[:4]...), true, nil
	case 'f':
		pattern.order.PutUint32(scratch[:4], math.Float32bits(float32(sleepFloat64(value))))
		return append(output, scratch[:4]...), true, nil
	case 'd':
		pattern.order.PutUint64(scratch[:8], math.Float64bits(sleepFloat64(value)))
		return append(output, scratch[:8]...), true, nil
	case 'l':
		pattern.order.PutUint64(scratch[:8], uint64(sleepInt64(value)))
		return append(output, scratch[:8]...), true, nil
	case 'o':
		serialized, err := encodeSleepScalarStream(value)
		if err != nil {
			return nil, false, err
		}
		return append(output, serialized...), true, nil
	default:
		// BasicIO silently consumes a scalar for unknown write patterns.
		return output, true, nil
	}
}

func builtinSleepUnpack(ctx context.Context, invocation Invocation) (Value, error) {
	patterns, err := parseSleepBinaryPattern(invocation.Arg(0).String())
	if err != nil {
		// BasicIO.unpack places both byte conversion and ReadFormatted (which
		// parses the pattern) inside a catch-all and returns an empty array on
		// every failure. In particular, malformed pattern counts are not fatal
		// bridge errors.
		return ArrayValue(NewArray()), nil
	}
	reader := &sleepByteFormattedReader{data: sleepStringLowBytes(invocation.Arg(1))}
	values, _, err := sleepReadFormatted(ctx, patterns, reader, serializationReceivingScript(ctx, invocation), invocation.Runtime)
	if err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	return ArrayValue(NewArray(values...)), nil
}

// bread mirrors BasicIO.bread's arity-sensitive console selection. A closed
// handle returns the empty scalar; an open handle always returns an array, and
// any formatted read failure closes the IOObject after preserving completed
// values in that array.
func (state *ioBuiltinState) bread(ctx context.Context, invocation Invocation) (Value, error) {
	handle, formatIndex, err := state.chooseHandle(invocation, 2)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation, err)
	}
	format := ""
	if formatIndex < len(invocation.Arguments) {
		format = invocation.Arg(formatIndex).String()
	}
	if format == "" && currentFiber(ctx) != nil {
		return Null(), sleepBridgeNullValue()
	}
	patterns, err := parseSleepBinaryPattern(format)
	if err != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}

	handle.readMu.Lock()
	handle.mu.Lock()
	open := handle.reader != nil
	handle.mu.Unlock()
	if !open {
		handle.readMu.Unlock()
		return Null(), nil
	}
	reader := &sleepHandleFormattedReader{ctx: ctx, handle: handle}
	values, stopped, readErr := sleepReadFormatted(ctx, patterns, reader, state.serializationScript(ctx, invocation), invocation.Runtime)
	if stopped {
		_ = state.closeFormattedHandle(handle)
	}
	handle.readMu.Unlock()
	if reader.err != nil {
		return Null(), reader.err
	}
	if readErr != nil {
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), readErr)
	}
	return ArrayValue(NewArray(values...)), nil
}

// bwrite shares pack's writer engine while retaining BasicIO's stream error
// policy: format errors escape the bridge, object serialization errors enter
// checkError, and ordinary output failures close the handle and are swallowed.
func (state *ioBuiltinState) bwrite(ctx context.Context, invocation Invocation) (Value, error) {
	handle, formatIndex, err := state.chooseHandle(invocation, 3)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation, err)
	}
	format := ""
	if formatIndex < len(invocation.Arguments) {
		format = invocation.Arg(formatIndex).String()
		formatIndex++
	}
	if format == "" && currentFiber(ctx) != nil {
		return Null(), sleepBridgeNullValue()
	}
	patterns, err := parseSleepBinaryPattern(format)
	if err != nil {
		return Null(), sleepIOBridgeWarning(ctx, invocation,
			fmt.Errorf("&%s: %w", builtinName(invocation.Name), err))
	}
	arguments := invocation.Values()
	if formatIndex < len(arguments) {
		arguments = arguments[formatIndex:]
	} else {
		arguments = nil
	}

	failure, writeErr := sleepWriteFormatted(ctx, patterns, arguments, handle)
	switch failure {
	case sleepFormattedWriteOK:
		_ = handle.flushFormattedWrite()
		return Null(), nil
	case sleepFormattedWriteHex:
		_ = state.closeFormattedHandle(handle)
		return Null(), sleepIOBridgeWarning(ctx, invocation,
			fmt.Errorf("&%s: %w", builtinName(invocation.Name), writeErr))
	case sleepFormattedWriteSerialization:
		return state.flagSerializationError(ctx, invocation, handle, writeErr)
	default:
		_ = state.closeFormattedHandle(handle)
		if errors.Is(writeErr, ErrResourceLimit) {
			return Null(), writeErr
		}
		if writeErr != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return Null(), ctxErr
			}
		}
		return Null(), nil
	}
}

func (state *ioBuiltinState) closeFormattedHandle(handle *sleepIOHandle) error {
	if handle == nil {
		return nil
	}
	if process := handle.getProcess(); process != nil {
		return process.close()
	}
	return handle.close()
}

func (handle *sleepIOHandle) flushFormattedWrite() error {
	if handle == nil {
		return io.ErrClosedPipe
	}
	handle.writeMu.Lock()
	defer handle.writeMu.Unlock()
	handle.mu.Lock()
	writer := handle.writer
	handle.mu.Unlock()
	return flushWriter(writer)
}

// sleepReadFormatted is the common ReadFormatted engine used by unpack and
// bread. A read failure returns the values completed so far and reports
// stopped=true; callers with a live IOObject then close that handle, while
// unpack simply returns the partial array.
func sleepReadFormatted(ctx context.Context, patterns []sleepBinaryPattern, reader sleepFormattedReader, script *Script, runtime *Runtime) ([]Value, bool, error) {
	values := make([]Value, 0)
	appendValue := func(value Value) error {
		if err := reserveCollectionEntries(runtime, 1); err != nil {
			return err
		}
		values = append(values, value)
		return nil
	}
	for _, pattern := range patterns {
		if err := ctx.Err(); err != nil {
			return nil, false, err
		}
		switch pattern.value {
		case 'M':
			limit := pattern.count
			if limit == 1 {
				limit = sleepFormattedDefaultMarkLimit
			}
			if err := reader.Mark(limit); err != nil {
				return values, true, nil
			}
			continue
		case 'x':
			// BasicIO suppresses exceptions from InputStream.skip and does not
			// require the requested count to be reached.
			_, _ = reader.Skip(int64(pattern.count))
			if err := ctx.Err(); err != nil {
				return nil, false, err
			}
			continue
		case 'h', 'H':
			var text strings.Builder
			for repetition := 0; pattern.count < 0 || repetition < pattern.count; repetition++ {
				if repetition&0x3fff == 0 {
					if err := ctx.Err(); err != nil {
						return nil, false, err
					}
				}
				chunk, complete := sleepReadFormattedBytes(reader, 1)
				if !complete {
					if err := appendValue(String(text.String())); err != nil {
						return nil, false, err
					}
					return values, true, nil
				}
				octet := chunk[0]
				high := "0123456789abcdef"[octet>>4]
				low := "0123456789abcdef"[octet&0x0f]
				if pattern.value == 'h' {
					text.WriteByte(low)
					text.WriteByte(high)
				} else {
					text.WriteByte(high)
					text.WriteByte(low)
				}
			}
			if err := appendValue(String(text.String())); err != nil {
				return nil, false, err
			}
			continue
		case 'z', 'Z', 'u', 'U':
			text, exhausted, readErr := sleepReadFormattedString(ctx, reader, pattern)
			if readErr != nil {
				return nil, false, readErr
			}
			if err := appendValue(text); err != nil {
				return nil, false, err
			}
			if exhausted {
				return values, true, nil
			}
			continue
		}

		if (pattern.value == 'R' || sleepBinaryPatternSize(pattern.value) == 0) && pattern.count < 0 {
			return nil, false, fmt.Errorf("unbounded %c* pattern would not consume input", pattern.value)
		}
		for repetition := 0; pattern.count < 0 || repetition < pattern.count; repetition++ {
			if repetition&0x3fff == 0 {
				if err := ctx.Err(); err != nil {
					return nil, false, err
				}
			}
			if pattern.value == 'R' {
				if err := reader.Reset(); err != nil {
					return values, true, nil
				}
				continue
			}
			if pattern.value == 'o' {
				value, consumed, decodeErr := decodeSleepScalarStreamForScript(reader, script)
				if decodeErr != nil || consumed <= 0 {
					return values, true, nil
				}
				if err := appendValue(value); err != nil {
					return nil, false, err
				}
				continue
			}
			width := sleepBinaryPatternSize(pattern.value)
			chunk, complete := sleepReadFormattedBytes(reader, width)
			if !complete {
				return values, true, nil
			}
			value, _, ok, decodeErr := sleepUnpackScalar(chunk, 0, pattern)
			if decodeErr != nil {
				return nil, false, decodeErr
			}
			if !ok {
				return values, true, nil
			}
			if !value.IsNull() {
				if err := appendValue(value); err != nil {
					return nil, false, err
				}
			}
		}
	}
	return values, false, nil
}

func sleepReadFormattedBytes(reader io.Reader, count int) ([]byte, bool) {
	if count == 0 {
		return nil, true
	}
	data := make([]byte, count)
	read, err := io.ReadFull(reader, data)
	return data[:read], err == nil
}

func sleepReadFormattedString(ctx context.Context, reader sleepFormattedReader, pattern sleepBinaryPattern) (Value, bool, error) {
	wide := pattern.value == 'u' || pattern.value == 'U'
	unitSize := 1
	if wide {
		unitSize = 2
	}
	readUnit := func() (uint16, bool) {
		chunk, complete := sleepReadFormattedBytes(reader, unitSize)
		if !complete {
			return 0, false
		}
		if wide {
			return pattern.order.Uint16(chunk), true
		}
		return uint16(chunk[0]), true
	}

	current, ok := readUnit()
	if !ok {
		return String(""), true, nil
	}
	var units []uint16
	count := 1
	for current != 0 && count != pattern.count {
		if count&0x3fff == 0 {
			if err := ctx.Err(); err != nil {
				return Null(), false, err
			}
		}
		units = append(units, current)
		current, ok = readUnit()
		if !ok {
			if wide {
				return sleepStringValueFromUnits(units, nil), true, nil
			}
			return BinaryString(uint16UnitsToBytes(units)), true, nil
		}
		count++
	}
	if current != 0 {
		if wide {
			units = append(units, current)
		} else {
			units = append(units, current)
		}
	}
	if (pattern.value == 'Z' || pattern.value == 'U') && count < pattern.count {
		_, _ = reader.Skip(int64(pattern.count-count) * int64(unitSize))
	}
	if wide {
		return sleepStringValueFromUnits(units, nil), false, nil
	}
	return BinaryString(uint16UnitsToBytes(units)), false, nil
}

func uint16UnitsToBytes(units []uint16) []byte {
	result := make([]byte, len(units))
	for index, unit := range units {
		result[index] = byte(unit)
	}
	return result
}

func sleepUnpackScalar(data []byte, position int, pattern sleepBinaryPattern) (Value, int, bool, error) {
	width := sleepBinaryPatternSize(pattern.value)
	if width == 0 {
		return Null(), position, true, nil
	}
	if position+width > len(data) {
		return Null(), position, false, nil
	}
	chunk := data[position : position+width]
	next := position + width
	switch pattern.value {
	case 'C':
		return BinaryString(chunk[:1]), next, true, nil
	case 'c':
		return sleepUTF16CharacterValue(pattern.order.Uint16(chunk)), next, true, nil
	case 'b':
		// Sleep 2.1 casts InputStream.read() to byte before checking for EOF,
		// making an actual 0xff indistinguishable from -1 in this format.
		value := int8(chunk[0])
		if value == -1 {
			return Null(), next, false, nil
		}
		return Int(int32(value)), next, true, nil
	case 'B':
		return Int(int32(chunk[0])), next, true, nil
	case 's':
		return Int(int32(int16(pattern.order.Uint16(chunk)))), next, true, nil
	case 'S':
		return Int(int32(pattern.order.Uint16(chunk))), next, true, nil
	case 'i':
		return Int(int32(pattern.order.Uint32(chunk))), next, true, nil
	case 'I':
		return Long(int64(pattern.order.Uint32(chunk))), next, true, nil
	case 'f':
		return Double(float64(math.Float32frombits(pattern.order.Uint32(chunk)))), next, true, nil
	case 'd':
		return Double(math.Float64frombits(pattern.order.Uint64(chunk))), next, true, nil
	case 'l':
		return Long(int64(pattern.order.Uint64(chunk))), next, true, nil
	default:
		return Null(), next, true, nil
	}
}

func builtinSleepDigest(ctx context.Context, invocation Invocation) (Value, error) {
	if object, ok := invocation.Arg(0).Object(); ok {
		switch object := object.(type) {
		case *sleepDigestState:
			return BinaryString(object.sumAndReset()), nil
		case *sleepIOHandle:
			algorithm := sleepBinaryAlgorithm(invocation, "MD5")
			write, algorithm, err := sleepStreamDirection(algorithm)
			if err != nil {
				return Null(), sleepBinaryEmptyAlgorithmWarning(ctx, invocation, err)
			}
			digest, err := sleepMessageDigest(algorithm)
			if err != nil {
				return sleepDigestAlgorithmError(ctx, invocation, algorithm)
			}
			state := &sleepDigestState{digest: digest}
			if write {
				err = object.wrapWrite(func(writer io.Writer) io.Writer {
					return &sleepDigestWriter{writer: writer, state: state}
				})
			} else {
				err = object.wrapRead(func(reader io.Reader) io.Reader {
					return &sleepDigestReader{reader: reader, state: state}
				})
			}
			if err != nil {
				return Null(), fmt.Errorf("&%s: wrap stream: %w", builtinName(invocation.Name), err)
			}
			return ObjectValue(state), nil
		}
	}

	algorithm := sleepBinaryAlgorithm(invocation, "MD5")
	digest, err := sleepMessageDigest(algorithm)
	if err != nil {
		return sleepDigestAlgorithmError(ctx, invocation, algorithm)
	}
	_, _ = digest.Write(sleepStringLowBytes(invocation.Arg(0)))
	return BinaryString(digest.Sum(nil)), nil
}

func sleepDigestAlgorithmError(ctx context.Context, invocation Invocation, algorithm string) (Value, error) {
	err := fmt.Errorf("java.security.NoSuchAlgorithmException: %s MessageDigest not available", algorithm)
	// BasicIO catches NoSuchAlgorithmException and calls flagError. That is a
	// checkError soft failure: the current block continues after digest(), and
	// debug(2) reports it immediately.
	if currentFiber(ctx) != nil && invocation.Runtime != nil {
		return invocation.Runtime.flagSourceError(invocation, err)
	}
	return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
}

func sleepMessageDigest(algorithm string) (hash.Hash, error) {
	// BasicIO delegates digest names to MessageDigest.getInstance. The pinned
	// OpenJDK provider accepts the two truncated SHA-512 spellings both with
	// and without the first hyphen, while SHA-3 requires its canonical hyphen.
	// All are case-insensitive. Keep these cases ahead of the older compact
	// alias normalization so unsupported spellings such as SHA3224 remain a
	// source-visible NoSuchAlgorithmException, as they are in the reference.
	switch strings.ToUpper(algorithm) {
	case "SHA-512/224", "SHA512/224":
		return sha512.New512_224(), nil
	case "SHA-512/256", "SHA512/256":
		return sha512.New512_256(), nil
	case "SHA3-224":
		return sha3.New224(), nil
	case "SHA3-256":
		return sha3.New256(), nil
	case "SHA3-384":
		return sha3.New384(), nil
	case "SHA3-512":
		return sha3.New512(), nil
	}

	normalized := strings.ToUpper(strings.ReplaceAll(algorithm, "-", ""))
	switch normalized {
	case "MD2":
		return newSleepMD2(), nil
	case "MD5":
		return md5.New(), nil
	case "SHA", "SHA1":
		return sha1.New(), nil
	case "SHA224":
		return sha256.New224(), nil
	case "SHA256":
		return sha256.New(), nil
	case "SHA384":
		return sha512.New384(), nil
	case "SHA512":
		return sha512.New(), nil
	default:
		return nil, fmt.Errorf("unsupported digest algorithm %q", algorithm)
	}
}

func builtinSleepChecksum(ctx context.Context, invocation Invocation) (Value, error) {
	if object, ok := invocation.Arg(0).Object(); ok {
		switch object := object.(type) {
		case *sleepChecksumState:
			return Long(int64(object.value())), nil
		case *sleepIOHandle:
			algorithm := sleepBinaryAlgorithm(invocation, "CRC32")
			write, algorithm, err := sleepStreamDirection(algorithm)
			if err != nil {
				return Null(), sleepBinaryEmptyAlgorithmWarning(ctx, invocation, err)
			}
			checksum, err := sleepChecksumHash(algorithm)
			if err != nil {
				return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
			}
			state := &sleepChecksumState{checksum: checksum}
			if write {
				err = object.wrapWrite(func(writer io.Writer) io.Writer {
					return &sleepChecksumWriter{writer: writer, state: state}
				})
			} else {
				err = object.wrapRead(func(reader io.Reader) io.Reader {
					return &sleepChecksumReader{reader: reader, state: state}
				})
			}
			if err != nil {
				return Null(), fmt.Errorf("&%s: wrap stream: %w", builtinName(invocation.Name), err)
			}
			return ObjectValue(state), nil
		}
	}

	algorithm := sleepBinaryAlgorithm(invocation, "CRC32")
	checksum, err := sleepChecksumHash(algorithm)
	if err != nil {
		if currentFiber(ctx) != nil {
			// getChecksum returns null for an unknown spelling; the direct
			// update then raises the NullPointerException which Sleep reports as
			// its canonical null-value bridge warning.
			return Null(), sleepBridgeNullValue()
		}
		return Null(), fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
	}
	_, _ = checksum.Write(sleepStringLowBytes(invocation.Arg(0)))
	return Long(int64(checksum.Sum32())), nil
}

func sleepBinaryAlgorithm(invocation Invocation, fallback string) string {
	if len(invocation.Arguments) > 1 {
		return invocation.Arg(1).String()
	}
	return fallback
}

func sleepStreamDirection(algorithm string) (write bool, normalized string, err error) {
	if algorithm == "" {
		return false, "", fmt.Errorf("digest or checksum algorithm is empty")
	}
	if algorithm[0] == '>' {
		return true, algorithm[1:], nil
	}
	return false, algorithm, nil
}

func sleepBinaryEmptyAlgorithmWarning(ctx context.Context, invocation Invocation, err error) error {
	if currentFiber(ctx) != nil {
		// BasicIO calls String.charAt(0) before entering its algorithm-specific
		// catch block. Sleep normalizes that exception as an invalid-index
		// warning and aborts only the active block.
		return sleepBridgeInvalidIndex("Index 0 out of bounds for length 0")
	}
	return fmt.Errorf("&%s: %w", builtinName(invocation.Name), err)
}

func sleepChecksumHash(algorithm string) (hash.Hash32, error) {
	// BasicIO.getChecksum compares these names literally rather than through a
	// provider registry. Unlike MessageDigest, checksum spellings are therefore
	// case-sensitive.
	switch algorithm {
	case "CRC32":
		return crc32.NewIEEE(), nil
	case "Adler32":
		return adler32.New(), nil
	default:
		return nil, fmt.Errorf("unsupported checksum algorithm %q", algorithm)
	}
}

type sleepDigestState struct {
	mu     sync.Mutex
	digest hash.Hash
}

func (*sleepDigestState) String() string { return "<message-digest>" }

func (state *sleepDigestState) write(data []byte) {
	state.mu.Lock()
	defer state.mu.Unlock()
	_, _ = state.digest.Write(data)
}

func (state *sleepDigestState) sumAndReset() []byte {
	state.mu.Lock()
	defer state.mu.Unlock()
	result := state.digest.Sum(nil)
	state.digest.Reset()
	return result
}

type sleepChecksumState struct {
	mu       sync.Mutex
	checksum hash.Hash32
}

func (*sleepChecksumState) String() string { return "<checksum>" }

func (state *sleepChecksumState) write(data []byte) {
	state.mu.Lock()
	defer state.mu.Unlock()
	_, _ = state.checksum.Write(data)
}

func (state *sleepChecksumState) value() uint32 {
	state.mu.Lock()
	defer state.mu.Unlock()
	return state.checksum.Sum32()
}

type sleepDigestReader struct {
	reader io.Reader
	state  *sleepDigestState
}

func (reader *sleepDigestReader) Read(data []byte) (int, error) {
	read, err := reader.reader.Read(data)
	if read > 0 {
		reader.state.write(data[:read])
	}
	return read, err
}

func (reader *sleepDigestReader) sleepUnderlyingReader() io.Reader { return reader.reader }

type sleepChecksumReader struct {
	reader io.Reader
	state  *sleepChecksumState
}

func (reader *sleepChecksumReader) Read(data []byte) (int, error) {
	read, err := reader.reader.Read(data)
	if read > 0 {
		reader.state.write(data[:read])
	}
	return read, err
}

func (reader *sleepChecksumReader) sleepUnderlyingReader() io.Reader { return reader.reader }

type sleepDigestWriter struct {
	writer io.Writer
	state  *sleepDigestState
}

func (writer *sleepDigestWriter) Write(data []byte) (int, error) {
	written, err := writer.writer.Write(data)
	if written > 0 {
		writer.state.write(data[:written])
	}
	return written, err
}

func (writer *sleepDigestWriter) Flush() error { return flushWriter(writer.writer) }

func (writer *sleepDigestWriter) runtimeOutputAccount() *runtimeResourceAccount {
	if writer == nil {
		return nil
	}
	return runtimeOutputAccountOf(writer.writer)
}

type sleepChecksumWriter struct {
	writer io.Writer
	state  *sleepChecksumState
}

func (writer *sleepChecksumWriter) Write(data []byte) (int, error) {
	written, err := writer.writer.Write(data)
	if written > 0 {
		writer.state.write(data[:written])
	}
	return written, err
}

func (writer *sleepChecksumWriter) Flush() error { return flushWriter(writer.writer) }

func (writer *sleepChecksumWriter) runtimeOutputAccount() *runtimeResourceAccount {
	if writer == nil {
		return nil
	}
	return runtimeOutputAccountOf(writer.writer)
}
