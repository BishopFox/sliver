package opfor

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"unicode/utf16"

	"github.com/sliverarmory/opfor/internal/lexer"
)

// mathExtraFunctions completes the portable BasicNumbers function surface.
// The functions are kept separate from stringNumberFunctions so the mapping
// remains easy to audit against Sleep 2.1's BasicNumbers bridge.
func (r *Runtime) mathExtraFunctions() map[string]NativeFunc {
	return map[string]NativeFunc{
		"acos":         builtinSleepAcos,
		"asin":         builtinSleepAsin,
		"atan":         builtinSleepAtan,
		"atan2":        builtinSleepAtan2,
		"radians":      builtinSleepRadians,
		"degrees":      builtinSleepDegrees,
		"exp":          builtinSleepExp,
		"not":          builtinSleepNot,
		"uint":         builtinSleepUint,
		"parseNumber":  builtinSleepParseNumber,
		"formatNumber": builtinSleepFormatNumber,
	}
}

func builtinSleepAcos(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Acos(sleepFloat64(invocation.Arg(0)))), nil
}

func builtinSleepAsin(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Asin(sleepFloat64(invocation.Arg(0)))), nil
}

func builtinSleepAtan(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Atan(sleepFloat64(invocation.Arg(0)))), nil
}

func builtinSleepAtan2(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Atan2(
		sleepFloat64(invocation.Arg(0)),
		sleepFloat64(invocation.Arg(1)),
	)), nil
}

func builtinSleepRadians(_ context.Context, invocation Invocation) (Value, error) {
	return Double(sleepFloat64(invocation.Arg(0)) * math.Pi / 180), nil
}

func builtinSleepDegrees(_ context.Context, invocation Invocation) (Value, error) {
	return Double(sleepFloat64(invocation.Arg(0)) * 180 / math.Pi), nil
}

func builtinSleepExp(_ context.Context, invocation Invocation) (Value, error) {
	return Double(math.Exp(sleepFloat64(invocation.Arg(0)))), nil
}

func builtinSleepNot(_ context.Context, invocation Invocation) (Value, error) {
	value, err := sleepBridgeArgument(invocation, 0)
	if err != nil {
		return Null(), err
	}
	if value.Kind() == KindInt {
		return Int(^sleepInt32(value)), nil
	}
	return Long(^sleepInt64(value)), nil
}

func builtinSleepUint(_ context.Context, invocation Invocation) (Value, error) {
	return Long(int64(uint32(sleepInt32(invocation.Arg(0))))), nil
}

func builtinSleepParseNumber(_ context.Context, invocation Invocation) (Value, error) {
	number := "0"
	if len(invocation.Arguments) > 0 {
		number = invocation.Arg(0).String()
	}
	radix := int32(10)
	if len(invocation.Arguments) > 1 {
		radix = sleepInt32(invocation.Arg(1))
	}
	parsed, err := sleepBigInteger(number, radix)
	if err != nil {
		return Null(), err
	}
	return Long(sleepBigIntegerLongValue(parsed)), nil
}

func builtinSleepFormatNumber(_ context.Context, invocation Invocation) (Value, error) {
	number := "0"
	if len(invocation.Arguments) > 0 {
		number = invocation.Arg(0).String()
	}
	from, to := int32(10), int32(10)
	switch len(invocation.Arguments) {
	case 0, 1:
	case 2:
		to = sleepInt32(invocation.Arg(1))
	case 3:
		from = sleepInt32(invocation.Arg(1))
		to = sleepInt32(invocation.Arg(2))
	default:
		// BasicNumbers tests args.size() after popping the number. Exactly
		// three source arguments select from/to radices; with four or more,
		// the second argument is only the output radix and the rest are
		// ignored.
		to = sleepInt32(invocation.Arg(1))
	}
	parsed, err := sleepBigInteger(number, from)
	if err != nil {
		return Null(), err
	}
	// BigInteger.toString(radix) silently substitutes base 10 for an output
	// radix outside 2..36. Its String/radix constructor does not do so for the
	// input radix, which is why sleepBigInteger remains strict above.
	if to < 2 || to > 36 {
		to = 10
	}
	return String(parsed.Text(int(to))), nil
}

func sleepBigInteger(number string, radix int32) (*big.Int, error) {
	if radix < 2 || radix > 36 {
		return nil, sleepBridgeIllegalArgument("Radix out of range")
	}

	// BigInteger(String, radix) consumes UTF-16 code units through
	// Character.digit(char, radix), rather than limiting input to the ASCII
	// alphabet accepted by math/big.SetString. Preserve that distinction so
	// Sleep accepts decimal digits from other BMP Unicode scripts as well as
	// Java's fullwidth Latin digits and letters.
	characters := utf16.Encode([]rune(number))
	if len(characters) == 0 {
		return nil, sleepBridgeIllegalArgument("Zero length BigInteger")
	}

	// BigInteger rejects every non-leading ASCII sign (and multiple signs)
	// before it examines the number's digits.
	minus, plus := -1, -1
	for index, character := range characters {
		switch character {
		case '-':
			minus = index
		case '+':
			plus = index
		}
	}
	sign, cursor := 1, 0
	if minus >= 0 {
		if minus != 0 || plus >= 0 {
			return nil, sleepBridgeIllegalArgument("Illegal embedded sign character")
		}
		sign, cursor = -1, 1
	} else if plus >= 0 {
		if plus != 0 {
			return nil, sleepBridgeIllegalArgument("Illegal embedded sign character")
		}
		cursor = 1
	}
	if cursor == len(characters) {
		return nil, sleepBridgeIllegalArgument("Zero length BigInteger")
	}

	for cursor < len(characters) && sleepJavaDigit(characters[cursor], radix) == 0 {
		cursor++
	}
	if cursor == len(characters) {
		return new(big.Int), nil
	}

	// BigInteger parses fixed-size groups through Integer.parseInt. Besides
	// validating the characters group-by-group, this determines the exact
	// NumberFormatException detail (the failing group, plus a non-decimal
	// radix suffix) that Sleep exposes as a warning.
	groupSize := sleepBigIntegerDigitsPerInt[radix]
	firstGroupSize := (len(characters) - cursor) % groupSize
	if firstGroupSize == 0 {
		firstGroupSize = groupSize
	}
	for start, size := cursor, firstGroupSize; start < len(characters); size = groupSize {
		end := start + size
		for _, character := range characters[start:end] {
			if sleepJavaDigit(character, radix) < 0 {
				group := string(utf16.Decode(characters[start:end]))
				message := `For input string: "` + group + `"`
				if radix != 10 {
					message += fmt.Sprintf(" under radix %d", radix)
				}
				return nil, sleepBridgeIllegalArgument(message)
			}
		}
		start = end
	}

	parsed := new(big.Int)
	bigRadix := big.NewInt(int64(radix))
	digit := new(big.Int)
	for _, character := range characters[cursor:] {
		parsed.Mul(parsed, bigRadix)
		digit.SetInt64(int64(sleepJavaDigit(character, radix)))
		parsed.Add(parsed, digit)
	}
	if sign < 0 {
		parsed.Neg(parsed)
	}
	return parsed, nil
}

var sleepBigIntegerDigitsPerInt = [...]int{
	0, 0, 30, 19, 15, 13, 11, 11, 10, 9, 9, 8, 8, 8, 8, 7, 7, 7, 7,
	7, 7, 7, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 6, 5,
}

func sleepJavaDigit(character uint16, radix int32) int32 {
	return int32(lexer.JavaDigit(rune(character), int(radix)))
}

func sleepBigIntegerLongValue(value *big.Int) int64 {
	// BigInteger.longValue returns the low 64 bits in two's-complement form,
	// including for values outside the signed-long range.
	modulus := new(big.Int).Lsh(big.NewInt(1), 64)
	low := new(big.Int).Mod(new(big.Int).Set(value), modulus)
	return int64(low.Uint64())
}
