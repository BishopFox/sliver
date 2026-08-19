//go:build (darwin && (amd64 || arm64)) || (windows && (386 || amd64 || arm64))

package native

import "github.com/sliverarmory/reflektor/memmod"

func loadPlatformLibrary(data []byte) (module, error) {
	return memmod.LoadLibrary(data)
}
