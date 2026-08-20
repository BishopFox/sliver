//go:build linux && (386 || amd64 || arm64)

package native

import "github.com/sliverarmory/reflektor/native/internal/linuxmem"

func loadPlatformLibrary(data []byte) (module, error) {
	return linuxmem.LoadLibrary(data)
}
