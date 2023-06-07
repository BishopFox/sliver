//go:build (amd64 || arm64) && windows

package sysfs

import (
	"io/fs"
	"os"
	"path"
	"syscall"

	"github.com/tetratelabs/wazero/internal/fsapi"
	"github.com/tetratelabs/wazero/internal/platform"
)

func lstat(path string) (fsapi.Stat_t, syscall.Errno) {
	attrs := uint32(syscall.FILE_FLAG_BACKUP_SEMANTICS)
	// Use FILE_FLAG_OPEN_REPARSE_POINT, otherwise CreateFile will follow symlink.
	// See https://docs.microsoft.com/en-us/windows/desktop/FileIO/symbolic-link-effects-on-file-systems-functions#createfile-and-createfiletransacted
	attrs |= syscall.FILE_FLAG_OPEN_REPARSE_POINT
	return statPath(attrs, path)
}

func stat(path string) (fsapi.Stat_t, syscall.Errno) {
	attrs := uint32(syscall.FILE_FLAG_BACKUP_SEMANTICS)
	return statPath(attrs, path)
}

func statPath(createFileAttrs uint32, path string) (fsapi.Stat_t, syscall.Errno) {
	if len(path) == 0 {
		return fsapi.Stat_t{}, syscall.ENOENT
	}
	pathp, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return fsapi.Stat_t{}, syscall.EINVAL
	}

	// open the file handle
	h, err := syscall.CreateFile(pathp, 0, 0, nil,
		syscall.OPEN_EXISTING, createFileAttrs, 0)
	if err != nil {
		// To match expectations of WASI, e.g. TinyGo TestStatBadDir, return
		// ENOENT, not ENOTDIR.
		if err == syscall.ENOTDIR {
			err = syscall.ENOENT
		}
		return fsapi.Stat_t{}, platform.UnwrapOSError(err)
	}
	defer syscall.CloseHandle(h)

	return statHandle(h)
}

func statFile(f *os.File) (fsapi.Stat_t, syscall.Errno) {
	// Attempt to get the stat by handle, which works for normal files
	st, err := statHandle(syscall.Handle(f.Fd()))

	// ERROR_INVALID_HANDLE happens before Go 1.20. Don't fail as we only
	// use that approach to fill in inode data, which is not critical.
	//
	// Note: statHandle uses UnwrapOSError which coerces
	// ERROR_INVALID_HANDLE to EBADF.
	if err != syscall.EBADF {
		return st, err
	}
	return defaultStatFile(f)
}

// inoFromFileInfo uses stat to get the inode information of the file.
func inoFromFileInfo(filePath string, t fs.FileInfo) (ino uint64, errno syscall.Errno) {
	if filePath == "" {
		// This is a fs.File backed implementation which doesn't have access to
		// the original file path.
		return
	}
	// ino is no not in Win32FileAttributeData
	inoPath := path.Clean(path.Join(filePath, t.Name()))
	var st fsapi.Stat_t
	if st, errno = lstat(inoPath); errno == 0 {
		ino = st.Ino
	}
	return
}

func statFromFileInfo(t fs.FileInfo) fsapi.Stat_t {
	if d, ok := t.Sys().(*syscall.Win32FileAttributeData); ok {
		st := fsapi.Stat_t{}
		st.Ino = 0 // not in Win32FileAttributeData
		st.Dev = 0 // not in Win32FileAttributeData
		st.Mode = t.Mode()
		st.Nlink = 1 // not in Win32FileAttributeData
		st.Size = t.Size()
		st.Atim = d.LastAccessTime.Nanoseconds()
		st.Mtim = d.LastWriteTime.Nanoseconds()
		st.Ctim = d.CreationTime.Nanoseconds()
		return st
	} else {
		return StatFromDefaultFileInfo(t)
	}
}

func statHandle(h syscall.Handle) (fsapi.Stat_t, syscall.Errno) {
	winFt, err := syscall.GetFileType(h)
	if err != nil {
		return fsapi.Stat_t{}, platform.UnwrapOSError(err)
	}

	var fi syscall.ByHandleFileInformation
	if err = syscall.GetFileInformationByHandle(h, &fi); err != nil {
		return fsapi.Stat_t{}, platform.UnwrapOSError(err)
	}

	var m fs.FileMode
	if fi.FileAttributes&syscall.FILE_ATTRIBUTE_READONLY != 0 {
		m |= 0o444
	} else {
		m |= 0o666
	}

	switch { // check whether this is a symlink first
	case fi.FileAttributes&syscall.FILE_ATTRIBUTE_REPARSE_POINT != 0:
		m |= fs.ModeSymlink
	case winFt == syscall.FILE_TYPE_PIPE:
		m |= fs.ModeNamedPipe
	case winFt == syscall.FILE_TYPE_CHAR:
		m |= fs.ModeDevice | fs.ModeCharDevice
	case fi.FileAttributes&syscall.FILE_ATTRIBUTE_DIRECTORY != 0:
		m |= fs.ModeDir | 0o111 // e.g. 0o444 -> 0o555
	}

	st := fsapi.Stat_t{}
	// FileIndex{High,Low} can be combined and used as a unique identifier like inode.
	// https://learn.microsoft.com/en-us/windows/win32/api/fileapi/ns-fileapi-by_handle_file_information
	st.Dev = uint64(fi.VolumeSerialNumber)
	st.Ino = (uint64(fi.FileIndexHigh) << 32) | uint64(fi.FileIndexLow)
	st.Mode = m
	st.Nlink = uint64(fi.NumberOfLinks)
	st.Size = int64(fi.FileSizeHigh)<<32 + int64(fi.FileSizeLow)
	st.Atim = fi.LastAccessTime.Nanoseconds()
	st.Mtim = fi.LastWriteTime.Nanoseconds()
	st.Ctim = fi.CreationTime.Nanoseconds()
	return st, 0
}
