# implant/sliver/extension

## Overview

Extension host runtime allowing implants to load optional capabilities. Manages sandboxing, lifecycle, and communication with extension modules. Runtime components handle extension darwin, extension windows, memfs, and WASM for implant-side extension features.

## Go Files

- `extension.go`
- `extension_darwin.go`
- `extension_windows.go`
- `memfs.go`
- `memfs_test.go` *(tests)*
- `wasm.go`
- `wasm_generic.go`
