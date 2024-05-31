//go:build !(unix || windows) || sqlite3_nosys

package util

import "github.com/tetratelabs/wazero/experimental"

func virtualAlloc(cap, max uint64) experimental.LinearMemory {
	return sliceAlloc(cap, max)
}
