package javaser

import (
	"fmt"
	"unicode/utf16"
)

func encodeModifiedUTF8(value string, rawUTF16 []uint16) []byte {
	units := rawUTF16
	if units == nil {
		units = utf16.Encode([]rune(value))
	}
	out := make([]byte, 0, len(units))
	for _, unit := range units {
		switch {
		case unit != 0 && unit <= 0x7f:
			out = append(out, byte(unit))
		case unit <= 0x07ff:
			out = append(out,
				0xc0|byte(unit>>6),
				0x80|byte(unit&0x3f),
			)
		default:
			out = append(out,
				0xe0|byte(unit>>12),
				0x80|byte((unit>>6)&0x3f),
				0x80|byte(unit&0x3f),
			)
		}
	}
	return out
}

func decodeModifiedUTF8(data []byte) (string, []uint16, error) {
	units := make([]uint16, 0, len(data))
	for i := 0; i < len(data); {
		first := data[i]
		switch {
		case first >= 0x01 && first <= 0x7f:
			units = append(units, uint16(first))
			i++
		case first >= 0xc0 && first <= 0xdf:
			if i+1 >= len(data) || data[i+1]&0xc0 != 0x80 {
				return "", nil, fmt.Errorf("invalid modified UTF-8 two-byte sequence at byte %d", i)
			}
			unit := uint16(first&0x1f)<<6 | uint16(data[i+1]&0x3f)
			if unit < 0x80 && unit != 0 {
				return "", nil, fmt.Errorf("overlong modified UTF-8 sequence at byte %d", i)
			}
			units = append(units, unit)
			i += 2
		case first >= 0xe0 && first <= 0xef:
			if i+2 >= len(data) || data[i+1]&0xc0 != 0x80 || data[i+2]&0xc0 != 0x80 {
				return "", nil, fmt.Errorf("invalid modified UTF-8 three-byte sequence at byte %d", i)
			}
			unit := uint16(first&0x0f)<<12 |
				uint16(data[i+1]&0x3f)<<6 |
				uint16(data[i+2]&0x3f)
			if unit < 0x0800 {
				return "", nil, fmt.Errorf("overlong modified UTF-8 sequence at byte %d", i)
			}
			units = append(units, unit)
			i += 3
		default:
			return "", nil, fmt.Errorf("invalid modified UTF-8 leading byte 0x%02x at byte %d", first, i)
		}
	}
	return string(utf16.Decode(units)), units, nil
}
