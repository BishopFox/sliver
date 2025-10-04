# server/db/wasmsqlite

## Overview

WebAssembly SQLite bridge for embedded database support. Adapts wasm-sqlite bindings for browser-compatible runtimes. Key routines cover ddlmod, migrator, and sqlite within the wasmsqlite subsystem.

## Go Files

- `ddlmod.go` – Implements dynamic DDL module support for wasm-sqlite.
- `ddlmod_test.go` *(tests)* – Tests DDL module behavior in WASM.
- `errors.go` – Defines error helpers for wasm-sqlite operations.
- `migrator.go` – Applies database migrations inside the WASM environment.
- `sqlite.go` – Wraps the wasm-sqlite engine with Go-friendly APIs.
- `sqlite_test.go` *(tests)* – Validates core wasm-sqlite functionality.
- `sqlite_version_test.go` *(tests)* – Ensures sqlite version detection works in WASM builds.
