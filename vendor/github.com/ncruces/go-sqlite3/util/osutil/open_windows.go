package osutil

import (
	"io/fs"
	"os"
	. "syscall"
	"unsafe"
)

// OpenFile behaves the same as [os.OpenFile],
// except on Windows it sets [syscall.FILE_SHARE_DELETE].
//
// See: https://go.dev/issue/32088#issuecomment-502850674
func OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	if name == "" {
		return nil, &os.PathError{Op: "open", Path: name, Err: ENOENT}
	}
	r, e := syscallOpen(name, flag|O_CLOEXEC, uint32(perm.Perm()))
	if e != nil {
		return nil, &os.PathError{Op: "open", Path: name, Err: e}
	}
	return os.NewFile(uintptr(r), name), nil
}

// syscallOpen is a copy of [syscall.Open]
// that uses [syscall.FILE_SHARE_DELETE].
//
// https://go.dev/src/syscall/syscall_windows.go
func syscallOpen(path string, mode int, perm uint32) (fd Handle, err error) {
	if len(path) == 0 {
		return InvalidHandle, ERROR_FILE_NOT_FOUND
	}
	pathp, err := UTF16PtrFromString(path)
	if err != nil {
		return InvalidHandle, err
	}
	var access uint32
	switch mode & (O_RDONLY | O_WRONLY | O_RDWR) {
	case O_RDONLY:
		access = GENERIC_READ
	case O_WRONLY:
		access = GENERIC_WRITE
	case O_RDWR:
		access = GENERIC_READ | GENERIC_WRITE
	}
	if mode&O_CREAT != 0 {
		access |= GENERIC_WRITE
	}
	if mode&O_APPEND != 0 {
		access &^= GENERIC_WRITE
		access |= FILE_APPEND_DATA
	}
	sharemode := uint32(FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE)
	var sa *SecurityAttributes
	if mode&O_CLOEXEC == 0 {
		sa = makeInheritSa()
	}
	var createmode uint32
	switch {
	case mode&(O_CREAT|O_EXCL) == (O_CREAT | O_EXCL):
		createmode = CREATE_NEW
	case mode&(O_CREAT|O_TRUNC) == (O_CREAT | O_TRUNC):
		createmode = CREATE_ALWAYS
	case mode&O_CREAT == O_CREAT:
		createmode = OPEN_ALWAYS
	case mode&O_TRUNC == O_TRUNC:
		createmode = TRUNCATE_EXISTING
	default:
		createmode = OPEN_EXISTING
	}
	var attrs uint32 = FILE_ATTRIBUTE_NORMAL
	if perm&S_IWRITE == 0 {
		attrs = FILE_ATTRIBUTE_READONLY
		if createmode == CREATE_ALWAYS {
			const _ERROR_BAD_NETPATH = Errno(53)
			// We have been asked to create a read-only file.
			// If the file already exists, the semantics of
			// the Unix open system call is to preserve the
			// existing permissions. If we pass CREATE_ALWAYS
			// and FILE_ATTRIBUTE_READONLY to CreateFile,
			// and the file already exists, CreateFile will
			// change the file permissions.
			// Avoid that to preserve the Unix semantics.
			h, e := CreateFile(pathp, access, sharemode, sa, TRUNCATE_EXISTING, FILE_ATTRIBUTE_NORMAL, 0)
			switch e {
			case ERROR_FILE_NOT_FOUND, _ERROR_BAD_NETPATH, ERROR_PATH_NOT_FOUND:
				// File does not exist. These are the same
				// errors as Errno.Is checks for ErrNotExist.
				// Carry on to create the file.
			default:
				// Success or some different error.
				return h, e
			}
		}
	}
	if createmode == OPEN_EXISTING && access == GENERIC_READ {
		// Necessary for opening directory handles.
		attrs |= FILE_FLAG_BACKUP_SEMANTICS
	}
	if mode&O_SYNC != 0 {
		const _FILE_FLAG_WRITE_THROUGH = 0x80000000
		attrs |= _FILE_FLAG_WRITE_THROUGH
	}
	return CreateFile(pathp, access, sharemode, sa, createmode, attrs, 0)
}

func makeInheritSa() *SecurityAttributes {
	var sa SecurityAttributes
	sa.Length = uint32(unsafe.Sizeof(sa))
	sa.InheritHandle = 1
	return &sa
}
