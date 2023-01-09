package encoders

/*
	Sliver Implant Framework
	Copyright (C) 2023  Bishop Fox

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"fmt"
	"runtime"

	"github.com/bishopfox/sliver/util"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// TrafficEncoder - Implements the `Encoder` interface using a wasm backend
type TrafficEncoder struct {
	ctx     context.Context
	runtime wazero.Runtime
	mod     api.Module

	// WASM functions
	encoder api.Function
	decoder api.Function
	malloc  api.Function
	free    api.Function
}

func (t *TrafficEncoder) Encode(data []byte) ([]byte, error) {
	// Allocate a buffer in the wasm runtime for the input data
	size := uint64(len(data))
	buf, err := t.malloc.Call(t.ctx, size)
	if err != nil {
		return nil, err
	}
	bufPtr := buf[0]
	defer t.free.Call(t.ctx, bufPtr)

	// Copy input data into wasm memory
	if !t.mod.Memory().Write(uint32(bufPtr), data) {
		return nil, fmt.Errorf("Memory.Write(%d, %d) out of range of memory size %d",
			bufPtr, size, t.mod.Memory().Size())
	}

	// Call the encoder function
	ptrSize, err := t.encoder.Call(t.ctx, bufPtr, size)
	if err != nil {
		return nil, err
	}

	// Read the output buffer from wasm memory
	encodeResultPtr := uint32(ptrSize[0] >> 32)
	encodeResultSize := uint32(ptrSize[0])
	var encodeResult []byte
	var ok bool
	if encodeResult, ok = t.mod.Memory().Read(encodeResultPtr, encodeResultSize); !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d",
			encodeResultPtr, encodeResultSize, t.mod.Memory().Size())
	}
	return encodeResult, nil
}

func (t *TrafficEncoder) Decode(data []byte) ([]byte, error) {
	size := uint64(len(data))
	buf, err := t.malloc.Call(t.ctx, size)
	if err != nil {
		return nil, err
	}
	bufPtr := buf[0]
	defer t.free.Call(t.ctx, bufPtr)

	if !t.mod.Memory().Write(uint32(bufPtr), data) {
		return nil, fmt.Errorf("Memory.Write(%d, %d) out of range of memory size %d",
			bufPtr, size, t.mod.Memory().Size())
	}

	// Call the decoder function
	ptrSize, err := t.decoder.Call(t.ctx, bufPtr, size)
	if err != nil {
		return nil, err
	}
	decodeResultPtr := uint32(ptrSize[0] >> 32)
	decodeResultSize := uint32(ptrSize[0])
	var decodeResult []byte
	var ok bool
	if decodeResult, ok = t.mod.Memory().Read(decodeResultPtr, decodeResultSize); !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d",
			decodeResultPtr, decodeResultSize, t.mod.Memory().Size())
	}

	return decodeResult, nil
}

func (t *TrafficEncoder) Close() error {
	return t.runtime.Close(t.ctx)
}

type TrafficEncoderLogCallback func(string)

func CreateTrafficEncoder(name string, wasm []byte, logString TrafficEncoderLogCallback) (*TrafficEncoder, error) {
	ctx := context.Background()
	var wasmRuntime wazero.Runtime
	if util.Contains([]string{"amd64", "arm64"}, runtime.GOARCH) && util.Contains([]string{"darwin", "linux", "windows"}, runtime.GOOS) {
		wasmRuntime = wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler())
	} else {
		wasmRuntime = wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter())
	}

	_, err := wasmRuntime.NewHostModuleBuilder(name).
		NewFunctionBuilder().WithFunc(func(_ context.Context, m api.Module, offset, byteCount uint32) {
		buf, ok := m.Memory().Read(offset, byteCount)
		if !ok {
			logString(fmt.Sprintf("Log error: Memory.Read(%d, %d) out of range", offset, byteCount))
		}
		logString(string(buf))
	}).Export("log").
		Instantiate(ctx, wasmRuntime)
	if err != nil {
		return nil, err
	}
	_, err = wasi.Instantiate(ctx, wasmRuntime)
	if err != nil {
		return nil, err
	}
	mod, err := wasmRuntime.InstantiateModuleFromBinary(ctx, wasm)
	if err != nil {
		return nil, err
	}

	return &TrafficEncoder{
		ctx:     ctx,
		runtime: wasmRuntime,
		mod:     mod,

		encoder: mod.ExportedFunction("encode"),
		decoder: mod.ExportedFunction("decode"),

		// These are undocumented, but exported. See tinygo-org/tinygo#2788
		malloc: mod.ExportedFunction("malloc"),
		free:   mod.ExportedFunction("free"),
	}, nil
}
