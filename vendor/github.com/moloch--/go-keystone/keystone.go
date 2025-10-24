package keystone

import (
	"bytes"
	"context"
	"embed"
	"fmt"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// just for prevent [import _ "embed"] :)
var _ embed.FS

//go:embed wasm/keystone.wasm
var module []byte

// Engine contain wasm runtime and keystone engine.
type Engine struct {
	arch Arch
	mode Mode

	context context.Context
	runtime wazero.Runtime
	module  api.Module
	memory  api.Memory

	_malloc     api.Function
	_free       api.Function
	_ksOpen     api.Function
	_ksOption   api.Function
	_ksAsm      api.Function
	_ksFree     api.Function
	_ksClose    api.Function
	_ksErrno    api.Function
	_ksStrerror api.Function
	_ksVersion  api.Function

	engine  uint64
	version string
}

// NewEngine is used to create keystone engine above wasm interpreter.
func NewEngine(arch Arch, mode Mode) (*Engine, error) {
	ctx := context.Background()
	// prevent generate RWX memory
	rc := wazero.NewRuntimeConfigInterpreter()
	runtime := wazero.NewRuntimeWithConfig(ctx, rc)
	// if failed to create engine, close the wasm runtime
	var ok bool
	defer func() {
		if !ok {
			_ = runtime.Close(ctx)
		}
	}()
	// load keystone wasm module
	err := processImport(runtime)
	if err != nil {
		return nil, fmt.Errorf("failed to process wasm module import: %s", err)
	}
	mod, err := runtime.Instantiate(ctx, module)
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate wasm module: %s", err)
	}
	// initialize keystone engine
	engine := Engine{
		arch: arch,
		mode: mode,

		context: ctx,
		runtime: runtime,
		module:  mod,
		memory:  mod.Memory(),

		_malloc: mod.ExportedFunction(_malloc),
		_free:   mod.ExportedFunction(_free),

		_ksOpen:     mod.ExportedFunction(_ks_open),
		_ksOption:   mod.ExportedFunction(_ks_option),
		_ksAsm:      mod.ExportedFunction(_ks_asm),
		_ksFree:     mod.ExportedFunction(_ks_free),
		_ksClose:    mod.ExportedFunction(_ks_close),
		_ksErrno:    mod.ExportedFunction(_ks_errno),
		_ksStrerror: mod.ExportedFunction(_ks_strerror),
		_ksVersion:  mod.ExportedFunction(_ks_version),
	}
	err = engine.initialize()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize keystone engine: %s", err)
	}
	ok = true
	return &engine, nil
}

// processImport is used to create a module with padding
// functions for call runtime.InstantiateModule.
func processImport(runtime wazero.Runtime) error {
	builder := runtime.NewHostModuleBuilder(importModule)
	fb := builder.NewFunctionBuilder()

	padFn1 := func(int32, int32, int32) {
	}
	fb.WithFunc(padFn1).Export(___cxa_throw)

	padFn2 := func(int32, int32) int32 {
		return 0
	}
	fb.WithFunc(padFn2).Export(___syscall_fstat64)

	padFn3 := func(buf int32, size int32) int32 {
		return 1
	}
	fb.WithFunc(padFn3).Export(___syscall_getcwd)

	padFn4 := func(int32, int32) int32 {
		return 0
	}
	fb.WithFunc(padFn4).Export(___syscall_lstat64)

	padFn5 := func(int32, int32, int32, int32) int32 {
		return 0
	}
	fb.WithFunc(padFn5).Export(___syscall_newfstatat)

	padFn6 := func(int32, int32, int32, int32) int32 {
		return 0
	}
	fb.WithFunc(padFn6).Export(___syscall_openat)

	padFn7 := func(int32, int32) int32 {
		return 0
	}
	fb.WithFunc(padFn7).Export(___syscall_stat64)

	padFn8 := func() {
	}
	fb.WithFunc(padFn8).Export(__abort_js)

	padFn9 := func(int32, int32, int32, int32, int64, int32, int32) int32 {
		return 1
	}
	fb.WithFunc(padFn9).Export(__mmap_js)

	padFn10 := func(int32, int32, int32, int32, int32, int64) int32 {
		return 1
	}
	fb.WithFunc(padFn10).Export(__munmap_js)

	padFn11 := func(v int32) int32 {
		return 0
	}
	fb.WithFunc(padFn11).Export(_emscripten_resize_heap)

	padFn12 := func(int32, int32) int32 {
		return 1
	}
	fb.WithFunc(padFn12).Export(_environ_get)

	padFn13 := func(int32, int32) int32 {
		return 1
	}
	fb.WithFunc(padFn13).Export(_environ_sizes_get)

	padFn14 := func(int32) {
	}
	fb.WithFunc(padFn14).Export(_exit)

	padFn15 := func(int32) int32 {
		return 1
	}
	fb.WithFunc(padFn15).Export(_fd_close)

	padFn16 := func(int32, int32) int32 {
		return 1
	}
	fb.WithFunc(padFn16).Export(_fd_fdstat_get)

	padFn17 := func(int32, int32, int32, int64, int32) int32 {
		return 1
	}
	fb.WithFunc(padFn17).Export(_fd_pread)

	padFn18 := func(int32, int32, int32, int32) int32 {
		return 1
	}
	fb.WithFunc(padFn18).Export(_fd_read)

	padFn19 := func(int32, int64, int32, int32) int32 {
		return 1
	}
	fb.WithFunc(padFn19).Export(_fd_seek)

	padFn20 := func(int32, int32, int32, int32) int32 {
		return 1
	}
	fb.WithFunc(padFn20).Export(_fd_write)

	_, err := builder.Instantiate(context.Background())
	return err
}

func (e *Engine) malloc(n uint32) uint32 {
	rets, err := e._malloc.Call(e.context, uint64(n))
	if err != nil {
		return 0
	}
	return uint32(rets[0])
}

func (e *Engine) free(ptr uint32) {
	_, err := e._malloc.Call(e.context, uint64(ptr))
	if err != nil {
		panic(fmt.Sprintf("failed to free 0x%X: %s", ptr, err))
	}
}

func (e *Engine) errno() uint32 {
	rets, err := e._ksErrno.Call(e.context, e.engine)
	if err != nil {
		panic(fmt.Sprintf("failed to get errno: %s", err))
	}
	return uint32(rets[0])
}

func (e *Engine) errnoStr(errno uint32) string {
	rets, err := e._ksStrerror.Call(e.context, uint64(errno))
	if err != nil {
		panic(fmt.Sprintf("failed to get errno string: %s", err))
	}
	s := uint32(rets[0])
	if s == 0 {
		return ""
	}
	eb := make([]byte, 0, 64)
	for {
		b, _ := e.memory.ReadByte(s)
		if b == 0x00 {
			break
		}
		eb = append(eb, b)
		s++
	}
	return string(eb)
}

func (e *Engine) initialize() error {
	// open keystone engine
	enginePtr := e.malloc(4)
	defer e.free(enginePtr)
	rets, err := e._ksOpen.Call(e.context,
		uint64(e.arch), uint64(e.mode), uint64(enginePtr),
	)
	if err != nil {
		return fmt.Errorf("failed to call ks_open: %s", err)
	}
	errno := Error(rets[0])
	if errno != ERR_OK {
		return fmt.Errorf("failed to open keystone engine: %s", e.errnoStr(errno))
	}
	engine, _ := e.memory.ReadUint32Le(enginePtr)
	e.engine = uint64(engine)
	// get keystone engine version
	rets, err = e._ksVersion.Call(e.context, 0, 0)
	if err != nil {
		return fmt.Errorf("failed to call ks_version: %s", err)
	}
	ver := rets[0]
	marjo := ver >> 8
	minor := ver & 0xFF
	e.version = fmt.Sprintf("%d.%d", marjo, minor)
	return nil
}

// Option is used to set the assembly option.
func (e *Engine) Option(typ OptionType, val OptionValue) error {
	rets, err := e._ksOption.Call(e.context,
		e.engine, uint64(typ), uint64(val),
	)
	if err != nil {
		return fmt.Errorf("failed to call ks_option: %s", err)
	}
	errno := Error(rets[0])
	if errno != ERR_OK {
		return fmt.Errorf("failed to set keystone option: %s", e.errnoStr(errno))
	}
	return nil
}

// Assemble is used to assemble input source code.
func (e *Engine) Assemble(src string, addr uint64) ([]byte, error) {
	// allocate memory and write source code
	src += "\x00"
	srcPtr := e.malloc(uint32(len(src)))
	defer e.free(srcPtr)
	e.memory.WriteString(srcPtr, src)
	// allocate memory for store pointer to output instruction
	instAddr := e.malloc(4)
	defer e.free(instAddr)
	instSize := e.malloc(4)
	defer e.free(instSize)
	statCount := e.malloc(4)
	defer e.free(statCount)
	// assemble input source code
	rets, err := e._ksAsm.Call(e.context,
		e.engine, uint64(srcPtr), addr,
		uint64(instAddr), uint64(instSize), uint64(statCount),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to call ks_asm: %s", err)
	}
	errno := Error(rets[0])
	if errno != ERR_OK {
		return nil, fmt.Errorf("failed to assemble: %s", e.errnoStr(e.errno()))
	}
	// copy output instruction to host memory
	instPtr, _ := e.memory.ReadUint32Le(instAddr)
	instLen, _ := e.memory.ReadUint32Le(instSize)
	inst, _ := e.memory.Read(instPtr, instLen)
	inst = bytes.Clone(inst)
	// free output instruction memory
	_, err = e._ksFree.Call(e.context, uint64(instPtr))
	if err != nil {
		return nil, fmt.Errorf("failed to call ks_free: %s", err)
	}
	return inst, nil
}

// Version is used to get the keystone engine version.
func (e *Engine) Version() string {
	return e.version
}

// Close is used to close keystone engine and wasm runtime.
func (e *Engine) Close() error {
	// close keystone engine
	rets, err := e._ksClose.Call(e.context, e.engine)
	if err != nil {
		return fmt.Errorf("failed to call ks_close: %s", err)
	}
	errno := Error(rets[0])
	if errno != ERR_OK {
		return fmt.Errorf("failed to close keystone engine: %s", e.errnoStr(errno))
	}
	// close wasm module
	err = e.module.Close(e.context)
	if err != nil {
		return fmt.Errorf("failed to close wasm module: %s", err)
	}
	// close wasm runtime
	err = e.runtime.Close(e.context)
	if err != nil {
		return fmt.Errorf("failed to close wasm runtime: %s", err)
	}
	return nil
}
