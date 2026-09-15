//go:build aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris

package opfor

import (
	"os"
	"syscall"
)

func portableJavaFileRootPaths() []string { return []string{"/"} }

func portableJavaFileSetPermission(path string, permission portableJavaFilePermission, enabled, ownerOnly bool) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	var owner, everyone os.FileMode
	switch permission {
	case portableJavaFilePermissionRead:
		owner, everyone = 0o400, 0o444
	case portableJavaFilePermissionWrite:
		owner, everyone = 0o200, 0o222
	case portableJavaFilePermissionExecute:
		owner, everyone = 0o100, 0o111
	default:
		return false
	}
	mask := everyone
	if ownerOnly {
		mask = owner
	}
	mode := info.Mode()
	if enabled {
		mode |= mask
	} else {
		mode &^= mask
	}
	return os.Chmod(path, mode) == nil
}

func portableJavaFileSetReadOnly(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return os.Chmod(path, info.Mode()&^0o222) == nil
}

func portableJavaFileCanExecute(path string) bool {
	// UnixFileSystem.checkAccess0 delegates to access(path, X_OK), including
	// effective credential and directory-search semantics.
	return syscall.Access(path, 1) == nil
}
