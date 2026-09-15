//go:build (linux && !android && (386 || amd64 || (arm && arm.7) || arm64 || ppc64le || riscv64)) || (freebsd && (amd64 || arm64))

package native

import "github.com/sliverarmory/reflektor/native/internal/linuxmem"

func loadPlatformLibrary(data []byte) (module, error) {
	return linuxmem.LoadLibrary(data)
}
