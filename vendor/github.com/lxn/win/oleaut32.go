// Copyright 2010 The win Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package win

import (
	"fmt"
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

type DISPID int32

const (
	DISPID_BEFORENAVIGATE             DISPID = 100
	DISPID_NAVIGATECOMPLETE           DISPID = 101
	DISPID_STATUSTEXTCHANGE           DISPID = 102
	DISPID_QUIT                       DISPID = 103
	DISPID_DOWNLOADCOMPLETE           DISPID = 104
	DISPID_COMMANDSTATECHANGE         DISPID = 105
	DISPID_DOWNLOADBEGIN              DISPID = 106
	DISPID_NEWWINDOW                  DISPID = 107
	DISPID_PROGRESSCHANGE             DISPID = 108
	DISPID_WINDOWMOVE                 DISPID = 109
	DISPID_WINDOWRESIZE               DISPID = 110
	DISPID_WINDOWACTIVATE             DISPID = 111
	DISPID_PROPERTYCHANGE             DISPID = 112
	DISPID_TITLECHANGE                DISPID = 113
	DISPID_TITLEICONCHANGE            DISPID = 114
	DISPID_FRAMEBEFORENAVIGATE        DISPID = 200
	DISPID_FRAMENAVIGATECOMPLETE      DISPID = 201
	DISPID_FRAMENEWWINDOW             DISPID = 204
	DISPID_BEFORENAVIGATE2            DISPID = 250
	DISPID_NEWWINDOW2                 DISPID = 251
	DISPID_NAVIGATECOMPLETE2          DISPID = 252
	DISPID_ONQUIT                     DISPID = 253
	DISPID_ONVISIBLE                  DISPID = 254
	DISPID_ONTOOLBAR                  DISPID = 255
	DISPID_ONMENUBAR                  DISPID = 256
	DISPID_ONSTATUSBAR                DISPID = 257
	DISPID_ONFULLSCREEN               DISPID = 258
	DISPID_DOCUMENTCOMPLETE           DISPID = 259
	DISPID_ONTHEATERMODE              DISPID = 260
	DISPID_ONADDRESSBAR               DISPID = 261
	DISPID_WINDOWSETRESIZABLE         DISPID = 262
	DISPID_WINDOWCLOSING              DISPID = 263
	DISPID_WINDOWSETLEFT              DISPID = 264
	DISPID_WINDOWSETTOP               DISPID = 265
	DISPID_WINDOWSETWIDTH             DISPID = 266
	DISPID_WINDOWSETHEIGHT            DISPID = 267
	DISPID_CLIENTTOHOSTWINDOW         DISPID = 268
	DISPID_SETSECURELOCKICON          DISPID = 269
	DISPID_FILEDOWNLOAD               DISPID = 270
	DISPID_NAVIGATEERROR              DISPID = 271
	DISPID_PRIVACYIMPACTEDSTATECHANGE DISPID = 272
	DISPID_NEWWINDOW3                 DISPID = 273
)

var (
	IID_IDispatch = IID{0x00020400, 0x0000, 0x0000, [8]byte{0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46}}
)

const (
	DISP_E_MEMBERNOTFOUND = 0x80020003
)

const (
	CSC_UPDATECOMMANDS  = ^0x0
	CSC_NAVIGATEFORWARD = 0x1
	CSC_NAVIGATEBACK    = 0x2
)

type IDispatchVtbl struct {
	QueryInterface   uintptr
	AddRef           uintptr
	Release          uintptr
	GetTypeInfoCount uintptr
	GetTypeInfo      uintptr
	GetIDsOfNames    uintptr
	Invoke           uintptr
}

type IDispatch struct {
	LpVtbl *IDispatchVtbl
}

type VARTYPE uint16

const (
	VT_EMPTY            VARTYPE = 0
	VT_NULL             VARTYPE = 1
	VT_I2               VARTYPE = 2
	VT_I4               VARTYPE = 3
	VT_R4               VARTYPE = 4
	VT_R8               VARTYPE = 5
	VT_CY               VARTYPE = 6
	VT_DATE             VARTYPE = 7
	VT_BSTR             VARTYPE = 8
	VT_DISPATCH         VARTYPE = 9
	VT_ERROR            VARTYPE = 10
	VT_BOOL             VARTYPE = 11
	VT_VARIANT          VARTYPE = 12
	VT_UNKNOWN          VARTYPE = 13
	VT_DECIMAL          VARTYPE = 14
	VT_I1               VARTYPE = 16
	VT_UI1              VARTYPE = 17
	VT_UI2              VARTYPE = 18
	VT_UI4              VARTYPE = 19
	VT_I8               VARTYPE = 20
	VT_UI8              VARTYPE = 21
	VT_INT              VARTYPE = 22
	VT_UINT             VARTYPE = 23
	VT_VOID             VARTYPE = 24
	VT_HRESULT          VARTYPE = 25
	VT_PTR              VARTYPE = 26
	VT_SAFEARRAY        VARTYPE = 27
	VT_CARRAY           VARTYPE = 28
	VT_USERDEFINED      VARTYPE = 29
	VT_LPSTR            VARTYPE = 30
	VT_LPWSTR           VARTYPE = 31
	VT_RECORD           VARTYPE = 36
	VT_INT_PTR          VARTYPE = 37
	VT_UINT_PTR         VARTYPE = 38
	VT_FILETIME         VARTYPE = 64
	VT_BLOB             VARTYPE = 65
	VT_STREAM           VARTYPE = 66
	VT_STORAGE          VARTYPE = 67
	VT_STREAMED_OBJECT  VARTYPE = 68
	VT_STORED_OBJECT    VARTYPE = 69
	VT_BLOB_OBJECT      VARTYPE = 70
	VT_CF               VARTYPE = 71
	VT_CLSID            VARTYPE = 72
	VT_VERSIONED_STREAM VARTYPE = 73
	VT_BSTR_BLOB        VARTYPE = 0xfff
	VT_VECTOR           VARTYPE = 0x1000
	VT_ARRAY            VARTYPE = 0x2000
	VT_BYREF            VARTYPE = 0x4000
	VT_RESERVED         VARTYPE = 0x8000
	VT_ILLEGAL          VARTYPE = 0xffff
	VT_ILLEGALMASKED    VARTYPE = 0xfff
	VT_TYPEMASK         VARTYPE = 0xfff
)

type VARIANTARG struct {
	VARIANT
}

type VARIANT_BOOL int16

const (
	VARIANT_TRUE  VARIANT_BOOL = -1
	VARIANT_FALSE VARIANT_BOOL = 0
)

type SAFEARRAYBOUND struct {
	CElements uint32
	LLbound   int32
}

type SAFEARRAY struct {
	CDims      uint16
	FFeatures  uint16
	CbElements uint32
	CLocks     uint32
	PvData     uintptr
	Rgsabound  [1]SAFEARRAYBOUND
}

//type BSTR *uint16

func StringToBSTR(value string) *uint16 /*BSTR*/ {
	// IMPORTANT: Don't forget to free the BSTR value when no longer needed!
	return SysAllocString(value)
}

func BSTRToString(value *uint16 /*BSTR*/) string {
	// ISSUE: Is this really ok?
	bstrArrPtr := (*[200000000]uint16)(unsafe.Pointer(value))

	bstrSlice := make([]uint16, SysStringLen(value))
	copy(bstrSlice, bstrArrPtr[:])

	return syscall.UTF16ToString(bstrSlice)
}

func IntToVariantI4(value int32) *VAR_I4 {
	return &VAR_I4{vt: VT_I4, lVal: value}
}

func VariantI4ToInt(value *VAR_I4) int32 {
	return value.lVal
}

func BoolToVariantBool(value bool) *VAR_BOOL {
	return &VAR_BOOL{vt: VT_BOOL, boolVal: VARIANT_BOOL(BoolToBOOL(value))}
}

func VariantBoolToBool(value *VAR_BOOL) bool {
	return value.boolVal != 0
}

func StringToVariantBSTR(value string) *VAR_BSTR {
	// IMPORTANT: Don't forget to free the BSTR value when no longer needed!
	return &VAR_BSTR{vt: VT_BSTR, bstrVal: StringToBSTR(value)}
}

func VariantBSTRToString(value *VAR_BSTR) string {
	return BSTRToString(value.bstrVal)
}

func (v *VARIANT) MustLong() int32 {
	value, err := v.Long()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *VARIANT) Long() (int32, error) {
	if v.Vt != VT_I4 {
		return 0, fmt.Errorf("Error: Long() v.Vt !=  VT_I4, ptr=%p, value=%+v", v, v)
	}
	p := (*VAR_I4)(unsafe.Pointer(v))
	return p.lVal, nil
}

func (v *VARIANT) SetLong(value int32) {
	v.Vt = VT_I4
	p := (*VAR_I4)(unsafe.Pointer(v))
	p.lVal = value
}

func (v *VARIANT) MustULong() uint32 {
	value, err := v.ULong()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *VARIANT) ULong() (uint32, error) {
	if v.Vt != VT_UI4 {
		return 0, fmt.Errorf("Error: ULong() v.Vt !=  VT_UI4, ptr=%p, value=%+v", v, v)
	}
	p := (*VAR_UI4)(unsafe.Pointer(v))
	return p.ulVal, nil
}

func (v *VARIANT) SetULong(value uint32) {
	v.Vt = VT_UI4
	p := (*VAR_UI4)(unsafe.Pointer(v))
	p.ulVal = value
}

func (v *VARIANT) MustBool() VARIANT_BOOL {
	value, err := v.Bool()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *VARIANT) Bool() (VARIANT_BOOL, error) {
	if v.Vt != VT_BOOL {
		return VARIANT_FALSE, fmt.Errorf("Error: Bool() v.Vt !=  VT_BOOL, ptr=%p, value=%+v", v, v)
	}
	p := (*VAR_BOOL)(unsafe.Pointer(v))
	return p.boolVal, nil
}

func (v *VARIANT) SetBool(value VARIANT_BOOL) {
	v.Vt = VT_BOOL
	p := (*VAR_BOOL)(unsafe.Pointer(v))
	p.boolVal = value
}

func (v *VARIANT) MustBSTR() *uint16 {
	value, err := v.BSTR()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *VARIANT) BSTR() (*uint16, error) {
	if v.Vt != VT_BSTR {
		return nil, fmt.Errorf("Error: BSTR() v.Vt !=  VT_BSTR, ptr=%p, value=%+v", v, v)
	}
	p := (*VAR_BSTR)(unsafe.Pointer(v))
	return p.bstrVal, nil
}

func (v *VARIANT) SetBSTR(value *uint16) {
	v.Vt = VT_BSTR
	p := (*VAR_BSTR)(unsafe.Pointer(v))
	p.bstrVal = value
}

func (v *VARIANT) MustPDispatch() *IDispatch {
	value, err := v.PDispatch()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *VARIANT) PDispatch() (*IDispatch, error) {
	if v.Vt != VT_DISPATCH {
		return nil, fmt.Errorf("Error: PDispatch() v.Vt !=  VT_DISPATCH, ptr=%p, value=%+v", v, v)
	}
	p := (*VAR_PDISP)(unsafe.Pointer(v))
	return p.pdispVal, nil
}

func (v *VARIANT) SetPDispatch(value *IDispatch) {
	v.Vt = VT_DISPATCH
	p := (*VAR_PDISP)(unsafe.Pointer(v))
	p.pdispVal = value
}

func (v *VARIANT) MustPVariant() *VARIANT {
	value, err := v.PVariant()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *VARIANT) PVariant() (*VARIANT, error) {
	if v.Vt != VT_BYREF|VT_VARIANT {
		return nil, fmt.Errorf("Error: PVariant() v.Vt !=  VT_BYREF|VT_VARIANT, ptr=%p, value=%+v", v, v)
	}
	p := (*VAR_PVAR)(unsafe.Pointer(v))
	return p.pvarVal, nil
}

func (v *VARIANT) SetPVariant(value *VARIANT) {
	v.Vt = VT_BYREF | VT_VARIANT
	p := (*VAR_PVAR)(unsafe.Pointer(v))
	p.pvarVal = value
}

func (v *VARIANT) MustPBool() *VARIANT_BOOL {
	value, err := v.PBool()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *VARIANT) PBool() (*VARIANT_BOOL, error) {
	if v.Vt != VT_BYREF|VT_BOOL {
		return nil, fmt.Errorf("Error: PBool() v.Vt !=  VT_BYREF|VT_BOOL, ptr=%p, value=%+v", v, v)
	}
	p := (*VAR_PBOOL)(unsafe.Pointer(v))
	return p.pboolVal, nil
}

func (v *VARIANT) SetPBool(value *VARIANT_BOOL) {
	v.Vt = VT_BYREF | VT_BOOL
	p := (*VAR_PBOOL)(unsafe.Pointer(v))
	p.pboolVal = value
}

func (v *VARIANT) MustPPDispatch() **IDispatch {
	value, err := v.PPDispatch()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *VARIANT) PPDispatch() (**IDispatch, error) {
	if v.Vt != VT_BYREF|VT_DISPATCH {
		return nil, fmt.Errorf("PPDispatch() v.Vt !=  VT_BYREF|VT_DISPATCH, ptr=%p, value=%+v", v, v)
	}
	p := (*VAR_PPDISP)(unsafe.Pointer(v))
	return p.ppdispVal, nil
}

func (v *VARIANT) SetPPDispatch(value **IDispatch) {
	v.Vt = VT_BYREF | VT_DISPATCH
	p := (*VAR_PPDISP)(unsafe.Pointer(v))
	p.ppdispVal = value
}

func (v *VARIANT) MustPSafeArray() *SAFEARRAY {
	value, err := v.PSafeArray()
	if err != nil {
		panic(err)
	}
	return value
}

func (v *VARIANT) PSafeArray() (*SAFEARRAY, error) {
	if (v.Vt & VT_ARRAY) != VT_ARRAY {
		return nil, fmt.Errorf("Error: PSafeArray() (v.Vt & VT_ARRAY) != VT_ARRAY, ptr=%p, value=%+v", v, v)
	}
	p := (*VAR_PSAFEARRAY)(unsafe.Pointer(v))
	return p.parray, nil
}

func (v *VARIANT) SetPSafeArray(value *SAFEARRAY, elementVt VARTYPE) {
	v.Vt = VT_ARRAY | elementVt
	p := (*VAR_PSAFEARRAY)(unsafe.Pointer(v))
	p.parray = value
}

type DISPPARAMS struct {
	Rgvarg            *VARIANTARG
	RgdispidNamedArgs *DISPID
	CArgs             int32
	CNamedArgs        int32
}

var (
	// Library
	liboleaut32 *windows.LazyDLL

	// Functions
	sysAllocString *windows.LazyProc
	sysFreeString  *windows.LazyProc
	sysStringLen   *windows.LazyProc
)

func init() {
	// Library
	liboleaut32 = windows.NewLazySystemDLL("oleaut32.dll")

	// Functions
	sysAllocString = liboleaut32.NewProc("SysAllocString")
	sysFreeString = liboleaut32.NewProc("SysFreeString")
	sysStringLen = liboleaut32.NewProc("SysStringLen")
}

func SysAllocString(s string) *uint16 /*BSTR*/ {
	ret, _, _ := syscall.Syscall(sysAllocString.Addr(), 1,
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(s))),
		0,
		0)

	return (*uint16) /*BSTR*/ (unsafe.Pointer(ret))
}

func SysFreeString(bstr *uint16 /*BSTR*/) {
	syscall.Syscall(sysFreeString.Addr(), 1,
		uintptr(unsafe.Pointer(bstr)),
		0,
		0)
}

func SysStringLen(bstr *uint16 /*BSTR*/) uint32 {
	ret, _, _ := syscall.Syscall(sysStringLen.Addr(), 1,
		uintptr(unsafe.Pointer(bstr)),
		0,
		0)

	return uint32(ret)
}
