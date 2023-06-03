package platform

import "syscall"

const nfdbits = 0x20

// FdSet re-exports syscall.FdSet with utility methods.
type FdSet syscall.FdSet
