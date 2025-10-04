# client/command/wasm

## Overview

Implements the 'wasm' command group for the Sliver client console. Handlers map Cobra invocations to wasm workflows such as memfs.

## Go Files

- `commands.go` – Registers WebAssembly-related commands.
- `memfs.go` – Manages the WASM in-memory filesystem cache for implants.
- `wasm.go` – Lists installed WASM modules and orchestrates execution requests.
