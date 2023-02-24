// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

package purego

import (
	"syscall"
)

func syscall_syscall9X(fn, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) (r1, r2, err uintptr) {
	r1, r2, errno := syscall.Syscall9(fn, 9, a1, a2, a3, a4, a5, a6, a7, a8, a9)
	return r1, r2, uintptr(errno)
}

// NewCallback converts a Go function to a function pointer conforming to the stdcall calling convention.
// This is useful when interoperating with Windows code requiring callbacks. The argument is expected to be a
// function with one uintptr-sized result. The function must not have arguments with size larger than the
// size of uintptr. Only a limited number of callbacks may be created in a single Go process, and any memory
// allocated for these callbacks is never released. Between NewCallback and NewCallbackCDecl, at least 1024
// callbacks can always be created. Although this function is similiar to the darwin version it may act
// differently.
func NewCallback(fn interface{}) uintptr {
	return syscall.NewCallback(fn)
}
