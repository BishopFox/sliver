package lexer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// javaDecimalDigitRanges is the BMP portion of the Unicode 17 DIGIT table
// already pinned by OPFOR's Java-character compatibility data. Sleep's
// integer decoders consume UTF-16 char values through Character.digit, so a
// supplementary-code-point digit must not be treated as one numeric digit.
var javaDecimalDigitRanges = [...]struct {
	lo rune
	hi rune
}{
	{0x0030, 0x0039}, {0x0660, 0x0669}, {0x06f0, 0x06f9},
	{0x07c0, 0x07c9}, {0x0966, 0x096f}, {0x09e6, 0x09ef},
	{0x0a66, 0x0a6f}, {0x0ae6, 0x0aef}, {0x0b66, 0x0b6f},
	{0x0be6, 0x0bef}, {0x0c66, 0x0c6f}, {0x0ce6, 0x0cef},
	{0x0d66, 0x0d6f}, {0x0de6, 0x0def}, {0x0e50, 0x0e59},
	{0x0ed0, 0x0ed9}, {0x0f20, 0x0f29}, {0x1040, 0x1049},
	{0x1090, 0x1099}, {0x17e0, 0x17e9}, {0x1810, 0x1819},
	{0x1946, 0x194f}, {0x19d0, 0x19d9}, {0x1a80, 0x1a89},
	{0x1a90, 0x1a99}, {0x1b50, 0x1b59}, {0x1bb0, 0x1bb9},
	{0x1c40, 0x1c49}, {0x1c50, 0x1c59}, {0xa620, 0xa629},
	{0xa8d0, 0xa8d9}, {0xa900, 0xa909}, {0xa9d0, 0xa9d9},
	{0xa9f0, 0xa9f9}, {0xaa50, 0xaa59}, {0xabf0, 0xabf9},
	{0xff10, 0xff19},
}

// JavaDigit mirrors Character.digit(char, radix) for the source characters
// that Sleep's numeric and BigInteger paths accept. A rune above the BMP is
// rejected because Java iterates its UTF-16 surrogate code units separately.
func JavaDigit(character rune, radix int) int {
	if radix < 2 || radix > 36 || character > 0xffff {
		return -1
	}

	value := -1
	switch {
	case character >= 'A' && character <= 'Z':
		value = int(character-'A') + 10
	case character >= 'a' && character <= 'z':
		value = int(character-'a') + 10
	case character >= 0xff21 && character <= 0xff3a:
		value = int(character-0xff21) + 10
	case character >= 0xff41 && character <= 0xff5a:
		value = int(character-0xff41) + 10
	default:
		for _, digitRange := range javaDecimalDigitRanges {
			if character < digitRange.lo || character > digitRange.hi {
				continue
			}
			value = int(character-digitRange.lo) % 10
			break
		}
	}
	if value >= radix {
		return -1
	}
	return value
}

// IsJavaDecimalDigit mirrors Character.isDigit(char) for a source rune.
func IsJavaDecimalDigit(character rune) bool {
	return JavaDigit(character, 10) >= 0
}

// ClassifyNumericLiteral applies Sleep Checkers' numeric ordering to a term:
// Integer/Long.decode first, followed by Double.parseDouble for non-long
// numeric terms. The lexer kind is a grammar hint, not the final value kind.
func ClassifyNumericLiteral(raw string, hinted Kind) (Kind, bool) {
	switch hinted {
	case Long:
		if !strings.HasSuffix(raw, "L") {
			return Invalid, false
		}
		if _, err := ParseJavaIntegerLiteral(strings.TrimSuffix(raw, "L"), 64); err != nil {
			return Invalid, false
		}
		return Long, true
	case Integer:
		if _, err := ParseJavaIntegerLiteral(raw, 32); err == nil {
			return Integer, true
		}
		if _, err := ParseJavaDoubleLiteral(raw); err == nil {
			return Double, true
		}
		return Invalid, false
	case Double:
		if _, err := ParseJavaDoubleLiteral(raw); err != nil {
			return Invalid, false
		}
		return Double, true
	default:
		return Invalid, false
	}
}

// ParseJavaIntegerLiteral implements the Integer.decode and Long.decode forms
// used by Sleep. It accepts a sign before an ASCII hexadecimal/octal prefix,
// decodes digits with Character.digit, and preserves the signed minimum value.
func ParseJavaIntegerLiteral(raw string, bits int) (int64, error) {
	if bits != 32 && bits != 64 {
		return 0, fmt.Errorf("unsupported Java integer width %d", bits)
	}
	if raw == "" {
		return 0, strconv.ErrSyntax
	}

	negative := false
	index := 0
	switch raw[0] {
	case '-':
		negative = true
		index++
	case '+':
		index++
	}
	if index == len(raw) {
		return 0, strconv.ErrSyntax
	}

	radix := 10
	if strings.HasPrefix(raw[index:], "0x") || strings.HasPrefix(raw[index:], "0X") {
		radix = 16
		index += 2
	} else if raw[index] == '0' && len(raw) > index+1 {
		radix = 8
		index++
	}
	if index == len(raw) {
		return 0, strconv.ErrSyntax
	}

	limit := uint64(1)<<(bits-1) - 1
	if negative {
		limit++
	}
	magnitude := uint64(0)
	for _, character := range raw[index:] {
		digit := JavaDigit(character, radix)
		if digit < 0 {
			return 0, strconv.ErrSyntax
		}
		unsignedDigit := uint64(digit)
		if magnitude > (limit-unsignedDigit)/uint64(radix) {
			return 0, strconv.ErrRange
		}
		magnitude = magnitude*uint64(radix) + unsignedDigit
	}

	if negative {
		if bits == 64 && magnitude == uint64(1)<<63 {
			return -1 << 63, nil
		}
		return -int64(magnitude), nil
	}
	return int64(magnitude), nil
}

// ParseJavaDoubleLiteral implements the Double.parseDouble forms emitted by
// the lexer. ParseFloat returns a useful IEEE infinity or zero together with
// ErrRange for overflow and underflow; Java accepts those results.
func ParseJavaDoubleLiteral(raw string) (float64, error) {
	unsuffixed := strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(raw, "D"), "d"), "F"), "f")
	if unsuffixed != raw {
		switch unsuffixed {
		case "NaN", "+NaN", "-NaN", "Infinity", "+Infinity", "-Infinity":
			return 0, strconv.ErrSyntax
		}
	}
	raw = unsuffixed
	switch raw {
	case "+NaN", "-NaN":
		raw = "NaN"
	case "Infinity", "+Infinity":
		raw = "+Inf"
	case "-Infinity":
		raw = "-Inf"
	case "Inf", "+Inf", "-Inf":
		return 0, strconv.ErrSyntax
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return 0, err
	}
	return value, nil
}
