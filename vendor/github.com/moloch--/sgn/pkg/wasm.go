package sgn

import (
	"context"
	cryptorand "crypto/rand"
	_ "embed"
	"errors"
	"fmt"
	"math"
	"slices"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// RandomSeedSize is the seed size used by the Rust encoder's pinned ChaCha20
// random number generator.
const RandomSeedSize = 32

// RandomSeed completely determines the polymorphic choices made by
// EncodeWithSeed. Encoder.Seed remains the independent one-byte ADFL key for
// compatibility with the original Go API.
type RandomSeed [RandomSeedSize]byte

//go:embed sgn.wasm
var embeddedRustWASM []byte

var rustWASMEngine struct {
	once     sync.Once
	runtime  wazero.Runtime
	compiled wazero.CompiledModule
	err      error
}

type rustWASMInstance struct {
	module             api.Module
	memory             api.Memory
	alloc              api.Function
	free               api.Function
	encode             api.Function
	outputPtr          api.Function
	outputLen          api.Function
	errorPtr           api.Function
	errorLen           api.Function
	finalSeed          api.Function
	finalEncodingCount api.Function
}

// EncodeWithSeed encodes payload with the embedded upstream Rust core and a
// caller-supplied deterministic random seed. The same seed, Encoder fields,
// payload, Rust source, and dependency lockfile produce byte-identical output
// in native Rust and WebAssembly builds.
func (encoder *Encoder) EncodeWithSeed(payload []byte, randomSeed RandomSeed) ([]byte, error) {
	if encoder == nil {
		return nil, errors.New("nil encoder")
	}
	if encoder.architecture != 32 && encoder.architecture != 64 {
		return nil, errors.New("invalid architecture")
	}
	if encoder.ObfuscationLimit < 0 || encoder.ObfuscationLimit > math.MaxInt32 {
		return nil, errors.New("invalid obfuscation limit")
	}
	if encoder.EncodingCount < 1 || encoder.EncodingCount > math.MaxInt32 {
		return nil, errors.New("invalid encoding count")
	}
	if len(payload) > math.MaxInt32 {
		return nil, errors.New("payload is too large for WebAssembly memory")
	}

	ctx := context.Background()
	instance, err := newRustWASMInstance(ctx)
	if err != nil {
		return nil, err
	}
	defer instance.close(context.WithoutCancel(ctx))

	inputPtr, err := instance.writeBytes(ctx, payload)
	if err != nil {
		return nil, fmt.Errorf("copy payload into Rust WebAssembly: %w", err)
	}
	if inputPtr != 0 {
		defer instance.release(context.WithoutCancel(ctx), inputPtr, uint32(len(payload)))
	}

	randomSeedPtr, err := instance.writeBytes(ctx, randomSeed[:])
	if err != nil {
		return nil, fmt.Errorf("copy random seed into Rust WebAssembly: %w", err)
	}
	defer instance.release(context.WithoutCancel(ctx), randomSeedPtr, RandomSeedSize)

	results, err := instance.encode.Call(
		ctx,
		uint64(inputPtr),
		uint64(uint32(len(payload))),
		uint64(uint32(encoder.architecture)),
		uint64(uint32(int32(encoder.ObfuscationLimit))),
		boolValue(encoder.PlainDecoder),
		uint64(uint32(encoder.Seed)),
		uint64(uint32(encoder.EncodingCount)),
		boolValue(encoder.SaveRegisters),
		uint64(randomSeedPtr),
		RandomSeedSize,
	)
	if err != nil {
		return nil, fmt.Errorf("call Rust WebAssembly encoder: %w", err)
	}
	if len(results) != 1 {
		return nil, fmt.Errorf("Rust WebAssembly encoder returned %d results, want 1", len(results))
	}
	if status := uint32(results[0]); status != 0 {
		message := instance.readString(ctx, instance.errorPtr, instance.errorLen)
		if message == "" {
			message = fmt.Sprintf("status %d", status)
		}
		return nil, fmt.Errorf("Rust encoder: %s", message)
	}

	output, err := instance.readResult(ctx)
	if err != nil {
		return nil, err
	}
	finalSeed, err := callI32(ctx, instance.finalSeed)
	if err != nil {
		return nil, fmt.Errorf("read final ADFL seed: %w", err)
	}
	finalCount, err := callI32(ctx, instance.finalEncodingCount)
	if err != nil {
		return nil, fmt.Errorf("read final encoding count: %w", err)
	}
	encoder.Seed = byte(finalSeed)
	encoder.EncodingCount = int(finalCount)
	return output, nil
}

func newRustWASMInstance(ctx context.Context) (*rustWASMInstance, error) {
	rustWASMEngine.once.Do(func() {
		if len(embeddedRustWASM) == 0 {
			rustWASMEngine.err = errors.New("embedded Rust WebAssembly module is empty")
			return
		}

		runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfig().WithCloseOnContextDone(true))
		if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
			_ = runtime.Close(context.WithoutCancel(ctx))
			rustWASMEngine.err = fmt.Errorf("instantiate WASI: %w", err)
			return
		}
		compiled, err := runtime.CompileModule(ctx, embeddedRustWASM)
		if err != nil {
			_ = runtime.Close(context.WithoutCancel(ctx))
			rustWASMEngine.err = fmt.Errorf("compile embedded Rust WebAssembly: %w", err)
			return
		}
		if err := validateRustWASM(compiled); err != nil {
			_ = compiled.Close(context.WithoutCancel(ctx))
			_ = runtime.Close(context.WithoutCancel(ctx))
			rustWASMEngine.err = err
			return
		}
		rustWASMEngine.runtime = runtime
		rustWASMEngine.compiled = compiled
	})
	if rustWASMEngine.err != nil {
		return nil, rustWASMEngine.err
	}

	module, err := rustWASMEngine.runtime.InstantiateModule(
		ctx,
		rustWASMEngine.compiled,
		wazero.NewModuleConfig().WithName("").WithRandSource(cryptorand.Reader),
	)
	if err != nil {
		return nil, fmt.Errorf("instantiate Rust WebAssembly: %w", err)
	}
	if initialize := findExportedFunction(module, "_initialize", "__initialize"); initialize != nil {
		if _, err := initialize.Call(ctx); err != nil {
			_ = module.Close(context.WithoutCancel(ctx))
			return nil, fmt.Errorf("initialize Rust WebAssembly: %w", err)
		}
	}

	instance := &rustWASMInstance{
		module:             module,
		memory:             module.Memory(),
		alloc:              module.ExportedFunction("sgn_alloc"),
		free:               module.ExportedFunction("sgn_free"),
		encode:             module.ExportedFunction("sgn_encode"),
		outputPtr:          module.ExportedFunction("sgn_output_ptr"),
		outputLen:          module.ExportedFunction("sgn_output_len"),
		errorPtr:           module.ExportedFunction("sgn_error_ptr"),
		errorLen:           module.ExportedFunction("sgn_error_len"),
		finalSeed:          module.ExportedFunction("sgn_final_seed"),
		finalEncodingCount: module.ExportedFunction("sgn_final_encoding_count"),
	}
	return instance, nil
}

func validateRustWASM(compiled wazero.CompiledModule) error {
	if _, ok := compiled.ExportedMemories()["memory"]; !ok {
		return errors.New("Rust WebAssembly does not export memory")
	}

	i32 := api.ValueTypeI32
	required := []struct {
		name    string
		params  []api.ValueType
		results []api.ValueType
	}{
		{name: "sgn_alloc", params: []api.ValueType{i32}, results: []api.ValueType{i32}},
		{name: "sgn_free", params: []api.ValueType{i32, i32}},
		{name: "sgn_encode", params: []api.ValueType{i32, i32, i32, i32, i32, i32, i32, i32, i32, i32}, results: []api.ValueType{i32}},
		{name: "sgn_output_ptr", results: []api.ValueType{i32}},
		{name: "sgn_output_len", results: []api.ValueType{i32}},
		{name: "sgn_error_ptr", results: []api.ValueType{i32}},
		{name: "sgn_error_len", results: []api.ValueType{i32}},
		{name: "sgn_final_seed", results: []api.ValueType{i32}},
		{name: "sgn_final_encoding_count", results: []api.ValueType{i32}},
	}

	exports := compiled.ExportedFunctions()
	for _, expected := range required {
		definition, ok := exports[expected.name]
		if !ok {
			return fmt.Errorf("Rust WebAssembly does not export %s", expected.name)
		}
		if !slices.Equal(definition.ParamTypes(), expected.params) || !slices.Equal(definition.ResultTypes(), expected.results) {
			return fmt.Errorf(
				"Rust WebAssembly export %s has params=%v results=%v; want params=%v results=%v",
				expected.name,
				definition.ParamTypes(),
				definition.ResultTypes(),
				expected.params,
				expected.results,
			)
		}
	}
	return nil
}

func (instance *rustWASMInstance) writeBytes(ctx context.Context, value []byte) (uint32, error) {
	if len(value) == 0 {
		return 0, nil
	}
	results, err := instance.alloc.Call(ctx, uint64(uint32(len(value))))
	if err != nil {
		return 0, err
	}
	if len(results) != 1 || uint32(results[0]) == 0 {
		return 0, errors.New("guest allocation failed")
	}
	pointer := uint32(results[0])
	if !instance.memory.Write(pointer, value) {
		instance.release(context.WithoutCancel(ctx), pointer, uint32(len(value)))
		return 0, errors.New("guest memory write was out of bounds")
	}
	return pointer, nil
}

func (instance *rustWASMInstance) readResult(ctx context.Context) ([]byte, error) {
	pointer, err := callI32(ctx, instance.outputPtr)
	if err != nil {
		return nil, fmt.Errorf("read Rust output pointer: %w", err)
	}
	length, err := callI32(ctx, instance.outputLen)
	if err != nil {
		return nil, fmt.Errorf("read Rust output length: %w", err)
	}
	if length == 0 {
		return []byte{}, nil
	}
	data, ok := instance.memory.Read(pointer, length)
	if !ok {
		return nil, errors.New("Rust output is outside WebAssembly memory")
	}
	return append([]byte(nil), data...), nil
}

func (instance *rustWASMInstance) readString(ctx context.Context, pointerFunction, lengthFunction api.Function) string {
	pointer, err := callI32(ctx, pointerFunction)
	if err != nil {
		return ""
	}
	length, err := callI32(ctx, lengthFunction)
	if err != nil || length == 0 {
		return ""
	}
	data, ok := instance.memory.Read(pointer, length)
	if !ok {
		return ""
	}
	return string(data)
}

func (instance *rustWASMInstance) release(ctx context.Context, pointer uint32, length uint32) {
	if pointer == 0 {
		return
	}
	_, _ = instance.free.Call(ctx, uint64(pointer), uint64(length))
}

func (instance *rustWASMInstance) close(ctx context.Context) {
	_ = instance.module.Close(ctx)
}

func callI32(ctx context.Context, function api.Function) (uint32, error) {
	results, err := function.Call(ctx)
	if err != nil {
		return 0, err
	}
	if len(results) != 1 {
		return 0, fmt.Errorf("function returned %d results, want 1", len(results))
	}
	return uint32(results[0]), nil
}

func boolValue(value bool) uint64 {
	if value {
		return 1
	}
	return 0
}

func findExportedFunction(module api.Module, names ...string) api.Function {
	for _, name := range names {
		if function := module.ExportedFunction(name); function != nil {
			return function
		}
	}
	return nil
}
