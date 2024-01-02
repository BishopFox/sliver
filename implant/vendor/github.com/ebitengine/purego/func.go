// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build darwin || freebsd || linux || windows

package purego

import (
	"math"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego/internal/strings"
)

// RegisterLibFunc is a wrapper around RegisterFunc that uses the C function returned from Dlsym(handle, name).
// It panics if it can't find the name symbol.
func RegisterLibFunc(fptr interface{}, handle uintptr, name string) {
	sym, err := loadSymbol(handle, name)
	if err != nil {
		panic(err)
	}
	RegisterFunc(fptr, sym)
}

// RegisterFunc takes a pointer to a Go function representing the calling convention of the C function.
// fptr will be set to a function that when called will call the C function given by cfn with the
// parameters passed in the correct registers and stack.
//
// A panic is produced if the type is not a function pointer or if the function returns more than 1 value.
//
// These conversions describe how a Go type in the fptr will be used to call
// the C function. It is important to note that there is no way to verify that fptr
// matches the C function. This also holds true for struct types where the padding
// needs to be ensured to match that of C; RegisterFunc does not verify this.
//
// # Type Conversions (Go <=> C)
//
//	string <=> char*
//	bool <=> _Bool
//	uintptr <=> uintptr_t
//	uint <=> uint32_t or uint64_t
//	uint8 <=> uint8_t
//	uint16 <=> uint16_t
//	uint32 <=> uint32_t
//	uint64 <=> uint64_t
//	int <=> int32_t or int64_t
//	int8 <=> int8_t
//	int16 <=> int16_t
//	int32 <=> int32_t
//	int64 <=> int64_t
//	float32 <=> float (WIP)
//	float64 <=> double (WIP)
//	struct <=> struct (WIP)
//	func <=> C function
//	unsafe.Pointer, *T <=> void*
//	[]T => void*
//
// There is a special case when the last argument of fptr is a variadic interface (or []interface}
// it will be expanded into a call to the C function as if it had the arguments in that slice.
// This means that using arg ...interface{} is like a cast to the function with the arguments inside arg.
// This is not the same as C variadic.
//
// There are some limitations when using RegisterFunc on Linux. First, there is no support for function arguments.
// Second, float32 and float64 arguments and return values do not work when CGO_ENABLED=1. Otherwise, Linux
// has the same feature parity as Darwin.
//
// # Memory
//
// In general it is not possible for purego to guarantee the lifetimes of objects returned or received from
// calling functions using RegisterFunc. For arguments to a C function it is important that the C function doesn't
// hold onto a reference to Go memory. This is the same as the [Cgo rules].
//
// However, there are some special cases. When passing a string as an argument if the string does not end in a null
// terminated byte (\x00) then the string will be copied into memory maintained by purego. The memory is only valid for
// that specific call. Therefore, if the C code keeps a reference to that string it may become invalid at some
// undefined time. However, if the string does already contain a null-terminated byte then no copy is done.
// It is then the responsibility of the caller to ensure the string stays alive as long as it's needed in C memory.
// This can be done using runtime.KeepAlive or allocating the string in C memory using malloc. When a C function
// returns a null-terminated pointer to char a Go string can be used. Purego will allocate a new string in Go memory
// and copy the data over. This string will be garbage collected whenever Go decides it's no longer referenced.
// This C created string will not be freed by purego. If the pointer to char is not null-terminated or must continue
// to point to C memory (because it's a buffer for example) then use a pointer to byte and then convert that to a slice
// using unsafe.Slice. Doing this means that it becomes the responsibility of the caller to care about the lifetime
// of the pointer
//
// # Example
//
// All functions below call this C function:
//
//	char *foo(char *str);
//
//	// Let purego convert types
//	var foo func(s string) string
//	goString := foo("copied")
//	// Go will garbage collect this string
//
//	// Manually, handle allocations
//	var foo2 func(b string) *byte
//	mustFree := foo2("not copied\x00")
//	defer free(mustFree)
//
// [Cgo rules]: https://pkg.go.dev/cmd/cgo#hdr-Go_references_to_C
func RegisterFunc(fptr interface{}, cfn uintptr) {
	fn := reflect.ValueOf(fptr).Elem()
	ty := fn.Type()
	if ty.Kind() != reflect.Func {
		panic("purego: fptr must be a function pointer")
	}
	if ty.NumOut() > 1 {
		panic("purego: function can only return zero or one values")
	}
	if cfn == 0 {
		panic("purego: cfn is nil")
	}
	{
		// this code checks how many registers and stack this function will use
		// to avoid crashing with too many arguments
		var ints int
		var floats int
		var stack int
		for i := 0; i < ty.NumIn(); i++ {
			arg := ty.In(i)
			switch arg.Kind() {
			case reflect.String, reflect.Uintptr, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Ptr, reflect.UnsafePointer, reflect.Slice,
				reflect.Func, reflect.Bool:
				if ints < numOfIntegerRegisters() {
					ints++
				} else {
					stack++
				}
			case reflect.Float32, reflect.Float64:
				if floats < numOfFloats {
					floats++
				} else {
					stack++
				}
			default:
				panic("purego: unsupported kind " + arg.Kind().String())
			}
		}
		sizeOfStack := maxArgs - numOfIntegerRegisters()
		if stack > sizeOfStack {
			panic("purego: too many arguments")
		}
	}
	v := reflect.MakeFunc(ty, func(args []reflect.Value) (results []reflect.Value) {
		if len(args) > 0 {
			if variadic, ok := args[len(args)-1].Interface().([]interface{}); ok {
				// subtract one from args bc the last argument in args is []interface{}
				// which we are currently expanding
				tmp := make([]reflect.Value, len(args)-1+len(variadic))
				n := copy(tmp, args[:len(args)-1])
				for i, v := range variadic {
					tmp[n+i] = reflect.ValueOf(v)
				}
				args = tmp
			}
		}
		var sysargs [maxArgs]uintptr
		stack := sysargs[numOfIntegerRegisters():]
		var floats [numOfFloats]uintptr
		var numInts int
		var numFloats int
		var numStack int
		var addStack, addInt, addFloat func(x uintptr)
		if runtime.GOARCH == "arm64" || runtime.GOOS != "windows" {
			// Windows arm64 uses the same calling convention as macOS and Linux
			addStack = func(x uintptr) {
				stack[numStack] = x
				numStack++
			}
			addInt = func(x uintptr) {
				if numInts >= numOfIntegerRegisters() {
					addStack(x)
				} else {
					sysargs[numInts] = x
					numInts++
				}
			}
			addFloat = func(x uintptr) {
				if numFloats < len(floats) {
					floats[numFloats] = x
					numFloats++
				} else {
					addStack(x)
				}
			}
		} else {
			// On Windows amd64 the arguments are passed in the numbered registered.
			// So the first int is in the first integer register and the first float
			// is in the second floating register if there is already a first int.
			// This is in contrast to how macOS and Linux pass arguments which
			// tries to use as many registers as possible in the calling convention.
			addStack = func(x uintptr) {
				sysargs[numStack] = x
				numStack++
			}
			addInt = addStack
			addFloat = addStack
		}

		var keepAlive []interface{}
		defer func() {
			runtime.KeepAlive(keepAlive)
			runtime.KeepAlive(args)
		}()
		for _, v := range args {
			switch v.Kind() {
			case reflect.String:
				ptr := strings.CString(v.String())
				keepAlive = append(keepAlive, ptr)
				addInt(uintptr(unsafe.Pointer(ptr)))
			case reflect.Uintptr, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				addInt(uintptr(v.Uint()))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				addInt(uintptr(v.Int()))
			case reflect.Ptr, reflect.UnsafePointer, reflect.Slice:
				// There is no need to keepAlive this pointer separately because it is kept alive in the args variable
				addInt(v.Pointer())
			case reflect.Func:
				addInt(NewCallback(v.Interface()))
			case reflect.Bool:
				if v.Bool() {
					addInt(1)
				} else {
					addInt(0)
				}
			case reflect.Float32:
				addFloat(uintptr(math.Float32bits(float32(v.Float()))))
			case reflect.Float64:
				addFloat(uintptr(math.Float64bits(v.Float())))
			default:
				panic("purego: unsupported kind: " + v.Kind().String())
			}
		}
		// TODO: support structs
		var r1, r2 uintptr
		if runtime.GOARCH == "arm64" || runtime.GOOS != "windows" {
			// Use the normal arm64 calling convention even on Windows
			syscall := syscall9Args{
				cfn,
				sysargs[0], sysargs[1], sysargs[2], sysargs[3], sysargs[4], sysargs[5], sysargs[6], sysargs[7], sysargs[8],
				floats[0], floats[1], floats[2], floats[3], floats[4], floats[5], floats[6], floats[7],
				0, 0, 0,
			}
			runtime_cgocall(syscall9XABI0, unsafe.Pointer(&syscall))
			r1, r2 = syscall.r1, syscall.r2
		} else {
			// This is a fallback for amd64, 386, and arm. Note this may not support floats
			r1, r2, _ = syscall_syscall9X(cfn, sysargs[0], sysargs[1], sysargs[2], sysargs[3], sysargs[4], sysargs[5], sysargs[6], sysargs[7], sysargs[8])
		}
		if ty.NumOut() == 0 {
			return nil
		}
		outType := ty.Out(0)
		v := reflect.New(outType).Elem()
		switch outType.Kind() {
		case reflect.Uintptr, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v.SetUint(uint64(r1))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v.SetInt(int64(r1))
		case reflect.Bool:
			v.SetBool(byte(r1) != 0)
		case reflect.UnsafePointer:
			// We take the address and then dereference it to trick go vet from creating a possible miss-use of unsafe.Pointer
			v.SetPointer(*(*unsafe.Pointer)(unsafe.Pointer(&r1)))
		case reflect.Ptr:
			// It is safe to have the address of r1 not escape because it is immediately dereferenced with .Elem()
			v = reflect.NewAt(outType, runtime_noescape(unsafe.Pointer(&r1))).Elem()
		case reflect.Func:
			// wrap this C function in a nicely typed Go function
			v = reflect.New(outType)
			RegisterFunc(v.Interface(), r1)
		case reflect.String:
			v.SetString(strings.GoString(r1))
		case reflect.Float32:
			// NOTE: r2 is only the floating return value on 64bit platforms.
			// On 32bit platforms r2 is the upper part of a 64bit return.
			v.SetFloat(float64(math.Float32frombits(uint32(r2))))
		case reflect.Float64:
			// NOTE: r2 is only the floating return value on 64bit platforms.
			// On 32bit platforms r2 is the upper part of a 64bit return.
			v.SetFloat(math.Float64frombits(uint64(r2)))
		default:
			panic("purego: unsupported return kind: " + outType.Kind().String())
		}
		return []reflect.Value{v}
	})
	fn.Set(v)
}

func numOfIntegerRegisters() int {
	switch runtime.GOARCH {
	case "arm64":
		return 8
	case "amd64":
		return 6
	// TODO: figure out why 386 tests are not working
	/*case "386":
		return 0
	case "arm":
		return 4*/
	default:
		panic("purego: unknown GOARCH (" + runtime.GOARCH + ")")
	}
}
