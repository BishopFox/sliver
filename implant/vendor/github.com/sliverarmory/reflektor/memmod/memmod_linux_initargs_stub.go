//go:build linux && !cgo && (386 || amd64 || arm64)

package memmod

func linuxInitCallArgs(includeEnvironment bool) (uintptr, uintptr, uintptr) {
	_ = includeEnvironment
	return 0, 0, 0
}
