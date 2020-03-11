// +build solaris

package shm

// #include <sys/ipc.h>
// #include <sys/shm.h>
// #include <errno.h>
import "C"

import (
	"unsafe"
)

// Perm is used to pass permission information to IPC operations.
type Perm struct {
	// Owner's user ID.
	Uid uint32
	// Owner's group ID.
	Gid uint32
	// Creator's user ID.
	Cuid uint32
	// Creator's group ID.
	Cgid uint32
	// Read/write permission.
	Mode uint32
	// Sequence number.
	Seq uint32
	// Key.
	Key int32
}

// IdDs describes shared memory segment.
type IdDs struct {
	// Operation permission struct.
	Perm Perm
	// Padding.
	PadCgo0 [4]byte
	// Size of segment in bytes.
	SegSz uint64
	// Flags.
	Flags uint64
	// Internal.
	Lkcnt uint16
	// Padding.
	PadCgo1 [2]byte
	// Pid of last shmat/shmdt.
	Lpid int32
	// Pid of creator.
	Cpid int32
	// Padding.
	PadCgo2 [4]byte
	// Number of current attaches.
	Nattch uint64
	// Internal.
	Cnattch uint64
	// Last attach time.
	Atime int64
	// Last detach time.
	Dtime int64
	// Last change time.
	Ctime int64
	// Internal.
	Amp *byte
	// Internal.
	Gransize uint64
	// Internal.
	Allocated uint64
	// Padding.
	Pad4 [1]int64
}

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
	IPC_RMID = 10
	// Set `ipc_perm` options.
	IPC_SET = 11
	// Get `ipc_perm' options.
	IPC_STAT = 12
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
	id, errno := C.shmget(C.key_t(key), C.size_t(size), C.int(shmFlg))

	if int(id) == -1 {
		return -1, errno
	}

	return int(id), nil
}

// At attaches the shared memory segment identified by shmId.
//
// Using At() with shmAddr equal to NULL is the preferred, portable way of attaching a shared memory segment.
func At(shmId int, shmAddr uintptr, shmFlg int) (data []byte, err error) {
	addr, errno := C.shmat(C.int(shmId), unsafe.Pointer(shmAddr), C.int(shmFlg))
	if int(uintptr(addr)) == -1 {
		return nil, errno
	}

	length, err := Size(shmId)
	if err != nil {
		return nil, err
	}

	var b = struct {
		addr uintptr
		len  int
		cap  int
	}{uintptr(addr), int(length), int(length)}

	data = *(*[]byte)(unsafe.Pointer(&b))
	return data, nil
}

// Dt detaches the shared memory segment.
//
// The to-be-detached segment must be currently attached with shmAddr equal to the value returned by the attaching At() call.
func Dt(data []byte) error {
	result, errno := C.shmdt(unsafe.Pointer(&data[0]))
	if int(result) == -1 {
		return errno
	}

	return nil
}

// Ctl performs the control operation specified by cmd on the shared memory segment whose identifier is given in shmId.
//
// The buf argument is a pointer to a IdDs structure.
func Ctl(shmId int, cmd int, buf *IdDs) (int, error) {
	result, errno := C.shmctl(C.int(shmId), C.int(cmd), (*C.struct_shmid_ds)(unsafe.Pointer(buf)))
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
