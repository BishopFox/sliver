// +build windows

// Copyright (C) 2018, Rapid7 LLC, Boston, MA, USA.
// All rights reserved. This material contains unpublished, copyrighted
// work including confidential and proprietary information of Rapid7.
package winhttp

import (
	"github.com/stretchr/testify/assert"
	"golang.org/x/sys/windows"
	"testing"
	"unicode/utf16"
	"unsafe"
)

func TestStringToLpwstr_empty(t *testing.T) {
	a := assert.New(t)
	a.Nil(StringToLpwstr(""))
}

func TestStringToLpwstr_NUL(t *testing.T) {
	a := assert.New(t)
	a.Nil(StringToLpwstr("te\x00st"))
}

func TestStringToLpwstr(t *testing.T) {
	a := assert.New(t)
	p := StringToLpwstr("test123")
	if a.NotNil(p) {
		a.Equal("t", string(utf16.Decode([]uint16{*(*uint16)(unsafe.Pointer(p))})))
	}
}

func TestLpwStrToString(t *testing.T) {
	a := assert.New(t)
	p, err := windows.UTF16PtrFromString("test")
	if a.Nil(err) && a.NotNil(p) {
		a.Equal("test", LpwstrToString(p))
	}
}

func TestLpwStrToString_empty(t *testing.T) {
	a := assert.New(t)
	p, err := windows.UTF16PtrFromString("")
	if a.Nil(err) && a.NotNil(p) {
		a.Equal("", LpwstrToString(p))
	}
}
