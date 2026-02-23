package beignet

import (
	"fmt"
	"strings"
	"sync"

	keystone "github.com/moloch--/go-keystone"
)

const amd64BootstrapLen = 59

var (
	amd64KeystoneOnce sync.Once
	amd64KeystoneEng  *keystone.Engine
	amd64KeystoneErr  error
	amd64KeystoneMu   sync.Mutex
)

func amd64Assembler() (*keystone.Engine, error) {
	amd64KeystoneOnce.Do(func() {
		amd64KeystoneEng, amd64KeystoneErr = keystone.NewEngine(keystone.ARCH_X86, keystone.MODE_64)
		if amd64KeystoneErr != nil {
			return
		}
		amd64KeystoneErr = amd64KeystoneEng.Option(keystone.OPT_SYNTAX, keystone.OPT_SYNTAX_INTEL)
	})
	return amd64KeystoneEng, amd64KeystoneErr
}

func assembleAMD64(src string) ([]byte, error) {
	eng, err := amd64Assembler()
	if err != nil {
		return nil, err
	}
	amd64KeystoneMu.Lock()
	defer amd64KeystoneMu.Unlock()
	return eng.Assemble(src, 0)
}

// buildAMD64Bootstrap returns a small x86_64 stub which:
// - sets rdi = base + payloadOffset
// - sets rsi = payloadSize
// - sets rdx = base + symbolOffset
// - calls base + loaderEntryOffsetAbs (then returns to the caller)
func buildAMD64Bootstrap(payloadOffset, payloadSize, symbolOffset, loaderEntryOffsetAbs uint64) ([]byte, error) {
	var sb strings.Builder
	sb.Grow(256)
	sb.WriteString(".code64\n")

	// Recover the shellcode base as the start of this bootstrap.
	sb.WriteString("lea r9, [rip - 7]\n")

	// rdi = base + payloadOffset
	fmt.Fprintf(&sb, "movabs rdi, 0x%X\n", payloadOffset)
	sb.WriteString("add rdi, r9\n")

	// rsi = payloadSize
	fmt.Fprintf(&sb, "movabs rsi, 0x%X\n", payloadSize)

	// rdx = base + symbolOffset
	fmt.Fprintf(&sb, "movabs rdx, 0x%X\n", symbolOffset)
	sb.WriteString("add rdx, r9\n")

	// rax = base + loaderEntryOffsetAbs
	fmt.Fprintf(&sb, "movabs rax, 0x%X\n", loaderEntryOffsetAbs)
	sb.WriteString("add rax, r9\n")
	sb.WriteString("call rax\n") // keeps stack alignment for System V ABI
	sb.WriteString("ret\n")

	b, err := assembleAMD64(sb.String())
	if err != nil {
		return nil, err
	}

	if len(b) != amd64BootstrapLen {
		return nil, fmt.Errorf("beignet: unexpected bootstrap length: got=%d want=%d", len(b), amd64BootstrapLen)
	}
	return b, nil
}
