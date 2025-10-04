# server/db

## Overview

Database access layer including migrations and helpers. Establishes database connections, adapters, and query utilities. Key routines cover SQL CGO, SQL GO, SQL WASM, and SQL within the db subsystem.

## Go Files

- `db.go` – Initializes database connections and manages migrations.
- `helpers.go` – Utility functions shared across the database layer.
- `logger.go` – Database logging adapter for structured output.
- `sql-cgo.go` – CGO-backed SQLite driver wiring.
- `sql-go.go` – Pure-Go SQLite driver wiring.
- `sql-wasm.go` – WebAssembly SQLite integration for embedded builds.
- `sql.go` – Common SQL helpers and abstractions.

## Sub-packages

- `models/` – Database models and ORM definitions for server state. Defines data schemas, relationships, and query helpers.
- `wasmsqlite/` – WebAssembly SQLite bridge for embedded database support. Adapts wasm-sqlite bindings for browser-compatible runtimes.
