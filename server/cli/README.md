# server/cli

## Overview

CLI entrypoint for running the server daemon. Parses flags, configures logging, and launches the service runtime. Key routines cover certs, daemon, operator, and unpack within the cli subsystem.

## Go Files

- `builder.go` – CLI commands for server-side payload building.
- `certs.go` – CLI actions for certificate management.
- `cli.go` – Root command setup and flag parsing for the server CLI.
- `daemon.go` – Commands to run the server daemon and control lifecycle.
- `operator.go` – Operator management commands (create/list/etc.).
- `unpack.go` – Extracts embedded assets from the server binary.
- `version.go` – Prints server build/version information.
