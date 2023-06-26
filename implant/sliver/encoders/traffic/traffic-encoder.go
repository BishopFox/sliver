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
	"fmt"
	"sync"

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
	runtime wazero.Runtime
	mod     api.Module
	lock    sync.Mutex

	// WASM functions
	encoder api.Function
	decoder api.Function
	malloc  api.Function
	free    api.Function
}

// Encode - Encode data using the wasm backend
func (t *TrafficEncoder) Encode(data []byte) ([]byte, error) {
	t.lock.Lock()
	defer t.lock.Unlock()

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

// Decode - Decode bytes using the wasm backend
func (t *TrafficEncoder) Decode(data []byte) ([]byte, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
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

// TrafficEncoderLogCallback - Callback function exposed to the wasm runtime to log messages
type TrafficEncoderLogCallback func(string)
