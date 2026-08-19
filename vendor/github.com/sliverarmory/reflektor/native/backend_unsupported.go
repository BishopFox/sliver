//go:build !(darwin && (amd64 || arm64)) && !(linux && (386 || amd64 || arm64)) && !(windows && (386 || amd64 || arm64))

package native

import (
	"errors"
	"runtime"
)

func loadPlatformLibrary(data []byte) (module, error) {
	_ = data
	return nil, errors.New("unsupported platform " + runtime.GOOS + "/" + runtime.GOARCH)
}
