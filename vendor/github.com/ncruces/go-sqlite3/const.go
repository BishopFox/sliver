package sqlite3

import "strconv"

const (
	_OK   = 0   /* Successful result */
	_ROW  = 100 /* sqlite3_step() has another row ready */
	_DONE = 101 /* sqlite3_step() has finished executing */

	_OK_SYMLINK = (_OK | (2 << 8)) /* internal use only */

	_UTF8 = 1

	_MAX_STRING   = 512 // Used for short strings: names, error messages…
	_MAX_PATHNAME = 512

	_MAX_ALLOCATION_SIZE = 0x7ffffeff

	ptrlen = 4
)

// ErrorCode is a result code that [Error.Code] might return.
//
// https://www.sqlite.org/rescode.html
type ErrorCode uint8

const (
	ERROR      ErrorCode = 1  /* Generic error */
	INTERNAL   ErrorCode = 2  /* Internal logic error in SQLite */
	PERM       ErrorCode = 3  /* Access permission denied */
	ABORT      ErrorCode = 4  /* Callback routine requested an abort */
	BUSY       ErrorCode = 5  /* The database file is locked */
	LOCKED     ErrorCode = 6  /* A table in the database is locked */
	NOMEM      ErrorCode = 7  /* A malloc() failed */
	READONLY   ErrorCode = 8  /* Attempt to write a readonly database */
	INTERRUPT  ErrorCode = 9  /* Operation terminated by sqlite3_interrupt() */
	IOERR      ErrorCode = 10 /* Some kind of disk I/O error occurred */
	CORRUPT    ErrorCode = 11 /* The database disk image is malformed */
	NOTFOUND   ErrorCode = 12 /* Unknown opcode in sqlite3_file_control() */
	FULL       ErrorCode = 13 /* Insertion failed because database is full */
	CANTOPEN   ErrorCode = 14 /* Unable to open the database file */
	PROTOCOL   ErrorCode = 15 /* Database lock protocol error */
	EMPTY      ErrorCode = 16 /* Internal use only */
	SCHEMA     ErrorCode = 17 /* The database schema changed */
	TOOBIG     ErrorCode = 18 /* String or BLOB exceeds size limit */
	CONSTRAINT ErrorCode = 19 /* Abort due to constraint violation */
	MISMATCH   ErrorCode = 20 /* Data type mismatch */
	MISUSE     ErrorCode = 21 /* Library used incorrectly */
	NOLFS      ErrorCode = 22 /* Uses OS features not supported on host */
	AUTH       ErrorCode = 23 /* Authorization denied */
	FORMAT     ErrorCode = 24 /* Not used */
	RANGE      ErrorCode = 25 /* 2nd parameter to sqlite3_bind out of range */
	NOTADB     ErrorCode = 26 /* File opened that is not a database file */
	NOTICE     ErrorCode = 27 /* Notifications from sqlite3_log() */
	WARNING    ErrorCode = 28 /* Warnings from sqlite3_log() */
)

// ExtendedErrorCode is a result code that [Error.ExtendedCode] might return.
//
// https://www.sqlite.org/rescode.html
type (
	ExtendedErrorCode uint16
	xErrorCode        = ExtendedErrorCode
)

const (
	ERROR_MISSING_COLLSEQ   ExtendedErrorCode = xErrorCode(ERROR) | (1 << 8)
	ERROR_RETRY             ExtendedErrorCode = xErrorCode(ERROR) | (2 << 8)
	ERROR_SNAPSHOT          ExtendedErrorCode = xErrorCode(ERROR) | (3 << 8)
	IOERR_READ              ExtendedErrorCode = xErrorCode(IOERR) | (1 << 8)
	IOERR_SHORT_READ        ExtendedErrorCode = xErrorCode(IOERR) | (2 << 8)
	IOERR_WRITE             ExtendedErrorCode = xErrorCode(IOERR) | (3 << 8)
	IOERR_FSYNC             ExtendedErrorCode = xErrorCode(IOERR) | (4 << 8)
	IOERR_DIR_FSYNC         ExtendedErrorCode = xErrorCode(IOERR) | (5 << 8)
	IOERR_TRUNCATE          ExtendedErrorCode = xErrorCode(IOERR) | (6 << 8)
	IOERR_FSTAT             ExtendedErrorCode = xErrorCode(IOERR) | (7 << 8)
	IOERR_UNLOCK            ExtendedErrorCode = xErrorCode(IOERR) | (8 << 8)
	IOERR_RDLOCK            ExtendedErrorCode = xErrorCode(IOERR) | (9 << 8)
	IOERR_DELETE            ExtendedErrorCode = xErrorCode(IOERR) | (10 << 8)
	IOERR_BLOCKED           ExtendedErrorCode = xErrorCode(IOERR) | (11 << 8)
	IOERR_NOMEM             ExtendedErrorCode = xErrorCode(IOERR) | (12 << 8)
	IOERR_ACCESS            ExtendedErrorCode = xErrorCode(IOERR) | (13 << 8)
	IOERR_CHECKRESERVEDLOCK ExtendedErrorCode = xErrorCode(IOERR) | (14 << 8)
	IOERR_LOCK              ExtendedErrorCode = xErrorCode(IOERR) | (15 << 8)
	IOERR_CLOSE             ExtendedErrorCode = xErrorCode(IOERR) | (16 << 8)
	IOERR_DIR_CLOSE         ExtendedErrorCode = xErrorCode(IOERR) | (17 << 8)
	IOERR_SHMOPEN           ExtendedErrorCode = xErrorCode(IOERR) | (18 << 8)
	IOERR_SHMSIZE           ExtendedErrorCode = xErrorCode(IOERR) | (19 << 8)
	IOERR_SHMLOCK           ExtendedErrorCode = xErrorCode(IOERR) | (20 << 8)
	IOERR_SHMMAP            ExtendedErrorCode = xErrorCode(IOERR) | (21 << 8)
	IOERR_SEEK              ExtendedErrorCode = xErrorCode(IOERR) | (22 << 8)
	IOERR_DELETE_NOENT      ExtendedErrorCode = xErrorCode(IOERR) | (23 << 8)
	IOERR_MMAP              ExtendedErrorCode = xErrorCode(IOERR) | (24 << 8)
	IOERR_GETTEMPPATH       ExtendedErrorCode = xErrorCode(IOERR) | (25 << 8)
	IOERR_CONVPATH          ExtendedErrorCode = xErrorCode(IOERR) | (26 << 8)
	IOERR_VNODE             ExtendedErrorCode = xErrorCode(IOERR) | (27 << 8)
	IOERR_AUTH              ExtendedErrorCode = xErrorCode(IOERR) | (28 << 8)
	IOERR_BEGIN_ATOMIC      ExtendedErrorCode = xErrorCode(IOERR) | (29 << 8)
	IOERR_COMMIT_ATOMIC     ExtendedErrorCode = xErrorCode(IOERR) | (30 << 8)
	IOERR_ROLLBACK_ATOMIC   ExtendedErrorCode = xErrorCode(IOERR) | (31 << 8)
	IOERR_DATA              ExtendedErrorCode = xErrorCode(IOERR) | (32 << 8)
	IOERR_CORRUPTFS         ExtendedErrorCode = xErrorCode(IOERR) | (33 << 8)
	LOCKED_SHAREDCACHE      ExtendedErrorCode = xErrorCode(LOCKED) | (1 << 8)
	LOCKED_VTAB             ExtendedErrorCode = xErrorCode(LOCKED) | (2 << 8)
	BUSY_RECOVERY           ExtendedErrorCode = xErrorCode(BUSY) | (1 << 8)
	BUSY_SNAPSHOT           ExtendedErrorCode = xErrorCode(BUSY) | (2 << 8)
	BUSY_TIMEOUT            ExtendedErrorCode = xErrorCode(BUSY) | (3 << 8)
	CANTOPEN_NOTEMPDIR      ExtendedErrorCode = xErrorCode(CANTOPEN) | (1 << 8)
	CANTOPEN_ISDIR          ExtendedErrorCode = xErrorCode(CANTOPEN) | (2 << 8)
	CANTOPEN_FULLPATH       ExtendedErrorCode = xErrorCode(CANTOPEN) | (3 << 8)
	CANTOPEN_CONVPATH       ExtendedErrorCode = xErrorCode(CANTOPEN) | (4 << 8)
	CANTOPEN_DIRTYWAL       ExtendedErrorCode = xErrorCode(CANTOPEN) | (5 << 8) /* Not Used */
	CANTOPEN_SYMLINK        ExtendedErrorCode = xErrorCode(CANTOPEN) | (6 << 8)
	CORRUPT_VTAB            ExtendedErrorCode = xErrorCode(CORRUPT) | (1 << 8)
	CORRUPT_SEQUENCE        ExtendedErrorCode = xErrorCode(CORRUPT) | (2 << 8)
	CORRUPT_INDEX           ExtendedErrorCode = xErrorCode(CORRUPT) | (3 << 8)
	READONLY_RECOVERY       ExtendedErrorCode = xErrorCode(READONLY) | (1 << 8)
	READONLY_CANTLOCK       ExtendedErrorCode = xErrorCode(READONLY) | (2 << 8)
	READONLY_ROLLBACK       ExtendedErrorCode = xErrorCode(READONLY) | (3 << 8)
	READONLY_DBMOVED        ExtendedErrorCode = xErrorCode(READONLY) | (4 << 8)
	READONLY_CANTINIT       ExtendedErrorCode = xErrorCode(READONLY) | (5 << 8)
	READONLY_DIRECTORY      ExtendedErrorCode = xErrorCode(READONLY) | (6 << 8)
	ABORT_ROLLBACK          ExtendedErrorCode = xErrorCode(ABORT) | (2 << 8)
	CONSTRAINT_CHECK        ExtendedErrorCode = xErrorCode(CONSTRAINT) | (1 << 8)
	CONSTRAINT_COMMITHOOK   ExtendedErrorCode = xErrorCode(CONSTRAINT) | (2 << 8)
	CONSTRAINT_FOREIGNKEY   ExtendedErrorCode = xErrorCode(CONSTRAINT) | (3 << 8)
	CONSTRAINT_FUNCTION     ExtendedErrorCode = xErrorCode(CONSTRAINT) | (4 << 8)
	CONSTRAINT_NOTNULL      ExtendedErrorCode = xErrorCode(CONSTRAINT) | (5 << 8)
	CONSTRAINT_PRIMARYKEY   ExtendedErrorCode = xErrorCode(CONSTRAINT) | (6 << 8)
	CONSTRAINT_TRIGGER      ExtendedErrorCode = xErrorCode(CONSTRAINT) | (7 << 8)
	CONSTRAINT_UNIQUE       ExtendedErrorCode = xErrorCode(CONSTRAINT) | (8 << 8)
	CONSTRAINT_VTAB         ExtendedErrorCode = xErrorCode(CONSTRAINT) | (9 << 8)
	CONSTRAINT_ROWID        ExtendedErrorCode = xErrorCode(CONSTRAINT) | (10 << 8)
	CONSTRAINT_PINNED       ExtendedErrorCode = xErrorCode(CONSTRAINT) | (11 << 8)
	CONSTRAINT_DATATYPE     ExtendedErrorCode = xErrorCode(CONSTRAINT) | (12 << 8)
	NOTICE_RECOVER_WAL      ExtendedErrorCode = xErrorCode(NOTICE) | (1 << 8)
	NOTICE_RECOVER_ROLLBACK ExtendedErrorCode = xErrorCode(NOTICE) | (2 << 8)
	NOTICE_RBU              ExtendedErrorCode = xErrorCode(NOTICE) | (3 << 8)
	WARNING_AUTOINDEX       ExtendedErrorCode = xErrorCode(WARNING) | (1 << 8)
	AUTH_USER               ExtendedErrorCode = xErrorCode(AUTH) | (1 << 8)
)

// OpenFlag is a flag for a file open operation.
//
// https://www.sqlite.org/c3ref/c_open_autoproxy.html
type OpenFlag uint32

const (
	OPEN_READONLY      OpenFlag = 0x00000001 /* Ok for sqlite3_open_v2() */
	OPEN_READWRITE     OpenFlag = 0x00000002 /* Ok for sqlite3_open_v2() */
	OPEN_CREATE        OpenFlag = 0x00000004 /* Ok for sqlite3_open_v2() */
	OPEN_DELETEONCLOSE OpenFlag = 0x00000008 /* VFS only */
	OPEN_EXCLUSIVE     OpenFlag = 0x00000010 /* VFS only */
	OPEN_AUTOPROXY     OpenFlag = 0x00000020 /* VFS only */
	OPEN_URI           OpenFlag = 0x00000040 /* Ok for sqlite3_open_v2() */
	OPEN_MEMORY        OpenFlag = 0x00000080 /* Ok for sqlite3_open_v2() */
	OPEN_MAIN_DB       OpenFlag = 0x00000100 /* VFS only */
	OPEN_TEMP_DB       OpenFlag = 0x00000200 /* VFS only */
	OPEN_TRANSIENT_DB  OpenFlag = 0x00000400 /* VFS only */
	OPEN_MAIN_JOURNAL  OpenFlag = 0x00000800 /* VFS only */
	OPEN_TEMP_JOURNAL  OpenFlag = 0x00001000 /* VFS only */
	OPEN_SUBJOURNAL    OpenFlag = 0x00002000 /* VFS only */
	OPEN_SUPER_JOURNAL OpenFlag = 0x00004000 /* VFS only */
	OPEN_NOMUTEX       OpenFlag = 0x00008000 /* Ok for sqlite3_open_v2() */
	OPEN_FULLMUTEX     OpenFlag = 0x00010000 /* Ok for sqlite3_open_v2() */
	OPEN_SHAREDCACHE   OpenFlag = 0x00020000 /* Ok for sqlite3_open_v2() */
	OPEN_PRIVATECACHE  OpenFlag = 0x00040000 /* Ok for sqlite3_open_v2() */
	OPEN_WAL           OpenFlag = 0x00080000 /* VFS only */
	OPEN_NOFOLLOW      OpenFlag = 0x01000000 /* Ok for sqlite3_open_v2() */
	OPEN_EXRESCODE     OpenFlag = 0x02000000 /* Extended result codes */
)

// PrepareFlag is a flag that can be passed to [Conn.PrepareFlags].
//
// https://www.sqlite.org/c3ref/c_prepare_normalize.html
type PrepareFlag uint32

const (
	PREPARE_PERSISTENT PrepareFlag = 0x01
	PREPARE_NORMALIZE  PrepareFlag = 0x02
	PREPARE_NO_VTAB    PrepareFlag = 0x04
)

// Datatype is a fundamental datatype of SQLite.
//
// https://www.sqlite.org/c3ref/c_blob.html
type Datatype uint32

const (
	INTEGER Datatype = 1
	FLOAT   Datatype = 2
	TEXT    Datatype = 3
	BLOB    Datatype = 4
	NULL    Datatype = 5
)

// String implements the [fmt.Stringer] interface.
func (t Datatype) String() string {
	const name = "INTEGERFLOATTEXTBLOBNULL"
	switch t {
	case INTEGER:
		return name[0:7]
	case FLOAT:
		return name[7:12]
	case TEXT:
		return name[12:16]
	case BLOB:
		return name[16:20]
	case NULL:
		return name[20:24]
	}
	return strconv.FormatUint(uint64(t), 10)
}

type _AccessFlag uint32

const (
	_ACCESS_EXISTS    _AccessFlag = 0
	_ACCESS_READWRITE _AccessFlag = 1 /* Used by PRAGMA temp_store_directory */
	_ACCESS_READ      _AccessFlag = 2 /* Unused */
)

type _SyncFlag uint32

const (
	_SYNC_NORMAL   _SyncFlag = 0x00002
	_SYNC_FULL     _SyncFlag = 0x00003
	_SYNC_DATAONLY _SyncFlag = 0x00010
)

type _FcntlOpcode uint32

const (
	_FCNTL_LOCKSTATE             = 1
	_FCNTL_GET_LOCKPROXYFILE     = 2
	_FCNTL_SET_LOCKPROXYFILE     = 3
	_FCNTL_LAST_ERRNO            = 4
	_FCNTL_SIZE_HINT             = 5
	_FCNTL_CHUNK_SIZE            = 6
	_FCNTL_FILE_POINTER          = 7
	_FCNTL_SYNC_OMITTED          = 8
	_FCNTL_WIN32_AV_RETRY        = 9
	_FCNTL_PERSIST_WAL           = 10
	_FCNTL_OVERWRITE             = 11
	_FCNTL_VFSNAME               = 12
	_FCNTL_POWERSAFE_OVERWRITE   = 13
	_FCNTL_PRAGMA                = 14
	_FCNTL_BUSYHANDLER           = 15
	_FCNTL_TEMPFILENAME          = 16
	_FCNTL_MMAP_SIZE             = 18
	_FCNTL_TRACE                 = 19
	_FCNTL_HAS_MOVED             = 20
	_FCNTL_SYNC                  = 21
	_FCNTL_COMMIT_PHASETWO       = 22
	_FCNTL_WIN32_SET_HANDLE      = 23
	_FCNTL_WAL_BLOCK             = 24
	_FCNTL_ZIPVFS                = 25
	_FCNTL_RBU                   = 26
	_FCNTL_VFS_POINTER           = 27
	_FCNTL_JOURNAL_POINTER       = 28
	_FCNTL_WIN32_GET_HANDLE      = 29
	_FCNTL_PDB                   = 30
	_FCNTL_BEGIN_ATOMIC_WRITE    = 31
	_FCNTL_COMMIT_ATOMIC_WRITE   = 32
	_FCNTL_ROLLBACK_ATOMIC_WRITE = 33
	_FCNTL_LOCK_TIMEOUT          = 34
	_FCNTL_DATA_VERSION          = 35
	_FCNTL_SIZE_LIMIT            = 36
	_FCNTL_CKPT_DONE             = 37
	_FCNTL_RESERVE_BYTES         = 38
	_FCNTL_CKPT_START            = 39
	_FCNTL_EXTERNAL_READER       = 40
	_FCNTL_CKSM_FILE             = 41
	_FCNTL_RESET_CACHE           = 42
)
