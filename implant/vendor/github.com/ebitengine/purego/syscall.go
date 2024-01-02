// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build darwin || freebsd || linux || windows

package purego

const (
	maxArgs     = 9
	numOfFloats = 8 // arm64 and amd64 both have 8 float registers
)

// SyscallN takes fn, a C function pointer and a list of arguments as uintptr.
// There is an internal maximum number of arguments that SyscallN can take. It panics
// when the maximum is exceeded. It returns the result and the libc error code if there is one.
//
// NOTE: SyscallN does not properly call functions that have both integer and float parameters.
// See discussion comment https://github.com/ebiten/purego/pull/1#issuecomment-1128057607
// for an explanation of why that is.
//
// On amd64, if there are more than 8 floats the 9th and so on will be placed incorrectly on the
// stack.
//
// The pragma go:nosplit is not needed at this function declaration because it uses go:uintptrescapes
// which forces all the objects that the uintptrs point to onto the heap where a stack split won't affect
// their memory location.
//
//go:uintptrescapes
func SyscallN(fn uintptr, args ...uintptr) (r1, r2, err uintptr) {
	if fn == 0 {
		panic("purego: fn is nil")
	}
	if len(args) > maxArgs {
		panic("purego: too many arguments to SyscallN")
	}
	// add padding so there is no out-of-bounds slicing
	var tmp [maxArgs]uintptr
	copy(tmp[:], args)
	return syscall_syscall9X(fn, tmp[0], tmp[1], tmp[2], tmp[3], tmp[4], tmp[5], tmp[6], tmp[7], tmp[8])
}
