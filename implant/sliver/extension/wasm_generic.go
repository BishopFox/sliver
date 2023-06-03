//go:build !((windows && (amd64 || 386)) || (darwin && (arm64 || amd64)) || (linux && (amd64 || 386)))

package extension

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

// While in theory Wazero is pure go and can compile to any platform, in practice
// this isn't the case just yet, so adding build constraints to disable Wasm on
// unsupported platforms.

import (
	"errors"
	"io"
)

type wasmPipe struct {
	Reader *io.PipeReader
	Writer *io.PipeWriter
}

type WasmExtension struct {
	Name string

	Stdin  *wasmPipe
	Stdout *wasmPipe
	Stderr *wasmPipe
}

func (w *WasmExtension) IsExecuting() bool {
	return false
}

func (w *WasmExtension) Execute(args []string) (uint32, error) {
	return 0, errors.New("Wasm extensions are not yet supported on this platform")
}

func (w *WasmExtension) Close() error {
	return nil
}

// NewWasmExtension - For platforms that don't support Wasm extensions
func NewWasmExtension(name string, wasm []byte, memFS map[string][]byte) (*WasmExtension, error) {
	return nil, errors.New("Wasm extensions are not supported on this platform")
}
