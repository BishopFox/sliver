package amd64

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
	------------------------------------------------------------------------

	Based on the Metasploit x64/xor_dynamic encoder by lupman, phra

*/

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/moloch--/go-keystone"
)

const (
	xorDynamicKeyPlaceholder     = 0x41
	xorDynamicPayloadPlaceholder = 0x42
	xorDynamicStubSize           = 46
)

var xorDynamicBadchars = map[byte]bool{
	0x00: true,
	0x0a: true,
	0x0d: true,
}

// XorDynamic encodes an amd64 payload using the Metasploit x64/xor_dynamic scheme.
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

	if containsBadchars(final, xorDynamicBadchars) {
		return nil, fmt.Errorf("xor_dynamic encoder: badchars present in output")
	}

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

	engine, err := keystone.NewEngine(keystone.ARCH_X86, keystone.MODE_64)
	if err != nil {
		return nil, err
	}
	defer func() { _ = engine.Close() }()

	if err := engine.Option(keystone.OPT_SYNTAX, keystone.OPT_SYNTAX_INTEL); err != nil {
		return nil, err
	}

	src := strings.Join([]string{
		".code64",
		"jmp _call",
		"_ret:",
		"pop rbx",
		"push rbx",
		"pop rdi",
		"mov al, 0x41",
		"cld",
		"_lp1:",
		"scasb",
		"jne _lp1",
		"push rdi",
		"pop rcx",
		"_lp2:",
		"push rbx",
		"pop rsi",
		"_lp3:",
		"mov al, byte ptr [rsi]",
		"xor byte ptr [rdi], al",
		"inc rdi",
		"inc rsi",
		"cmp word ptr [rdi], 0x4242",
		"je _jmp",
		"cmp byte ptr [rsi], 0x41",
		"jne _lp3",
		"jmp _lp2",
		"_jmp:",
		"jmp rcx",
		"_call:",
		"call _ret",
	}, "\n")

	inst, err := engine.Assemble(src, 0)
	if err != nil {
		return nil, err
	}

	if len(inst) != xorDynamicStubSize {
		return nil, fmt.Errorf("xor_dynamic encoder: unexpected stub length %d", len(inst))
	}

	keyReplaced := 0
	for i := range inst {
		if inst[i] == xorDynamicKeyPlaceholder {
			inst[i] = keyTerm
			keyReplaced++
		}
	}
	if keyReplaced == 0 {
		return nil, fmt.Errorf("xor_dynamic encoder: key placeholder not found")
	}

	payloadReplaced := 0
	for i := 0; i+1 < len(inst); i++ {
		if inst[i] == xorDynamicPayloadPlaceholder && inst[i+1] == xorDynamicPayloadPlaceholder {
			inst[i] = payloadTerm[0]
			inst[i+1] = payloadTerm[1]
			payloadReplaced++
			i++
		}
	}
	if payloadReplaced == 0 {
		return nil, fmt.Errorf("xor_dynamic encoder: payload placeholder not found")
	}

	if containsBadchars(inst, xorDynamicBadchars) {
		return nil, fmt.Errorf("xor_dynamic encoder: badchars present in stub")
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
