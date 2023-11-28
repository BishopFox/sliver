package util

import (
	"context"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

type i32 interface{ ~int32 | ~uint32 }
type i64 interface{ ~int64 | ~uint64 }

type funcII[TR, T0 i32] func(context.Context, api.Module, T0) TR

func (fn funcII[TR, T0]) Call(ctx context.Context, mod api.Module, stack []uint64) {
	stack[0] = uint64(fn(ctx, mod, T0(stack[0])))
}

func ExportFuncII[TR, T0 i32](mod wazero.HostModuleBuilder, name string, fn func(context.Context, api.Module, T0) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(funcII[TR, T0](fn),
			[]api.ValueType{api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

type funcIII[TR, T0, T1 i32] func(context.Context, api.Module, T0, T1) TR

func (fn funcIII[TR, T0, T1]) Call(ctx context.Context, mod api.Module, stack []uint64) {
	stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1])))
}

func ExportFuncIII[TR, T0, T1 i32](mod wazero.HostModuleBuilder, name string, fn func(context.Context, api.Module, T0, T1) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(funcIII[TR, T0, T1](fn),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

type funcIIII[TR, T0, T1, T2 i32] func(context.Context, api.Module, T0, T1, T2) TR

func (fn funcIIII[TR, T0, T1, T2]) Call(ctx context.Context, mod api.Module, stack []uint64) {
	stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1]), T2(stack[2])))
}

func ExportFuncIIII[TR, T0, T1, T2 i32](mod wazero.HostModuleBuilder, name string, fn func(context.Context, api.Module, T0, T1, T2) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(funcIIII[TR, T0, T1, T2](fn),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

type funcIIIII[TR, T0, T1, T2, T3 i32] func(context.Context, api.Module, T0, T1, T2, T3) TR

func (fn funcIIIII[TR, T0, T1, T2, T3]) Call(ctx context.Context, mod api.Module, stack []uint64) {
	stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1]), T2(stack[2]), T3(stack[3])))
}

func ExportFuncIIIII[TR, T0, T1, T2, T3 i32](mod wazero.HostModuleBuilder, name string, fn func(context.Context, api.Module, T0, T1, T2, T3) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(funcIIIII[TR, T0, T1, T2, T3](fn),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

type funcIIIIII[TR, T0, T1, T2, T3, T4 i32] func(context.Context, api.Module, T0, T1, T2, T3, T4) TR

func (fn funcIIIIII[TR, T0, T1, T2, T3, T4]) Call(ctx context.Context, mod api.Module, stack []uint64) {
	stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1]), T2(stack[2]), T3(stack[3]), T4(stack[4])))
}

func ExportFuncIIIIII[TR, T0, T1, T2, T3, T4 i32](mod wazero.HostModuleBuilder, name string, fn func(context.Context, api.Module, T0, T1, T2, T3, T4) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(funcIIIIII[TR, T0, T1, T2, T3, T4](fn),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

type funcIIIIJ[TR, T0, T1, T2 i32, T3 i64] func(context.Context, api.Module, T0, T1, T2, T3) TR

func (fn funcIIIIJ[TR, T0, T1, T2, T3]) Call(ctx context.Context, mod api.Module, stack []uint64) {
	stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1]), T2(stack[2]), T3(stack[3])))
}

func ExportFuncIIIIJ[TR, T0, T1, T2 i32, T3 i64](mod wazero.HostModuleBuilder, name string, fn func(context.Context, api.Module, T0, T1, T2, T3) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(funcIIIIJ[TR, T0, T1, T2, T3](fn),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}

type funcIIJ[TR, T0 i32, T1 i64] func(context.Context, api.Module, T0, T1) TR

func (fn funcIIJ[TR, T0, T1]) Call(ctx context.Context, mod api.Module, stack []uint64) {
	stack[0] = uint64(fn(ctx, mod, T0(stack[0]), T1(stack[1])))
}

func ExportFuncIIJ[TR, T0 i32, T1 i64](mod wazero.HostModuleBuilder, name string, fn func(context.Context, api.Module, T0, T1) TR) {
	mod.NewFunctionBuilder().
		WithGoModuleFunction(funcIIJ[TR, T0, T1](fn),
			[]api.ValueType{api.ValueTypeI32, api.ValueTypeI64}, []api.ValueType{api.ValueTypeI32}).
		Export(name)
}
