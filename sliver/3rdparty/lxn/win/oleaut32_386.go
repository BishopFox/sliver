// Copyright 2012 The win Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package win

type VARIANT struct {
	Vt       VARTYPE
	reserved [14]byte
}

type VAR_I4 struct {
	vt        VARTYPE
	reserved1 [6]byte
	lVal      int32
	reserved2 [4]byte
}

type VAR_UI4 struct {
	vt        VARTYPE
	reserved1 [6]byte
	ulVal     uint32
	reserved2 [4]byte
}

type VAR_BOOL struct {
	vt        VARTYPE
	reserved1 [6]byte
	boolVal   VARIANT_BOOL
	reserved2 [6]byte
}

type VAR_BSTR struct {
	vt        VARTYPE
	reserved1 [6]byte
	bstrVal   *uint16 /*BSTR*/
	reserved2 [4]byte
}

type VAR_PDISP struct {
	vt        VARTYPE
	reserved1 [6]byte
	pdispVal  *IDispatch
	reserved2 [4]byte
}

type VAR_PSAFEARRAY struct {
	vt        VARTYPE
	reserved1 [6]byte
	parray    *SAFEARRAY
	reserved2 [4]byte
}

type VAR_PVAR struct {
	vt        VARTYPE
	reserved1 [6]byte
	pvarVal   *VARIANT
	reserved2 [4]byte
}

type VAR_PBOOL struct {
	vt        VARTYPE
	reserved1 [6]byte
	pboolVal  *VARIANT_BOOL
	reserved2 [4]byte
}

type VAR_PPDISP struct {
	vt        VARTYPE
	reserved1 [6]byte
	ppdispVal **IDispatch
	reserved2 [4]byte
}
