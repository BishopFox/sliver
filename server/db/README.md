# server/db

## Overview

Database access layer including migrations and helpers. Establishes database connections, adapters, and query utilities. Key routines cover SQL CGO, SQL GO, SQL WASM, and SQL within the db subsystem.

## Go Files

- `db.go`
- `helpers.go`
- `logger.go`
- `sql-cgo.go`
- `sql-go.go`
- `sql-wasm.go`
- `sql.go`

## Sub-packages

- `models/` – Database models and ORM definitions for server state. Defines data schemas, relationships, and query helpers.
- `wasmsqlite/` – WebAssembly SQLite bridge for embedded database support. Adapts wasm-sqlite bindings for browser-compatible runtimes.
