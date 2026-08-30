package opfor

import (
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"
)

// sleepTextCharset is the finite, portable subset of Java's charset registry
// exposed by BasicIO.setEncoding. The JVM bridge accepts provider-installed
// charsets too; OPFOR deliberately limits that open-ended surface to the six
// Java-required Unicode/ASCII families plus Windows-1252.
type sleepTextCharset uint8

const (
	sleepCharsetUTF8 sleepTextCharset = iota
	sleepCharsetASCII
	sleepCharsetLatin1
	sleepCharsetWindows1252
	sleepCharsetUTF16
	sleepCharsetUTF16BE
	sleepCharsetUTF16LE
)

func sleepLookupTextCharset(name string) (sleepTextCharset, error) {
	switch strings.ToLower(name) {
	case "utf-8", "utf8", "unicode-1-1-utf-8":
		return sleepCharsetUTF8, nil
	case "us-ascii", "ascii", "ascii7", "646", "iso646-us", "iso_646.irv:1991", "iso_646.irv:1983", "iso-ir-6", "ansi_x3.4-1986", "ansi_x3.4-1968", "us", "ibm367", "cp367", "csascii":
		return sleepCharsetASCII, nil
	case "iso-8859-1", "iso8859_1", "iso8859-1", "iso_8859-1", "iso_8859_1", "iso_8859-1:1987", "8859_1", "iso-ir-100", "latin1", "l1", "ibm819", "ibm-819", "cp819", "csisolatin1", "819":
		return sleepCharsetLatin1, nil
	case "windows-1252", "cp1252", "cp5348", "ibm-1252", "ibm1252":
		return sleepCharsetWindows1252, nil
	case "utf-16", "utf_16", "utf16", "unicode", "unicodebig":
		return sleepCharsetUTF16, nil
	case "utf-16be", "utf_16be", "x-utf-16be", "unicodebigunmarked", "iso-10646-ucs-2":
		return sleepCharsetUTF16BE, nil
	case "utf-16le", "utf_16le", "x-utf-16le", "unicodelittleunmarked":
		return sleepCharsetUTF16LE, nil
	default:
		return sleepCharsetUTF8, fmt.Errorf("unsupported charset %q", name)
	}
}

type sleepTextDecoder struct {
	charset sleepTextCharset
	pending []byte

	utf16Order       binary.ByteOrder
	utf16OrderChosen bool
	utf16High        uint16
}

func (decoder *sleepTextDecoder) reset(charset sleepTextCharset) {
	decoder.charset = charset
	decoder.pending = nil
	decoder.utf16High = 0
	switch charset {
	case sleepCharsetUTF16:
		decoder.utf16Order = binary.BigEndian
		decoder.utf16OrderChosen = false
	case sleepCharsetUTF16LE:
		decoder.utf16Order = binary.LittleEndian
		decoder.utf16OrderChosen = true
	default:
		decoder.utf16Order = binary.BigEndian
		decoder.utf16OrderChosen = true
	}
}

func (decoder *sleepTextDecoder) decode(data []byte, atEOF bool) []uint16 {
	capacity := len(data)
	if decoder.charset == sleepCharsetUTF16 || decoder.charset == sleepCharsetUTF16BE || decoder.charset == sleepCharsetUTF16LE {
		capacity = len(data)/2 + 2
	}
	units := make([]uint16, 0, capacity)
	decoder.decodeTo(data, atEOF, func(unit uint16) {
		units = append(units, unit)
	})
	return units
}

// decodedRenderedLength applies the same state machine as decode while only
// counting the bytes sleepRenderStringUnits will need. Callers can therefore
// reserve an exact UTF-8 expansion before allocating decoded code units.
func (decoder *sleepTextDecoder) decodedRenderedLength(data []byte, atEOF bool) int {
	length := 0
	var high uint16
	decoder.decodeTo(data, atEOF, func(unit uint16) {
		if high != 0 {
			if unit >= 0xdc00 && unit <= 0xdfff {
				length += 4
				high = 0
				return
			}
			length += 3
			high = 0
		}
		if unit >= 0xd800 && unit <= 0xdbff {
			high = unit
			return
		}
		switch {
		case unit < 0x80:
			length++
		case unit < 0x800:
			length += 2
		default:
			length += 3
		}
	})
	if high != 0 {
		length += 3
	}
	return length
}

func (decoder *sleepTextDecoder) decodeTo(data []byte, atEOF bool, emit func(uint16)) {
	switch decoder.charset {
	case sleepCharsetASCII:
		for _, value := range data {
			if value <= 0x7f {
				emit(uint16(value))
			} else {
				emit(0xfffd)
			}
		}
	case sleepCharsetLatin1:
		for _, value := range data {
			emit(uint16(value))
		}
	case sleepCharsetWindows1252:
		for _, value := range data {
			emit(sleepWindows1252Decode(value))
		}
	case sleepCharsetUTF16, sleepCharsetUTF16BE, sleepCharsetUTF16LE:
		decoder.decodeUTF16To(data, atEOF, emit)
	default:
		decoder.decodeUTF8To(data, atEOF, emit)
	}
}

func (decoder *sleepTextDecoder) decodeUTF8To(data []byte, atEOF bool, emit func(uint16)) {
	input := data
	if len(decoder.pending) != 0 {
		input = append(append(make([]byte, 0, len(decoder.pending)+len(data)), decoder.pending...), data...)
	}
	decoder.pending = nil

	for position := 0; position < len(input); {
		first := input[position]
		if first < 0x80 {
			emit(uint16(first))
			position++
			continue
		}

		width := 0
		switch {
		case first >= 0xc2 && first <= 0xdf:
			width = 2
		case first >= 0xe0 && first <= 0xef:
			width = 3
		case first >= 0xf0 && first <= 0xf4:
			width = 4
		default:
			emit(0xfffd)
			position++
			continue
		}

		available := len(input) - position
		if available >= 2 {
			second := input[position+1]
			if first == 0xe0 && second >= 0x80 && second < 0xa0 ||
				first == 0xf0 && second >= 0x80 && second < 0x90 ||
				first == 0xf4 && second >= 0x90 && second <= 0xbf {
				// Java rejects an out-of-range second byte as soon as it is
				// available. Only the lead byte belongs to that malformed
				// input; continuation bytes are decoded independently even
				// when the sequence ends before its nominal width.
				emit(0xfffd)
				position++
				continue
			}
		}
		validPrefix := 1
		for validPrefix < width && validPrefix < available && input[position+validPrefix]&0xc0 == 0x80 {
			validPrefix++
		}
		if validPrefix < width {
			if validPrefix == available && !atEOF {
				decoder.pending = append(decoder.pending, input[position:]...)
				break
			}
			// Java consumes the valid prefix of an otherwise well-formed
			// sequence as one malformed input, leaving the first non-
			// continuation byte to be decoded normally.
			emit(0xfffd)
			position += validPrefix
			continue
		}

		second := input[position+1]
		switch {
		case first == 0xed && second >= 0xa0:
			// The JVM decoder treats a complete UTF-8 spelling of a UTF-16
			// surrogate as one malformed sequence.
			emit(0xfffd)
			position += width
			continue
		}

		var character rune
		switch width {
		case 2:
			character = rune(first&0x1f)<<6 | rune(second&0x3f)
		case 3:
			character = rune(first&0x0f)<<12 | rune(second&0x3f)<<6 | rune(input[position+2]&0x3f)
		case 4:
			character = rune(first&0x07)<<18 | rune(second&0x3f)<<12 |
				rune(input[position+2]&0x3f)<<6 | rune(input[position+3]&0x3f)
		}
		if character <= 0xffff {
			emit(uint16(character))
		} else {
			high, low := utf16.EncodeRune(character)
			emit(uint16(high))
			emit(uint16(low))
		}
		position += width
	}

	if atEOF && len(decoder.pending) != 0 {
		emit(0xfffd)
		decoder.pending = nil
	}
}

func (decoder *sleepTextDecoder) decodeUTF16To(data []byte, atEOF bool, emit func(uint16)) {
	input := data
	if len(decoder.pending) != 0 {
		input = append(append(make([]byte, 0, len(decoder.pending)+len(data)), decoder.pending...), data...)
	}
	decoder.pending = nil
	position := 0

	for position+1 < len(input) {
		if !decoder.utf16OrderChosen {
			switch {
			case input[position] == 0xfe && input[position+1] == 0xff:
				decoder.utf16Order = binary.BigEndian
				decoder.utf16OrderChosen = true
				position += 2
				continue
			case input[position] == 0xff && input[position+1] == 0xfe:
				decoder.utf16Order = binary.LittleEndian
				decoder.utf16OrderChosen = true
				position += 2
				continue
			default:
				decoder.utf16Order = binary.BigEndian
				decoder.utf16OrderChosen = true
			}
		}

		unit := decoder.utf16Order.Uint16(input[position : position+2])
		position += 2
		decoder.emitUTF16Unit(emit, unit)
	}

	if position < len(input) {
		if atEOF {
			if decoder.utf16High != 0 {
				emit(0xfffd)
				decoder.utf16High = 0
			} else {
				emit(0xfffd)
			}
		} else {
			decoder.pending = append(decoder.pending, input[position])
		}
	} else if atEOF && decoder.utf16High != 0 {
		emit(0xfffd)
		decoder.utf16High = 0
	}
}

func (decoder *sleepTextDecoder) emitUTF16Unit(emit func(uint16), unit uint16) {
	if decoder.utf16High != 0 {
		if unit >= 0xdc00 && unit <= 0xdfff {
			emit(decoder.utf16High)
			emit(unit)
			decoder.utf16High = 0
			return
		}
		emit(0xfffd)
		decoder.utf16High = 0
		// The JDK UTF-16 decoder reports the pending high surrogate and
		// following non-low unit as one malformed input. Its replacement
		// therefore consumes both code units.
		return
	}
	if unit >= 0xd800 && unit <= 0xdbff {
		decoder.utf16High = unit
		return
	}
	if unit >= 0xdc00 && unit <= 0xdfff {
		emit(0xfffd)
		return
	}
	emit(unit)
}

type sleepTextEncoder struct {
	charset     sleepTextCharset
	pendingHigh uint16
	bomWritten  bool
}

func (encoder *sleepTextEncoder) reset(charset sleepTextCharset) {
	encoder.charset = charset
	encoder.pendingHigh = 0
	encoder.bomWritten = false
}

func (encoder *sleepTextEncoder) encode(text string) []byte {
	return encoder.encodeUnits(sleepFormattedUTF16Units(text), false)
}

func (encoder *sleepTextEncoder) finish() []byte {
	return encoder.encodeUnits(nil, true)
}

func (encoder *sleepTextEncoder) encodeUnits(units []uint16, final bool) []byte {
	output := make([]byte, 0, len(units)*3+4)
	if encoder.charset == sleepCharsetUTF16 && len(units) != 0 {
		// StreamEncoder writes UTF-16's BOM when its first non-empty write is
		// accepted, even if that call contains only a high surrogate retained
		// for the next write.
		output = encoder.appendUTF16BOM(output)
	}
	for _, unit := range units {
		if encoder.pendingHigh != 0 {
			if unit >= 0xdc00 && unit <= 0xdfff {
				output = encoder.appendPair(output, encoder.pendingHigh, unit)
				encoder.pendingHigh = 0
				continue
			}
			output = encoder.appendReplacement(output)
			encoder.pendingHigh = 0
		}
		if unit >= 0xd800 && unit <= 0xdbff {
			encoder.pendingHigh = unit
			continue
		}
		if unit >= 0xdc00 && unit <= 0xdfff {
			output = encoder.appendReplacement(output)
			continue
		}
		output = encoder.appendBMP(output, unit)
	}
	if final && encoder.pendingHigh != 0 {
		output = encoder.appendReplacement(output)
		encoder.pendingHigh = 0
	}
	return output
}

func (encoder *sleepTextEncoder) appendBMP(output []byte, unit uint16) []byte {
	switch encoder.charset {
	case sleepCharsetASCII:
		if unit <= 0x7f {
			return append(output, byte(unit))
		}
		return append(output, '?')
	case sleepCharsetLatin1:
		if unit <= 0xff {
			return append(output, byte(unit))
		}
		return append(output, '?')
	case sleepCharsetWindows1252:
		if value, ok := sleepWindows1252Encode(unit); ok {
			return append(output, value)
		}
		return append(output, '?')
	case sleepCharsetUTF16, sleepCharsetUTF16BE, sleepCharsetUTF16LE:
		output = encoder.appendUTF16BOM(output)
		return appendSleepEncodedUint16(output, encoder.utf16Order(), unit)
	default:
		return append(output, string(rune(unit))...)
	}
}

func (encoder *sleepTextEncoder) appendPair(output []byte, high, low uint16) []byte {
	switch encoder.charset {
	case sleepCharsetUTF16, sleepCharsetUTF16BE, sleepCharsetUTF16LE:
		output = encoder.appendUTF16BOM(output)
		order := encoder.utf16Order()
		output = appendSleepEncodedUint16(output, order, high)
		return appendSleepEncodedUint16(output, order, low)
	case sleepCharsetUTF8:
		return append(output, string(utf16.DecodeRune(rune(high), rune(low)))...)
	default:
		return append(output, '?')
	}
}

func (encoder *sleepTextEncoder) appendReplacement(output []byte) []byte {
	switch encoder.charset {
	case sleepCharsetUTF16, sleepCharsetUTF16BE, sleepCharsetUTF16LE:
		output = encoder.appendUTF16BOM(output)
		return appendSleepEncodedUint16(output, encoder.utf16Order(), 0xfffd)
	default:
		return append(output, '?')
	}
}

func (encoder *sleepTextEncoder) appendUTF16BOM(output []byte) []byte {
	if encoder.charset == sleepCharsetUTF16 && !encoder.bomWritten {
		output = append(output, 0xfe, 0xff)
		encoder.bomWritten = true
	}
	return output
}

func (encoder *sleepTextEncoder) utf16Order() binary.ByteOrder {
	if encoder.charset == sleepCharsetUTF16LE {
		return binary.LittleEndian
	}
	return binary.BigEndian
}

func appendSleepEncodedUint16(output []byte, order binary.ByteOrder, unit uint16) []byte {
	var encoded [2]byte
	order.PutUint16(encoded[:], unit)
	return append(output, encoded[:]...)
}

func sleepWindows1252Decode(value byte) uint16 {
	if value < 0x80 || value >= 0xa0 {
		return uint16(value)
	}
	if decoded, ok := sleepWindows1252Special[value]; ok {
		return decoded
	}
	return 0xfffd
}

func sleepWindows1252Encode(unit uint16) (byte, bool) {
	if unit < 0x80 || unit >= 0xa0 && unit <= 0xff {
		return byte(unit), true
	}
	for encoded, decoded := range sleepWindows1252Special {
		if decoded == unit {
			return encoded, true
		}
	}
	return 0, false
}

var sleepWindows1252Special = map[byte]uint16{
	0x80: 0x20ac,
	0x82: 0x201a,
	0x83: 0x0192,
	0x84: 0x201e,
	0x85: 0x2026,
	0x86: 0x2020,
	0x87: 0x2021,
	0x88: 0x02c6,
	0x89: 0x2030,
	0x8a: 0x0160,
	0x8b: 0x2039,
	0x8c: 0x0152,
	0x8e: 0x017d,
	0x91: 0x2018,
	0x92: 0x2019,
	0x93: 0x201c,
	0x94: 0x201d,
	0x95: 0x2022,
	0x96: 0x2013,
	0x97: 0x2014,
	0x98: 0x02dc,
	0x99: 0x2122,
	0x9a: 0x0161,
	0x9b: 0x203a,
	0x9c: 0x0153,
	0x9e: 0x017e,
	0x9f: 0x0178,
}

// sleepUTF16CharacterValue retains the exact Java code unit carried by readc
// and chr, including an unpaired surrogate.
func sleepUTF16CharacterValue(unit uint16) Value {
	return sleepStringValueFromUnits([]uint16{unit}, nil)
}
