//go:build (windows && (arm64 || amd64 || 386)) || (darwin && (arm64 || amd64)) || (linux && (arm64 || amd64 || 386))

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
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"runtime"
	"sync"

	"github.com/bishopfox/sliver/implant/sliver/wasmnet"
	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/tetratelabs/wazero"
	experimentalsysfs "github.com/tetratelabs/wazero/experimental/sysfs"
	wasi "github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	"github.com/tetratelabs/wazero/sys"
)

const wasmMemoryLimitPages = 4096 // 256 MiB

type wasmPipe struct {
	Reader *io.PipeReader
	Writer *io.PipeWriter
}

// WasmExtension - Wasm extension
type WasmExtension struct {
	Name string
	ctx  context.Context
	stop context.CancelFunc
	lock sync.Mutex

	mod     wazero.CompiledModule
	config  wazero.ModuleConfig
	runtime wazero.Runtime
	network *wasmnet.Host
	memFS   *WasmMemoryFS

	closeOnce sync.Once
	closeErr  error

	Stdin  *wasmPipe
	Stdout *wasmPipe
	Stderr *wasmPipe
}

// IsExecuting - Check if the Wasm module runtime is currently executing
func (w *WasmExtension) IsExecuting() bool {
	if !w.lock.TryLock() {
		return true
	}
	w.lock.Unlock()
	return false
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
	module, err := w.runtime.InstantiateModule(w.ctx, w.mod, conf)
	if module != nil {
		defer func() { _ = module.Close(w.ctx) }()
	}
	if err != nil {
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
	w.closeOnce.Do(func() {
		if w.stop != nil {
			w.stop()
		}
		_ = w.Stdin.Writer.Close()
		_ = w.Stdout.Writer.Close()
		_ = w.Stderr.Writer.Close()
		_ = w.Stdin.Reader.Close()
		_ = w.Stdout.Reader.Close()
		_ = w.Stderr.Reader.Close()
		if w.network != nil {
			w.closeErr = errors.Join(w.closeErr, w.network.Close())
		}
		if w.mod != nil {
			w.closeErr = errors.Join(w.closeErr, w.mod.Close(context.Background()))
		}
		if w.runtime != nil {
			w.closeErr = errors.Join(w.closeErr, w.runtime.Close(context.Background()))
		}
	})
	return w.closeErr
}

// NewWasmExtension - Create a new Wasm extension
func NewWasmExtension(name string, wasm []byte, memFS map[string][]byte) (*WasmExtension, error) {
	return NewWasmExtensionWithOptions(name, wasm, memFS)
}

// NewWasmExtensionWithOptions creates a Wasm extension with optional runtime
// features while preserving the historical NewWasmExtension API.
func NewWasmExtensionWithOptions(name string, wasm []byte, memFS map[string][]byte, options ...WasmExtensionOption) (*WasmExtension, error) {
	extensionConfig := applyWasmExtensionOptions(options)
	memoryFS, err := makeWasmMemFS(memFS, extensionConfig.memoryFSOptions...)
	if err != nil {
		return nil, err
	}
	fsConfig, ok := wazero.NewFSConfig().(experimentalsysfs.FSConfig)
	if !ok {
		return nil, errors.New("wazero writable filesystem mounting is unavailable")
	}

	stdinReader, stdinWriter := io.Pipe()
	stdoutReader, stdoutWriter := io.Pipe()
	stderrReader, stderrWriter := io.Pipe()
	ctx, stop := context.WithCancel(context.Background())
	wasmExt := &WasmExtension{
		Name:  name,
		ctx:   ctx,
		stop:  stop,
		lock:  sync.Mutex{},
		memFS: memoryFS,

		Stdin:  &wasmPipe{Reader: stdinReader, Writer: stdinWriter},
		Stdout: &wasmPipe{Reader: stdoutReader, Writer: stdoutWriter},
		Stderr: &wasmPipe{Reader: stderrReader, Writer: stderrWriter},
	}
	runtimeConfig := wazero.NewRuntimeConfig().
		WithCloseOnContextDone(true).
		WithMemoryLimitPages(wasmMemoryLimitPages)
	wasmExt.runtime = wazero.NewRuntimeWithConfig(wasmExt.ctx, runtimeConfig)
	wasmExt.config = wazero.NewModuleConfig().
		WithStdin(wasmExt.Stdin.Reader).
		WithStdout(wasmExt.Stdout.Writer).
		WithStderr(wasmExt.Stderr.Writer).
		WithSysWalltime().
		WithSysNanotime().
		WithSysNanosleep().
		WithOsyield(runtime.Gosched).
		WithRandSource(rand.Reader).
		WithFSConfig(fsConfig.WithSysFSMount(memoryFS, "/"))

	if _, err := wasi.Instantiate(wasmExt.ctx, wasmExt.runtime); err != nil {
		_ = wasmExt.Close()
		return nil, err
	}
	wasmExt.network = wasmnet.New(wasmExt.ctx)
	if _, err := wasmExt.network.Instantiate(wasmExt.ctx, wasmExt.runtime); err != nil {
		_ = wasmExt.Close()
		return nil, err
	}
	wasmExt.mod, err = wasmExt.runtime.CompileModule(wasmExt.ctx, wasm)
	if err != nil {
		_ = wasmExt.Close()
		return nil, err
	}
	return wasmExt, nil
}
