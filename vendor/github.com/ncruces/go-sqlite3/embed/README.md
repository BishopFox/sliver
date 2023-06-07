# Embeddable WASM build of SQLite

This folder includes an embeddable WASM build of SQLite 3.42.0 for use with
[`github.com/ncruces/go-sqlite3`](https://pkg.go.dev/github.com/ncruces/go-sqlite3).

The following optional features are compiled in:
- [math functions](https://www.sqlite.org/lang_mathfunc.html)
- [FTS3/4](https://www.sqlite.org/fts3.html)/[5](https://www.sqlite.org/fts5.html)
- [JSON](https://www.sqlite.org/json1.html)
- [R*Tree](https://www.sqlite.org/rtree.html)
- [GeoPoly](https://www.sqlite.org/geopoly.html)
- [base64](https://github.com/sqlite/sqlite/blob/master/ext/misc/base64.c)
- [decimal](https://github.com/sqlite/sqlite/blob/master/ext/misc/decimal.c)
- [regexp](https://github.com/sqlite/sqlite/blob/master/ext/misc/regexp.c)
- [series](https://github.com/sqlite/sqlite/blob/master/ext/misc/series.c)
- [uint](https://github.com/sqlite/sqlite/blob/master/ext/misc/uint.c)
- [uuid](https://github.com/sqlite/sqlite/blob/master/ext/misc/uuid.c)
- [time](../sqlite3/time.c)

See the [configuration options](../sqlite3/sqlite_cfg.h),
and [patches](../sqlite3) applied.

Built using [`wasi-sdk`](https://github.com/WebAssembly/wasi-sdk),
and [`binaryen`](https://github.com/WebAssembly/binaryen).