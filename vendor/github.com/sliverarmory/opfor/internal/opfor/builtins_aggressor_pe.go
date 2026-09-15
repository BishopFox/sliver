package opfor

import (
	"context"
	"encoding/binary"
	"fmt"
	"unicode/utf16"
)

const (
	aggressorPEDOSHeaderSize             = 0x40
	aggressorPEHeaderPointerOffset       = 0x3c
	aggressorPESignatureSize             = 4
	aggressorPECOFFHeaderSize            = 20
	aggressorPECOFFTimeDateStampOffset   = 4
	aggressorPECOFFOptionalSizeOffset    = 16
	aggressorPEOptionalChecksumOffset    = 64
	aggressorPE32MinimumOptionalSize     = 96
	aggressorPE32PlusMinimumOptionalSize = 112
	aggressorPE32Magic                   = 0x10b
	aggressorPE32PlusMagic               = 0x20b
)

// aggressorPEFunctions returns the raw PE mutation helpers implemented by
// OPFOR. Registration is intentionally left to runtimeFunctions so importers
// retain the same override precedence as the other native function families.
func (r *Runtime) aggressorPEFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"pe_mask":                       r.peMask,
		"pe_mask_string":                r.peMaskString,
		"pe_set_compile_time_with_long": r.peSetCompileTimeWithLong,
		"pe_set_long":                   r.peSetLong,
		"pe_set_short":                  r.peSetShort,
		"pe_set_string":                 r.peSetString,
		"pe_set_stringz":                r.peSetStringZ,
		"pe_stomp":                      r.peStomp,
		"pe_update_checksum":            r.peUpdateChecksum,
	}
}

// The official Aggressor documentation specifies successful mutations but
// does not publish the native bridge or define its invalid-input behavior.
// Exact arity, checked no-extension ranges, numeric narrowing, and errors below
// are therefore explicit provisional OPFOR policy rather than licensed-runtime
// parity claims. Every operation works on a fresh low-byte copy of its input.

func (*Runtime) peSetCompileTimeWithLong(ctx context.Context, invocation Invocation) (Value, error) {
	ctx, arguments, err := aggressorPEArguments(ctx, invocation, 2)
	if err != nil {
		return Null(), err
	}
	output, err := aggressorPELowBytes(ctx, arguments[0])
	if err != nil {
		return Null(), err
	}
	layout, err := parseAggressorPELayout(invocation, output)
	if err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	// The public ABI supplies Java milliseconds while the COFF field stores the
	// low 32 bits of whole Unix seconds. Division truncates toward zero and the
	// uint32 conversion deliberately follows Java/Sleep narrowing for dates
	// outside the unsigned 32-bit COFF range; those undocumented edges remain
	// provisional until licensed-runtime differential evidence is available.
	seconds := sleepInt64(arguments[1]) / 1000
	binary.LittleEndian.PutUint32(
		output[layout.timeDateStampOffset:layout.timeDateStampOffset+4],
		uint32(seconds),
	)
	return BinaryString(output), nil
}

func (*Runtime) peUpdateChecksum(ctx context.Context, invocation Invocation) (Value, error) {
	ctx, arguments, err := aggressorPEArguments(ctx, invocation, 1)
	if err != nil {
		return Null(), err
	}
	output, err := aggressorPELowBytes(ctx, arguments[0])
	if err != nil {
		return Null(), err
	}
	layout, err := parseAggressorPELayout(invocation, output)
	if err != nil {
		return Null(), err
	}
	checksum, err := aggressorPEImageChecksum(ctx, output, layout.checksumOffset)
	if err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	binary.LittleEndian.PutUint32(output[layout.checksumOffset:layout.checksumOffset+4], checksum)
	return BinaryString(output), nil
}

func (*Runtime) peMask(ctx context.Context, invocation Invocation) (Value, error) {
	ctx, arguments, err := aggressorPEArguments(ctx, invocation, 4)
	if err != nil {
		return Null(), err
	}

	output, err := aggressorPELowBytes(ctx, arguments[0])
	if err != nil {
		return Null(), err
	}
	length := sleepInt32(arguments[2])
	if length < 0 {
		return Null(), aggressorPEArgumentError(invocation, 3, "length must be non-negative")
	}
	start, end, err := aggressorPECheckedRange(
		invocation,
		len(output),
		sleepInt32(arguments[1]),
		uint64(length),
		3,
	)
	if err != nil {
		return Null(), err
	}

	// XOR is the conservative interpretation of the documented property that
	// applying pe_mask again with the same key restores the original content.
	// Truncating the integer key to its low byte is provisional Sleep-style
	// narrowing for the documented "byte value mask key (int)" argument.
	key := byte(sleepInt32(arguments[3]))
	for position := start; position < end; position++ {
		if (position-start)%aggressorUtilityChunkSize == 0 {
			if err := ctx.Err(); err != nil {
				return Null(), err
			}
		}
		output[position] ^= key
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	return BinaryString(output), nil
}

func (*Runtime) peMaskString(ctx context.Context, invocation Invocation) (Value, error) {
	ctx, arguments, err := aggressorPEArguments(ctx, invocation, 3)
	if err != nil {
		return Null(), err
	}

	output, err := aggressorPELowBytes(ctx, arguments[0])
	if err != nil {
		return Null(), err
	}
	start, err := aggressorPEStringStart(invocation, len(output), sleepInt32(arguments[1]))
	if err != nil {
		return Null(), err
	}
	terminator, err := aggressorPEFindTerminator(ctx, invocation, output, start)
	if err != nil {
		return Null(), err
	}

	// The official pe_mask_string example expressly includes the original NUL
	// terminator in the masked range. Finding it before mutation also guarantees
	// that a missing terminator fails without producing partially changed data.
	key := byte(sleepInt32(arguments[2]))
	end := terminator + 1
	for position := start; position < end; position++ {
		if (position-start)%aggressorUtilityChunkSize == 0 {
			if err := ctx.Err(); err != nil {
				return Null(), err
			}
		}
		output[position] ^= key
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	return BinaryString(output), nil
}

func (*Runtime) peSetLong(ctx context.Context, invocation Invocation) (Value, error) {
	ctx, arguments, err := aggressorPEArguments(ctx, invocation, 3)
	if err != nil {
		return Null(), err
	}

	output, err := aggressorPELowBytes(ctx, arguments[0])
	if err != nil {
		return Null(), err
	}
	start, end, err := aggressorPECheckedRange(
		invocation,
		len(output),
		sleepInt32(arguments[1]),
		4,
		2,
	)
	if err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	// The documented example writes a PE DWORD. Four-byte little endian and
	// modulo-2^32 narrowing are high-confidence PE/Sleep inferences, but remain
	// provisional until compared with a licensed runtime.
	binary.LittleEndian.PutUint32(output[start:end], uint32(sleepInt32(arguments[2])))
	return BinaryString(output), nil
}

func (*Runtime) peSetShort(ctx context.Context, invocation Invocation) (Value, error) {
	ctx, arguments, err := aggressorPEArguments(ctx, invocation, 3)
	if err != nil {
		return Null(), err
	}

	output, err := aggressorPELowBytes(ctx, arguments[0])
	if err != nil {
		return Null(), err
	}
	start, end, err := aggressorPECheckedRange(
		invocation,
		len(output),
		sleepInt32(arguments[1]),
		2,
		2,
	)
	if err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	// The documented example writes a PE WORD. Two-byte little endian and
	// modulo-2^16 narrowing are provisional compatibility interpretations.
	binary.LittleEndian.PutUint16(output[start:end], uint16(sleepInt32(arguments[2])))
	return BinaryString(output), nil
}

func (*Runtime) peSetString(ctx context.Context, invocation Invocation) (Value, error) {
	ctx, arguments, err := aggressorPEArguments(ctx, invocation, 3)
	if err != nil {
		return Null(), err
	}

	output, err := aggressorPELowBytes(ctx, arguments[0])
	if err != nil {
		return Null(), err
	}
	value, err := aggressorPELowBytes(ctx, arguments[2])
	if err != nil {
		return Null(), err
	}
	start, end, err := aggressorPECheckedRange(
		invocation,
		len(output),
		sleepInt32(arguments[1]),
		uint64(len(value)),
		3,
	)
	if err != nil {
		return Null(), err
	}
	if err := aggressorPECopy(ctx, output[start:end], value); err != nil {
		return Null(), err
	}
	return BinaryString(output), nil
}

func (*Runtime) peSetStringZ(ctx context.Context, invocation Invocation) (Value, error) {
	ctx, arguments, err := aggressorPEArguments(ctx, invocation, 3)
	if err != nil {
		return Null(), err
	}

	output, err := aggressorPELowBytes(ctx, arguments[0])
	if err != nil {
		return Null(), err
	}
	value, err := aggressorPELowBytes(ctx, arguments[2])
	if err != nil {
		return Null(), err
	}
	// Writing the complete value, preserving embedded NULs, and then adding
	// exactly one terminator is OPFOR's provisional literal interpretation of
	// "places a string ... and adds a zero terminator."
	width := uint64(len(value)) + 1
	start, end, err := aggressorPECheckedRange(
		invocation,
		len(output),
		sleepInt32(arguments[1]),
		width,
		3,
	)
	if err != nil {
		return Null(), err
	}
	if err := aggressorPECopy(ctx, output[start:end-1], value); err != nil {
		return Null(), err
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	output[end-1] = 0
	return BinaryString(output), nil
}

func (*Runtime) peStomp(ctx context.Context, invocation Invocation) (Value, error) {
	ctx, arguments, err := aggressorPEArguments(ctx, invocation, 2)
	if err != nil {
		return Null(), err
	}

	output, err := aggressorPELowBytes(ctx, arguments[0])
	if err != nil {
		return Null(), err
	}
	start, err := aggressorPEStringStart(invocation, len(output), sleepInt32(arguments[1]))
	if err != nil {
		return Null(), err
	}
	terminator, err := aggressorPEFindTerminator(ctx, invocation, output, start)
	if err != nil {
		return Null(), err
	}

	// The original terminator is already zero, so stopping immediately before
	// it and writing it again are observably equivalent. Missing-terminator
	// rejection is provisional and is completed before this mutation begins.
	for position := start; position < terminator; {
		if err := ctx.Err(); err != nil {
			return Null(), err
		}
		end := position + aggressorUtilityChunkSize
		if end > terminator {
			end = terminator
		}
		clear(output[position:end])
		position = end
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	return BinaryString(output), nil
}

func aggressorPEArguments(
	ctx context.Context,
	invocation Invocation,
	arity int,
) (context.Context, []Value, error) {
	if err := requireAggressorCommandArguments(invocation, arity); err != nil {
		return ctx, nil, err
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return ctx, nil, err
	}
	return ctx, invocation.Values(), nil
}

// aggressorPELowBytes is the cancellation-aware PE mutation boundary for a
// Sleep string. Unlike sleepStringLowBytes, it does not first clone the whole
// UTF-16 unit slice: binary/raw strings are copied directly from their stored
// units, while ordinary text is streamed into UTF-16 code units rune by rune.
// Each path is byte-for-byte equivalent to taking the low eight bits of
// sleepStringUnits. These checks cover source and replacement conversion; the
// shared BinaryString constructor separately materializes the final result.
func aggressorPELowBytes(ctx context.Context, value Value) ([]byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	value = sleepStringCoercion(value)
	if value.stringUnits != nil {
		output := make([]byte, len(value.stringUnits))
		for position, unit := range value.stringUnits {
			// The preflight above is the first chunk check. Check again before
			// starting each subsequent chunk without making an extra unit copy.
			if position != 0 && position%aggressorUtilityChunkSize == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			output[position] = byte(unit)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		return output, nil
	}

	text := value.data.(string)
	output := make([]byte, 0, len(text))
	units := 0
	appendUnit := func(unit uint16) error {
		if units != 0 && units%aggressorUtilityChunkSize == 0 {
			if err := ctx.Err(); err != nil {
				return err
			}
		}
		output = append(output, byte(unit))
		units++
		return nil
	}
	for _, character := range text {
		if character <= 0xffff {
			if err := appendUnit(uint16(character)); err != nil {
				return nil, err
			}
			continue
		}
		first, second := utf16.EncodeRune(character)
		if err := appendUnit(uint16(first)); err != nil {
			return nil, err
		}
		if err := appendUnit(uint16(second)); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return output, nil
}

type aggressorPELayout struct {
	peHeaderOffset       int
	coffHeaderOffset     int
	optionalHeaderOffset int
	timeDateStampOffset  int
	checksumOffset       int
	optionalMagic        uint16
}

// parseAggressorPELayout validates the structural path to the two PE fields
// used by the portable convenience helpers. The public Aggressor reference
// defines only successful Beacon-DLL inputs, so rejecting malformed/truncated
// images with a typed argument error is explicit OPFOR policy.
func parseAggressorPELayout(invocation Invocation, content []byte) (aggressorPELayout, error) {
	fail := func(reason string) (aggressorPELayout, error) {
		return aggressorPELayout{}, aggressorPEArgumentError(invocation, 1, reason)
	}
	if len(content) < aggressorPEDOSHeaderSize {
		return fail(fmt.Sprintf("content is %d bytes; a PE image requires at least a %d-byte DOS header", len(content), aggressorPEDOSHeaderSize))
	}
	if content[0] != 'M' || content[1] != 'Z' {
		return fail("content does not begin with the MZ image signature")
	}

	peOffset := uint64(binary.LittleEndian.Uint32(
		content[aggressorPEHeaderPointerOffset : aggressorPEHeaderPointerOffset+4],
	))
	if peOffset < aggressorPEDOSHeaderSize {
		return fail(fmt.Sprintf("PE header offset %d overlaps the %d-byte DOS header", peOffset, aggressorPEDOSHeaderSize))
	}
	peHeaderWidth := uint64(aggressorPESignatureSize + aggressorPECOFFHeaderSize)
	if peOffset > uint64(len(content)) || peHeaderWidth > uint64(len(content))-peOffset {
		return fail(fmt.Sprintf("PE header offset %d does not identify a complete signature and COFF header", peOffset))
	}
	peHeaderOffset := int(peOffset)
	if content[peHeaderOffset] != 'P' || content[peHeaderOffset+1] != 'E' ||
		content[peHeaderOffset+2] != 0 || content[peHeaderOffset+3] != 0 {
		return fail(fmt.Sprintf("content at PE header offset %d does not contain the PE\\x00\\x00 signature", peOffset))
	}

	coffHeaderOffset := peHeaderOffset + aggressorPESignatureSize
	optionalSizeOffset := coffHeaderOffset + aggressorPECOFFOptionalSizeOffset
	optionalSize := int(binary.LittleEndian.Uint16(content[optionalSizeOffset : optionalSizeOffset+2]))
	optionalHeaderOffset := coffHeaderOffset + aggressorPECOFFHeaderSize
	if optionalSize < 2 {
		return fail(fmt.Sprintf("COFF optional-header size %d cannot contain a PE magic value", optionalSize))
	}
	if optionalHeaderOffset > len(content) || optionalSize > len(content)-optionalHeaderOffset {
		return fail(fmt.Sprintf("declared %d-byte optional header at offset %d exceeds %d-byte content", optionalSize, optionalHeaderOffset, len(content)))
	}
	optionalMagic := binary.LittleEndian.Uint16(content[optionalHeaderOffset : optionalHeaderOffset+2])
	minimumOptionalSize := 0
	switch optionalMagic {
	case aggressorPE32Magic:
		minimumOptionalSize = aggressorPE32MinimumOptionalSize
	case aggressorPE32PlusMagic:
		minimumOptionalSize = aggressorPE32PlusMinimumOptionalSize
	default:
		return fail(fmt.Sprintf("optional header has unsupported magic 0x%x; want PE32 0x10b or PE32+ 0x20b", optionalMagic))
	}
	if optionalSize < minimumOptionalSize {
		return fail(fmt.Sprintf("%#x optional header is %d bytes; format requires at least %d", optionalMagic, optionalSize, minimumOptionalSize))
	}

	return aggressorPELayout{
		peHeaderOffset:       peHeaderOffset,
		coffHeaderOffset:     coffHeaderOffset,
		optionalHeaderOffset: optionalHeaderOffset,
		timeDateStampOffset:  coffHeaderOffset + aggressorPECOFFTimeDateStampOffset,
		checksumOffset:       optionalHeaderOffset + aggressorPEOptionalChecksumOffset,
		optionalMagic:        optionalMagic,
	}, nil
}

// aggressorPEImageChecksum implements the image checksum returned by Windows'
// CheckSumMappedFile: sum little-endian 16-bit words with end-around carry,
// treat the four-byte CheckSum field as zero, then add the complete file size.
// An odd final byte is the low byte of the last word.
func aggressorPEImageChecksum(ctx context.Context, content []byte, checksumOffset int) (uint32, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if checksumOffset < 0 || checksumOffset > len(content) || len(content)-checksumOffset < 4 {
		return 0, fmt.Errorf("opfor: PE checksum offset %d is outside %d-byte content", checksumOffset, len(content))
	}
	var sum uint64
	for position := 0; position < len(content); position += 2 {
		if position%aggressorUtilityChunkSize == 0 {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
		}
		low := content[position]
		if position >= checksumOffset && position < checksumOffset+4 {
			low = 0
		}
		high := byte(0)
		if position+1 < len(content) {
			high = content[position+1]
			if position+1 >= checksumOffset && position+1 < checksumOffset+4 {
				high = 0
			}
		}
		sum += uint64(uint16(low) | uint16(high)<<8)
		sum = (sum & 0xffff) + (sum >> 16)
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	sum = (sum & 0xffff) + (sum >> 16)
	return uint32(sum) + uint32(len(content)), nil
}

// aggressorPECheckedRange validates without adding offset and width until both
// fit within the input. This keeps the check safe from integer overflow and
// deliberately forbids extending the supplied content.
func aggressorPECheckedRange(
	invocation Invocation,
	contentLength int,
	start int32,
	width uint64,
	widthPosition int,
) (int, int, error) {
	if start < 0 {
		return 0, 0, aggressorPEArgumentError(invocation, 2, "offset must be non-negative")
	}
	contentWidth := uint64(contentLength)
	offset := uint64(start)
	if offset > contentWidth {
		return 0, 0, aggressorPEArgumentError(
			invocation,
			2,
			fmt.Sprintf("offset %d exceeds content length %d", start, contentLength),
		)
	}
	available := contentWidth - offset
	if width > available {
		return 0, 0, aggressorPEArgumentError(
			invocation,
			widthPosition,
			fmt.Sprintf("%d-byte mutation exceeds the %d byte(s) available at offset %d", width, available, start),
		)
	}
	return int(offset), int(offset + width), nil
}

func aggressorPEStringStart(invocation Invocation, contentLength int, start int32) (int, error) {
	position, _, err := aggressorPECheckedRange(invocation, contentLength, start, 0, 2)
	if err != nil {
		return 0, err
	}
	if position == contentLength {
		return 0, aggressorPEArgumentError(
			invocation,
			2,
			fmt.Sprintf("offset %d does not identify a byte in content of length %d", start, contentLength),
		)
	}
	return position, nil
}

func aggressorPEFindTerminator(
	ctx context.Context,
	invocation Invocation,
	content []byte,
	start int,
) (int, error) {
	for position := start; position < len(content); position++ {
		if (position-start)%aggressorUtilityChunkSize == 0 {
			if err := ctx.Err(); err != nil {
				return 0, err
			}
		}
		if content[position] == 0 {
			return position, nil
		}
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return 0, aggressorPEArgumentError(
		invocation,
		1,
		fmt.Sprintf("content has no NUL terminator at or after offset %d", start),
	)
}

func aggressorPECopy(ctx context.Context, destination, source []byte) error {
	for position := 0; position < len(source); {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := position + aggressorUtilityChunkSize
		if end > len(source) {
			end = len(source)
		}
		copy(destination[position:end], source[position:end])
		position = end
	}
	return ctx.Err()
}

func aggressorPEArgumentError(invocation Invocation, position int, reason string) error {
	return &PortableUtilityArgumentError{
		Function: invocation.Name,
		Position: position,
		Reason:   reason,
	}
}
