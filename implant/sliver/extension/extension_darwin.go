package extension

import (
	"bytes"
	"unsafe"

	"github.com/Binject/universal"
)

type DarwinExtension struct {
	id          string
	data        []byte
	module      *universal.Library
	arch        string
	serverStore bool
	init        string
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
	extArgs := extensionArguments{}
	if len(arguments) > 0 {
		extArgs.inDataBuff = uintptr(unsafe.Pointer(&arguments[0]))
		extArgs.inDataSize = uintptr(uint64(len(arguments)))
	}
	_, err := d.module.Call(export, uintptr(unsafe.Pointer(&extArgs)))
	if err != nil {
		return err
	}
	outData := new(bytes.Buffer)
	for i := 0; i < int(extArgs.outDataSize); i++ {
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
