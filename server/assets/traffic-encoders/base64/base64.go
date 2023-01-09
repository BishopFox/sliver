package main

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"encoding/base64"
	"reflect"
	"unsafe"
)

// main is required for TinyGo to compile to Wasm.
func main() {}

func encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

//export encode
func _encode(ptr, size uint32) (ptrSize uint64) {
	data := ptrToBuf(ptr, size)
	encodedData := encode(data)
	ptr, size = stringToPtr(encodedData)
	return (uint64(ptr) << uint64(32)) | uint64(size)
}

func decode(data string) []byte {
	result, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		log(err.Error())
		return []byte{0}
	}
	return result
}

//export decode
func _decode(ptr, size uint32) (ptrSize uint64) {
	encodedData := ptrToString(ptr, size)
	decodedData := decode(encodedData)
	ptr, size = bufToPtr(decodedData)
	return (uint64(ptr) << uint64(32)) | uint64(size)
}

func log(message string) {
	ptr, size := stringToPtr(message)
	_log(ptr, size)
}

//go:wasm-module base64
//export log
func _log(ptr uint32, size uint32)

//
// *** Helper Functions ***
//

func ptrToBuf(ptr uint32, size uint32) []byte {
	return *(*[]byte)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(ptr),
		Len:  uintptr(size), // Tinygo requires these as uintptrs even if they are int fields.
		Cap:  uintptr(size), // ^^ See https://github.com/tinygo-org/tinygo/issues/1284
	}))
}

func ptrToString(ptr uint32, size uint32) string {
	return *(*string)(unsafe.Pointer(&reflect.SliceHeader{
		Data: uintptr(ptr),
		Len:  uintptr(size), // Tinygo requires these as uintptrs even if they are int fields.
		Cap:  uintptr(size), // ^^ See https://github.com/tinygo-org/tinygo/issues/1284
	}))
}

func stringToPtr(s string) (uint32, uint32) {
	buf := []byte(s)
	ptr := &buf[0]
	unsafePtr := uintptr(unsafe.Pointer(ptr))
	return uint32(unsafePtr), uint32(len(buf))
}

func bufToPtr(buf []byte) (uint32, uint32) {
	ptr := &buf[0]
	unsafePtr := uintptr(unsafe.Pointer(ptr))
	return uint32(unsafePtr), uint32(len(buf))
}
