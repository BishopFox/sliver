# Go SQLite VFS API

This package implements the SQLite [OS Interface](https://sqlite.org/vfs.html) (aka VFS).

It replaces the default SQLite VFS with a **pure Go** implementation.

It also exposes [interfaces](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs#VFS)
that should allow you to implement your own custom VFSes.

Since it is a from scratch reimplementation,
there are naturally some ways it deviates from the original.

The main differences are [file locking](#file-locking) and [WAL mode](write-ahead-logging) support.

### File Locking

POSIX advisory locks, which SQLite uses on Unix, are
[broken by design](https://github.com/sqlite/sqlite/blob/b74eb0/src/os_unix.c#L1073-L1161).

On Linux and macOS, this module uses
[OFD locks](https://www.gnu.org/software/libc/manual/html_node/Open-File-Description-Locks.html)
to synchronize access to database files.
OFD locks are fully compatible with POSIX advisory locks.

On BSD Unixes, this module uses
[BSD locks](https://man.freebsd.org/cgi/man.cgi?query=flock&sektion=2).
On BSD, these locks are fully compatible with POSIX advisory locks.
However, concurrency is reduced with BSD locks
(`BEGIN IMMEDIATE` behaves the same as `BEGIN EXCLUSIVE`). 

On Windows, this module uses `LockFileEx` and `UnlockFileEx`,
like SQLite.

On all other platforms, file locking is not supported, and you must use
[`nolock=1`](https://sqlite.org/uri.html#urinolock)
(or [`immutable=1`](https://sqlite.org/uri.html#uriimmutable))
to open database files.\
To use the [`database/sql`](https://pkg.go.dev/database/sql) driver
with `nolock=1` you must disable connection pooling by calling
[`db.SetMaxOpenConns(1)`](https://pkg.go.dev/database/sql#DB.SetMaxOpenConns).

You can use [`vfs.SupportsFileLocking`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs#SupportsFileLocking)
to check if your platform supports file locking.

### Write-Ahead Logging

On 64-bit Linux and macOS, this module uses `mmap` to implement
[shared-memory for the WAL-index](https://sqlite.org/wal.html#implementation_of_shared_memory_for_the_wal_index),
like SQLite.

To allow `mmap` to work, each connection needs to reserve up to 4GB of address space.\
To limit the amount of address space each connection needs,
use [`WithMemoryLimitPages`](../tests/testcfg/testcfg.go).

On Windows and BSD, [WAL](https://sqlite.org/wal.html) support is
[limited](https://sqlite.org/wal.html#noshm).
`EXCLUSIVE` locking mode can be set to create, read, and write WAL databases.\
To use `EXCLUSIVE` locking mode with the
[`database/sql`](https://pkg.go.dev/database/sql) driver
you must disable connection pooling by calling
[`db.SetMaxOpenConns(1)`](https://pkg.go.dev/database/sql#DB.SetMaxOpenConns).

On all other platforms, where file locking is not supported, WAL mode does not work.

You can use [`vfs.SupportsSharedMemory`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs#SupportsSharedMemory)
to check if your platform supports shared memory.

### Batch-Atomic Write

On 64-bit Linux, this module supports [batch-atomic writes](https://sqlite.org/cgi/src/technote/714)
with the F2FS filesystem.

### Build tags

The VFS can be customized with a few build tags:
- `sqlite3_flock` forces the use of BSD locks; it can be used on macOS to test the BSD locking implementation.
- `sqlite3_nosys` prevents importing [`x/sys`](https://pkg.go.dev/golang.org/x/sys);
  disables locking _and_ shared memory on all platforms.
- `sqlite3_noshm` disables shared memory on all platforms.
