# Go bindings to SQLite using Wazero

[![Go Reference](https://pkg.go.dev/badge/image)](https://pkg.go.dev/github.com/ncruces/go-sqlite3)
[![Go Report](https://goreportcard.com/badge/github.com/ncruces/go-sqlite3)](https://goreportcard.com/report/github.com/ncruces/go-sqlite3)
[![Go Coverage](https://github.com/ncruces/go-sqlite3/wiki/coverage.svg)](https://github.com/ncruces/go-sqlite3/wiki/Test-coverage-report)

Go module `github.com/ncruces/go-sqlite3` is a `cgo`-free [SQLite](https://sqlite.org/) wrapper.\
It provides a [`database/sql`](https://pkg.go.dev/database/sql) compatible driver,
as well as direct access to most of the [C SQLite API](https://sqlite.org/cintro.html).

It wraps a [Wasm](https://webassembly.org/) [build](embed/) of SQLite,
and uses [wazero](https://wazero.io/) as the runtime.\
Go, wazero and [`x/sys`](https://pkg.go.dev/golang.org/x/sys) are the _only_ runtime dependencies [^1].

### Getting started

Using the [`database/sql`](https://pkg.go.dev/database/sql) driver:
```go

import "database/sql"
import _ "github.com/ncruces/go-sqlite3/driver"
import _ "github.com/ncruces/go-sqlite3/embed"

var version string
db, _ := sql.Open("sqlite3", "file:demo.db")
db.QueryRow(`SELECT sqlite_version()`).Scan(&version)
```

### Packages

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

### Extensions

- [`github.com/ncruces/go-sqlite3/ext/array`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/array)
  provides the [`array`](https://sqlite.org/carray.html) table-valued function.
- [`github.com/ncruces/go-sqlite3/ext/blobio`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/blobio)
  simplifies [incremental BLOB I/O](https://sqlite.org/c3ref/blob_open.html).
- [`github.com/ncruces/go-sqlite3/ext/bloom`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/bloom)
  provides a [Bloom filter](https://github.com/nalgeon/sqlean/issues/27#issuecomment-1002267134) virtual table.
- [`github.com/ncruces/go-sqlite3/ext/csv`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/csv)
  reads [comma-separated values](https://sqlite.org/csv.html).
- [`github.com/ncruces/go-sqlite3/ext/fileio`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/fileio)
  reads, writes and lists files.
- [`github.com/ncruces/go-sqlite3/ext/hash`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/hash)
  provides cryptographic hash functions.
- [`github.com/ncruces/go-sqlite3/ext/lines`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/lines)
  reads data [line-by-line](https://github.com/asg017/sqlite-lines).
- [`github.com/ncruces/go-sqlite3/ext/pivot`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/pivot)
  creates [pivot tables](https://github.com/jakethaw/pivot_vtab).
- [`github.com/ncruces/go-sqlite3/ext/regexp`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/regexp)
  provides regular expression functions.
- [`github.com/ncruces/go-sqlite3/ext/statement`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/statement)
  creates [parameterized views](https://github.com/0x09/sqlite-statement-vtab).
- [`github.com/ncruces/go-sqlite3/ext/stats`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/stats)
  provides [statistics](https://www.oreilly.com/library/view/sql-in-a/9780596155322/ch04s02.html) functions.
- [`github.com/ncruces/go-sqlite3/ext/unicode`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/unicode)
  provides [Unicode aware](https://sqlite.org/src/dir/ext/icu) functions.
- [`github.com/ncruces/go-sqlite3/ext/uuid`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/uuid)
  generates [UUIDs](https://en.wikipedia.org/wiki/Universally_unique_identifier).
- [`github.com/ncruces/go-sqlite3/ext/zorder`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/ext/zorder)
  maps multidimensional data to one dimension.
- [`github.com/ncruces/go-sqlite3/vfs/adiantum`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/adiantum)
  wraps a VFS to offer encryption at rest.
- [`github.com/ncruces/go-sqlite3/vfs/memdb`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/memdb)
  implements an in-memory VFS.
- [`github.com/ncruces/go-sqlite3/vfs/readervfs`](https://pkg.go.dev/github.com/ncruces/go-sqlite3/vfs/readervfs)
  implements a VFS for immutable databases.

### Advanced features

- [incremental BLOB I/O](https://sqlite.org/c3ref/blob_open.html)
- [nested transactions](https://sqlite.org/lang_savepoint.html)
- [custom functions](https://sqlite.org/c3ref/create_function.html)
- [virtual tables](https://sqlite.org/vtab.html)
- [custom VFSes](https://sqlite.org/vfs.html)
- [online backup](https://sqlite.org/backup.html)
- [JSON support](https://sqlite.org/json1.html)
- [math functions](https://sqlite.org/lang_mathfunc.html)
- [full-text search](https://sqlite.org/fts5.html)
- [geospatial search](https://sqlite.org/geopoly.html)
- [encryption at rest](vfs/adiantum/README.md)
- [and more…](embed/README.md)

### Caveats

This module replaces the SQLite [OS Interface](https://sqlite.org/vfs.html)
(aka VFS) with a [pure Go](vfs/) implementation,
which has advantages and disadvantages.

Read more about the Go VFS design [here](vfs/README.md).

### Testing

This project aims for [high test coverage](https://github.com/ncruces/go-sqlite3/wiki/Test-coverage-report).
It also benefits greatly from [SQLite's](https://sqlite.org/testing.html) and
[wazero's](https://tetrate.io/blog/introducing-wazero-from-tetrate/#:~:text=Rock%2Dsolid%20test%20approach) thorough testing.

Every commit is [tested](https://github.com/ncruces/go-sqlite3/wiki/Test-matrix) on
Linux (amd64/arm64/386/riscv64/s390x), macOS (amd64/arm64),
Windows (amd64), FreeBSD (amd64), OpenBSD (amd64), NetBSD (amd64),
illumos (amd64), and Solaris (amd64).

The Go VFS is tested by running SQLite's
[mptest](https://github.com/sqlite/sqlite/blob/master/mptest/mptest.c).

### Performance

Perfomance of the [`database/sql`](https://pkg.go.dev/database/sql) driver is
[competitive](https://github.com/cvilsmeier/go-sqlite-bench) with alternatives.

The Wasm and VFS layers are also tested by running SQLite's
[speedtest1](https://github.com/sqlite/sqlite/blob/master/test/speedtest1.c).

### Alternatives

- [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite)
- [`crawshaw.io/sqlite`](https://pkg.go.dev/crawshaw.io/sqlite)
- [`github.com/mattn/go-sqlite3`](https://pkg.go.dev/github.com/mattn/go-sqlite3)
- [`github.com/zombiezen/go-sqlite`](https://pkg.go.dev/github.com/zombiezen/go-sqlite)

[^1]: anything else you find in `go.mod` is either a test dependency,
      or needed by one of the extensions.
