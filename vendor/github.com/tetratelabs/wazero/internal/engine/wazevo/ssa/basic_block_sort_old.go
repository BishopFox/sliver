//go:build !go1.21

// TODO: delete after the floor Go version is 1.21

package ssa

import "sort"

func sortBlocks(blocks []*basicBlock) {
	sort.SliceStable(blocks, func(i, j int) bool {
		iBlk, jBlk := blocks[i], blocks[j]
		if jBlk.ReturnBlock() {
			return true
		}
		if iBlk.ReturnBlock() {
			return false
		}
		iRoot, jRoot := iBlk.rootInstr, jBlk.rootInstr
		if iRoot == nil || jRoot == nil { // For testing.
			return true
		}
		return iBlk.rootInstr.id < jBlk.rootInstr.id
	})
}
