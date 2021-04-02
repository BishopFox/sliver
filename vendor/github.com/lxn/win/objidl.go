// Copyright 2010 The win Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build windows

package win

type IDataObjectVtbl struct {
	IUnknownVtbl
	GetData               uintptr
	GetDataHere           uintptr
	QueryGetData          uintptr
	GetCanonicalFormatEtc uintptr
	SetData               uintptr
	EnumFormatEtc         uintptr
	DAdvise               uintptr
	DUnadvise             uintptr
	EnumDAdvise           uintptr
}

type IDataObject struct {
	LpVtbl *IDataObjectVtbl
}

type IStorageVtbl struct {
	IUnknownVtbl
	CreateStream    uintptr
	OpenStream      uintptr
	CreateStorage   uintptr
	OpenStorage     uintptr
	CopyTo          uintptr
	MoveElementTo   uintptr
	Commit          uintptr
	Revert          uintptr
	EnumElements    uintptr
	DestroyElement  uintptr
	RenameElement   uintptr
	SetElementTimes uintptr
	SetClass        uintptr
	SetStateBits    uintptr
	Stat            uintptr
}

type IStorage struct {
	LpVtbl *IStorageVtbl
}
