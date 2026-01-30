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

	Based on the Metasploit x64/xor encoder by stephen_fewer@harmonysecurity.com

*/

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/moloch--/go-keystone"
)

const (
	xorKeySize         = 8
	xorBlockSize       = 8
	decoderPayloadSize = 0x27
)

// Xor encodes an amd64 payload using the Metasploit x64/xor scheme.
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
	engine, err := keystone.NewEngine(keystone.ARCH_X86, keystone.MODE_64)
	if err != nil {
		return nil, err
	}
	defer func() { _ = engine.Close() }()

	err = engine.Option(keystone.OPT_SYNTAX, keystone.OPT_SYNTAX_INTEL)
	if err != nil {
		return nil, err
	}

	src := strings.Join([]string{
		".code64",
		"xor rcx, rcx",
		"sub rcx, 0x81",
		"lea rax, [rip - 0x11]",
		"mov rbx, 0x1122334455667788",
		"decode:",
		"xor qword ptr [rax + 0x27], rbx",
		"sub rax, 0x80",
		"loop decode",
	}, "\n")
	inst, err := engine.Assemble(src, 0)
	if err != nil {
		return nil, err
	}

	if len(inst) != decoderPayloadSize {
		return nil, fmt.Errorf("xor encoder: unexpected decoder stub length %d", len(inst))
	}

	blockValue := uint32(int32(-blockCount))
	if err := patchImm32(inst, []byte{0x48, 0x81, 0xE9}, blockValue); err != nil {
		return nil, err
	}

	keyOffset := bytes.Index(inst, []byte{0x48, 0xBB})
	if keyOffset == -1 || keyOffset+10 > len(inst) {
		return nil, fmt.Errorf("xor encoder: failed to locate key offset")
	}
	copy(inst[keyOffset+2:keyOffset+10], key)

	if err := patchImm32(inst, []byte{0x48, 0x2D}, 0xFFFFFFF8); err != nil {
		return nil, err
	}

	return inst, nil
}

func patchImm32(stub []byte, opcode []byte, value uint32) error {
	offset := bytes.Index(stub, opcode)
	if offset == -1 {
		return fmt.Errorf("xor encoder: opcode %x not found", opcode)
	}
	offset += len(opcode)
	if offset+4 > len(stub) {
		return fmt.Errorf("xor encoder: opcode %x out of range", opcode)
	}
	binary.LittleEndian.PutUint32(stub[offset:offset+4], value)
	return nil
}
