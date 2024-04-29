//go:build (amd64 || arm64 || riscv64) && !sqlite3_nosys

package vfs

import (
	"os"

	"golang.org/x/sys/unix"
)

const (
	_F2FS_IOC_START_ATOMIC_WRITE  = 62721
	_F2FS_IOC_COMMIT_ATOMIC_WRITE = 62722
	_F2FS_IOC_ABORT_ATOMIC_WRITE  = 62725
	_F2FS_IOC_GET_FEATURES        = 2147808524
	_F2FS_FEATURE_ATOMIC_WRITE    = 4
)

func osBatchAtomic(file *os.File) bool {
	flags, err := unix.IoctlGetInt(int(file.Fd()), _F2FS_IOC_GET_FEATURES)
	return err == nil && flags&_F2FS_FEATURE_ATOMIC_WRITE != 0
}

func (f *vfsFile) BeginAtomicWrite() error {
	return unix.IoctlSetInt(int(f.Fd()), _F2FS_IOC_START_ATOMIC_WRITE, 0)
}

func (f *vfsFile) CommitAtomicWrite() error {
	return unix.IoctlSetInt(int(f.Fd()), _F2FS_IOC_COMMIT_ATOMIC_WRITE, 0)
}

func (f *vfsFile) RollbackAtomicWrite() error {
	return unix.IoctlSetInt(int(f.Fd()), _F2FS_IOC_ABORT_ATOMIC_WRITE, 0)
}
