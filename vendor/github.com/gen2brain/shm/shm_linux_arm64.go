package shm

// System call constants.
const (
	sysShmAt  = 21
	sysShmCtl = 24
	sysShmDt  = 22
	sysShmGet = 23
)

// Perm is used to pass permission information to IPC operations.
type Perm struct {
	// Key.
	Key int32
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
	Seq uint16
	// Padding.
	Pad1 uint16
	// Reserved.
	GlibcReserved1 uint64
	// Reserved.
	GlibcReserved2 uint64
}

// IdDs describes shared memory segment.
type IdDs struct {
	// Operation permission struct.
	Perm Perm
	// Size of segment in bytes.
	SegSz uint64
	// Last attach time.
	Atime int64
	// Last detach time.
	Dtime int64
	// Last change time.
	Ctime int64
	// Pid of creator.
	Cpid int32
	// Pid of last shmat/shmdt.
	Lpid int32
	// Number of current attaches.
	Nattch uint64
	// Reserved.
	GlibcReserved5 uint64
	// Reserved.
	GlibcReserved6 uint64
}
