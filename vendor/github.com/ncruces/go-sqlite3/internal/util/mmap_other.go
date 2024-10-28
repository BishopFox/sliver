//go:build !unix || !(386 || arm || amd64 || arm64 || riscv64 || ppc64le) || sqlite3_noshm || sqlite3_nosys

package util

type mmapState struct{}
