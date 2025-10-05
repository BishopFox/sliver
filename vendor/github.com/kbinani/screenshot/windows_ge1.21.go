//go:build windows && go1.21

package screenshot

import (
	"image"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/lxn/win"
)

func NumActiveDisplays() int {
	count := new(int)
	pinner := new(runtime.Pinner)
	pinner.Pin(count)
	defer pinner.Unpin()
	*count = 0
	ptr := unsafe.Pointer(count)
	enumDisplayMonitors(win.HDC(0), nil, syscall.NewCallback(countupMonitorCallback), uintptr(ptr))
	return *count
}

func GetDisplayBounds(displayIndex int) image.Rectangle {
	ctx := new(getMonitorBoundsContext)
	pinner := new(runtime.Pinner)
	pinner.Pin(ctx)
	defer pinner.Unpin()
	ctx.Index = displayIndex
	ctx.Count = 0
	ptr := unsafe.Pointer(ctx)
	enumDisplayMonitors(win.HDC(0), nil, syscall.NewCallback(getMonitorBoundsCallback), uintptr(ptr))
	return image.Rect(
		int(ctx.Rect.Left), int(ctx.Rect.Top),
		int(ctx.Rect.Right), int(ctx.Rect.Bottom))
}
