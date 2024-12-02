# Go SQLite VFS API

This package implements the SQLite [OS Interface](https://sqlite.org/vfs.html) (aka VFS).

It replaces the default SQLite VFS with a **pure Go** implementation,
and exposes [interfaces](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs#VFS)
that should allow you to implement your own [custom VFSes](#custom-vfses).

Since it is a from scratch reimplementation,
there are naturally some ways it deviates from the original.

The main differences are [file locking](#file-locking) and [WAL mode](#write-ahead-logging) support.

### File Locking

POSIX advisory locks, which SQLite uses on Unix, are
[broken by design](https://github.com/sqlite/sqlite/blob/b74eb0/src/os_unix.c#L1073-L1161).
Instead, on Linux and macOS, this package uses
[OFD locks](https://www.gnu.org/software/libc/manual/html_node/Open-File-Description-Locks.html)
to synchronize access to database files.

This package can also use
[BSD locks](https://man.freebsd.org/cgi/man.cgi?query=flock&sektion=2),
albeit with reduced concurrency (`BEGIN IMMEDIATE` behaves like `BEGIN EXCLUSIVE`).
BSD locks are the default on BSD and illumos,
but you can opt into them with the `sqlite3_flock` build tag.

On Windows, this package uses `LockFileEx` and `UnlockFileEx`,
like SQLite.

You can also opt into a cross-platform locking implementation
with the `sqlite3_dotlk` build tag.

Otherwise, file locking is not supported, and you must use
[`nolock=1`](https://sqlite.org/uri.html#urinolock)
(or [`immutable=1`](https://sqlite.org/uri.html#uriimmutable))
to open database files.
To use the [`database/sql`](https://pkg.go.dev/database/sql) driver
with `nolock=1` you must disable connection pooling by calling
[`db.SetMaxOpenConns(1)`](https://pkg.go.dev/database/sql#DB.SetMaxOpenConns).

You can use [`vfs.SupportsFileLocking`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs#SupportsFileLocking)
to check if your build supports file locking.

### Write-Ahead Logging

On Unix, this package may use `mmap` to implement
[shared-memory for the WAL-index](https://sqlite.org/wal.html#implementation_of_shared_memory_for_the_wal_index),
like SQLite.

With [BSD locks](https://man.freebsd.org/cgi/man.cgi?query=flock&sektion=2)
a WAL database can only be accessed by a single proccess.
Other processes that attempt to access a database locked with BSD locks,
will fail with the [`SQLITE_PROTOCOL`](https://sqlite.org/rescode.html#protocol) error code.

On Windows, this package may use `MapViewOfFile`, like SQLite.

You can also opt into a cross-platform, in-process, memory sharing implementation
with the `sqlite3_dotlk` build tag.

Otherwise, [WAL support is limited](https://sqlite.org/wal.html#noshm),
and `EXCLUSIVE` locking mode must be set to create, read, and write WAL databases.
To use `EXCLUSIVE` locking mode with the
[`database/sql`](https://pkg.go.dev/database/sql) driver
you must disable connection pooling by calling
[`db.SetMaxOpenConns(1)`](https://pkg.go.dev/database/sql#DB.SetMaxOpenConns).

You can use [`vfs.SupportsSharedMemory`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs#SupportsSharedMemory)
to check if your build supports shared memory.

### Batch-Atomic Write

On Linux, this package may support
[batch-atomic writes](https://sqlite.org/cgi/src/technote/714)
on the F2FS filesystem.

### Checksums

This package can be [configured](https://pkg.go.dev/github.com/ncruces/go-sqlite3#Conn.EnableChecksums)
to add an 8-byte checksum to the end of every page in an SQLite database.
The checksum is added as each page is written
and verified as each page is read.\
The checksum is intended to help detect database corruption
caused by random bit-flips in the mass storage device.

The implementation is compatible with SQLite's
[Checksum VFS Shim](https://sqlite.org/cksumvfs.html).

### Build Tags

The VFS can be customized with a few build tags:
- `sqlite3_flock` forces the use of BSD locks.
- `sqlite3_dotlk` forces the use of dot-file locks.
- `sqlite3_nosys` prevents importing [`x/sys`](https://pkg.go.dev/golang.org/x/sys).

> [!IMPORTANT]
> The default configuration of this package is compatible with the standard
> [Unix and Windows SQLite VFSes](https://sqlite.org/vfs.html#multiple_vfses);
> `sqlite3_flock` builds are compatible with the
> [`unix-flock` VFS](https://sqlite.org/compile.html#enable_locking_style);
> `sqlite3_dotlk` builds are compatible with the
> [`unix-dotfile` VFS](https://sqlite.org/compile.html#enable_locking_style).
> If incompatible file locking is used, accessing databases concurrently with
> _other_ SQLite libraries will eventually corrupt data.

### Custom VFSes

- [`github.com/ncruces/go-sqlite3/vfs/memdb`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/memdb)
  implements an in-memory VFS.
- [`github.com/ncruces/go-sqlite3/vfs/readervfs`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/readervfs)
  implements a VFS for immutable databases.
- [`github.com/ncruces/go-sqlite3/vfs/adiantum`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/adiantum)
  wraps a VFS to offer encryption at rest.
- [`github.com/ncruces/go-sqlite3/vfs/xts`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/xts)
  wraps a VFS to offer encryption at rest.
