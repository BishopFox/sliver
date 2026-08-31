//go:build darwin

package opfor

import "syscall"

func portableJavaFileNameMax(path string) int {
	// Darwin exposes _PC_NAME_MAX as selector 4. Pathconf is the same native
	// query used by OpenJDK's UnixFileSystem implementation.
	if maximum, err := syscall.Pathconf(path, 4); err == nil && maximum > 0 {
		return maximum
	}
	return 255
}

func portableJavaFileSpace(path string, kind portableJavaFileSpaceKind) int64 {
	var status syscall.Statfs_t
	if err := syscall.Statfs(path, &status); err != nil {
		return 0
	}
	blockSize := uint64(status.Bsize)
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
