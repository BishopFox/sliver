// Note: The build constraints here are about the compiler, which is more
// narrow than the architectures supported by the assembler.
//
// Constraints here must match platform.CompilerSupported.
//
// Meanwhile, users who know their runtime.GOOS can operate with the compiler
// may choose to use NewRuntimeConfigCompiler explicitly.
//go:build (amd64 || arm64) && (linux || darwin || freebsd || netbsd || dragonfly || solaris || windows)

package wazero

import "github.com/tetratelabs/wazero/internal/platform"

func newRuntimeConfig() RuntimeConfig {
	if platform.CompilerSupported() {
		return NewRuntimeConfigCompiler()
	}
	return NewRuntimeConfigInterpreter()
}
