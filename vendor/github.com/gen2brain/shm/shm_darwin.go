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
	Uid uint32
	// Owner's group ID.
	Gid uint32
	// Creator's user ID.
	Cuid uint32
	// Creator's group ID.
	Cgid uint32
	// Read/write permission.
	Mode uint16
	// Sequence number.
	Seq uint16
	// Key.
	Key int32
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
	Nattch uint16
	// Padding.
	PadCgo0 [2]byte
	// Padding.
	PadCgo1 [8]byte
	// Padding.
	PadCgo2 [8]byte
	// Padding.
	PadCgo3 [8]byte
	// Padding.
	PadCgo4 [8]byte
}
