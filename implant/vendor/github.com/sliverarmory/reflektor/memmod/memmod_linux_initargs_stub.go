//go:build linux && !cgo && (386 || amd64 || arm64)

package memmod

func linuxInitCallArgs() (uintptr, uintptr, uintptr) {
	return 0, 0, 0
}
