// Copyright 2010 The win Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package win

import (
	"syscall"
	"unsafe"
)

func (idProp *MSAAPROPID) split() (uintptr, uintptr, uintptr, uintptr) {
	if idProp == nil {
		return 0, 0, 0, 0
	}
	x := (*struct { a, b, c, d uintptr })(unsafe.Pointer(idProp))
	return x.a, x.b, x.c, x.d
}

// SetPropValue identifies the accessible element to be annotated, specify the property to be annotated, and provide a new value for that property.
// If server developers know the HWND of the accessible element they want to annotate, they can use one of the following methods: SetHwndPropStr, SetHwndProp, or SetHwndPropServer
func (obj *IAccPropServices) SetPropValue(idString []byte, idProp *MSAAPROPID, v *VARIANT) HRESULT {
	var idStringPtr unsafe.Pointer
	idStringLen := len(idString)
	if idStringLen != 0 {
		idStringPtr = unsafe.Pointer(&idString[0])
	}
	propA, propB, propC, propD := idProp.split()
	ret, _, _ := syscall.Syscall9(obj.LpVtbl.SetPropValue, 8,
		uintptr(unsafe.Pointer(obj)),
		uintptr(idStringPtr),
		uintptr(idStringLen),
		propA, propB, propC, propD,
		uintptr(unsafe.Pointer(v)),
		0)
	return HRESULT(ret)
}

// SetHwndProp wraps SetPropValue, providing a convenient entry point for callers who are annotating HWND-based accessible elements. If the new value is a string, you can use SetHwndPropStr instead.
func (obj *IAccPropServices) SetHwndProp(hwnd HWND, idObject int32, idChild uint32, idProp *MSAAPROPID, v *VARIANT) HRESULT {
	propA, propB, propC, propD := idProp.split()
	ret, _, _ := syscall.Syscall9(obj.LpVtbl.SetHwndProp, 9,
		uintptr(unsafe.Pointer(obj)),
		uintptr(hwnd),
		uintptr(idObject),
		uintptr(idChild),
		propA, propB, propC, propD,
		uintptr(unsafe.Pointer(v)))
	return HRESULT(ret)
}

// SetHwndPropStr wraps SetPropValue, providing a more convenient entry point for callers who are annotating HWND-based accessible elements.
func (obj *IAccPropServices) SetHwndPropStr(hwnd HWND, idObject int32, idChild uint32, idProp *MSAAPROPID, str string) HRESULT {
	str16, err := syscall.UTF16PtrFromString(str)
	if err != nil {
		return -((E_INVALIDARG ^ 0xFFFFFFFF) + 1)
	}
	propA, propB, propC, propD := idProp.split()
	ret, _, _ := syscall.Syscall9(obj.LpVtbl.SetHwndPropStr, 9,
		uintptr(unsafe.Pointer(obj)),
		uintptr(hwnd),
		uintptr(idObject),
		uintptr(idChild),
		propA, propB, propC, propD,
		uintptr(unsafe.Pointer(str16)))
	return HRESULT(ret)
}

// SetHmenuProp wraps SetPropValue, providing a convenient entry point for callers who are annotating HMENU-based accessible elements. If the new value is a string, you can use IAccPropServices::SetHmenuPropStr instead.
func (obj *IAccPropServices) SetHmenuProp(hmenu HMENU, idChild uint32, idProp *MSAAPROPID, v *VARIANT) HRESULT {
	propA, propB, propC, propD := idProp.split()
	ret, _, _ := syscall.Syscall9(obj.LpVtbl.SetHmenuProp, 8,
		uintptr(unsafe.Pointer(obj)),
		uintptr(hmenu),
		uintptr(idChild),
		propA, propB, propC, propD,
		uintptr(unsafe.Pointer(v)),
		0)
	return HRESULT(ret)
}

// SetHmenuPropStr wraps SetPropValue, providing a more convenient entry point for callers who are annotating HMENU-based accessible elements.
func (obj *IAccPropServices) SetHmenuPropStr(hmenu HMENU, idChild uint32, idProp *MSAAPROPID, str string) HRESULT {
	str16, err := syscall.UTF16PtrFromString(str)
	if err != nil {
		return -((E_INVALIDARG ^ 0xFFFFFFFF) + 1)
	}
	propA, propB, propC, propD := idProp.split()
	ret, _, _ := syscall.Syscall9(obj.LpVtbl.SetHmenuPropStr, 8,
		uintptr(unsafe.Pointer(obj)),
		uintptr(hmenu),
		uintptr(idChild),
		propA, propB, propC, propD,
		uintptr(unsafe.Pointer(str16)),
		0)
	return HRESULT(ret)
}
