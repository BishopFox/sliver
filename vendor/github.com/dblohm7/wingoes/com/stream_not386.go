// Copyright (c) 2023 Tailscale Inc & AUTHORS. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows && !386

package com

import (
	"math"
	"syscall"
	"unsafe"

	"github.com/dblohm7/wingoes"
)

const maxStreamRWLen = math.MaxUint32

func (abi *IStreamABI) Seek(offset int64, whence int) (n int64, _ error) {
	var hr wingoes.HRESULT
	method := unsafe.Slice(abi.Vtbl, 14)[5]

	rc, _, _ := syscall.SyscallN(
		method,
		uintptr(unsafe.Pointer(abi)),
		uintptr(offset),
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

	rc, _, _ := syscall.SyscallN(
		method,
		uintptr(unsafe.Pointer(abi)),
		uintptr(newSize),
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

	rc, _, _ := syscall.SyscallN(
		method,
		uintptr(unsafe.Pointer(abi)),
		uintptr(unsafe.Pointer(dest)),
		uintptr(numBytesToCopy),
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

	rc, _, _ := syscall.SyscallN(
		method,
		uintptr(unsafe.Pointer(abi)),
		uintptr(offset),
		uintptr(numBytes),
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

	rc, _, _ := syscall.SyscallN(
		method,
		uintptr(unsafe.Pointer(abi)),
		uintptr(offset),
		uintptr(numBytes),
		uintptr(lockType),
	)
	hr = wingoes.HRESULT(rc)

	if e := wingoes.ErrorFromHRESULT(hr); e.Failed() {
		return e
	}

	return nil
}
