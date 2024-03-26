//go:build !go1.21

// TODO: delete after the floor Go version is 1.21

package frontend

import (
	"sort"

	"github.com/tetratelabs/wazero/internal/engine/wazevo/ssa"
)

func sortSSAValueIDs(IDs []ssa.ValueID) {
	sort.SliceStable(IDs, func(i, j int) bool {
		return int(IDs[i]) < int(IDs[j])
	})
}
