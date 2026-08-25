//go:build !(darwin && !ios && (amd64 || arm64)) && !(linux && !android && (386 || amd64 || (arm && arm.7) || arm64 || ppc64le || riscv64)) && !(freebsd && (amd64 || arm64)) && !(windows && (386 || amd64 || arm64))

package native

import (
	"errors"
	"runtime"
)

func loadPlatformLibrary(data []byte) (module, error) {
	_ = data
	return nil, errors.New("unsupported platform " + runtime.GOOS + "/" + runtime.GOARCH)
}
