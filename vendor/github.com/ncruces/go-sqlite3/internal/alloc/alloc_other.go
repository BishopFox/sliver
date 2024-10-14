//go:build !(unix || windows) || sqlite3_nosys

package alloc

import "github.com/tetratelabs/wazero/experimental"

func Virtual(cap, max uint64) experimental.LinearMemory {
	return Slice(cap, max)
}
