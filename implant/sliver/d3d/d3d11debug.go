package d3d

import (
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

type ID3D11Debug struct {
	vtbl *iD3D11DebugVtbl
}

func (obj *ID3D11Debug) QueryInterface(iid windows.GUID, pp interface{}) int32 {
	return reflectQueryInterface(obj, obj.vtbl.QueryInterface, &iid, pp)
}
func (obj *ID3D11Debug) ReportLiveDeviceObjects(flags uint32) int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.ReportLiveDeviceObjects,
		2,
		uintptr(unsafe.Pointer(obj)),
		uintptr(flags),
		0,
	)
	return int32(ret)
}
func (obj *ID3D11Debug) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}

type ID3D11InfoQueue struct {
	vtbl *iD3D11InfoQueueVtbl
}

func (obj *ID3D11InfoQueue) Release() int32 {
	ret, _, _ := syscall.Syscall(
		obj.vtbl.Release,
		1,
		uintptr(unsafe.Pointer(obj)),
		0,
		0,
	)
	return int32(ret)
}
