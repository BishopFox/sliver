// Copyright 2010 The win Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package win

import (
	"syscall"
	"unsafe"
)

type AnnoScope int

const (
	ANNO_THIS      = AnnoScope(0)
	ANNO_CONTAINER = AnnoScope(1)
)

type MSAAPROPID syscall.GUID

var (
	PROPID_ACC_NAME             = MSAAPROPID{0x608d3df8, 0x8128, 0x4aa7, [8]byte{0xa4, 0x28, 0xf5, 0x5e, 0x49, 0x26, 0x72, 0x91}}
	PROPID_ACC_VALUE            = MSAAPROPID{0x123fe443, 0x211a, 0x4615, [8]byte{0x95, 0x27, 0xc4, 0x5a, 0x7e, 0x93, 0x71, 0x7a}}
	PROPID_ACC_DESCRIPTION      = MSAAPROPID{0x4d48dfe4, 0xbd3f, 0x491f, [8]byte{0xa6, 0x48, 0x49, 0x2d, 0x6f, 0x20, 0xc5, 0x88}}
	PROPID_ACC_ROLE             = MSAAPROPID{0xcb905ff2, 0x7bd1, 0x4c05, [8]byte{0xb3, 0xc8, 0xe6, 0xc2, 0x41, 0x36, 0x4d, 0x70}}
	PROPID_ACC_STATE            = MSAAPROPID{0xa8d4d5b0, 0x0a21, 0x42d0, [8]byte{0xa5, 0xc0, 0x51, 0x4e, 0x98, 0x4f, 0x45, 0x7b}}
	PROPID_ACC_HELP             = MSAAPROPID{0xc831e11f, 0x44db, 0x4a99, [8]byte{0x97, 0x68, 0xcb, 0x8f, 0x97, 0x8b, 0x72, 0x31}}
	PROPID_ACC_KEYBOARDSHORTCUT = MSAAPROPID{0x7d9bceee, 0x7d1e, 0x4979, [8]byte{0x93, 0x82, 0x51, 0x80, 0xf4, 0x17, 0x2c, 0x34}}
	PROPID_ACC_DEFAULTACTION    = MSAAPROPID{0x180c072b, 0xc27f, 0x43c7, [8]byte{0x99, 0x22, 0xf6, 0x35, 0x62, 0xa4, 0x63, 0x2b}}
	PROPID_ACC_HELPTOPIC        = MSAAPROPID{0x787d1379, 0x8ede, 0x440b, [8]byte{0x8a, 0xec, 0x11, 0xf7, 0xbf, 0x90, 0x30, 0xb3}}
	PROPID_ACC_FOCUS            = MSAAPROPID{0x6eb335df, 0x1c29, 0x4127, [8]byte{0xb1, 0x2c, 0xde, 0xe9, 0xfd, 0x15, 0x7f, 0x2b}}
	PROPID_ACC_SELECTION        = MSAAPROPID{0xb99d073c, 0xd731, 0x405b, [8]byte{0x90, 0x61, 0xd9, 0x5e, 0x8f, 0x84, 0x29, 0x84}}
	PROPID_ACC_PARENT           = MSAAPROPID{0x474c22b6, 0xffc2, 0x467a, [8]byte{0xb1, 0xb5, 0xe9, 0x58, 0xb4, 0x65, 0x73, 0x30}}
	PROPID_ACC_NAV_UP           = MSAAPROPID{0x016e1a2b, 0x1a4e, 0x4767, [8]byte{0x86, 0x12, 0x33, 0x86, 0xf6, 0x69, 0x35, 0xec}}
	PROPID_ACC_NAV_DOWN         = MSAAPROPID{0x031670ed, 0x3cdf, 0x48d2, [8]byte{0x96, 0x13, 0x13, 0x8f, 0x2d, 0xd8, 0xa6, 0x68}}
	PROPID_ACC_NAV_LEFT         = MSAAPROPID{0x228086cb, 0x82f1, 0x4a39, [8]byte{0x87, 0x05, 0xdc, 0xdc, 0x0f, 0xff, 0x92, 0xf5}}
	PROPID_ACC_NAV_RIGHT        = MSAAPROPID{0xcd211d9f, 0xe1cb, 0x4fe5, [8]byte{0xa7, 0x7c, 0x92, 0x0b, 0x88, 0x4d, 0x09, 0x5b}}
	PROPID_ACC_NAV_PREV         = MSAAPROPID{0x776d3891, 0xc73b, 0x4480, [8]byte{0xb3, 0xf6, 0x07, 0x6a, 0x16, 0xa1, 0x5a, 0xf6}}
	PROPID_ACC_NAV_NEXT         = MSAAPROPID{0x1cdc5455, 0x8cd9, 0x4c92, [8]byte{0xa3, 0x71, 0x39, 0x39, 0xa2, 0xfe, 0x3e, 0xee}}
	PROPID_ACC_NAV_FIRSTCHILD   = MSAAPROPID{0xcfd02558, 0x557b, 0x4c67, [8]byte{0x84, 0xf9, 0x2a, 0x09, 0xfc, 0xe4, 0x07, 0x49}}
	PROPID_ACC_NAV_LASTCHILD    = MSAAPROPID{0x302ecaa5, 0x48d5, 0x4f8d, [8]byte{0xb6, 0x71, 0x1a, 0x8d, 0x20, 0xa7, 0x78, 0x32}}
	PROPID_ACC_ROLEMAP          = MSAAPROPID{0xf79acda2, 0x140d, 0x4fe6, [8]byte{0x89, 0x14, 0x20, 0x84, 0x76, 0x32, 0x82, 0x69}}
	PROPID_ACC_VALUEMAP         = MSAAPROPID{0xda1c3d79, 0xfc5c, 0x420e, [8]byte{0xb3, 0x99, 0x9d, 0x15, 0x33, 0x54, 0x9e, 0x75}}
	PROPID_ACC_STATEMAP         = MSAAPROPID{0x43946c5e, 0x0ac0, 0x4042, [8]byte{0xb5, 0x25, 0x07, 0xbb, 0xdb, 0xe1, 0x7f, 0xa7}}
	PROPID_ACC_DESCRIPTIONMAP   = MSAAPROPID{0x1ff1435f, 0x8a14, 0x477b, [8]byte{0xb2, 0x26, 0xa0, 0xab, 0xe2, 0x79, 0x97, 0x5d}}
	PROPID_ACC_DODEFAULTACTION  = MSAAPROPID{0x1ba09523, 0x2e3b, 0x49a6, [8]byte{0xa0, 0x59, 0x59, 0x68, 0x2a, 0x3c, 0x48, 0xfd}}
)

const (
	STATE_SYSTEM_NORMAL          = 0
	STATE_SYSTEM_UNAVAILABLE     = 0x1
	STATE_SYSTEM_SELECTED        = 0x2
	STATE_SYSTEM_FOCUSED         = 0x4
	STATE_SYSTEM_PRESSED         = 0x8
	STATE_SYSTEM_CHECKED         = 0x10
	STATE_SYSTEM_MIXED           = 0x20
	STATE_SYSTEM_INDETERMINATE   = STATE_SYSTEM_MIXED
	STATE_SYSTEM_READONLY        = 0x40
	STATE_SYSTEM_HOTTRACKED      = 0x80
	STATE_SYSTEM_DEFAULT         = 0x100
	STATE_SYSTEM_EXPANDED        = 0x200
	STATE_SYSTEM_COLLAPSED       = 0x400
	STATE_SYSTEM_BUSY            = 0x800
	STATE_SYSTEM_FLOATING        = 0x1000
	STATE_SYSTEM_MARQUEED        = 0x2000
	STATE_SYSTEM_ANIMATED        = 0x4000
	STATE_SYSTEM_INVISIBLE       = 0x8000
	STATE_SYSTEM_OFFSCREEN       = 0x10000
	STATE_SYSTEM_SIZEABLE        = 0x20000
	STATE_SYSTEM_MOVEABLE        = 0x40000
	STATE_SYSTEM_SELFVOICING     = 0x80000
	STATE_SYSTEM_FOCUSABLE       = 0x100000
	STATE_SYSTEM_SELECTABLE      = 0x200000
	STATE_SYSTEM_LINKED          = 0x400000
	STATE_SYSTEM_TRAVERSED       = 0x800000
	STATE_SYSTEM_MULTISELECTABLE = 0x1000000
	STATE_SYSTEM_EXTSELECTABLE   = 0x2000000
	STATE_SYSTEM_ALERT_LOW       = 0x4000000
	STATE_SYSTEM_ALERT_MEDIUM    = 0x8000000
	STATE_SYSTEM_ALERT_HIGH      = 0x10000000
	STATE_SYSTEM_PROTECTED       = 0x20000000
	STATE_SYSTEM_HASPOPUP        = 0x40000000
	STATE_SYSTEM_VALID           = 0x7fffffff
)

const (
	ROLE_SYSTEM_TITLEBAR           = 0x1
	ROLE_SYSTEM_MENUBAR            = 0x2
	ROLE_SYSTEM_SCROLLBAR          = 0x3
	ROLE_SYSTEM_GRIP               = 0x4
	ROLE_SYSTEM_SOUND              = 0x5
	ROLE_SYSTEM_CURSOR             = 0x6
	ROLE_SYSTEM_CARET              = 0x7
	ROLE_SYSTEM_ALERT              = 0x8
	ROLE_SYSTEM_WINDOW             = 0x9
	ROLE_SYSTEM_CLIENT             = 0xa
	ROLE_SYSTEM_MENUPOPUP          = 0xb
	ROLE_SYSTEM_MENUITEM           = 0xc
	ROLE_SYSTEM_TOOLTIP            = 0xd
	ROLE_SYSTEM_APPLICATION        = 0xe
	ROLE_SYSTEM_DOCUMENT           = 0xf
	ROLE_SYSTEM_PANE               = 0x10
	ROLE_SYSTEM_CHART              = 0x11
	ROLE_SYSTEM_DIALOG             = 0x12
	ROLE_SYSTEM_BORDER             = 0x13
	ROLE_SYSTEM_GROUPING           = 0x14
	ROLE_SYSTEM_SEPARATOR          = 0x15
	ROLE_SYSTEM_TOOLBAR            = 0x16
	ROLE_SYSTEM_STATUSBAR          = 0x17
	ROLE_SYSTEM_TABLE              = 0x18
	ROLE_SYSTEM_COLUMNHEADER       = 0x19
	ROLE_SYSTEM_ROWHEADER          = 0x1a
	ROLE_SYSTEM_COLUMN             = 0x1b
	ROLE_SYSTEM_ROW                = 0x1c
	ROLE_SYSTEM_CELL               = 0x1d
	ROLE_SYSTEM_LINK               = 0x1e
	ROLE_SYSTEM_HELPBALLOON        = 0x1f
	ROLE_SYSTEM_CHARACTER          = 0x20
	ROLE_SYSTEM_LIST               = 0x21
	ROLE_SYSTEM_LISTITEM           = 0x22
	ROLE_SYSTEM_OUTLINE            = 0x23
	ROLE_SYSTEM_OUTLINEITEM        = 0x24
	ROLE_SYSTEM_PAGETAB            = 0x25
	ROLE_SYSTEM_PROPERTYPAGE       = 0x26
	ROLE_SYSTEM_INDICATOR          = 0x27
	ROLE_SYSTEM_GRAPHIC            = 0x28
	ROLE_SYSTEM_STATICTEXT         = 0x29
	ROLE_SYSTEM_TEXT               = 0x2a
	ROLE_SYSTEM_PUSHBUTTON         = 0x2b
	ROLE_SYSTEM_CHECKBUTTON        = 0x2c
	ROLE_SYSTEM_RADIOBUTTON        = 0x2d
	ROLE_SYSTEM_COMBOBOX           = 0x2e
	ROLE_SYSTEM_DROPLIST           = 0x2f
	ROLE_SYSTEM_PROGRESSBAR        = 0x30
	ROLE_SYSTEM_DIAL               = 0x31
	ROLE_SYSTEM_HOTKEYFIELD        = 0x32
	ROLE_SYSTEM_SLIDER             = 0x33
	ROLE_SYSTEM_SPINBUTTON         = 0x34
	ROLE_SYSTEM_DIAGRAM            = 0x35
	ROLE_SYSTEM_ANIMATION          = 0x36
	ROLE_SYSTEM_EQUATION           = 0x37
	ROLE_SYSTEM_BUTTONDROPDOWN     = 0x38
	ROLE_SYSTEM_BUTTONMENU         = 0x39
	ROLE_SYSTEM_BUTTONDROPDOWNGRID = 0x3a
	ROLE_SYSTEM_WHITESPACE         = 0x3b
	ROLE_SYSTEM_PAGETABLIST        = 0x3c
	ROLE_SYSTEM_CLOCK              = 0x3d
	ROLE_SYSTEM_SPLITBUTTON        = 0x3e
	ROLE_SYSTEM_IPADDRESS          = 0x3f
	ROLE_SYSTEM_OUTLINEBUTTON      = 0x40
)

var (
	IID_IAccPropServer    = IID{0x76c0dbbb, 0x15e0, 0x4e7b, [8]byte{0xb6, 0x1b, 0x20, 0xee, 0xea, 0x20, 0x01, 0xe0}}
	IID_IAccPropServices  = IID{0x6e26e776, 0x04f0, 0x495d, [8]byte{0x80, 0xe4, 0x33, 0x30, 0x35, 0x2e, 0x31, 0x69}}
	CLSID_AccPropServices = CLSID{0xb5f8350b, 0x0548, 0x48b1, [8]byte{0xa6, 0xee, 0x88, 0xbd, 0x00, 0xb4, 0xa5, 0xe7}}
)

type IAccPropServerVtbl struct {
	QueryInterface uintptr
	AddRef         uintptr
	Release        uintptr
	GetPropValue   uintptr
}

type IAccPropServer struct {
	LpVtbl *IAccPropServerVtbl
}

type IAccPropServicesVtbl struct {
	QueryInterface               uintptr
	AddRef                       uintptr
	Release                      uintptr
	SetPropValue                 uintptr
	SetPropServer                uintptr
	ClearProps                   uintptr
	SetHwndProp                  uintptr
	SetHwndPropStr               uintptr
	SetHwndPropServer            uintptr
	ClearHwndProps               uintptr
	ComposeHwndIdentityString    uintptr
	DecomposeHwndIdentityString  uintptr
	SetHmenuProp                 uintptr
	SetHmenuPropStr              uintptr
	SetHmenuPropServer           uintptr
	ClearHmenuProps              uintptr
	ComposeHmenuIdentityString   uintptr
	DecomposeHmenuIdentityString uintptr
}

type IAccPropServices struct {
	LpVtbl *IAccPropServicesVtbl
}

func (obj *IAccPropServices) QueryInterface(riid REFIID, ppvObject *unsafe.Pointer) HRESULT {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.QueryInterface, 3,
		uintptr(unsafe.Pointer(obj)),
		uintptr(unsafe.Pointer(riid)),
		uintptr(unsafe.Pointer(ppvObject)))
	return HRESULT(ret)
}

func (obj *IAccPropServices) AddRef() uint32 {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.AddRef, 1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return uint32(ret)
}

func (obj *IAccPropServices) Release() uint32 {
	ret, _, _ := syscall.Syscall(obj.LpVtbl.Release, 1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0)
	return uint32(ret)
}

// SetPropServer specifies a callback object to be used to annotate an array of properties for the accessible element. You can also specify whether the annotation is to be applied to this accessible element or to the element and its children. This method is used for server annotation.
// If server developers know the HWND of the accessible element they want to annotate, they can use SetHwndPropServer.
func (obj *IAccPropServices) SetPropServer(idString []byte, idProps []MSAAPROPID, server *IAccPropServer, annoScope AnnoScope) HRESULT {
	var idStringPtr unsafe.Pointer
	idStringLen := len(idString)
	if idStringLen != 0 {
		idStringPtr = unsafe.Pointer(&idString[0])
	}
	var idPropsPtr unsafe.Pointer
	idPropsLen := len(idProps)
	if idPropsLen != 0 {
		idPropsPtr = unsafe.Pointer(&idProps[0])
	}
	ret, _, _ := syscall.Syscall9(obj.LpVtbl.SetPropServer, 7,
		uintptr(unsafe.Pointer(obj)),
		uintptr(idStringPtr),
		uintptr(idStringLen),
		uintptr(idPropsPtr),
		uintptr(idPropsLen),
		uintptr(unsafe.Pointer(server)),
		uintptr(annoScope),
		0,
		0)
	return HRESULT(ret)
}

// ClearProps restores default values to properties of accessible elements that they had previously annotated.
// If servers know the HWND of the object they want to clear, they can use ClearHwndProps.
func (obj *IAccPropServices) ClearProps(idString []byte, idProps []MSAAPROPID) HRESULT {
	var idStringPtr unsafe.Pointer
	idStringLen := len(idString)
	if idStringLen != 0 {
		idStringPtr = unsafe.Pointer(&idString[0])
	}
	var idPropsPtr unsafe.Pointer
	idPropsLen := len(idProps)
	if idPropsLen != 0 {
		idPropsPtr = unsafe.Pointer(&idProps[0])
	}
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.ClearProps, 5,
		uintptr(unsafe.Pointer(obj)),
		uintptr(idStringPtr),
		uintptr(idStringLen),
		uintptr(idPropsPtr),
		uintptr(idPropsLen),
		0)
	return HRESULT(ret)
}

// SetHwndPropServer wraps SetPropServer, providing a convenient entry point for callers who are annotating HWND-based accessible elements.
func (obj *IAccPropServices) SetHwndPropServer(hwnd HWND, idObject int32, idChild uint32, idProps []MSAAPROPID, server *IAccPropServer, annoScope AnnoScope) HRESULT {
	var idPropsPtr unsafe.Pointer
	idPropsLen := len(idProps)
	if idPropsLen != 0 {
		idPropsPtr = unsafe.Pointer(&idProps[0])
	}
	ret, _, _ := syscall.Syscall9(obj.LpVtbl.SetHwndPropServer, 8,
		uintptr(unsafe.Pointer(obj)),
		uintptr(hwnd),
		uintptr(idObject),
		uintptr(idChild),
		uintptr(idPropsPtr),
		uintptr(idPropsLen),
		uintptr(unsafe.Pointer(server)),
		uintptr(annoScope),
		0)
	return HRESULT(ret)
}

// ClearHwndProps wraps SetPropValue, SetPropServer, and ClearProps, and provides a convenient entry point for callers who are annotating HWND-based accessible elements.
func (obj *IAccPropServices) ClearHwndProps(hwnd HWND, idObject int32, idChild uint32, idProps []MSAAPROPID) HRESULT {
	var idPropsPtr unsafe.Pointer
	idPropsLen := len(idProps)
	if idPropsLen != 0 {
		idPropsPtr = unsafe.Pointer(&idProps[0])
	}
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.ClearHwndProps, 6,
		uintptr(unsafe.Pointer(obj)),
		uintptr(hwnd),
		uintptr(idObject),
		uintptr(idChild),
		uintptr(idPropsPtr),
		uintptr(idPropsLen))
	return HRESULT(ret)
}

// ComposeHwndIdentityString retrievs an identity string.
func (obj *IAccPropServices) ComposeHwndIdentityString(hwnd HWND, idObject int32, idChild uint32) (hr HRESULT, idString []byte) {
	var data *[1<<31 - 1]byte
	var len uint32
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.ComposeHwndIdentityString, 6,
		uintptr(unsafe.Pointer(obj)),
		uintptr(hwnd),
		uintptr(idObject),
		uintptr(idChild),
		uintptr(unsafe.Pointer(&data)),
		uintptr(unsafe.Pointer(&len)))
	hr = HRESULT(ret)
	if FAILED(hr) {
		return
	}
	defer CoTaskMemFree(uintptr(unsafe.Pointer(data)))
	idString = make([]byte, len)
	copy(idString, data[:len])
	return
}

// DecomposeHwndIdentityString determines the HWND, object ID, and child ID for the accessible element identified by the identity string.
func (obj *IAccPropServices) DecomposeHwndIdentityString(idString []byte) (hr HRESULT, hwnd HWND, idObject int32, idChild uint32) {
	var idStringPtr unsafe.Pointer
	idStringLen := len(idString)
	if idStringLen != 0 {
		idStringPtr = unsafe.Pointer(&idString[0])
	}
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.DecomposeHwndIdentityString, 6,
		uintptr(unsafe.Pointer(obj)),
		uintptr(idStringPtr),
		uintptr(idStringLen),
		uintptr(unsafe.Pointer(&hwnd)),
		uintptr(unsafe.Pointer(&idObject)),
		uintptr(unsafe.Pointer(&idChild)))
	hr = HRESULT(ret)
	return
}

// SetHmenuPropServer wraps SetPropServer, providing a convenient entry point for callers who are annotating HMENU-based accessible elements.
func (obj *IAccPropServices) SetHmenuPropServer(hmenu HMENU, idChild uint32, idProps []MSAAPROPID, server *IAccPropServer, annoScope AnnoScope) HRESULT {
	var idPropsPtr unsafe.Pointer
	idPropsLen := len(idProps)
	if idPropsLen != 0 {
		idPropsPtr = unsafe.Pointer(&idProps[0])
	}
	ret, _, _ := syscall.Syscall9(obj.LpVtbl.SetHmenuPropServer, 7,
		uintptr(unsafe.Pointer(obj)),
		uintptr(hmenu),
		uintptr(idChild),
		uintptr(idPropsPtr),
		uintptr(idPropsLen),
		uintptr(unsafe.Pointer(server)),
		uintptr(annoScope),
		0,
		0)
	return HRESULT(ret)
}

// ClearHmenuProps wraps ClearProps, and provides a convenient entry point for callers who are annotating HMENU-based accessible elements.
func (obj *IAccPropServices) ClearHmenuProps(hmenu HMENU, idChild uint32, idProps []MSAAPROPID) HRESULT {
	var idPropsPtr unsafe.Pointer
	idPropsLen := len(idProps)
	if idPropsLen != 0 {
		idPropsPtr = unsafe.Pointer(&idProps[0])
	}
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.ClearHmenuProps, 5,
		uintptr(unsafe.Pointer(obj)),
		uintptr(hmenu),
		uintptr(idChild),
		uintptr(idPropsPtr),
		uintptr(idPropsLen),
		0)
	return HRESULT(ret)
}

// ComposeHmenuIdentityString retrieves an identity string for an HMENU-based accessible element.
func (obj *IAccPropServices) ComposeHmenuIdentityString(hmenu HMENU, idChild uint32) (hr HRESULT, idString []byte) {
	var data *[1<<31 - 1]byte
	var len uint32
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.ComposeHmenuIdentityString, 5,
		uintptr(unsafe.Pointer(obj)),
		uintptr(hmenu),
		uintptr(idChild),
		uintptr(unsafe.Pointer(&data)),
		uintptr(unsafe.Pointer(&len)),
		0)
	hr = HRESULT(ret)
	if FAILED(hr) {
		return
	}
	defer CoTaskMemFree(uintptr(unsafe.Pointer(data)))
	idString = make([]byte, len)
	copy(idString, data[:len])
	return
}

// DecomposeHmenuIdentityString determines the HMENU, object ID, and child ID for the accessible element identified by the identity string.
func (obj *IAccPropServices) DecomposeHmenuIdentityString(idString []byte) (hr HRESULT, hmenu HMENU, idChild uint32) {
	var idStringPtr unsafe.Pointer
	idStringLen := len(idString)
	if idStringLen != 0 {
		idStringPtr = unsafe.Pointer(&idString[0])
	}
	ret, _, _ := syscall.Syscall6(obj.LpVtbl.DecomposeHmenuIdentityString, 5,
		uintptr(unsafe.Pointer(obj)),
		uintptr(idStringPtr),
		uintptr(idStringLen),
		uintptr(unsafe.Pointer(&hmenu)),
		uintptr(unsafe.Pointer(&idChild)),
		0)
	hr = HRESULT(ret)
	return
}
