// Copyright (c) 2023 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows

package com

import (
	"math"
	"syscall"
	"unsafe"

	"github.com/dblohm7/wingoes"
)

const maxStreamRWLen = math.MaxInt32

func (abi *IStreamABI) Seek(offset int64, whence int) (n int64, _ error) {
	var hr wingoes.HRESULT
	method := unsafe.Slice(abi.Vtbl, 14)[5]

	words := (*[2]uintptr)(unsafe.Pointer(&offset))
	rc, _, _ := syscall.SyscallN(
		method,
		uintptr(unsafe.Pointer(abi)),
		words[0],
		words[1],
		uintptr(uint32(whence)),
		uintptr(unsafe.Pointer(&n)),
	)
	hr = wingoes.HRESULT(rc)

	if e := wingoes.ErrorFromHRESULT(hr); e.Failed() {
		return 0, e
	}

	return n, nil
}

func (abi *IStreamABI) SetSize(newSize uint64) error {
	var hr wingoes.HRESULT
	method := unsafe.Slice(abi.Vtbl, 14)[6]

	words := (*[2]uintptr)(unsafe.Pointer(&newSize))
	rc, _, _ := syscall.SyscallN(
		method,
		uintptr(unsafe.Pointer(abi)),
		words[0],
		words[1],
	)
	hr = wingoes.HRESULT(rc)

	if e := wingoes.ErrorFromHRESULT(hr); e.Failed() {
		return e
	}

	return nil
}

func (abi *IStreamABI) CopyTo(dest *IStreamABI, numBytesToCopy uint64) (bytesRead, bytesWritten uint64, _ error) {
	var hr wingoes.HRESULT
	method := unsafe.Slice(abi.Vtbl, 14)[7]

	words := (*[2]uintptr)(unsafe.Pointer(&numBytesToCopy))
	rc, _, _ := syscall.SyscallN(
		method,
		uintptr(unsafe.Pointer(abi)),
		uintptr(unsafe.Pointer(dest)),
		words[0],
		words[1],
		uintptr(unsafe.Pointer(&bytesRead)),
		uintptr(unsafe.Pointer(&bytesWritten)),
	)
	hr = wingoes.HRESULT(rc)

	if e := wingoes.ErrorFromHRESULT(hr); e.Failed() {
		return bytesRead, bytesWritten, e
	}

	return bytesRead, bytesWritten, nil
}

func (abi *IStreamABI) LockRegion(offset, numBytes uint64, lockType LOCKTYPE) error {
	var hr wingoes.HRESULT
	method := unsafe.Slice(abi.Vtbl, 14)[10]

	oWords := (*[2]uintptr)(unsafe.Pointer(&offset))
	nWords := (*[2]uintptr)(unsafe.Pointer(&numBytes))
	rc, _, _ := syscall.SyscallN(
		method,
		uintptr(unsafe.Pointer(abi)),
		oWords[0],
		oWords[1],
		nWords[0],
		nWords[1],
		uintptr(lockType),
	)
	hr = wingoes.HRESULT(rc)

	if e := wingoes.ErrorFromHRESULT(hr); e.Failed() {
		return e
	}

	return nil
}

func (abi *IStreamABI) UnlockRegion(offset, numBytes uint64, lockType LOCKTYPE) error {
	var hr wingoes.HRESULT
	method := unsafe.Slice(abi.Vtbl, 14)[11]

	oWords := (*[2]uintptr)(unsafe.Pointer(&offset))
	nWords := (*[2]uintptr)(unsafe.Pointer(&numBytes))
	rc, _, _ := syscall.SyscallN(
		method,
		uintptr(unsafe.Pointer(abi)),
		oWords[0],
		oWords[1],
		nWords[0],
		nWords[1],
		uintptr(lockType),
	)
	hr = wingoes.HRESULT(rc)

	if e := wingoes.ErrorFromHRESULT(hr); e.Failed() {
		return e
	}

	return nil
}
