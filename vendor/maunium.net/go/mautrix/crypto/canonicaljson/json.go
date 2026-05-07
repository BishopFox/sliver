/* Copyright 2016-2017 Vector Creations Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package canonicaljson

import (
	"encoding/binary"
	"fmt"
	"sort"
	"unicode/utf8"

	"github.com/tidwall/gjson"
)

// CanonicalJSON re-encodes the JSON in a canonical encoding. The encoding is
// the shortest possible encoding using integer values with sorted object keys.
// https://matrix.org/docs/spec/appendices#canonical-json
func CanonicalJSON(input []byte) ([]byte, error) {
	if !gjson.Valid(string(input)) {
		return nil, fmt.Errorf("invalid json")
	}

	return CanonicalJSONAssumeValid(input), nil
}

// CanonicalJSONAssumeValid is the same as CanonicalJSON, but assumes the
// input is valid JSON
func CanonicalJSONAssumeValid(input []byte) []byte {
	input = CompactJSON(input, make([]byte, 0, len(input)))
	return SortJSON(input, make([]byte, 0, len(input)))
}

// SortJSON reencodes the JSON with the object keys sorted by lexicographically
// by codepoint. The input must be valid JSON.
func SortJSON(input, output []byte) []byte {
	result := gjson.ParseBytes(input)

	return sortJSONValue(result, input, output)
}

// sortJSONValue takes a gjson.Result and sorts it. inputJSON must be the
// raw JSON bytes that gjson.Result points to.
func sortJSONValue(input gjson.Result, inputJSON, output []byte) []byte {
	if input.IsArray() {
		return sortJSONArray(input, inputJSON, output)
	}

	if input.IsObject() {
		return sortJSONObject(input, inputJSON, output)
	}

	// If its neither an object nor an array then there is no sub structure
	// to sort, so just append the raw bytes.
	return append(output, input.Raw...)
}

// sortJSONArray takes a gjson.Result and sorts it, assuming its an array.
// inputJSON must be the raw JSON bytes that gjson.Result points to.
func sortJSONArray(input gjson.Result, inputJSON, output []byte) []byte {
	sep := byte('[')

	// Iterate over each value in the array and sort it.
	input.ForEach(func(_, value gjson.Result) bool {
		output = append(output, sep)
		sep = ','
		output = sortJSONValue(value, inputJSON, output)
		return true // keep iterating
	})

	if sep == '[' {
		// If sep is still '[' then the array was empty and we never wrote the
		// initial '[', so we write it now along with the closing ']'.
		output = append(output, '[', ']')
	} else {
		// Otherwise we end the array by writing a single ']'
		output = append(output, ']')
	}
	return output
}

// sortJSONObject takes a gjson.Result and sorts it, assuming its an object.
// inputJSON must be the raw JSON bytes that gjson.Result points to.
func sortJSONObject(input gjson.Result, inputJSON, output []byte) []byte {
	type entry struct {
		key    string // The parsed key string
		rawKey string // The raw, unparsed key JSON string
		value  gjson.Result
	}

	var entries []entry

	// Iterate over each key/value pair and add it to a slice
	// that we can sort
	input.ForEach(func(key, value gjson.Result) bool {
		entries = append(entries, entry{
			key:    key.String(),
			rawKey: key.Raw,
			value:  value,
		})
		return true // keep iterating
	})

	// Sort the slice based on the *parsed* key
	sort.Slice(entries, func(a, b int) bool {
		return entries[a].key < entries[b].key
	})

	sep := byte('{')

	for _, entry := range entries {
		output = append(output, sep)
		sep = ','

		// Append the raw unparsed JSON key, *not* the parsed key
		output = append(output, entry.rawKey...)
		output = append(output, ':')
		output = sortJSONValue(entry.value, inputJSON, output)
	}
	if sep == '{' {
		// If sep is still '{' then the object was empty and we never wrote the
		// initial '{', so we write it now along with the closing '}'.
		output = append(output, '{', '}')
	} else {
		// Otherwise we end the object by writing a single '}'
		output = append(output, '}')
	}
	return output
}

// CompactJSON makes the encoded JSON as small as possible by removing
// whitespace and unneeded unicode escapes
func CompactJSON(input, output []byte) []byte {
	var i int
	for i < len(input) {
		c := input[i]
		i++
		// The valid whitespace characters are all less than or equal to SPACE 0x20.
		// The valid non-white characters are all greater than SPACE 0x20.
		// So we can check for whitespace by comparing against SPACE 0x20.
		if c <= ' ' {
			// Skip over whitespace.
			continue
		}
		// Add the non-whitespace character to the output.
		output = append(output, c)
		if c == '"' {
			// We are inside a string.
			for i < len(input) {
				c = input[i]
				i++
				// Check if this is an escape sequence.
				if c == '\\' {
					escape := input[i]
					i++
					if escape == 'u' {
						// If this is a unicode escape then we need to handle it specially
						output, i = compactUnicodeEscape(input, output, i)
					} else if escape == '/' {
						// JSON does not require escaping '/', but allows encoders to escape it as a special case.
						// Since the escape isn't required we remove it.
						output = append(output, escape)
					} else {
						// All other permitted escapes are single charater escapes that are already in their shortest form.
						output = append(output, '\\', escape)
					}
				} else {
					output = append(output, c)
				}
				if c == '"' {
					break
				}
			}
		}
	}
	return output
}

// compactUnicodeEscape unpacks a 4 byte unicode escape starting at index.
// If the escape is a surrogate pair then decode the 6 byte \uXXXX escape
// that follows. Returns the output slice and a new input index.
func compactUnicodeEscape(input, output []byte, index int) ([]byte, int) {
	const (
		ESCAPES = "uuuuuuuubtnufruuuuuuuuuuuuuuuuuu"
		HEX     = "0123456789ABCDEF"
	)
	// If there aren't enough bytes to decode the hex escape then return.
	if len(input)-index < 4 {
		return output, len(input)
	}
	// Decode the 4 hex digits.
	c := readHexDigits(input[index:])
	index += 4
	if c < ' ' {
		// If the character is less than SPACE 0x20 then it will need escaping.
		escape := ESCAPES[c]
		output = append(output, '\\', escape)
		if escape == 'u' {
			output = append(output, '0', '0', byte('0'+(c>>4)), HEX[c&0xF])
		}
	} else if c == '\\' || c == '"' {
		// Otherwise the character only needs escaping if it is a QUOTE '"' or BACKSLASH '\\'.
		output = append(output, '\\', byte(c))
	} else if c < 0xD800 || c >= 0xE000 {
		// If the character isn't a surrogate pair then encoded it directly as UTF-8.
		var buffer [4]byte
		n := utf8.EncodeRune(buffer[:], rune(c))
		output = append(output, buffer[:n]...)
	} else {
		// Otherwise the escaped character was the first part of a UTF-16 style surrogate pair.
		// The next 6 bytes MUST be a '\uXXXX'.
		// If there aren't enough bytes to decode the hex escape then return.
		if len(input)-index < 6 {
			return output, len(input)
		}
		// Decode the 4 hex digits from the '\uXXXX'.
		surrogate := readHexDigits(input[index+2:])
		index += 6
		// Reconstruct the UCS4 codepoint from the surrogates.
		codepoint := 0x10000 + (((c & 0x3FF) << 10) | (surrogate & 0x3FF))
		// Encode the charater as UTF-8.
		var buffer [4]byte
		n := utf8.EncodeRune(buffer[:], rune(codepoint))
		output = append(output, buffer[:n]...)
	}
	return output, index
}

// Read 4 hex digits from the input slice.
// Taken from https://github.com/NegativeMjark/indolentjson-rust/blob/8b959791fe2656a88f189c5d60d153be05fe3deb/src/readhex.rs#L21
func readHexDigits(input []byte) uint32 {
	hex := binary.BigEndian.Uint32(input)
	// subtract '0'
	hex -= 0x30303030
	// strip the higher bits, maps 'a' => 'A'
	hex &= 0x1F1F1F1F
	mask := hex & 0x10101010
	// subtract 'A' - 10 - '9' - 9 = 7 from the letters.
	hex -= mask >> 1
	hex += mask >> 4
	// collect the nibbles
	hex |= hex >> 4
	hex &= 0xFF00FF
	hex |= hex >> 8
	return hex & 0xFFFF
}
