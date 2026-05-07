package emscripten

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/internal/wasm"
	"github.com/tetratelabs/wazero/sys"
)

const FunctionNotifyMemoryGrowth = "emscripten_notify_memory_growth"

var NotifyMemoryGrowth = &wasm.HostFunc{
	ExportName: FunctionNotifyMemoryGrowth,
	Name:       FunctionNotifyMemoryGrowth,
	ParamTypes: []wasm.ValueType{wasm.ValueTypeI32},
	ParamNames: []string{"memory_index"},
	Code:       wasm.Code{GoFunc: api.GoModuleFunc(func(context.Context, api.Module, []uint64) {})},
}

// Emscripten uses this host method to throw an error that can then be caught
// in the dynamic invoke functions. Emscripten uses this to allow for
// setjmp/longjmp support. When this error is seen in the invoke handler,
// it ignores the error and does not act on it.
const FunctionThrowLongjmp = "_emscripten_throw_longjmp"

var (
	ThrowLongjmpError = errors.New("_emscripten_throw_longjmp")
	ThrowLongjmp      = &wasm.HostFunc{
		ExportName: FunctionThrowLongjmp,
		Name:       FunctionThrowLongjmp,
		ParamTypes: []wasm.ValueType{},
		ParamNames: []string{},
		Code: wasm.Code{GoFunc: api.GoModuleFunc(func(context.Context, api.Module, []uint64) {
			panic(ThrowLongjmpError)
		})},
	}
)

// InvokePrefix is the naming convention of Emscripten dynamic functions.
//
// All `invoke_` functions have an initial "index" parameter of
// api.ValueTypeI32. This is the index of the desired funcref in the only table
// in the module. The type of the funcref is via naming convention. The first
// character after `invoke_` decides the result type: 'v' for no result or 'i'
// for api.ValueTypeI32. Any count of 'i' following that are api.ValueTypeI32
// parameters.
//
// For example, the function `invoke_iiiii` signature has five parameters, but
// also five i's. The five 'i's mean there are four parameters
//
//	(import "env" "invoke_iiiii" (func $invoke_iiiii
//		(param i32 i32 i32 i32 i32) (result i32))))
//
// So, the above function if invoked `invoke_iiiii(1234, 1, 2, 3, 4)` would
// look up the funcref at table index 1234, which has a type i32i32i3232_i32
// and invoke it with the remaining parameters.
const InvokePrefix = "invoke_"

func NewInvokeFunc(importName string, params, results []api.ValueType) *wasm.HostFunc {
	// The type we invoke is the same type as the import except without the
	// index parameter.
	fn := &InvokeFunc{&wasm.FunctionType{Results: results}}
	if len(params) > 1 {
		fn.FunctionType.Params = params[1:]
	}

	// Now, make friendly parameter names.
	paramNames := make([]string, len(params))
	paramNames[0] = "index"
	for i := 1; i < len(paramNames); i++ {
		paramNames[i] = "a" + strconv.Itoa(i)
	}
	return &wasm.HostFunc{
		ExportName:  importName,
		ParamTypes:  params,
		ParamNames:  paramNames,
		ResultTypes: results,
		Code:        wasm.Code{GoFunc: fn},
	}
}

type InvokeFunc struct {
	*wasm.FunctionType
}

// Call implements api.GoModuleFunction by special casing dynamic calls needed
// for emscripten `invoke_` functions such as `invoke_ii` or `invoke_v`.
func (v *InvokeFunc) Call(ctx context.Context, mod api.Module, stack []uint64) {
	m := mod.(*wasm.ModuleInstance)

	// Lookup the type of the function we are calling indirectly.
	typeID := m.GetFunctionTypeID(v.FunctionType)

	// This needs copy (not reslice) because the stack is reused for results.
	// Consider invoke_i (zero arguments, one result): index zero (tableOffset)
	// is needed to store the result.
	tableOffset := wasm.Index(stack[0]) // position in the module's only table.
	copy(stack, stack[1:])              // pop the tableOffset.

	// Lookup the table index we will call.
	t := m.Tables[0] // Note: Emscripten doesn't use multiple tables
	f := m.LookupFunction(t, typeID, tableOffset)

	// The Go implementation below mimics the Emscripten JS behaviour to support
	// longjmps from indirect function calls. The implementation of these
	// indirection function calls in Emscripten JS is like this:
	//
	// function invoke_iii(index,a1,a2) {
	//  var sp = emscripten_stack_get_current();
	//  try {
	//    return getWasmTableEntry(index)(a1,a2);
	//  } catch(e) {
	//    _emscripten_stack_restore(sp);
	//    if (e !== e+0) throw e;
	//    _setThrew(1, 0);
	//  }
	//}

	// This is the equivalent of "var sp = emscripten_stack_get_current();".
	// We reuse savedStack to save allocations. We allocate with a size of 2
	// here to accommodate for the input and output of setThrew.
	var savedStack [2]uint64
	callOrPanic(ctx, mod, "emscripten_stack_get_current", savedStack[:])

	err := f.CallWithStack(ctx, stack)
	if err != nil {
		// Module closed: any calls will just fail with the same error.
		if _, ok := err.(*sys.ExitError); ok {
			panic(err)
		}

		// This is the equivalent of "_emscripten_stack_restore(sp);".
		// Do not overwrite err here to preserve the original error.
		callOrPanic(ctx, mod, "_emscripten_stack_restore", savedStack[:])

		// If we encounter ThrowLongjmpError, this means that the C code did a
		// longjmp, which in turn called _emscripten_throw_longjmp and that is
		// a host function that panics with ThrowLongjmpError. In that case we
		// ignore the error because we have restored the stack to what it was
		// before the indirect function call, so the program can continue.
		// This is the equivalent of the "if (e !== e+0) throw e;" line in the
		// JS implementation, which checks if the error is not a number, which
		// is what the JS implementation throws (Infinity for
		// _emscripten_throw_longjmp, a memory address for C++ exceptions).
		if !errors.Is(err, ThrowLongjmpError) {
			panic(err)
		}

		// This is the equivalent of "_setThrew(1, 0);".
		savedStack[0] = 1
		savedStack[1] = 0
		callOrPanic(ctx, mod, "setThrew", savedStack[:])
	}
}

// maybeCallOrPanic calls a given function if it is exported, otherwise panics.
//
// This ensures if the given name is exported before calling it. In other words, this explicitly checks if an api.Function
// returned by api.Module.ExportedFunction is not nil. This is necessary because directly calling a method which is
// potentially nil interface can be fatal on some platforms due to a bug? in Go/QEMU.
// See https://github.com/tetratelabs/wazero/issues/1621
func callOrPanic(ctx context.Context, m api.Module, name string, stack []uint64) {
	if f := m.ExportedFunction(name); f != nil {
		err := f.CallWithStack(ctx, stack)
		if err != nil {
			panic(err)
		}
	} else {
		panic(fmt.Sprintf("%s not exported", name))
	}
}
