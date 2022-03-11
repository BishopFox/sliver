package d3d

import (
	"reflect"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

func reflectQueryInterface(self interface{}, method uintptr, interfaceID *windows.GUID, obj interface{}) int32 {
	selfValue := reflect.ValueOf(self).Elem()
	objValue := reflect.ValueOf(obj).Elem()

	hr, _, _ := syscall.Syscall(
		method,
		3,
		selfValue.UnsafeAddr(),
		uintptr(unsafe.Pointer(interfaceID)),
		objValue.Addr().Pointer())

	return int32(hr)
}
