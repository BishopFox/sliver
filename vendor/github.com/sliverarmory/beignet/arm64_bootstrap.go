package beignet

import (
	"fmt"
	"strings"
	"sync"

	keystone "github.com/moloch--/go-keystone"
)

const arm64BootstrapLen = 21 * 4

var (
	arm64KeystoneOnce sync.Once
	arm64KeystoneEng  *keystone.Engine
	arm64KeystoneErr  error
	arm64KeystoneMu   sync.Mutex
)

func arm64Assembler() (*keystone.Engine, error) {
	arm64KeystoneOnce.Do(func() {
		arm64KeystoneEng, arm64KeystoneErr = keystone.NewEngine(keystone.ARCH_ARM64, keystone.MODE_LITTLE_ENDIAN)
	})
	return arm64KeystoneEng, arm64KeystoneErr
}

func assembleArm64(src string) ([]byte, error) {
	eng, err := arm64Assembler()
	if err != nil {
		return nil, err
	}
	arm64KeystoneMu.Lock()
	defer arm64KeystoneMu.Unlock()
	return eng.Assemble(src, 0)
}

func emitMovImm64(sb *strings.Builder, reg string, imm uint64) {
	lo := uint16(imm & 0xffff)
	m16 := uint16((imm >> 16) & 0xffff)
	m32 := uint16((imm >> 32) & 0xffff)
	m48 := uint16((imm >> 48) & 0xffff)

	fmt.Fprintf(sb, "movz %s, #0x%X\n", reg, lo)
	fmt.Fprintf(sb, "movk %s, #0x%X, lsl #16\n", reg, m16)
	fmt.Fprintf(sb, "movk %s, #0x%X, lsl #32\n", reg, m32)
	fmt.Fprintf(sb, "movk %s, #0x%X, lsl #48\n", reg, m48)
}

// buildArm64Bootstrap returns a small aarch64 stub which:
// - sets x0 = base + payloadOffset
// - sets x1 = payloadSize
// - sets x2 = base + symbolOffset
// - jumps to base + loaderEntryOffsetAbs using br
//
// The loader expects payload pointer/size in x0/x1 and the entry symbol pointer in x2.
func buildArm64Bootstrap(payloadOffset, payloadSize, symbolOffset, loaderEntryOffsetAbs uint64) ([]byte, error) {
	var sb strings.Builder
	sb.Grow(512)

	sb.WriteString("adr x9, #0\n")

	emitMovImm64(&sb, "x0", payloadOffset)
	sb.WriteString("add x0, x0, x9\n")

	emitMovImm64(&sb, "x1", payloadSize)

	emitMovImm64(&sb, "x2", symbolOffset)
	sb.WriteString("add x2, x2, x9\n")

	emitMovImm64(&sb, "x16", loaderEntryOffsetAbs)
	sb.WriteString("add x16, x16, x9\n")

	sb.WriteString("br x16\n")

	b, err := assembleArm64(sb.String())
	if err != nil {
		return nil, err
	}
	if len(b) != arm64BootstrapLen {
		return nil, fmt.Errorf("beignet: unexpected bootstrap length: got=%d want=%d", len(b), arm64BootstrapLen)
	}
	return b, nil
}

