//go:build windows && (386 || amd64 || arm64)

package bofloader

import (
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

const (
	windowsCodePageACP        = 0
	windowsMBErrInvalidChars  = 0x00000008
	windowsNativeWideCharSize = 2
)

//go:nocheckptr
func toWideChar(sourceAddress, destinationAddress, maximumValue uintptr) (result uintptr) {
	defer callbackPanic("toWideChar")
	maximum := int64(int32(maximumValue))
	if sourceAddress == 0 || destinationAddress == 0 || maximum < windowsNativeWideCharSize || maximum > maxCallbackData {
		return 0
	}
	text, err := readCString(sourceAddress, maxFormatString)
	if err != nil {
		callbackError("toWideChar", "%v", err)
		return 0
	}
	source := append([]byte(text), 0)
	written, err := windows.MultiByteToWideChar(
		windowsCodePageACP,
		windowsMBErrInvalidChars,
		&source[0],
		-1,
		(*uint16)(unsafe.Pointer(destinationAddress)),
		int32(maximum/windowsNativeWideCharSize),
	)
	runtime.KeepAlive(source)
	if err != nil || written <= 0 {
		return 0
	}
	return 1
}
