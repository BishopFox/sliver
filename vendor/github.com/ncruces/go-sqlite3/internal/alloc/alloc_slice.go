//go:build !(darwin || linux) || !(amd64 || arm64 || riscv64 || ppc64le) || sqlite3_noshm || sqlite3_nosys

package alloc

import "github.com/tetratelabs/wazero/experimental"

func Slice(cap, _ uint64) experimental.LinearMemory {
	return &sliceMemory{make([]byte, 0, cap)}
}

type sliceMemory struct {
	buf []byte
}

func (b *sliceMemory) Free() {}

func (b *sliceMemory) Reallocate(size uint64) []byte {
	if cap := uint64(cap(b.buf)); size > cap {
		b.buf = append(b.buf[:cap], make([]byte, size-cap)...)
	} else {
		b.buf = b.buf[:size]
	}
	return b.buf
}
