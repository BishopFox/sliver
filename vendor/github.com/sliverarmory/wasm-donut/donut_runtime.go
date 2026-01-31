package wasmdonut

import (
	"context"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/emscripten"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	DonutArchX86 = 1
	DonutArchX64 = 2
	DonutArchX84 = 3

	DonutEntropyNone    = 1
	DonutEntropyRandom  = 2
	DonutEntropyDefault = 3

	DonutCompressNone  = 1
	DonutCompressAplib = 2

	DonutExitThread  = 1
	DonutExitProcess = 2
	DonutExitBlock   = 3
)

//go:embed donut/wasm/donut.wasm
var embeddedWasm []byte

// GenerateOptions configures Donut shellcode generation.
type GenerateOptions struct {
	Ext      string
	Args     string
	Class    string
	Method   string
	Arch     int
	Entropy  int
	Compress int
	ExitOpt  int
}

// Generate runs the embedded Donut wasm module and returns shellcode bytes.
func Generate(ctx context.Context, input []byte, ext string, opts GenerateOptions) ([]byte, error) {
	if len(input) == 0 {
		return nil, errors.New("input is empty")
	}
	if ext == "" {
		return nil, errors.New("extension is required")
	}
	if len(embeddedWasm) == 0 {
		return nil, errors.New("embedded wasm is empty; run donut/wasm/build.sh")
	}

	arch := opts.Arch
	if arch == 0 {
		arch = DonutArchX84
	}
	entropy := opts.Entropy
	if entropy == 0 {
		entropy = DonutEntropyNone
	}
	compress := opts.Compress
	if compress == 0 {
		compress = DonutCompressNone
	}
	exitOpt := opts.ExitOpt
	if exitOpt == 0 {
		exitOpt = DonutExitThread
	}

	runtime := wazero.NewRuntime(ctx)
	defer func() {
		_ = runtime.Close(ctx)
	}()

	wasi_snapshot_preview1.MustInstantiate(ctx, runtime)

	compiled, err := runtime.CompileModule(ctx, embeddedWasm)
	if err != nil {
		return nil, fmt.Errorf("compile wasm: %w", err)
	}

	envCloser, err := emscripten.InstantiateForModule(ctx, runtime, compiled)
	if err != nil {
		return nil, fmt.Errorf("instantiate emscripten env: %w", err)
	}
	defer func() {
		_ = envCloser.Close(ctx)
	}()

	module, err := runtime.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		return nil, fmt.Errorf("instantiate wasm: %w", err)
	}
	defer func() {
		_ = module.Close(ctx)
	}()

	mallocFn, err := exportedFunc(module, "malloc", "_malloc")
	if err != nil {
		return nil, fmt.Errorf("malloc export: %w", err)
	}
	freeFn, err := exportedFunc(module, "free", "_free")
	if err != nil {
		return nil, fmt.Errorf("free export: %w", err)
	}
	genFn, err := exportedFunc(module, "donut_generate", "_donut_generate")
	if err != nil {
		return nil, fmt.Errorf("donut_generate export: %w", err)
	}

	mem := module.Memory()
	if mem == nil {
		return nil, errors.New("module has no exported memory")
	}

	inPtr, err := allocBytes(ctx, mallocFn, mem, input)
	if err != nil {
		return nil, err
	}
	defer callFree(ctx, freeFn, inPtr)

	extPtr, err := allocCString(ctx, mallocFn, mem, ext)
	if err != nil {
		return nil, err
	}
	defer callFree(ctx, freeFn, extPtr)

	argsPtr := uint32(0)
	if opts.Args != "" {
		argsPtr, err = allocCString(ctx, mallocFn, mem, opts.Args)
		if err != nil {
			return nil, err
		}
		defer callFree(ctx, freeFn, argsPtr)
	}

	clsPtr := uint32(0)
	if opts.Class != "" {
		clsPtr, err = allocCString(ctx, mallocFn, mem, opts.Class)
		if err != nil {
			return nil, err
		}
		defer callFree(ctx, freeFn, clsPtr)
	}

	methodPtr := uint32(0)
	if opts.Method != "" {
		methodPtr, err = allocCString(ctx, mallocFn, mem, opts.Method)
		if err != nil {
			return nil, err
		}
		defer callFree(ctx, freeFn, methodPtr)
	}

	resultPtr, err := allocBytes(ctx, mallocFn, mem, make([]byte, 12))
	if err != nil {
		return nil, err
	}
	defer callFree(ctx, freeFn, resultPtr)

	callArgs := []uint64{
		uint64(inPtr),
		uint64(len(input)),
		uint64(extPtr),
		uint64(argsPtr),
		uint64(clsPtr),
		uint64(methodPtr),
		uint64(arch),
		uint64(entropy),
		uint64(compress),
		uint64(exitOpt),
		uint64(resultPtr),
	}

	res, err := genFn.Call(ctx, callArgs...)
	if err != nil {
		return nil, fmt.Errorf("donut_generate call: %w", err)
	}
	if len(res) > 0 && res[0] != 0 {
		return nil, fmt.Errorf("donut_generate returned error code: %d", res[0])
	}

	resultBytes, ok := mem.Read(resultPtr, 12)
	if !ok {
		return nil, errors.New("read result struct")
	}

	outPtr := binary.LittleEndian.Uint32(resultBytes[0:4])
	outLen := binary.LittleEndian.Uint32(resultBytes[4:8])
	outErr := int32(binary.LittleEndian.Uint32(resultBytes[8:12]))

	if outErr != 0 {
		return nil, fmt.Errorf("donut_generate failed with error code: %d", outErr)
	}
	if outPtr == 0 || outLen == 0 {
		return nil, errors.New("donut_generate returned empty output")
	}

	outBytes, ok := mem.Read(outPtr, outLen)
	if !ok {
		return nil, errors.New("read output bytes")
	}

	callFree(ctx, freeFn, outPtr)

	return outBytes, nil
}

// GenerateFromFile loads input from disk and returns generated shellcode.
func GenerateFromFile(ctx context.Context, path string, opts GenerateOptions) ([]byte, error) {
	if path == "" {
		return nil, errors.New("input path is required")
	}

	input, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}

	ext := opts.Ext
	if ext == "" {
		ext = strings.ToLower(filepath.Ext(path))
		if ext == "" {
			return nil, errors.New("could not infer extension; pass Ext in options")
		}
	}

	return Generate(ctx, input, ext, opts)
}

// GenerateToFile writes shellcode generated from the input path to outPath.
func GenerateToFile(ctx context.Context, inPath, outPath string, opts GenerateOptions) error {
	if outPath == "" {
		return errors.New("output path is required")
	}

	outBytes, err := GenerateFromFile(ctx, inPath, opts)
	if err != nil {
		return err
	}

	if err := os.WriteFile(outPath, outBytes, 0644); err != nil {
		return fmt.Errorf("write output: %w", err)
	}

	return nil
}

func exportedFunc(module api.Module, names ...string) (api.Function, error) {
	for _, name := range names {
		if fn := module.ExportedFunction(name); fn != nil {
			return fn, nil
		}
	}
	return nil, fmt.Errorf("export not found (tried %s)", strings.Join(names, ", "))
}

func allocBytes(ctx context.Context, malloc api.Function, mem api.Memory, data []byte) (uint32, error) {
	res, err := malloc.Call(ctx, uint64(len(data)))
	if err != nil {
		return 0, fmt.Errorf("malloc: %w", err)
	}
	ptr := uint32(res[0])
	if ptr == 0 {
		return 0, errors.New("malloc returned null")
	}
	if len(data) == 0 {
		return ptr, nil
	}
	if ok := mem.Write(ptr, data); !ok {
		return 0, errors.New("memory write failed")
	}
	return ptr, nil
}

func allocCString(ctx context.Context, malloc api.Function, mem api.Memory, s string) (uint32, error) {
	if strings.ContainsRune(s, '\x00') {
		return 0, errors.New("string contains NUL byte")
	}
	data := append([]byte(s), 0)
	return allocBytes(ctx, malloc, mem, data)
}

func callFree(ctx context.Context, free api.Function, ptr uint32) {
	if ptr == 0 {
		return
	}
	_, _ = free.Call(ctx, uint64(ptr))
}
