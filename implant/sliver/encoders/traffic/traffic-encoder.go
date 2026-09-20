package traffic

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
	"crypto/sha256"
	"errors"
	"fmt"
	"sync"

	"github.com/bishopfox/sliver/implant/sliver/wasmnet"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

// CalculateWasmEncoderID - Creates an Encoder ID based on the hash of the wasm bin
func CalculateWasmEncoderID(wasmEncoderData []byte) uint64 {
	digest := sha256.Sum256(wasmEncoderData)
	// The encoder id must be less than 65537 (the encoder modulo)
	return uint64(uint16(digest[0])<<8 + uint16(digest[1]))
}

// TrafficEncoder - Implements the `Encoder` interface using a wasm backend
type TrafficEncoder struct {
	ctx     context.Context
	stop    context.CancelFunc
	runtime wazero.Runtime
	mod     api.Module
	network *wasmnet.Host
	lock    sync.Mutex

	closeOnce sync.Once
	closeErr  error

	// WASM functions
	encoder api.Function
	decoder api.Function
	malloc  api.Function
	free    api.Function
}

func validateTrafficEncoderFunction(name string, function api.Function, params, results []api.ValueType) error {
	if function == nil {
		return fmt.Errorf("traffic encoder must export %s", name)
	}
	definition := function.Definition()
	if !sameValueTypes(definition.ParamTypes(), params) || !sameValueTypes(definition.ResultTypes(), results) {
		return fmt.Errorf("traffic encoder export %s has an incompatible function signature", name)
	}
	return nil
}

func validateTrafficEncoderMalloc(function api.Function) error {
	if function == nil {
		return fmt.Errorf("traffic encoder must export malloc")
	}
	definition := function.Definition()
	params := definition.ParamTypes()
	results := definition.ResultTypes()
	if len(params) != 1 || len(results) != 1 ||
		(params[0] != api.ValueTypeI32 && params[0] != api.ValueTypeI64) ||
		(results[0] != api.ValueTypeI32 && results[0] != api.ValueTypeI64) {
		return fmt.Errorf("traffic encoder export malloc has an incompatible function signature")
	}
	return nil
}

func sameValueTypes(left, right []api.ValueType) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// Encode - Encode data using the wasm backend
func (t *TrafficEncoder) Encode(data []byte) ([]byte, error) {
	return t.transform(t.encoder, data)
}

// Decode - Decode bytes using the wasm backend
func (t *TrafficEncoder) Decode(data []byte) ([]byte, error) {
	return t.transform(t.decoder, data)
}

func (t *TrafficEncoder) transform(transformer api.Function, data []byte) (result []byte, err error) {
	t.lock.Lock()
	defer t.lock.Unlock()

	// Allocate a buffer in the wasm runtime for the input data
	size := uint64(len(data))
	if size > uint64(^uint32(0)) {
		return nil, fmt.Errorf("traffic encoder input is too large: %d bytes", size)
	}
	allocationSize := size
	if allocationSize == 0 {
		allocationSize = 1
	}
	buf, err := t.malloc.Call(t.ctx, allocationSize)
	if err != nil {
		return nil, err
	}
	bufPtr := buf[0]
	if bufPtr > uint64(^uint32(0)) {
		return nil, fmt.Errorf("traffic encoder malloc returned an invalid pointer: %d", bufPtr)
	}
	defer func() {
		if _, freeErr := t.free.Call(t.ctx, bufPtr, allocationSize); freeErr != nil {
			err = errors.Join(err, fmt.Errorf("free traffic encoder input: %w", freeErr))
		}
	}()

	// Copy input data into wasm memory
	if !t.mod.Memory().Write(uint32(bufPtr), data) {
		return nil, fmt.Errorf("Memory.Write(%d, %d) out of range of memory size %d",
			bufPtr, size, t.mod.Memory().Size())
	}

	ptrSize, err := transformer.Call(t.ctx, bufPtr, size)
	if err != nil {
		return nil, err
	}

	// Read the output buffer from wasm memory
	encodeResultPtr := uint32(ptrSize[0] >> 32)
	encodeResultSize := uint32(ptrSize[0])
	if encodeResultSize > 0 && uint64(encodeResultPtr) != bufPtr {
		defer func() {
			if _, freeErr := t.free.Call(t.ctx, uint64(encodeResultPtr), uint64(encodeResultSize)); freeErr != nil {
				err = errors.Join(err, fmt.Errorf("free traffic encoder output: %w", freeErr))
			}
		}()
	}
	encodeResult, ok := t.mod.Memory().Read(encodeResultPtr, encodeResultSize)
	if !ok {
		return nil, fmt.Errorf("Memory.Read(%d, %d) out of range of memory size %d",
			encodeResultPtr, encodeResultSize, t.mod.Memory().Size())
	}
	return append([]byte(nil), encodeResult...), nil
}

func (t *TrafficEncoder) Close() error {
	t.closeOnce.Do(func() {
		if t.stop != nil {
			t.stop()
		}
		t.lock.Lock()
		defer t.lock.Unlock()
		if t.network != nil {
			t.closeErr = errors.Join(t.closeErr, t.network.Close())
		}
		if t.mod != nil {
			t.closeErr = errors.Join(t.closeErr, t.mod.Close(context.Background()))
		}
		if t.runtime != nil {
			t.closeErr = errors.Join(t.closeErr, t.runtime.Close(context.Background()))
		}
	})
	return t.closeErr
}

// TrafficEncoderLogCallback - Callback function exposed to the wasm runtime to log messages
type TrafficEncoderLogCallback func(string)
