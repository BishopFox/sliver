package sqlite3_wrap

type errorCode interface {
	~uint8 | ~uint16 | ~uint32 | ~int32
}

func ErrorCodeString[T errorCode](rc T) string {
	switch uint32(rc) {
	case ABORT_ROLLBACK:
		return "sqlite3: abort due to ROLLBACK"
	case ROW:
		return "sqlite3: another row available"
	case DONE:
		return "sqlite3: no more rows available"
	}
	switch uint8(rc) {
	case OK:
		return "sqlite3: not an error"
	case ERROR:
		return "sqlite3: SQL logic error"
	case INTERNAL:
		break
	case PERM:
		return "sqlite3: access permission denied"
	case ABORT:
		return "sqlite3: query aborted"
	case BUSY:
		return "sqlite3: database is locked"
	case LOCKED:
		return "sqlite3: database table is locked"
	case NOMEM:
		return "sqlite3: out of memory"
	case READONLY:
		return "sqlite3: attempt to write a readonly database"
	case INTERRUPT:
		return "sqlite3: interrupted"
	case IOERR:
		return "sqlite3: disk I/O error"
	case CORRUPT:
		return "sqlite3: database disk image is malformed"
	case NOTFOUND:
		return "sqlite3: unknown operation"
	case FULL:
		return "sqlite3: database or disk is full"
	case CANTOPEN:
		return "sqlite3: unable to open database file"
	case PROTOCOL:
		return "sqlite3: locking protocol"
	case EMPTY:
		break
	case SCHEMA:
		return "sqlite3: database schema has changed"
	case TOOBIG:
		return "sqlite3: string or blob too big"
	case CONSTRAINT:
		return "sqlite3: constraint failed"
	case MISMATCH:
		return "sqlite3: datatype mismatch"
	case MISUSE:
		return "sqlite3: bad parameter or other API misuse"
	case NOLFS:
		break
	case AUTH:
		return "sqlite3: authorization denied"
	case FORMAT:
		break
	case RANGE:
		return "sqlite3: column index out of range"
	case NOTADB:
		return "sqlite3: file is not a database"
	case NOTICE:
		return "sqlite3: notification message"
	case WARNING:
		return "sqlite3: warning message"
	}
	return "sqlite3: unknown error"
}
