# client/command/alias

## Overview

Implements the 'alias' command group for the Sliver client console. Handlers map Cobra invocations to alias workflows such as install, load, and remove.

## Go Files

- `alias.go` – Implements runtime alias management, printing installed aliases and exposing tab-completion helpers.
- `alias_test.go` *(tests)* – Exercises manifest parsing logic to ensure alias metadata is interpreted correctly.
- `commands.go` – Wires Cobra commands for listing, installing, loading, and removing aliases into the client console.
- `install.go` – Handles installing alias bundles from directories or archives, writing manifests and payload artifacts to disk.
- `load.go` – Loads alias manifests, validates target compatibility, and registers commands against the active console.
- `remove.go` – Removes installed aliases, prompting for confirmation and deleting associated on-disk assets.
