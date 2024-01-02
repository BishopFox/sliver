// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build darwin || freebsd || linux || windows

package purego

import (
	"unsafe"
)

//go:linkname runtime_cgocall runtime.cgocall
func runtime_cgocall(fn uintptr, arg unsafe.Pointer) int32 // from runtime/sys_libc.go

//go:linkname runtime_noescape runtime.noescape
//go:noescape
func runtime_noescape(p unsafe.Pointer) unsafe.Pointer // from runtime/stubs.go
