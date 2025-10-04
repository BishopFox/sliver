# server/db/wasmsqlite

## Overview

WebAssembly SQLite bridge for embedded database support. Adapts wasm-sqlite bindings for browser-compatible runtimes. Key routines cover ddlmod, migrator, and sqlite within the wasmsqlite subsystem.

## Go Files

- `ddlmod.go`
- `ddlmod_test.go` *(tests)*
- `errors.go`
- `migrator.go`
- `sqlite.go`
- `sqlite_test.go` *(tests)*
- `sqlite_version_test.go` *(tests)*
