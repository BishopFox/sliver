// Package basex provides fast base encoding / decoding of any given alphabet using bitcoin style leading zero compression.
// It is a GO port of https://github.com/cryptocoinjs/base-x
package basex

import (
	"bytes"
	"errors"
)

var (
	ErrNonBaseChar       = errors.New("non-base character")
	ErrAmbiguousAlphabet = errors.New("ambiguous alphabet")
)

// Encoding is a custom base encoding defined by an alphabet.
// It should bre created using NewEncoding function
type Encoding struct {
	base        int
	alphabet    []rune
	alphabetMap map[rune]int
}

// NewEncoding returns a custom base encoder defined by the alphabet string.
// The alphabet should contain non-repeating characters.
// Ordering is important.
// Example alphabets:
//   - base2: 01
//   - base16: 0123456789abcdef
//   - base32: 0123456789ABCDEFGHJKMNPQRSTVWXYZ
//   - base62: 0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ
func NewEncoding(alphabet string) (*Encoding, error) {
	runes := []rune(alphabet)
	runeMap := make(map[rune]int)

	for i := 0; i < len(runes); i++ {
		if _, ok := runeMap[runes[i]]; ok {
			return nil, ErrAmbiguousAlphabet
		}

		runeMap[runes[i]] = i
	}

	return &Encoding{
		base:        len(runes),
		alphabet:    runes,
		alphabetMap: runeMap,
	}, nil
}

// Encode function receives a byte slice and encodes it to a string using the alphabet provided
func (e *Encoding) Encode(source []byte) string {
	if len(source) == 0 {
		return ""
	}

	digits := []int{0}

	for i := 0; i < len(source); i++ {
		carry := int(source[i])

		for j := 0; j < len(digits); j++ {
			carry += digits[j] << 8
			digits[j] = carry % e.base
			carry = carry / e.base
		}

		for carry > 0 {
			digits = append(digits, carry%e.base)
			carry = carry / e.base
		}
	}

	var res bytes.Buffer

	for k := 0; source[k] == 0 && k < len(source)-1; k++ {
		res.WriteRune(e.alphabet[0])
	}

	for q := len(digits) - 1; q >= 0; q-- {
		res.WriteRune(e.alphabet[digits[q]])
	}

	return res.String()
}

// Decode function decodes a string previously obtained from Encode, using the same alphabet and returns a byte slice
// In case the input is not valid an arror will be returned
func (e *Encoding) Decode(source string) ([]byte, error) {
	if len(source) == 0 {
		return []byte{}, nil
	}

	runes := []rune(source)

	bytes := []byte{0}
	for i := 0; i < len(runes); i++ {
		value, ok := e.alphabetMap[runes[i]]

		if !ok {
			return nil, ErrNonBaseChar
		}

		carry := int(value)

		for j := 0; j < len(bytes); j++ {
			carry += int(bytes[j]) * e.base
			bytes[j] = byte(carry & 0xff)
			carry >>= 8
		}

		for carry > 0 {
			bytes = append(bytes, byte(carry&0xff))
			carry >>= 8
		}
	}

	for k := 0; runes[k] == e.alphabet[0] && k < len(runes)-1; k++ {
		bytes = append(bytes, 0)
	}

	// Reverse bytes
	for i, j := 0, len(bytes)-1; i < j; i, j = i+1, j-1 {
		bytes[i], bytes[j] = bytes[j], bytes[i]
	}

	return bytes, nil
}
