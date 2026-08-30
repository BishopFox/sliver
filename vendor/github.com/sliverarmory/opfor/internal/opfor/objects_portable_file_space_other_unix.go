//go:build aix || dragonfly || freebsd || netbsd || openbsd || solaris

package opfor

// These targets do not expose one portable pathconf/statfs field layout in the
// Go standard library. NAME_MAX is 255 on the supported mainstream filesystems;
// retain that conservative OpenJDK-compatible fallback without introducing
// CGO or a target-specific external dependency.
func portableJavaFileNameMax(string) int { return 255 }

// syscall.Statfs_t does not expose one field layout across these targets.
// Returning File's specified failure value keeps the limitation honest; an
// importer ObjectHost can provide a target-specific implementation first.
func portableJavaFileSpace(string, portableJavaFileSpaceKind) int64 { return 0 }
