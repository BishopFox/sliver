package shm

// System call constants.
const (
	sysShmAt  = 228
	sysShmCtl = 229
	sysShmDt  = 230
	sysShmGet = 231
)

// Perm is used to pass permission information to IPC operations.
type Perm struct {
	// Creator's user ID.
	Cuid uint32
	// Creator's group ID.
	Cgid uint32
	// Owner's user ID.
	Uid uint32
	// Owner's group ID.
	Gid uint32
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
	Nattch int32
	// Padding.
	PadCgo0 [4]byte
	// Last attach time.
	Atime int64
	// Last detach time.
	Dtime int64
	// Last change time.
	Ctime int64
}
