package wininet

import (
	"golang.org/x/sys/windows"
)

var (
	libuser32 = windows.NewLazySystemDLL("User32")
)

func GetDesktopWindow() uintptr {

	ret, _, _ := libuser32.NewProc("GetDesktopWindow").Call()

	return ret
}
