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
	// Creator's user ID.
	Cuid uint16
	// Creator's group ID.
	Cgid uint16
	// Owner's user ID.
	Uid uint16
	// Owner's group ID.
	Gid uint16
	// Read/write permission.
	Mode uint16
	// Sequence number.
	Seq uint16
	// Padding.
	PadCgo0 [4]byte
	// Key.
	Key int64
}

// IdDs describes shared memory segment.
type IdDs struct {
	// Operation permission struct.
	Perm Perm
	// Size of segment in bytes.
	SegSz uint64
	// Pid of last shmat/shmdt.
	Lpid int32
	// Pid of creator.
	Cpid int32
	// Number of current attaches.
	Nattch uint32
	// Padding.
	PadCgo0 [4]byte
	// Last attach time.
	Atime int64
	// Last detach time.
	Dtime int64
	// Last change time.
	Ctime int64
	// Internal.
	Internal *byte
}
