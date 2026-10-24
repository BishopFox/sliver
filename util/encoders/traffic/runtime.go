package traffic

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"runtime"
	"time"

	"github.com/bishopfox/sliver/implant/sliver/wasmnet"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const trafficEncoderMemoryLimitPages = 4096 // 256 MiB

//nolint:gocyclo // Runtime setup is intentionally kept together so every partial initialization is cleaned up.
func createTrafficEncoder(
	name string,
	wasm []byte,
	logger TrafficEncoderLogCallback,
	runtimeConfig wazero.RuntimeConfig,
) (_ *TrafficEncoder, err error) {
	if name == wasmnet.ModuleName {
		return nil, fmt.Errorf("traffic encoder name %q is reserved", name)
	}

	ctx, stop := context.WithCancel(context.Background())
	wasmRuntime := wazero.NewRuntimeWithConfig(
		ctx,
		runtimeConfig.
			WithCloseOnContextDone(true).
			WithMemoryLimitPages(trafficEncoderMemoryLimitPages),
	)
	network := wasmnet.New(ctx)
	var mod api.Module
	defer func() {
		if err == nil {
			return
		}
		stop()
		_ = network.Close()
		if mod != nil {
			_ = mod.Close(context.Background())
		}
		_ = wasmRuntime.Close(context.Background())
	}()

	_, err = wasmRuntime.NewHostModuleBuilder(name).
		NewFunctionBuilder().WithFunc(func() uint64 {
		var buffer [8]byte
		if _, readErr := rand.Read(buffer[:]); readErr != nil {
			return 0
		}
		return binary.LittleEndian.Uint64(buffer[:])
	}).Export("rand").
		NewFunctionBuilder().WithFunc(func() int64 {
		return time.Now().UnixNano()
	}).Export("time").
		NewFunctionBuilder().WithFunc(func(_ context.Context, module api.Module, offset, byteCount uint32) {
		buffer, ok := module.Memory().Read(offset, byteCount)
		if !ok {
			logger(fmt.Sprintf("Log error: Memory.Read(%d, %d) out of range", offset, byteCount))
			return
		}
		logger(string(buffer))
	}).Export("log").
		Instantiate(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = network.Instantiate(ctx, wasmRuntime); err != nil {
		return nil, err
	}
	if _, err = wasi.Instantiate(ctx, wasmRuntime); err != nil {
		return nil, err
	}

	compiledModule, err := wasmRuntime.CompileModule(ctx, wasm)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = compiledModule.Close(context.Background())
		}
	}()
	if _, ok := compiledModule.ExportedMemories()["memory"]; !ok {
		return nil, fmt.Errorf("traffic encoder must export memory")
	}

	moduleConfig := wazero.NewModuleConfig().
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader)
	if _, ok := compiledModule.ExportedFunctions()["_initialize"]; ok {
		moduleConfig = moduleConfig.WithStartFunctions("_initialize")
	}
	mod, err = wasmRuntime.InstantiateModule(ctx, compiledModule, moduleConfig)
	if err != nil {
		return nil, err
	}

	encoder := mod.ExportedFunction("encode")
	decoder := mod.ExportedFunction("decode")
	malloc := mod.ExportedFunction("malloc")
	free := mod.ExportedFunction("free")
	for _, function := range []struct {
		name     string
		function api.Function
		params   []api.ValueType
		results  []api.ValueType
	}{
		{"encode", encoder, []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}},
		{"decode", decoder, []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, []api.ValueType{api.ValueTypeI64}},
		{"free", free, []api.ValueType{api.ValueTypeI32, api.ValueTypeI32}, nil},
	} {
		if err = validateTrafficEncoderFunction(function.name, function.function, function.params, function.results); err != nil {
			return nil, err
		}
	}
	if err = validateTrafficEncoderMalloc(malloc); err != nil {
		return nil, err
	}

	return &TrafficEncoder{
		ID:   CalculateWasmEncoderID(wasm),
		Data: wasm,

		ctx:     ctx,
		stop:    stop,
		runtime: wasmRuntime,
		mod:     mod,
		network: network,

		encoder: encoder,
		decoder: decoder,
		malloc:  malloc,
		free:    free,
	}, nil
}
