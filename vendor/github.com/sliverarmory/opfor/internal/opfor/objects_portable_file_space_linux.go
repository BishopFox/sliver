//go:build linux

package opfor

import (
	"math"
	"syscall"
)

func portableJavaFileNameMax(path string) int {
	var status syscall.Statfs_t
	if err := syscall.Statfs(path, &status); err == nil {
		nameLength := int64(status.Namelen)
		if nameLength > 0 && nameLength <= int64(math.MaxInt32) {
			return int(nameLength)
		}
	}
	return 255
}

func portableJavaFileSpace(path string, kind portableJavaFileSpaceKind) int64 {
	var status syscall.Statfs_t
	if err := syscall.Statfs(path, &status); err != nil || status.Frsize < 0 {
		return 0
	}
	blockSize := uint64(status.Frsize)
	switch kind {
	case portableJavaFileSpaceTotal:
		return portableJavaFileSpaceBytes(status.Blocks, blockSize)
	case portableJavaFileSpaceFree:
		return portableJavaFileSpaceBytes(status.Bfree, blockSize)
	case portableJavaFileSpaceUsable:
		return portableJavaFileSpaceBytes(status.Bavail, blockSize)
	default:
		return 0
	}
}
