//go:build !darwin && !linux

package platform

const nfdbits = 0x40

// FdSet mocks syscall.FdSet on systems that do not support it.
type FdSet struct {
	Bits [16]int64
}
