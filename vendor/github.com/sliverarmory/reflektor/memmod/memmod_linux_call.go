//go:build linux && !cgo && (386 || amd64 || arm64)

package memmod

import "errors"

//go:noescape
func cCall0(fn uintptr) uintptr

//go:noescape
func cCall1(fn, a0 uintptr) uintptr

//go:noescape
func cCall2(fn, a0, a1 uintptr) uintptr

//go:noescape
func cCall3(fn, a0, a1, a2 uintptr) uintptr

func cCallVoid0(fn uintptr) {
	_ = callExportFunction(fn)
}

func cCallVoid3(fn, a0, a1, a2 uintptr) {
	_ = callExportFunction(fn, a0, a1, a2)
}

func cCallVoid0OnThread(fn uintptr) error {
	_ = fn
	return errors.New("calling a Go c-shared export requires cgo")
}

func linuxGoTLSSlotOffset(slot uintptr) (int64, error) {
	_ = slot
	return 0, errors.New("loading a Go c-shared image requires cgo TLS support")
}
