package strutil

import (
	"fmt"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	// Max integer value on 64 bit architecture.
	maxInt = 9223372036854775807
)

// KeywordSwitcher is a function modifying a given word, returning:
// @done     => If true, the handler performed a change.
// @switched => The updated word.
// @bpos     => Offset to begin position.
// @epos     => Offset to end position.
type KeywordSwitcher func(word string, increase bool, times int) (done bool, switched string, bpos, epos int)

// KeywordSwitchers returns all keywordSwitchers of the shell.
func KeywordSwitchers() []KeywordSwitcher {
	return []KeywordSwitcher{
		switchNumber,
		switchBoolean,
		switchWeekday,
		switchOperator,
	}
}

// AdjustNumberOperatorPos returns an adjusted cursor position when
// the word around the cursor is an expression with an operator.
func AdjustNumberOperatorPos(cpos int, line []rune) int {
	word := lineSlice(line, cpos, 2)

	if match, _ := regexp.MatchString(`[+-][0-9]`, word); match {
		// If cursor is on the `+` or `-`, we need to check if it is a
		// number with a sign or an operator, only the number needs to
		// forward the cursor.
		digit := regexp.MustCompile(`[^0-9]`)
		if cpos == 0 || digit.MatchString(string(line[cpos-1])) {
			cpos++
		}
	} else if match, _ := regexp.MatchString(`[+-][a-zA-Z]`, word); match {
		// If cursor is on the `+` or `-`, we need to check if it is a
		// short option, only the short option needs to forward the cursor.
		if cpos == 0 || line[cpos-1] == ' ' {
			cpos++
		}
	}

	return cpos
}

// lineSlice returns a subset of the current input line.
func lineSlice(line []rune, cpos, adjust int) (slice string) {
	switch {
	case cpos+adjust > len(line):
		slice = string(line[cpos:])
	case adjust < 0:
		if cpos+adjust < 0 {
			slice = string(line[:cpos])
		} else {
			slice = string(line[cpos+adjust : cpos])
		}
	default:
		slice = string(line[cpos : cpos+adjust])
	}

	return
}

func switchNumber(word string, _ bool, times int) (done bool, switched string, bpos, epos int) {
	if done, switched, bpos, epos = switchHexa(word, times); done {
		return
	}

	if done, switched, bpos, epos = switchBinary(word, times); done {
		return
	}

	if done, switched, bpos, epos = switchDecimal(word, times); done {
		return
	}

	return
}

// Hexadecimal cases:
//
// 1. Increment:
// 0xDe => 0xdf
// 0xdE => 0xDF
// 0xde0 => 0xddf
// 0xffffffffffffffff => 0x0000000000000000
// 0X9 => 0XA
// 0Xdf => 0Xe0
//
// 2. Decrement:
// 0xdE0 => 0xDDF
// 0xffFf0 => 0xfffef
// 0xfffF0 => 0xFFFEF
// 0x0 => 0xffffffffffffffff
// 0X0 => 0XFFFFFFFFFFFFFFFF
// 0Xf => 0Xe.
func switchHexa(word string, inc int) (done bool, switched string, bpos, epos int) {
	hexadecimal := regexp.MustCompile(`[^0-9]?(0[xX][0-9a-fA-F]*)`)
	match := hexadecimal.FindString(word)

	if match == "" {
		return
	}

	done = true

	number := match
	prefix := match[:2]
	hexVal := number[len(prefix):]
	indexes := hexadecimal.FindStringIndex(word)
	mbegin, mend := indexes[0], indexes[1]
	bpos, epos = mbegin, mend

	if match, _ := regexp.MatchString(`[A-Z][0-9]*$`, number); !match {
		hexVal = strings.ToUpper(hexVal)
	}

	num, err := strconv.ParseInt(hexVal, 16, 64)
	if err != nil {
		done = false
		return
	}

	max64Bit := big.NewInt(maxInt)
	bigNum := big.NewInt(num)
	bigInc := big.NewInt(int64(inc))
	sum := bigNum.Add(bigNum, bigInc)

	numBefore := num

	switch {
	case sum.Cmp(big.NewInt(0)) < 0:
		offset := bigInc.Sub(max64Bit, sum.Abs(sum))
		if offset.IsInt64() {
			num = offset.Int64()
		} else {
			num = math.MaxInt64
		}
	case sum.CmpAbs(max64Bit) >= 0:
		offset := bigInc.Sub(sum, max64Bit)
		if offset.IsInt64() {
			num = offset.Int64()
		} else {
			num = int64(inc) - (num - numBefore)
		}
	default:
		num = sum.Int64()
	}

	hexVal = fmt.Sprintf("%x", num)
	switched = prefix + hexVal

	return done, switched, bpos, epos
}

// Binary cases:
//
// 1. Increment:
// 0b1 => 0b10
// 0x1111111111111111111111111111111111111111111111111111111111111111 =>
// 0x0000000000000000000000000000000000000000000000000000000000000000
// 0B0 => 0B1
//
// 2. Decrement:
// 0b1 => 0b0
// 0b100 => 0b011
// 0B010 => 0B001
// 0b0 =>
// 0x1111111111111111111111111111111111111111111111111111111111111111.
func switchBinary(word string, inc int) (done bool, switched string, bpos, epos int) {
	binary := regexp.MustCompile(`[^0-9]?(0[bB][01]*)`)
	match := binary.FindString(word)

	if match == "" {
		return
	}

	done = true

	number := match
	prefix := match[:2]
	binVal := number[len(prefix):]
	indexes := binary.FindStringIndex(word)
	mbegin, mend := indexes[0], indexes[1]
	bpos, epos = mbegin, mend

	num, err := strconv.ParseInt(binVal, 2, 64)
	if err != nil {
		done = false
		return
	}

	max64Bit := big.NewInt(maxInt)
	bigNum := big.NewInt(num)
	bigInc := big.NewInt(int64(inc))
	sum := bigNum.Add(bigNum, bigInc)
	zero := big.NewInt(0)

	numBefore := num

	switch {
	case sum.Cmp(zero) < 0:
		offset := bigInc.Sub(max64Bit, sum.Abs(sum))
		if offset.IsInt64() {
			num = offset.Int64()
		} else {
			num = math.MaxInt64
		}
	case sum.CmpAbs(max64Bit) >= 0:
		offset := bigInc.Sub(sum, max64Bit)
		if offset.IsInt64() {
			num = offset.Int64()
		} else {
			num = int64(inc) - (num - numBefore)
		}
	default:
		num = sum.Int64()
	}

	binVal = fmt.Sprintf("%b", num)
	switched = prefix + binVal

	return done, switched, bpos, epos
}

// Decimal cases:
//
// 1. Increment:
// 0 => 1
// 99 => 100
//
// 2. Decrement:
// 0 => -1
// 10 => 9
// aa1230xa => aa1231xa // NOT WORKING => MATCHED BY HEXA
// aa1230bb => aa1231bb
// aa123a0bb => aa124a0bb.
func switchDecimal(word string, inc int) (done bool, switched string, bpos, epos int) {
	decimal := regexp.MustCompile(`([-+]?[0-9]+)`)
	match := decimal.FindString(word)

	if match == "" {
		return
	}

	done = true

	indexes := decimal.FindStringIndex(word)
	mbegin, mend := indexes[0], indexes[1]
	bpos, epos = mbegin, mend

	num, _ := strconv.Atoi(match)
	num += inc

	switched = strconv.Itoa(num)

	// Add prefix if needed
	if word[0] == '+' {
		switched = "+" + switched
	}

	// Don't consider anything done if result is empty.
	if switched == "" {
		done = false
	}

	return
}

func switchBoolean(word string, _ bool, _ int) (done bool, switched string, bpos, epos int) {
	epos = len(word)

	option := regexp.MustCompile(`(^[+-]{0,2})`)
	if match := option.FindString(word); match != "" {
		indexes := option.FindStringIndex(word)
		bpos = indexes[1]
		word = word[bpos:]
	}

	booleans := map[string]string{
		"true":  "false",
		"false": "true",
		"t":     "f",
		"f":     "t",
		"yes":   "no",
		"no":    "yes",
		"y":     "n",
		"n":     "y",
		"on":    "off",
		"off":   "on",
	}

	switched, done = booleans[strings.ToLower(word)]
	if !done {
		return
	}

	done = true

	// Transform case
	if match, _ := regexp.MatchString(`^[A-Z]+$`, word); match {
		switched = strings.ToLower(switched)
	} else if match, _ := regexp.MatchString(`^[A-Z]`, word); match {
		letter := switched[0]
		upper := unicode.ToUpper(rune(letter))
		switched = string(upper) + switched[1:]
	}

	return done, switched, bpos, epos
}

func switchWeekday(word string, inc bool, _ int) (done bool, switched string, bpos, epos int) {
	return
}

func switchOperator(word string, _ bool, _ int) (done bool, switched string, bpos, epos int) {
	epos = len(word)

	operators := map[string]string{
		"&&":  "||",
		"||":  "&&",
		"++":  "--",
		"--":  "++",
		"==":  "!=",
		"!=":  "==",
		"===": "!==",
		"!==": "===",
		"+":   "-",
		"-":   "*",
		"*":   "/",
		"/":   "+",
		"and": "or",
		"or":  "and",
	}

	switched, done = operators[strings.ToLower(word)]
	if !done {
		return
	}

	done = true

	// Transform case
	if match, _ := regexp.MatchString(`^[A-Z]+$`, word); match {
		switched = strings.ToLower(switched)
	} else if match, _ := regexp.MatchString(`^[A-Z]`, word); match {
		letter := switched[0]
		upper := unicode.ToUpper(rune(letter))
		switched = string(upper) + switched[1:]
	}

	return done, switched, bpos, epos
}
