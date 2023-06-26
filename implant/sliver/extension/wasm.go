//go:build (windows && (amd64 || 386)) || (darwin && (arm64 || amd64)) || (linux && (amd64 || 386))

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

import (
	"context"
	"fmt"
	"io"
	"sync"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

type wasmPipe struct {
	Reader *io.PipeReader
	Writer *io.PipeWriter
}

// WasmExtension - Wasm extension
type WasmExtension struct {
	Name string
	ctx  context.Context
	lock sync.Mutex

	mod     wazero.CompiledModule
	config  wazero.ModuleConfig
	runtime wazero.Runtime
	closer  api.Closer

	Stdin  *wasmPipe
	Stdout *wasmPipe
	Stderr *wasmPipe
}

// IsExecuting - Check if the Wasm module runtime is currently executing
func (w *WasmExtension) IsExecuting() bool {
	return w.lock.TryLock()
}

// Execute - Execute the Wasm module with arguments, blocks during execution, returns errors
func (w *WasmExtension) Execute(args []string) (uint32, error) {
	w.lock.Lock()
	defer w.lock.Unlock()

	// {{if .Config.Debug}}
	log.Printf("[wasm ext] '%s' execute with args: %s", w.Name, args)
	// {{end}}

	args = append([]string{"wasi"}, args...)
	conf := w.config.WithArgs(args...)
	if _, err := w.runtime.InstantiateModule(w.ctx, w.mod, conf); err != nil {
		// Note: Most compilers do not exit the module after running "_start",
		// unless there was an error. This allows you to call exported functions.
		if exitErr, ok := err.(*sys.ExitError); ok && exitErr.ExitCode() != 0 {
			fmt.Fprintf(w.Stderr.Writer, "exit_code: %d\n", exitErr.ExitCode())
			// {{if .Config.Debug}}
			log.Printf("[wasm ext] '%s' exited with non-zero code: %d", w.Name, exitErr.ExitCode())
			// {{end}}
			return exitErr.ExitCode(), nil
		} else if !ok {
			// {{if .Config.Debug}}
			log.Printf("[wasm ext] '%s' exited with error: %s", w.Name, err.Error())
			// {{end}}
			return 0, err
		}
	}
	return 0, nil
}

// Close - Close the Wasm module
func (w *WasmExtension) Close() error {
	w.Stdin.Reader.Close()
	w.Stdout.Reader.Close()
	w.Stderr.Reader.Close()
	return w.closer.Close(w.ctx)
}

// NewWasmExtension - Create a new Wasm extension
func NewWasmExtension(name string, wasm []byte, memFS map[string][]byte) (*WasmExtension, error) {
	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	wasmExt := &WasmExtension{
		Name: name,
		ctx:  context.Background(),
		lock: sync.Mutex{},

		Stdin:  &wasmPipe{Reader: stdinReader, Writer: stdinWriter},
		Stdout: &wasmPipe{Reader: stdoutReader, Writer: stdoutWriter},
		Stderr: &wasmPipe{Reader: stderrReader, Writer: stderrWriter},
	}
	wasmExt.runtime = wazero.NewRuntime(wasmExt.ctx)
	wasmExt.config = wazero.NewModuleConfig().
		WithStdin(wasmExt.Stdin.Reader).
		WithStdout(wasmExt.Stdout.Writer).
		WithStderr(wasmExt.Stderr.Writer).
		WithFS(makeWasmMemFS(memFS))

	var err error
	wasmExt.closer, err = wasi.Instantiate(wasmExt.ctx, wasmExt.runtime)
	if err != nil {
		return nil, err
	}
	wasmExt.mod, err = wasmExt.runtime.CompileModule(wasmExt.ctx, wasm)
	if err != nil {
		return nil, err
	}
	return wasmExt, nil
}
