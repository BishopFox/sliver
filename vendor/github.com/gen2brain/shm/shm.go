// +build darwin dragonfly freebsd linux netbsd openbsd

// Package shm implements System V shared memory functions (shmctl, shmget, shmat, shmdt).
package shm

import (
	"syscall"
	"unsafe"
)

// Constants.
const (
	// Mode bits for `shmget`.

	// Create key if key does not exist.
	IPC_CREAT = 01000
	// Fail if key exists.
	IPC_EXCL = 02000
	// Return error on wait.
	IPC_NOWAIT = 04000

	// Special key values.

	// Private key.
	IPC_PRIVATE = 0

	// Flags for `shmat`.

	// Attach read-only access.
	SHM_RDONLY = 010000
	// Round attach address to SHMLBA.
	SHM_RND = 020000
	// Take-over region on attach.
	SHM_REMAP = 040000
	// Execution access.
	SHM_EXEC = 0100000

	// Commands for `shmctl`.

	// Lock segment (root only).
	SHM_LOCK = 1
	// Unlock segment (root only).
	SHM_UNLOCK = 12

	// Control commands for `shmctl`.

	// Remove identifier.
	IPC_RMID = 0
	// Set `ipc_perm` options.
	IPC_SET = 1
	// Get `ipc_perm' options.
	IPC_STAT = 2
)

// Get allocates a shared memory segment.
//
// Get() returns the identifier of the shared memory segment associated with the value of the argument key.
// A new shared memory segment is created if key has the value IPC_PRIVATE or key isn't IPC_PRIVATE,
// no shared memory segment corresponding to key exists, and IPC_CREAT is specified in shmFlg.
//
// If shmFlg specifies both IPC_CREAT and IPC_EXCL and a shared memory segment already exists for key,
// then Get() fails with errno set to EEXIST.
func Get(key int, size int, shmFlg int) (shmId int, err error) {
	id, _, errno := syscall.Syscall(sysShmGet, uintptr(int32(key)), uintptr(int32(size)), uintptr(int32(shmFlg)))
	if int(id) == -1 {
		return -1, errno
	}

	return int(id), nil
}

// At attaches the shared memory segment identified by shmId.
//
// Using At() with shmAddr equal to NULL is the preferred, portable way of attaching a shared memory segment.
func At(shmId int, shmAddr uintptr, shmFlg int) (data []byte, err error) {
	addr, _, errno := syscall.Syscall(sysShmAt, uintptr(int32(shmId)), shmAddr, uintptr(int32(shmFlg)))
	if int(addr) == -1 {
		return nil, errno
	}

	length, err := Size(shmId)
	if err != nil {
		syscall.Syscall(sysShmDt, addr, 0, 0)
		return nil, err
	}

	var b = struct {
		addr uintptr
		len  int
		cap  int
	}{addr, int(length), int(length)}

	data = *(*[]byte)(unsafe.Pointer(&b))
	return data, nil
}

// Dt detaches the shared memory segment.
//
// The to-be-detached segment must be currently attached with shmAddr equal to the value returned by the attaching At() call.
func Dt(data []byte) error {
	result, _, errno := syscall.Syscall(sysShmDt, uintptr(unsafe.Pointer(&data[0])), 0, 0)
	if int(result) == -1 {
		return errno
	}

	return nil
}

// Ctl performs the control operation specified by cmd on the shared memory segment whose identifier is given in shmId.
//
// The buf argument is a pointer to a IdDs structure.
func Ctl(shmId int, cmd int, buf *IdDs) (int, error) {
	result, _, errno := syscall.Syscall(sysShmCtl, uintptr(int32(shmId)), uintptr(int32(cmd)), uintptr(unsafe.Pointer(buf)))
	if int(result) == -1 {
		return -1, errno
	}

	return int(result), nil
}

// Rm removes the shared memory segment.
func Rm(shmId int) error {
	_, err := Ctl(shmId, IPC_RMID, nil)
	return err
}

// Size returns size of shared memory segment.
func Size(shmId int) (int64, error) {
	var idDs IdDs

	_, err := Ctl(shmId, IPC_STAT, &idDs)
	if err != nil {
		return 0, err
	}

	return int64(idDs.SegSz), nil
}
