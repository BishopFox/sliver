package main

import "unsafe"

var allocations = map[uint32][]byte{}

func main() {}

//go:wasmexport malloc
func malloc(size uint32) uint32 {
	buffer := make([]byte, max(size, 1))
	pointer := uint32(uintptr(unsafe.Pointer(unsafe.SliceData(buffer))))
	allocations[pointer] = buffer
	return pointer
}

//go:wasmexport free
func free(pointer uint32) {
	delete(allocations, pointer)
}

//go:wasmexport encode
func encode(pointer, size uint32) uint64 {
	return uint64(pointer)<<32 | uint64(size)
}

//go:wasmexport decode
func decode(pointer, size uint32) uint64 {
	return uint64(pointer)<<32 | uint64(size)
}
