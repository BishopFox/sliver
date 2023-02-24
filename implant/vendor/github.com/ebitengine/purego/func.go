// SPDX-License-Identifier: Apache-2.0
// SPDX-FileCopyrightText: 2022 The Ebitengine Authors

//go:build darwin || linux

package purego

import (
	"math"
	"reflect"
	"runtime"
	"unsafe"

	"github.com/ebitengine/purego/internal/strings"
)

// RegisterLibFunc is a wrapper around RegisterFunc that uses the C function returned from Dlsym(handle, name).
// It panics if Dlsym fails.
//
// Windows does not support this function.
func RegisterLibFunc(fptr interface{}, handle uintptr, name string) {
	sym := Dlsym(handle, name)
	if sym == 0 {
		panic("purego: couldn't find symbol: " + Dlerror())
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
// Type Conversions (Go => C)
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
// Windows does not support this function.
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
				if floats < 8 {
					floats++
				} else {
					stack++
				}
			default:
				panic("purego: unsupported kind " + arg.Kind().String())
			}
		}
		if ints+stack > maxArgs || floats+stack > maxArgs {
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
		var stack = sysargs[numOfIntegerRegisters():]
		var floats [8]float64
		var numInts int
		var numFloats int
		var numStack int
		addStack := func(x uintptr) {
			stack[numStack] = x
			numStack++
		}
		addInt := func(x uintptr) {
			if numInts >= numOfIntegerRegisters() {
				addStack(x)
			} else {
				sysargs[numInts] = x
				numInts++
			}
		}
		var keepAlive []interface{}
		defer func() {
			runtime.KeepAlive(keepAlive)
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
				keepAlive = append(keepAlive, v.Pointer())
				addInt(v.Pointer())
			case reflect.Func:
				addInt(NewCallback(v.Interface()))
			case reflect.Bool:
				if v.Bool() {
					addInt(1)
				} else {
					addInt(0)
				}
			case reflect.Float32, reflect.Float64:
				if numFloats < len(floats) {
					floats[numFloats] = v.Float()
					numFloats++
				} else {
					addStack(uintptr(math.Float64bits(v.Float())))
				}
			default:
				panic("purego: unsupported kind: " + v.Kind().String())
			}
		}
		// TODO: support structs
		syscall := syscall9Args{
			cfn,
			sysargs[0], sysargs[1], sysargs[2], sysargs[3], sysargs[4], sysargs[5], sysargs[6], sysargs[7], sysargs[8],
			floats[0], floats[1], floats[2], floats[3], floats[4], floats[5], floats[6], floats[7],
			0, 0, 0}
		runtime_cgocall(syscall9XABI0, unsafe.Pointer(&syscall))
		r1, r2 := syscall.r1, syscall.r2

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
			v.SetBool(r1 != 0)
		case reflect.UnsafePointer:
			// We take the address and then dereference it to trick go vet from creating a possible miss-use of unsafe.Pointer
			v.SetPointer(*(*unsafe.Pointer)(unsafe.Pointer(&r1)))
		case reflect.Ptr:
			v = reflect.NewAt(outType, unsafe.Pointer(&r1)).Elem()
		case reflect.Func:
			// wrap this C function in a nicely typed Go function
			v = reflect.New(outType)
			RegisterFunc(v.Interface(), r1)
		case reflect.String:
			v.SetString(strings.GoString(r1))
		case reflect.Float32, reflect.Float64:
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
	default:
		panic("purego: unknown GOARCH (" + runtime.GOARCH + ")")
	}
}
