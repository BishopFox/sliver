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
	Mode uint16
	// Padding.
	Pad1 uint16
	// Sequence number.
	Seq uint16
	// Padding.
	Pad2 uint16
	// Reserved.
	GlibcReserved1 uint32
	// Reserved.
	GlibcReserved2 uint32
}

// IdDs describes shared memory segment.
type IdDs struct {
	// Operation permission struct.
	Perm Perm
	// Size of segment in bytes.
	SegSz uint32
	// Last attach time.
	Atime int32
	// Reserved.
	GlibcReserved1 uint32
	// Last detach time.
	Dtime int32
	// Reserved.
	GlibcReserved2 uint32
	// Last change time.
	Ctime int32
	// Reserved.
	GlibcReserved3 uint32
	// Pid of creator.
	Cpid int32
	// Pid of last shmat/shmdt.
	Lpid int32
	// Number of current attaches.
	Nattch uint32
	// Reserved.
	GlibcReserved4 uint32
	// Reserved.
	GlibcReserved5 uint32
}
