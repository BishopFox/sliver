package main

import (
	"encoding/base64"
	"reflect"
	"unsafe"
)

// main is required for TinyGo to compile to Wasm.
func main() {}

func encode(name []byte) string {
	return base64.RawStdEncoding.EncodeToString(name)
}

//export encode
func _encode(ptr, size uint32) (ptrSize uint64) {
	name := ptrToBuf(ptr, size)
	g := encode(name)
	ptr, size = stringToPtr(g)
	return (uint64(ptr) << uint64(32)) | uint64(size)
}

func decode(name string) []byte {
	result, err := base64.StdEncoding.DecodeString(name)
	if err != nil {
		log(err.Error())
		return []byte{0}
	}
	return result
}

//export decode
func _decode(ptr, size uint32) (ptrSize uint64) {
	name := ptrToString(ptr, size)
	g := decode(name)
	ptr, size = bufToPtr(g)
	return (uint64(ptr) << uint64(32)) | uint64(size)
}

func log(message string) {
	ptr, size := stringToPtr(message)
	_log(ptr, size)
}

//go:wasm-module base64
//export log
func _log(ptr uint32, size uint32)

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
