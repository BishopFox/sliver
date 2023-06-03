//go:build !amd64 && !arm64

package compiler

import (
	"fmt"
	"runtime"

	"github.com/tetratelabs/wazero/internal/asm"
)

// archContext is empty on an unsupported architecture.
type archContext struct{}

// newCompiler panics with an unsupported error.
func newCompiler() compiler {
	panic(fmt.Sprintf("unsupported GOARCH %s", runtime.GOARCH))
}

func registerMaskShift(r asm.Register) (ret int) {
	panic(fmt.Sprintf("unsupported GOARCH %s", runtime.GOARCH))
}

func registerFromMaskShift(s int) asm.Register {
	panic(fmt.Sprintf("unsupported GOARCH %s", runtime.GOARCH))
}
