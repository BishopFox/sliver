//go:build linux && !cgo && (386 || amd64 || arm64)

package memmod

//go:noescape
func cCall0(fn uintptr) uintptr

//go:noescape
func cCall1(fn, a0 uintptr) uintptr

//go:noescape
func cCall2(fn, a0, a1 uintptr) uintptr

//go:noescape
func cCall3(fn, a0, a1, a2 uintptr) uintptr
