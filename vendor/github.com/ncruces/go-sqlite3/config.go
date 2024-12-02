package sqlite3

import (
	"context"
	"fmt"
	"strconv"

	"github.com/tetratelabs/wazero/api"

	"github.com/ncruces/go-sqlite3/internal/util"
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

	defer c.arena.mark()()
	argsPtr := c.arena.new(intlen + ptrlen)

	var flag int
	switch {
	case len(arg) == 0:
		flag = -1
	case arg[0]:
		flag = 1
	}

	util.WriteUint32(c.mod, argsPtr+0*ptrlen, uint32(flag))
	util.WriteUint32(c.mod, argsPtr+1*ptrlen, argsPtr)

	r := c.call("sqlite3_db_config", uint64(c.handle),
		uint64(op), uint64(argsPtr))
	return util.ReadUint32(c.mod, argsPtr) != 0, c.error(r)
}

// ConfigLog sets up the error logging callback for the connection.
//
// https://sqlite.org/errlog.html
func (c *Conn) ConfigLog(cb func(code ExtendedErrorCode, msg string)) error {
	var enable uint64
	if cb != nil {
		enable = 1
	}
	r := c.call("sqlite3_config_log_go", enable)
	if err := c.error(r); err != nil {
		return err
	}
	c.log = cb
	return nil
}

func logCallback(ctx context.Context, mod api.Module, _, iCode, zMsg uint32) {
	if c, ok := ctx.Value(connKey{}).(*Conn); ok && c.log != nil {
		msg := util.ReadString(mod, zMsg, _MAX_LENGTH)
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
	defer c.arena.mark()()
	ptr := c.arena.new(max(ptrlen, intlen))

	var schemaPtr uint32
	if schema != "" {
		schemaPtr = c.arena.string(schema)
	}

	var rc uint64
	var res any
	switch op {
	default:
		return nil, MISUSE

	case FCNTL_RESET_CACHE:
		rc = c.call("sqlite3_file_control",
			uint64(c.handle), uint64(schemaPtr),
			uint64(op), 0)

	case FCNTL_PERSIST_WAL, FCNTL_POWERSAFE_OVERWRITE:
		var flag int
		switch {
		case len(arg) == 0:
			flag = -1
		case arg[0]:
			flag = 1
		}
		util.WriteUint32(c.mod, ptr, uint32(flag))
		rc = c.call("sqlite3_file_control",
			uint64(c.handle), uint64(schemaPtr),
			uint64(op), uint64(ptr))
		res = util.ReadUint32(c.mod, ptr) != 0

	case FCNTL_CHUNK_SIZE:
		util.WriteUint32(c.mod, ptr, uint32(arg[0].(int)))
		rc = c.call("sqlite3_file_control",
			uint64(c.handle), uint64(schemaPtr),
			uint64(op), uint64(ptr))

	case FCNTL_RESERVE_BYTES:
		bytes := -1
		if len(arg) > 0 {
			bytes = arg[0].(int)
		}
		util.WriteUint32(c.mod, ptr, uint32(bytes))
		rc = c.call("sqlite3_file_control",
			uint64(c.handle), uint64(schemaPtr),
			uint64(op), uint64(ptr))
		res = int(util.ReadUint32(c.mod, ptr))

	case FCNTL_DATA_VERSION:
		rc = c.call("sqlite3_file_control",
			uint64(c.handle), uint64(schemaPtr),
			uint64(op), uint64(ptr))
		res = util.ReadUint32(c.mod, ptr)

	case FCNTL_LOCKSTATE:
		rc = c.call("sqlite3_file_control",
			uint64(c.handle), uint64(schemaPtr),
			uint64(op), uint64(ptr))
		res = vfs.LockLevel(util.ReadUint32(c.mod, ptr))

	case FCNTL_VFS_POINTER:
		rc = c.call("sqlite3_file_control",
			uint64(c.handle), uint64(schemaPtr),
			uint64(op), uint64(ptr))
		if rc == _OK {
			const zNameOffset = 16
			ptr = util.ReadUint32(c.mod, ptr)
			ptr = util.ReadUint32(c.mod, ptr+zNameOffset)
			name := util.ReadString(c.mod, ptr, _MAX_NAME)
			res = vfs.Find(name)
		}

	case FCNTL_FILE_POINTER, FCNTL_JOURNAL_POINTER:
		rc = c.call("sqlite3_file_control",
			uint64(c.handle), uint64(schemaPtr),
			uint64(op), uint64(ptr))
		if rc == _OK {
			const fileHandleOffset = 4
			ptr = util.ReadUint32(c.mod, ptr)
			ptr = util.ReadUint32(c.mod, ptr+fileHandleOffset)
			res = util.GetHandle(c.ctx, ptr)
		}
	}

	if err := c.error(rc); err != nil {
		return nil, err
	}
	return res, nil
}

// Limit allows the size of various constructs to be
// limited on a connection by connection basis.
//
// https://sqlite.org/c3ref/limit.html
func (c *Conn) Limit(id LimitCategory, value int) int {
	r := c.call("sqlite3_limit", uint64(c.handle), uint64(id), uint64(value))
	return int(int32(r))
}

// SetAuthorizer registers an authorizer callback with the database connection.
//
// https://sqlite.org/c3ref/set_authorizer.html
func (c *Conn) SetAuthorizer(cb func(action AuthorizerActionCode, name3rd, name4th, schema, inner string) AuthorizerReturnCode) error {
	var enable uint64
	if cb != nil {
		enable = 1
	}
	r := c.call("sqlite3_set_authorizer_go", uint64(c.handle), enable)
	if err := c.error(r); err != nil {
		return err
	}
	c.authorizer = cb
	return nil

}

func authorizerCallback(ctx context.Context, mod api.Module, pDB uint32, action AuthorizerActionCode, zName3rd, zName4th, zSchema, zInner uint32) (rc AuthorizerReturnCode) {
	if c, ok := ctx.Value(connKey{}).(*Conn); ok && c.handle == pDB && c.authorizer != nil {
		var name3rd, name4th, schema, inner string
		if zName3rd != 0 {
			name3rd = util.ReadString(mod, zName3rd, _MAX_NAME)
		}
		if zName4th != 0 {
			name4th = util.ReadString(mod, zName4th, _MAX_NAME)
		}
		if zSchema != 0 {
			schema = util.ReadString(mod, zSchema, _MAX_NAME)
		}
		if zInner != 0 {
			inner = util.ReadString(mod, zInner, _MAX_NAME)
		}
		rc = c.authorizer(action, name3rd, name4th, schema, inner)
	}
	return rc
}

// Trace registers a trace callback function against the database connection.
//
// https://sqlite.org/c3ref/trace_v2.html
func (c *Conn) Trace(mask TraceEvent, cb func(evt TraceEvent, arg1 any, arg2 any) error) error {
	r := c.call("sqlite3_trace_go", uint64(c.handle), uint64(mask))
	if err := c.error(r); err != nil {
		return err
	}
	c.trace = cb
	return nil
}

func traceCallback(ctx context.Context, mod api.Module, evt TraceEvent, pDB, pArg1, pArg2 uint32) (rc uint32) {
	if c, ok := ctx.Value(connKey{}).(*Conn); ok && c.handle == pDB && c.trace != nil {
		var arg1, arg2 any
		if evt == TRACE_CLOSE {
			arg1 = c
		} else {
			for _, s := range c.stmts {
				if pArg1 == s.handle {
					arg1 = s
					switch evt {
					case TRACE_STMT:
						arg2 = s.SQL()
					case TRACE_PROFILE:
						arg2 = int64(util.ReadUint64(mod, pArg2))
					}
					break
				}
			}
		}
		if arg1 != nil {
			_, rc = errorCode(c.trace(evt, arg1, arg2), ERROR)
		}
	}
	return rc
}

// WALCheckpoint checkpoints a WAL database.
//
// https://sqlite.org/c3ref/wal_checkpoint_v2.html
func (c *Conn) WALCheckpoint(schema string, mode CheckpointMode) (nLog, nCkpt int, err error) {
	defer c.arena.mark()()
	nLogPtr := c.arena.new(ptrlen)
	nCkptPtr := c.arena.new(ptrlen)
	schemaPtr := c.arena.string(schema)
	r := c.call("sqlite3_wal_checkpoint_v2",
		uint64(c.handle), uint64(schemaPtr), uint64(mode),
		uint64(nLogPtr), uint64(nCkptPtr))
	nLog = int(int32(util.ReadUint32(c.mod, nLogPtr)))
	nCkpt = int(int32(util.ReadUint32(c.mod, nCkptPtr)))
	return nLog, nCkpt, c.error(r)
}

// WALAutoCheckpoint configures WAL auto-checkpoints.
//
// https://sqlite.org/c3ref/wal_autocheckpoint.html
func (c *Conn) WALAutoCheckpoint(pages int) error {
	r := c.call("sqlite3_wal_autocheckpoint", uint64(c.handle), uint64(pages))
	return c.error(r)
}

// WALHook registers a callback function to be invoked
// each time data is committed to a database in WAL mode.
//
// https://sqlite.org/c3ref/wal_hook.html
func (c *Conn) WALHook(cb func(db *Conn, schema string, pages int) error) {
	var enable uint64
	if cb != nil {
		enable = 1
	}
	c.call("sqlite3_wal_hook_go", uint64(c.handle), enable)
	c.wal = cb
}

func walCallback(ctx context.Context, mod api.Module, _, pDB, zSchema uint32, pages int32) (rc uint32) {
	if c, ok := ctx.Value(connKey{}).(*Conn); ok && c.handle == pDB && c.wal != nil {
		schema := util.ReadString(mod, zSchema, _MAX_NAME)
		err := c.wal(c, schema, int(pages))
		_, rc = errorCode(err, ERROR)
	}
	return rc
}

// AutoVacuumPages registers a autovacuum compaction amount callback.
//
// https://sqlite.org/c3ref/autovacuum_pages.html
func (c *Conn) AutoVacuumPages(cb func(schema string, dbPages, freePages, bytesPerPage uint) uint) error {
	var funcPtr uint32
	if cb != nil {
		funcPtr = util.AddHandle(c.ctx, cb)
	}
	r := c.call("sqlite3_autovacuum_pages_go", uint64(c.handle), uint64(funcPtr))
	return c.error(r)
}

func autoVacuumCallback(ctx context.Context, mod api.Module, pApp, zSchema, nDbPage, nFreePage, nBytePerPage uint32) uint32 {
	fn := util.GetHandle(ctx, pApp).(func(schema string, dbPages, freePages, bytesPerPage uint) uint)
	schema := util.ReadString(mod, zSchema, _MAX_NAME)
	return uint32(fn(schema, uint(nDbPage), uint(nFreePage), uint(nBytePerPage)))
}

// SoftHeapLimit imposes a soft limit on heap size.
//
// https://sqlite.org/c3ref/hard_heap_limit64.html
func (c *Conn) SoftHeapLimit(n int64) int64 {
	return int64(c.call("sqlite3_soft_heap_limit64", uint64(n)))
}

// HardHeapLimit imposes a hard limit on heap size.
//
// https://sqlite.org/c3ref/hard_heap_limit64.html
func (c *Conn) HardHeapLimit(n int64) int64 {
	return int64(c.call("sqlite3_hard_heap_limit64", uint64(n)))
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
		return util.ErrorString("sqlite3: reserve bytes must be 8, is: " + strconv.Itoa(r.(int)))
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
	_, _, err = c.WALCheckpoint(schema, CHECKPOINT_RESTART)
	return err
}
