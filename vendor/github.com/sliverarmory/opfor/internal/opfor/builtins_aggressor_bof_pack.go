package opfor

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
)

const bofPackMaximumFieldLength = uint64(^uint32(0))

// builtinAggressorBOFPack implements the documented BOF argument buffer
// formats. Each length belongs to its immediately following field; the
// complete argument buffer has no outer length prefix.
func (r *Runtime) builtinAggressorBOFPack(ctx context.Context, invocation Invocation) (Value, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return Null(), err
	}

	arguments := invocation.Values()
	if len(arguments) < 2 {
		return Null(), &PortableUtilityArgumentError{
			Function: invocation.Name,
			Reason: fmt.Sprintf(
				"requires a beacon ID and format, received %d argument(s)",
				len(arguments),
			),
		}
	}

	formats := sleepStringUnits(arguments[1])
	for index, format := range formats {
		if !bofPackFormatSupported(format) {
			return Null(), &PortableUtilityArgumentError{
				Function: invocation.Name,
				Position: 2,
				Reason: fmt.Sprintf(
					"format character %d (%q) is unsupported; expected one of b, i, s, z, or Z",
					index+1,
					rune(format),
				),
			}
		}
	}

	valueCount := len(arguments) - 2
	if valueCount != len(formats) {
		return Null(), &PortableUtilityArgumentError{
			Function: invocation.Name,
			Reason: fmt.Sprintf(
				"format requires exactly %d value argument(s), received %d",
				len(formats),
				valueCount,
			),
		}
	}

	beaconID := arguments[0]
	byteOrder := r.bofPackBinaryByteOrder()
	var output bytes.Buffer
	writer := newRuntimeOutputWriter(runtimeOutputAccountFor(ctx, r), &output)
	for index, format := range formats {
		if err := ctx.Err(); err != nil {
			return Null(), err
		}

		value := arguments[index+2]
		argumentPosition := index + 3
		switch format {
		case 'b':
			units := sleepStringUnits(value)
			length, err := bofPackFieldLength(
				invocation.Name,
				argumentPosition,
				format,
				uint64(len(units)),
				0,
			)
			if err != nil {
				return Null(), err
			}
			var header [4]byte
			byteOrder.PutUint32(header[:], length)
			if err := sleepWriteFormattedBytes(ctx, writer, header[:]); err != nil {
				return Null(), err
			}
			for position := 0; position < len(units); {
				end := position + aggressorUtilityChunkSize
				if end > len(units) {
					end = len(units)
				}
				chunk := make([]byte, end-position)
				for offset, unit := range units[position:end] {
					chunk[offset] = byte(unit)
				}
				if err := sleepWriteFormattedBytes(ctx, writer, chunk); err != nil {
					return Null(), err
				}
				position = end
			}

		case 'i':
			var data [4]byte
			byteOrder.PutUint32(data[:], uint32(sleepInt32(value)))
			if err := sleepWriteFormattedBytes(ctx, writer, data[:]); err != nil {
				return Null(), err
			}

		case 's':
			var data [2]byte
			byteOrder.PutUint16(data[:], uint16(sleepInt32(value)))
			if err := sleepWriteFormattedBytes(ctx, writer, data[:]); err != nil {
				return Null(), err
			}

		case 'z':
			encoder := BeaconStringEncoder(utf8BeaconStringEncoder{})
			importerEncoder := false
			if r != nil && !isNilInterface(r.beaconEncoder) {
				encoder = r.beaconEncoder
				_, stock := encoder.(utf8BeaconStringEncoder)
				importerEncoder = !stock
			}
			encoded, err := encoder.EncodeBeaconString(ctx, beaconID, value)
			if err != nil {
				if importerEncoder {
					err = preserveNativeBoundaryError(ctx, err)
				}
				return Null(), fmt.Errorf(
					"&%s: encode argument %d as a z string: %w",
					builtinName(invocation.Name),
					argumentPosition,
					err,
				)
			}
			if err := ctx.Err(); err != nil {
				return Null(), err
			}
			encoded, err = bofPackNarrowCStringPayload(ctx, encoded)
			if err != nil {
				return Null(), err
			}
			length, err := bofPackFieldLength(
				invocation.Name,
				argumentPosition,
				format,
				uint64(len(encoded)),
				1,
			)
			if err != nil {
				return Null(), err
			}
			var header [4]byte
			byteOrder.PutUint32(header[:], length)
			if err := sleepWriteFormattedBytes(ctx, writer, header[:]); err != nil {
				return Null(), err
			}
			if err := bofPackWriteBytes(ctx, writer, encoded); err != nil {
				return Null(), err
			}
			if err := sleepWriteFormattedBytes(ctx, writer, []byte{0}); err != nil {
				return Null(), err
			}

		case 'Z':
			units := sleepStringUnits(value)
			units, err := bofPackWideCStringPayload(ctx, units)
			if err != nil {
				return Null(), err
			}
			length, err := bofPackFieldLength(
				invocation.Name,
				argumentPosition,
				format,
				uint64(len(units))*2,
				2,
			)
			if err != nil {
				return Null(), err
			}
			var header [4]byte
			byteOrder.PutUint32(header[:], length)
			if err := sleepWriteFormattedBytes(ctx, writer, header[:]); err != nil {
				return Null(), err
			}
			const unitsPerChunk = aggressorUtilityChunkSize / 2
			for position := 0; position < len(units); {
				end := position + unitsPerChunk
				if end > len(units) {
					end = len(units)
				}
				chunk := make([]byte, (end-position)*2)
				for offset, unit := range units[position:end] {
					binary.LittleEndian.PutUint16(chunk[offset*2:], unit)
				}
				if err := sleepWriteFormattedBytes(ctx, writer, chunk); err != nil {
					return Null(), err
				}
				position = end
			}
			if err := sleepWriteFormattedBytes(ctx, writer, []byte{0, 0}); err != nil {
				return Null(), err
			}
		}
	}

	if err := ctx.Err(); err != nil {
		return Null(), err
	}
	return BinaryString(output.Bytes()), nil
}

func (r *Runtime) bofPackBinaryByteOrder() binary.ByteOrder {
	if r != nil && r.bofPackByteOrder == BOFPackLittleEndian {
		return binary.LittleEndian
	}
	return binary.BigEndian
}

// The official public BofData packer accepts C strings and measures them with
// strlen/wcslen. Keep the encoded and wide formats on that same first-NUL
// boundary even though their length-prefixed wire representation could carry
// inaccessible trailing data. Raw b fields deliberately do not use these
// helpers.
func bofPackNarrowCStringPayload(ctx context.Context, input []byte) ([]byte, error) {
	for position, octet := range input {
		if position%aggressorUtilityChunkSize == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		if octet == 0 {
			return input[:position], nil
		}
	}
	return input, nil
}

func bofPackWideCStringPayload(ctx context.Context, input []uint16) ([]uint16, error) {
	for position, unit := range input {
		if position%aggressorUtilityChunkSize == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		if unit == 0 {
			return input[:position], nil
		}
	}
	return input, nil
}

func bofPackFormatSupported(format uint16) bool {
	switch format {
	case 'b', 'i', 's', 'z', 'Z':
		return true
	default:
		return false
	}
}

func bofPackFieldLength(
	function string,
	argumentPosition int,
	format uint16,
	payloadLength uint64,
	terminatorLength uint64,
) (uint32, error) {
	if terminatorLength > bofPackMaximumFieldLength ||
		payloadLength > bofPackMaximumFieldLength-terminatorLength {
		return 0, &PortableUtilityArgumentError{
			Function: function,
			Position: argumentPosition,
			Reason: fmt.Sprintf(
				"%c field byte length exceeds the uint32 maximum of %d (%d payload byte(s), %d terminator byte(s))",
				format,
				bofPackMaximumFieldLength,
				payloadLength,
				terminatorLength,
			),
		}
	}
	return uint32(payloadLength + terminatorLength), nil
}

func bofPackWriteBytes(ctx context.Context, writer *runtimeOutputWriter, input []byte) error {
	for position := 0; position < len(input); {
		if err := ctx.Err(); err != nil {
			return err
		}
		end := position + aggressorUtilityChunkSize
		if end > len(input) {
			end = len(input)
		}
		if err := sleepWriteFormattedBytes(ctx, writer, input[position:end]); err != nil {
			return err
		}
		position = end
	}
	return nil
}
