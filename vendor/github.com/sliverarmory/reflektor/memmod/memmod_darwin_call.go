//go:build darwin && !ios && (amd64 || arm64) && !cgo

package memmod

import (
	"errors"
	_ "unsafe"

	"github.com/ebitengine/purego"
)

//go:noescape
func cCall10(fn, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) uintptr

//go:linkname runtimeSystemstack runtime.systemstack
func runtimeSystemstack(fn func())

func call0(fn uintptr) uintptr {
	return call10(fn, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
}

func callVoid0(fn uintptr) {
	_ = call0(fn)
}

func callVoid0OnThread(fn uintptr) error {
	_ = fn
	return errors.New("calling a Go c-shared export requires cgo")
}

func callExport(fn uintptr, args ...uintptr) uintptr {
	result, _, _ := purego.SyscallN(fn, args...)
	return result
}

func callDlopen(fn, name uintptr, flags int) uintptr {
	return call2(fn, name, uintptr(flags))
}

func callDlerror(fn uintptr) uintptr {
	return call0(fn)
}

func call1(fn, a0 uintptr) uintptr {
	return call10(fn, a0, 0, 0, 0, 0, 0, 0, 0, 0, 0)
}

func call2(fn, a0, a1 uintptr) uintptr {
	return call10(fn, a0, a1, 0, 0, 0, 0, 0, 0, 0, 0)
}

func call4(fn, a0, a1, a2, a3 uintptr) uintptr {
	return call10(fn, a0, a1, a2, a3, 0, 0, 0, 0, 0, 0)
}

func call6(fn, a0, a1, a2, a3, a4, a5 uintptr) uintptr {
	return call10(fn, a0, a1, a2, a3, a4, a5, 0, 0, 0, 0)
}

func call10(fn, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9 uintptr) uintptr {
	var ret uintptr
	runtimeSystemstack(func() {
		ret = cCall10(fn, a0, a1, a2, a3, a4, a5, a6, a7, a8, a9)
	})
	return ret
}
