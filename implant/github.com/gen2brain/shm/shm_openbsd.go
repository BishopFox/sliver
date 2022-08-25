package shm

import (
	"syscall"
)

// System call constants.
const (
	sysShmAt  = syscall.SYS_SHMAT
	sysShmCtl = syscall.SYS_SHMCTL
	sysShmDt  = syscall.SYS_SHMDT
	sysShmGet = syscall.SYS_SHMGET
)

// Perm is used to pass permission information to IPC operations.
type Perm struct {
	// Owner's user ID.
	Cuid uint32
	// Owner's group ID.
	Cgid uint32
	// Creator's user ID.
	Uid uint32
	// Creator's group ID.
	Gid uint32
	// Read/write permission.
	Mode uint32
	// Sequence number.
	Seq uint16
	// Padding.
	PadCgo0 [2]byte
	// Key.
	Key int64
}

// IdDs describes shared memory segment.
type IdDs struct {
	// Operation permission struct.
	Perm Perm
	// Size of segment in bytes.
	SegSz int32
	// Pid of last shmat/shmdt.
	Lpid int32
	// Pid of creator.
	Cpid int32
	// Number of current attaches.
	Nattch int16
	// Padding.
	PadCgo0 [2]byte
	// Last attach time.
	Atime int64
	// Internal.
	XShmAtimensec int64
	// Last detach time.
	Dtime int64
	// Internal.
	XShmDtimensec int64
	// Last change time.
	Ctime int64
	// Internal.
	XShmCtimensec int64
	// Internal.
	XShmInternal *byte
}
