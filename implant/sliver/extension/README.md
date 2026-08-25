# implant/sliver/extension

## Overview

Extension host runtime allowing implants to load optional capabilities. Manages native shared-library and WASM extension lifecycle, registration, and communication across supported implant platforms.

## Go Files

- `extension.go` – Thread-safe native extension registry and single-flight loading lifecycle.
- `extension_darwin.go` – macOS constructor for the shared Reflektor-backed Unix loader.
- `extension_linux.go` – Linux constructor for the shared Reflektor-backed Unix loader.
- `extension_unix.go` – CGO-free macOS/Linux loading, export invocation, and callback bridge.
- `extension_windows.go` – Windows-specific Reflektor integration for loading and invoking extensions.
- `memfs.go` – Implements the in-memory filesystem backing extension assets.
- `memfs_test.go` *(tests)* – Tests the in-memory filesystem behavior for extensions.
- `wasm.go` – Sets up the WASM runtime for implant extensions and handles module lifecycle.
- `wasm_generic.go` – Provides WASM runtime glue for non-platform-specific builds.
