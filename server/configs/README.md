# server/configs

## Overview

Configuration loading and validation for the server. Parses config files, environment overrides, and applies defaults. Key routines cover crack, database, and HTTP C2 within the configs subsystem.

## Go Files

- `crack.go` – Defines configuration structures for cracking infrastructure.
- `database.go` – Loads and validates database connection settings.
- `http-c2.go` – Parses HTTP C2 configuration blocks.
- `http-c2_test.go` *(tests)* – Tests HTTP C2 configuration parsing logic.
- `server.go` – Top-level server configuration loading and defaults.
