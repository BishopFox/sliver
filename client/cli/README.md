# client/cli

## Overview

Defines the Cobra root command and CLI initialization for the Sliver client binary. Sets up persistent flags, environment bootstrapping, and command wiring. Core logic addresses console, implant, import, and version within the cli package.

## Go Files

- `cli.go` – Builds the root Cobra command, wires global flags, and starts the console UI.
- `config.go` – Parses CLI configuration files and environment variables used during startup.
- `console.go` – Launches the interactive console mode and handles profile selection.
- `implant.go` – Implements CLI entry points for implant-specific operations without launching the console.
- `import.go` – Provides import routines for bringing external state into the client (e.g., implants or loot).
- `version.go` – Prints version/build information for the CLI binary.
