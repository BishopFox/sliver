# Embeddable Wasm build of SQLite

This folder includes an embeddable Wasm build of SQLite 3.45.3 for use with
[`github.com/ncruces/go-sqlite3`](https://pkg.go.dev/github.com/ncruces/go-sqlite3).

The following optional features are compiled in:
- [math functions](https://sqlite.org/lang_mathfunc.html)
- [FTS5](https://sqlite.org/fts5.html)
- [JSON](https://sqlite.org/json1.html)
- [R*Tree](https://sqlite.org/rtree.html)
- [GeoPoly](https://sqlite.org/geopoly.html)
- [soundex](https://sqlite.org/lang_corefunc.html#soundex)
- [base64](https://github.com/sqlite/sqlite/blob/master/ext/misc/base64.c)
- [decimal](https://github.com/sqlite/sqlite/blob/master/ext/misc/decimal.c)
- [ieee754](https://github.com/sqlite/sqlite/blob/master/ext/misc/ieee754.c)
- [regexp](https://github.com/sqlite/sqlite/blob/master/ext/misc/regexp.c)
- [series](https://github.com/sqlite/sqlite/blob/master/ext/misc/series.c)
- [uint](https://github.com/sqlite/sqlite/blob/master/ext/misc/uint.c)
- [uuid](https://github.com/sqlite/sqlite/blob/master/ext/misc/uuid.c)
- [time](../sqlite3/time.c)

See the [configuration options](../sqlite3/sqlite_cfg.h),
and [patches](../sqlite3) applied.

Built using [`wasi-sdk`](https://github.com/WebAssembly/wasi-sdk),
and [`binaryen`](https://github.com/WebAssembly/binaryen).