// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build cgo

package purego

import (
	_ "unsafe" // for go:linkname

	"github.com/ebitengine/purego/internal/cgo"
)

var syscall9XABI0 = uintptr(cgo.Syscall9XABI0)

// this is only here to make the assembly files happy :)
type syscall9Args struct {
	fn, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr
	f1, f2, f3, f4, f5, f6, f7, f8         uintptr
	r1, r2, err                            uintptr
}

//go:nosplit
func syscall_syscall9X(fn, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) (r1, r2, err uintptr) {
	return cgo.Syscall9X(fn, a1, a2, a3, a4, a5, a6, a7, a8, a9)
}

func NewCallback(_ interface{}) uintptr {
	panic("purego: NewCallback not supported")
}
