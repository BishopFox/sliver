package sqlite3

import (
	"fmt"
	"strconv"
	"sync/atomic"

	"github.com/ncruces/go-sqlite3/internal/errutil"
	"github.com/ncruces/go-sqlite3/vfs"
)

// Config makes configuration changes to a database connection.
// Only boolean configuration options are supported.
// Called with no arg reads the current configuration value,
// called with one arg sets and returns the new value.
//
// https://sqlite.org/c3ref/db_config.html
func (c *Conn) Config(op DBConfig, arg ...bool) (bool, error) {
	if op < DBCONFIG_ENABLE_FKEY || op > DBCONFIG_REVERSE_SCANORDER {
		return false, MISUSE
	}

	// We need to call sqlite3_db_config, a variadic function.
	// We only support the `int int*` variants.
	// The int is a three-valued bool: -1 queries, 0/1 sets false/true.
	// The int* points to where new state will be written to.
	// The vararg is a pointer to an array containing these arguments:
	// an int and an int* pointing to that int.

	defer c.wrp.StackMark()()
	argsPtr := c.wrp.StackAlloc(intlen + ptrlen)

	var flag int32
	switch {
	case len(arg) == 0:
		flag = -1
	case arg[0]:
		flag = 1
	}

	c.wrp.Write32(argsPtr+0*ptrlen, uint32(flag))
	c.wrp.Write32(argsPtr+1*ptrlen, uint32(argsPtr))

	rc := res_t(c.wrp.Xsqlite3_db_config(int32(c.handle),
		int32(op), int32(argsPtr)))
	return c.wrp.ReadBool(argsPtr), c.error(rc)
}

var defaultLogger atomic.Pointer[func(code ExtendedErrorCode, msg string)]

// ConfigLog sets up the default error logging callback for new connections.
//
// https://sqlite.org/errlog.html
func ConfigLog(cb func(code ExtendedErrorCode, msg string)) {
	defaultLogger.Store(&cb)
}

// ConfigLog sets up the error logging callback for the connection.
//
// https://sqlite.org/errlog.html
func (c *Conn) ConfigLog(cb func(code ExtendedErrorCode, msg string)) error {
	var enable int32
	if cb != nil {
		enable = 1
	}
	rc := res_t(c.wrp.Xsqlite3_config_log_go(enable))
	if err := c.error(rc); err != nil {
		return err
	}
	c.log = cb
	return nil
}

func (e env) Xgo_log(_, iCode, zMsg int32) {
	if c, ok := e.DB.(*Conn); ok && c.log != nil {
		msg := e.ReadString(ptr_t(zMsg), _MAX_LENGTH)
		c.log(xErrorCode(iCode), msg)
	}
}

// Log writes a message into the error log established by [Conn.ConfigLog].
//
// https://sqlite.org/c3ref/log.html
func (c *Conn) Log(code ExtendedErrorCode, format string, a ...any) {
	if c.log != nil {
		c.log(code, fmt.Sprintf(format, a...))
	}
}

// FileControl allows low-level control of database files.
// Only a subset of opcodes are supported.
//
// https://sqlite.org/c3ref/file_control.html
func (c *Conn) FileControl(schema string, op FcntlOpcode, arg ...any) (any, error) {
	defer c.arena.Mark()()
	ptr := c.arena.New(max(ptrlen, intlen))

	var schemaPtr ptr_t
	if schema != "" {
		schemaPtr = c.arena.String(schema)
	}

	var rc res_t
	var ret any
	switch op {
	default:
		return nil, MISUSE

	case FCNTL_RESET_CACHE, FCNTL_NULL_IO:
		rc = res_t(c.wrp.Xsqlite3_file_control(
			int32(c.handle), int32(schemaPtr),
			int32(op), 0))

	case FCNTL_PERSIST_WAL, FCNTL_POWERSAFE_OVERWRITE:
		var flag int32
		switch {
		case len(arg) == 0:
			flag = -1
		case arg[0]:
			flag = 1
		}
		c.wrp.Write32(ptr, uint32(flag))
		rc = res_t(c.wrp.Xsqlite3_file_control(
			int32(c.handle), int32(schemaPtr),
			int32(op), int32(ptr)))
		ret = c.wrp.ReadBool(ptr)

	case FCNTL_CHUNK_SIZE:
		c.wrp.Write32(ptr, uint32(arg[0].(int)))
		rc = res_t(c.wrp.Xsqlite3_file_control(
			int32(c.handle), int32(schemaPtr),
			int32(op), int32(ptr)))

	case FCNTL_RESERVE_BYTES:
		bytes := -1
		if len(arg) > 0 {
			bytes = arg[0].(int)
		}
		c.wrp.Write32(ptr, uint32(bytes))
		rc = res_t(c.wrp.Xsqlite3_file_control(
			int32(c.handle), int32(schemaPtr),
			int32(op), int32(ptr)))
		ret = int(int32(c.wrp.Read32(ptr)))

	case FCNTL_DATA_VERSION:
		rc = res_t(c.wrp.Xsqlite3_file_control(
			int32(c.handle), int32(schemaPtr),
			int32(op), int32(ptr)))
		ret = c.wrp.Read32(ptr)

	case FCNTL_LOCKSTATE:
		rc = res_t(c.wrp.Xsqlite3_file_control(
			int32(c.handle), int32(schemaPtr),
			int32(op), int32(ptr)))
		ret = vfs.LockLevel(c.wrp.Read32(ptr))

	case FCNTL_VFSNAME, FCNTL_VFS_POINTER:
		rc = res_t(c.wrp.Xsqlite3_file_control(
			int32(c.handle), int32(schemaPtr),
			int32(FCNTL_VFS_POINTER), int32(ptr)))
		if rc == _OK {
			const zNameOffset = 16
			ptr = ptr_t(c.wrp.Read32(ptr))
			ptr = ptr_t(c.wrp.Read32(ptr + zNameOffset))
			name := c.wrp.ReadString(ptr, _MAX_NAME)
			if op == FCNTL_VFS_POINTER {
				ret = vfs.Find(name)
			} else {
				ret = name
			}
		}

	case FCNTL_FILE_POINTER, FCNTL_JOURNAL_POINTER:
		rc = res_t(c.wrp.Xsqlite3_file_control(
			int32(c.handle), int32(schemaPtr),
			int32(op), int32(ptr)))
		if rc == _OK {
			const fileHandleOffset = 4
			ptr = ptr_t(c.wrp.Read32(ptr))
			ptr = ptr_t(c.wrp.Read32(ptr + fileHandleOffset))
			ret = c.wrp.GetHandle(ptr)
		}
	}

	if err := c.error(rc); err != nil {
		return nil, err
	}
	return ret, nil
}

// Limit allows the size of various constructs to be
// limited on a connection by connection basis.
//
// https://sqlite.org/c3ref/limit.html
func (c *Conn) Limit(id LimitCategory, value int) int {
	return int(c.wrp.Xsqlite3_limit(int32(c.handle), int32(id), int32(value)))
}

// SetAuthorizer registers an authorizer callback with the database connection.
//
// https://sqlite.org/c3ref/set_authorizer.html
func (c *Conn) SetAuthorizer(cb func(action AuthorizerActionCode, name3rd, name4th, schema, inner string) AuthorizerReturnCode) error {
	var enable int32
	if cb != nil {
		enable = 1
	}
	rc := res_t(c.wrp.Xsqlite3_set_authorizer_go(int32(c.handle), enable))
	if err := c.error(rc); err != nil {
		return err
	}
	c.authorizer = cb
	return nil
}

func (e env) Xgo_authorizer(pDB, action, zName3rd, zName4th, zSchema, zInner int32) (rc int32) {
	if c, ok := e.DB.(*Conn); ok && c.handle == ptr_t(pDB) && c.authorizer != nil {
		var name3rd, name4th, schema, inner string
		if zName3rd != 0 {
			name3rd = e.ReadString(ptr_t(zName3rd), _MAX_NAME)
		}
		if zName4th != 0 {
			name4th = e.ReadString(ptr_t(zName4th), _MAX_NAME)
		}
		if zSchema != 0 {
			schema = e.ReadString(ptr_t(zSchema), _MAX_NAME)
		}
		if zInner != 0 {
			inner = e.ReadString(ptr_t(zInner), _MAX_NAME)
		}
		return int32(c.authorizer(AuthorizerActionCode(action), name3rd, name4th, schema, inner))
	}
	return _OK
}

// Trace registers a trace callback function against the database connection.
//
// https://sqlite.org/c3ref/trace_v2.html
func (c *Conn) Trace(mask TraceEvent, cb func(evt TraceEvent, arg1, arg2 any) error) error {
	rc := res_t(c.wrp.Xsqlite3_trace_go(int32(c.handle), int32(mask)))
	if err := c.error(rc); err != nil {
		return err
	}
	c.trace = cb
	return nil
}

func (e env) Xgo_trace(evt, pDB, pArg1, pArg2 int32) int32 {
	if c, ok := e.DB.(*Conn); ok && c.handle == ptr_t(pDB) && c.trace != nil {
		var arg1, arg2 any
		if TraceEvent(evt) == TRACE_CLOSE {
			arg1 = c
		} else {
			for _, s := range c.stmts {
				if ptr_t(pArg1) == s.handle {
					arg1 = s
					switch TraceEvent(evt) {
					case TRACE_STMT:
						arg2 = s.SQL()
					case TRACE_PROFILE:
						arg2 = int64(e.Read64(ptr_t(pArg2)))
					}
					break
				}
			}
		}
		if arg1 != nil {
			_ = c.trace(TraceEvent(evt), arg1, arg2)
		}
	}
	return _OK
}

// WALCheckpoint checkpoints a WAL database.
//
// https://sqlite.org/c3ref/wal_checkpoint_v2.html
func (c *Conn) WALCheckpoint(schema string, mode CheckpointMode) (nLog, nCkpt int, err error) {
	if c.interrupt.Err() != nil {
		return 0, 0, INTERRUPT
	}

	defer c.arena.Mark()()
	nLogPtr := c.arena.New(ptrlen)
	nCkptPtr := c.arena.New(ptrlen)
	schemaPtr := c.arena.String(schema)
	rc := res_t(c.wrp.Xsqlite3_wal_checkpoint_v2(
		int32(c.handle), int32(schemaPtr), int32(mode),
		int32(nLogPtr), int32(nCkptPtr)))
	nLog = int(int32(c.wrp.Read32(nLogPtr)))
	nCkpt = int(int32(c.wrp.Read32(nCkptPtr)))
	return nLog, nCkpt, c.error(rc)
}

// WALAutoCheckpoint configures WAL auto-checkpoints.
//
// https://sqlite.org/c3ref/wal_autocheckpoint.html
func (c *Conn) WALAutoCheckpoint(pages int) error {
	rc := res_t(c.wrp.Xsqlite3_wal_autocheckpoint(int32(c.handle), int32(pages)))
	return c.error(rc)
}

// WALHook registers a callback function to be invoked
// each time data is committed to a database in WAL mode.
//
// https://sqlite.org/c3ref/wal_hook.html
func (c *Conn) WALHook(cb func(db *Conn, schema string, pages int) error) {
	var enable int32
	if cb != nil {
		enable = 1
	}
	c.wrp.Xsqlite3_wal_hook_go(int32(c.handle), enable)
	c.wal = cb
}

func (e env) Xgo_wal_hook(_, pDB, zSchema, pages int32) int32 {
	if c, ok := e.DB.(*Conn); ok && c.handle == ptr_t(pDB) && c.wal != nil {
		schema := e.ReadString(ptr_t(zSchema), _MAX_NAME)
		err := c.wal(c, schema, int(pages))
		_, rc := errorCode(err, ERROR)
		return int32(rc)
	}
	return _OK
}

// AutoVacuumPages registers a autovacuum compaction amount callback.
//
// https://sqlite.org/c3ref/autovacuum_pages.html
func (c *Conn) AutoVacuumPages(cb func(schema string, dbPages, freePages, bytesPerPage uint) uint) error {
	var funcPtr ptr_t
	if cb != nil {
		funcPtr = c.wrp.AddHandle(cb)
	}
	rc := res_t(c.wrp.Xsqlite3_autovacuum_pages_go(int32(c.handle), int32(funcPtr)))
	return c.error(rc)
}

func (e env) Xgo_autovacuum_pages(pApp, zSchema, nDbPage, nFreePage, nBytePerPage int32) int32 {
	fn := e.GetHandle(ptr_t(pApp)).(func(schema string, dbPages, freePages, bytesPerPage uint) uint)
	schema := e.ReadString(ptr_t(zSchema), _MAX_NAME)
	return int32(fn(schema, uint(uint32(nDbPage)), uint(uint32(nFreePage)), uint(uint32(nBytePerPage))))
}

// SoftHeapLimit imposes a soft limit on heap size.
//
// https://sqlite.org/c3ref/hard_heap_limit64.html
func (c *Conn) SoftHeapLimit(n int64) int64 {
	return c.wrp.Xsqlite3_soft_heap_limit64(n)
}

// HardHeapLimit imposes a hard limit on heap size.
//
// https://sqlite.org/c3ref/hard_heap_limit64.html
func (c *Conn) HardHeapLimit(n int64) int64 {
	return c.wrp.Xsqlite3_hard_heap_limit64(n)
}

// EnableChecksums enables checksums on a database.
//
// https://sqlite.org/cksumvfs.html
func (c *Conn) EnableChecksums(schema string) error {
	r, err := c.FileControl(schema, FCNTL_RESERVE_BYTES)
	if err != nil {
		return err
	}
	if r == 8 {
		// Correct value, enabled.
		return nil
	}
	if r == 0 {
		// Default value, enable.
		_, err = c.FileControl(schema, FCNTL_RESERVE_BYTES, 8)
		if err != nil {
			return err
		}
		r, err = c.FileControl(schema, FCNTL_RESERVE_BYTES)
		if err != nil {
			return err
		}
	}
	if r != 8 {
		// Invalid value.
		return errutil.ErrorString("sqlite3: reserve bytes must be 8, is: " + strconv.Itoa(r.(int)))
	}

	// VACUUM the database.
	if schema != "" {
		err = c.Exec(`VACUUM ` + QuoteIdentifier(schema))
	} else {
		err = c.Exec(`VACUUM`)
	}
	if err != nil {
		return err
	}

	// Checkpoint the WAL.
	_, _, err = c.WALCheckpoint(schema, CHECKPOINT_FULL)
	return err
}
