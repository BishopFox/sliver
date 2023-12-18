# Go bindings to SQLite using Wazero

[![Go Reference](https://pkg.go.dev/badge/image)](https://pkg.go.dev/github.com/ncruces/go-sqlite3)
[![Go Report](https://goreportcard.com/badge/github.com/ncruces/go-sqlite3)](https://goreportcard.com/report/github.com/ncruces/go-sqlite3)
[![Go Coverage](https://github.com/ncruces/go-sqlite3/wiki/coverage.svg)](https://github.com/ncruces/go-sqlite3/wiki/Test-coverage-report)

Go module `github.com/ncruces/go-sqlite3` wraps a [WASM](https://webassembly.org/) build of [SQLite](https://sqlite.org/),
and uses [wazero](https://wazero.io/) to provide `cgo`-free SQLite bindings.

- [`github.com/ncruces/go-sqlite3`](https://pkg.go.dev/github.com/ncruces/go-sqlite3)
  wraps the [C SQLite API](https://sqlite.org/cintro.html)
  ([example usage](https://pkg.go.dev/github.com/ncruces/go-sqlite3#example-package)).
- [`github.com/ncruces/go-sqlite3/driver`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/driver)
  provides a [`database/sql`](https://pkg.go.dev/database/sql) driver
  ([example usage](https://pkg.go.dev/github.com/ncruces/go-sqlite3/driver#example-package)).
- [`github.com/ncruces/go-sqlite3/embed`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/embed)
  embeds a build of SQLite into your application.
- [`github.com/ncruces/go-sqlite3/vfs`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs)
  wraps the [C SQLite VFS API](https://sqlite.org/vfs.html) and provides a pure Go implementation.
- [`github.com/ncruces/go-sqlite3/gormlite`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/gormlite)
  provides a [GORM](https://gorm.io) driver.

### Loadable extensions

- [`github.com/ncruces/go-sqlite3/ext/array`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/array)
  provides the [`array`](https://sqlite.org/carray.html) table-valued function.
- [`github.com/ncruces/go-sqlite3/ext/blobio`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/blobio)
  simplifies [incremental BLOB I/O](https://sqlite.org/c3ref/blob_open.html).
- [`github.com/ncruces/go-sqlite3/ext/csv`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/csv)
  reads [comma-separated values](https://sqlite.org/csv.html).
- [`github.com/ncruces/go-sqlite3/ext/fileio`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/fileio)
  reads, writes and lists files.
- [`github.com/ncruces/go-sqlite3/ext/lines`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/lines)
  reads data [line-by-line](https://github.com/asg017/sqlite-lines).
- [`github.com/ncruces/go-sqlite3/ext/pivot`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/pivot)
  creates [pivot tables](https://github.com/jakethaw/pivot_vtab).
- [`github.com/ncruces/go-sqlite3/ext/statement`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/statement)
  creates [parameterized views](https://github.com/0x09/sqlite-statement-vtab).
- [`github.com/ncruces/go-sqlite3/ext/stats`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/stats)
  provides [statistics](https://www.oreilly.com/library/view/sql-in-a/9780596155322/ch04s02.html) functions.
- [`github.com/ncruces/go-sqlite3/ext/unicode`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/unicode)
  provides [Unicode aware](https://sqlite.org/src/dir/ext/icu) functions.
- [`github.com/ncruces/go-sqlite3/vfs/memdb`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/memdb)
  implements an in-memory VFS.
- [`github.com/ncruces/go-sqlite3/vfs/readervfs`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/readervfs)
  implements a VFS for immutable databases.

### Advanced features

- [x] [incremental BLOB I/O](https://sqlite.org/c3ref/blob_open.html)
- [x] [nested transactions](https://sqlite.org/lang_savepoint.html)
- [x] [custom functions](https://sqlite.org/c3ref/create_function.html)
- [x] [virtual tables](https://sqlite.org/vtab.html)
- [x] [custom VFSes](https://sqlite.org/vfs.html)
- [x] [online backup](https://sqlite.org/backup.html)
- [x] [JSON support](https://sqlite.org/json1.html)
- [x] [Unicode support](https://sqlite.org/src/dir/ext/icu)

### Caveats

This module replaces the SQLite [OS Interface](https://sqlite.org/vfs.html)
(aka VFS) with a [pure Go](vfs/) implementation.
This has benefits, but also comes with some drawbacks.

#### Write-Ahead Logging

Because WASM does not support shared memory,
[WAL](https://sqlite.org/wal.html) support is [limited](https://sqlite.org/wal.html#noshm).

To work around this limitation, SQLite is [patched](sqlite3/locking_mode.patch)
to always use `EXCLUSIVE` locking mode for WAL databases.

Because connection pooling is incompatible with `EXCLUSIVE` locking mode,
to use the [`database/sql`](https://pkg.go.dev/database/sql) driver
with WAL mode databases you should disable connection pooling by calling
[`db.SetMaxOpenConns(1)`](https://pkg.go.dev/database/sql#DB.SetMaxOpenConns).

#### File Locking

POSIX advisory locks, which SQLite uses on Unix, are
[broken by design](https://sqlite.org/src/artifact/2e8b12?ln=1073-1161).

On Linux, macOS and illumos, this module uses
[OFD locks](https://www.gnu.org/software/libc/manual/html_node/Open-File-Description-Locks.html)
to synchronize access to database files.
OFD locks are fully compatible with POSIX advisory locks.

On BSD Unixes, this module uses
[BSD locks](https://man.freebsd.org/cgi/man.cgi?query=flock&sektion=2).
On BSD Unixes, BSD locks are fully compatible with POSIX advisory locks.

On Windows, this module uses `LockFile`, `LockFileEx`, and `UnlockFile`,
like SQLite.

On all other platforms, file locking is not supported, and you must use
[`nolock=1`](https://sqlite.org/uri.html#urinolock)
to open database files.
To use the [`database/sql`](https://pkg.go.dev/database/sql) driver
with `nolock=1` you must disable connection pooling by calling
[`db.SetMaxOpenConns(1)`](https://pkg.go.dev/database/sql#DB.SetMaxOpenConns).

#### Testing

The pure Go VFS is tested by running SQLite's
[mptest](https://github.com/sqlite/sqlite/blob/master/mptest/mptest.c)
on Linux, macOS, Windows and FreeBSD.
Performance is tested by running
[speedtest1](https://github.com/sqlite/sqlite/blob/master/test/speedtest1.c).

### Alternatives

- [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite)
- [`crawshaw.io/sqlite`](https://pkg.go.dev/crawshaw.io/sqlite)
- [`github.com/mattn/go-sqlite3`](https://pkg.go.dev/github.com/mattn/go-sqlite3)
- [`github.com/zombiezen/go-sqlite`](https://pkg.go.dev/github.com/zombiezen/go-sqlite)
