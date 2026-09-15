//go:build windows

package opfor

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

var (
	portableJavaFileKernel32            = syscall.NewLazyDLL("kernel32.dll")
	portableJavaFileGetLogicalDrives    = portableJavaFileKernel32.NewProc("GetLogicalDrives")
	portableJavaFileGetVolumePathNameW  = portableJavaFileKernel32.NewProc("GetVolumePathNameW")
	portableJavaFileGetVolumeInfoW      = portableJavaFileKernel32.NewProc("GetVolumeInformationW")
	portableJavaFileGetDiskFreeSpaceExW = portableJavaFileKernel32.NewProc("GetDiskFreeSpaceExW")
)

func portableJavaFileRootPaths() []string {
	mask, _, _ := portableJavaFileGetLogicalDrives.Call()
	if mask == 0 {
		return nil
	}
	roots := make([]string, 0, 26)
	for index := uint(0); index < 26; index++ {
		if mask&(uintptr(1)<<index) != 0 {
			roots = append(roots, string([]byte{byte('A' + index), ':', '\\'}))
		}
	}
	return roots
}

// WinNTFileSystem reports the maximum component length for the target volume.
// Query that volume rather than assuming NTFS: removable and network
// filesystems may advertise a different value. The directory passed by
// createTempFile already exists, so GetVolumePathNameW can resolve it.
func portableJavaFileNameMax(path string) int {
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 255
	}
	volume := make([]uint16, 32768)
	result, _, _ := portableJavaFileGetVolumePathNameW.Call(
		uintptr(unsafe.Pointer(pathPointer)),
		uintptr(unsafe.Pointer(&volume[0])),
		uintptr(len(volume)),
	)
	if result == 0 {
		return 255
	}
	var maximumComponentLength uint32
	result, _, _ = portableJavaFileGetVolumeInfoW.Call(
		uintptr(unsafe.Pointer(&volume[0])),
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&maximumComponentLength)),
		0,
		0,
		0,
	)
	if result == 0 || maximumComponentLength == 0 {
		return 255
	}
	return int(maximumComponentLength)
}

func portableJavaFileSetPermission(path string, permission portableJavaFilePermission, enabled, _ bool) bool {
	// WinNTFileSystem treats read and execute permission as unsupported and
	// returns the requested enable value without querying the pathname.
	if permission == portableJavaFilePermissionRead || permission == portableJavaFilePermissionExecute {
		return enabled
	}
	if permission != portableJavaFilePermissionWrite {
		return false
	}
	target, err := filepath.EvalSymlinks(path)
	if err != nil {
		return false
	}
	info, err := os.Stat(target)
	if err != nil || info.IsDir() {
		// The JDK read-only attribute adapter deliberately rejects directories.
		return false
	}
	mode := info.Mode()
	if enabled {
		mode |= 0o200
	} else {
		mode &^= 0o222
	}
	return os.Chmod(target, mode) == nil
}

func portableJavaFileSetReadOnly(path string) bool {
	return portableJavaFileSetPermission(path, portableJavaFilePermissionWrite, false, false)
}

func portableJavaFileCanExecute(path string) bool {
	// WinNTFileSystem.checkAccess0 reports execute access whenever final target
	// attributes exist; it does not inspect an executable extension or ACL.
	_, err := os.Stat(path)
	return err == nil
}

func portableJavaFileSpace(path string, kind portableJavaFileSpaceKind) int64 {
	// WinNTFileSystem checks File.exists before querying its volume. Without
	// this guard GetVolumePathNameW can resolve a nonexistent leaf and report
	// the containing volume's space instead of File's specified zero.
	if _, err := os.Stat(path); err != nil {
		return 0
	}
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}
	// WinNTFileSystem uses MAX_PATH_LENGTH for the volume-path result. The JDK
	// constant is 32,767 UTF-16 code units, including the terminator here.
	volume := make([]uint16, 32768)
	result, _, _ := portableJavaFileGetVolumePathNameW.Call(
		uintptr(unsafe.Pointer(pathPointer)),
		uintptr(unsafe.Pointer(&volume[0])),
		uintptr(len(volume)),
	)
	if result == 0 {
		return 0
	}
	var usable, total, free uint64
	result, _, _ = portableJavaFileGetDiskFreeSpaceExW.Call(
		uintptr(unsafe.Pointer(&volume[0])),
		uintptr(unsafe.Pointer(&usable)),
		uintptr(unsafe.Pointer(&total)),
		uintptr(unsafe.Pointer(&free)),
	)
	if result == 0 {
		return 0
	}
	switch kind {
	case portableJavaFileSpaceTotal:
		return portableJavaFileSpaceBytes(total, 1)
	case portableJavaFileSpaceFree:
		return portableJavaFileSpaceBytes(free, 1)
	case portableJavaFileSpaceUsable:
		return portableJavaFileSpaceBytes(usable, 1)
	default:
		return 0
	}
}
