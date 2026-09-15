//go:build !(aix || darwin || dragonfly || freebsd || linux || netbsd || openbsd || solaris || windows)

package opfor

import "os"

func portableJavaFileRootPaths() []string { return []string{"/"} }

func portableJavaFileNameMax(string) int { return 255 }

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
	return err == nil && os.Chmod(path, info.Mode()&^0o222) == nil
}

func portableJavaFileCanExecute(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().Perm()&0o111 != 0
}

// The Go standard library exposes no disk-space API on these targets. File's
// specified failure value is zero, so keep the limitation explicit rather
// than deriving misleading values from a parent or process-global filesystem.
func portableJavaFileSpace(string, portableJavaFileSpaceKind) int64 { return 0 }
