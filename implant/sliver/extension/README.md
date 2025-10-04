# implant/sliver/extension

## Overview

Extension host runtime allowing implants to load optional capabilities. Manages sandboxing, lifecycle, and communication with extension modules. Runtime components handle extension darwin, extension windows, memfs, and WASM for implant-side extension features.

## Go Files

- `extension.go` – Core extension host that loads modules and brokers RPC communication.
- `extension_darwin.go` – macOS-specific stubs and build tags for extension support.
- `extension_windows.go` – Windows-specific integration for loading and managing extensions.
- `memfs.go` – Implements the in-memory filesystem backing extension assets.
- `memfs_test.go` *(tests)* – Tests the in-memory filesystem behavior for extensions.
- `wasm.go` – Sets up the WASM runtime for implant extensions and handles module lifecycle.
- `wasm_generic.go` – Provides WASM runtime glue for non-platform-specific builds.
