# implant/sliver/extension

## Overview

Extension host runtime allowing implants to load optional capabilities. Manages sandboxing, lifecycle, and communication with extension modules. Runtime components handle extension darwin, extension windows, memfs, and WASM for implant-side extension features.

## Go Files

- `extension.go` – Core extension host that loads modules and brokers RPC communication.
- `extension_darwin.go` – macOS-specific stubs and build tags for extension support.
- `extension_windows.go` – Windows-specific integration for loading and managing extensions.
- `memfs.go` – Implements the writable in-memory filesystem backing extension assets and the unchanged host filesystem pass-through.
- `memfs*_test.go` *(tests)* – Exercise legacy compatibility, mutation, concurrency, resource limits, guest integration, and fuzz-seed behavior.
- `wasm.go` – Sets up the WASM runtime for implant extensions and handles module lifecycle.
- `memfs_wasi_integration_test.go` *(test)* – Runs a Go/WASI guest through create, read, write, append, rename, and removal operations.
- `wasm_network_integration_test.go` *(tests)* – Exercises the shared networking host with the wrapper-built HTTP example.
- `wasm_generic.go` – Provides WASM runtime glue for non-platform-specific builds.

## Go WASI networking

The Go standard library does not currently provide outbound networking on
`wasip1`. Sliver's bundled toolchain therefore installs
`$SLIVER_ROOT_DIR/go/bin/sliver-wasm-go`, a standalone wrapper that applies the
matching standard-library overlay and sets `GOOS=wasip1`, `GOARCH=wasm`, and
`CGO_ENABLED=0`.

Modules built by this wrapper import `sliver_wasi_net_v1` and must run in the
Sliver extension or traffic-encoder runtime. Both instantiate the shared host
implemented by `implant/sliver/wasmnet`, so TCP, UDP, DNS, deadlines, and
cancellation have the same behavior on the implant and server.

See `testdata/http-fetch` for an HTTPS extension example with an embedded
Mozilla public CA bundle. Traffic encoders use the same network imports but
must instead be built as WASI reactors with `-buildmode=c-shared` and export
the traffic-encoder ABI. The wrapper intentionally rejects `go run` because
the ordinary Go WASI runner does not provide Sliver's networking host.

## Go WASI memory filesystem

Extension modules see a persistent, concurrency-safe filesystem at `/memfs`.
The default is writable and supports regular and positioned I/O, sparse
writes, truncation, directories, metadata, rename and removal, hard links,
and relative symbolic links. State is shared by successive executions of the
same `WasmExtension`. Non-`/memfs` paths still use the historical read-only
host-root pass-through and are not made writable by this implementation.

`NewWasmExtension` remains signature-compatible; its `/memfs` default is now
writable. Hosts that require the old strict read-only behavior can use
`NewWasmExtensionWithOptions(..., WithReadOnlyMemFS())`. Code that needs the
filesystem independently can use `NewWasmMemoryFS`, optionally with
`WithWasmMemoryFSReadOnly()`. Read-only selection is currently a programmatic
host API; the existing console and RPC construction path keeps the writable
default. Initial maps and file contents are copied so guest mutations never
alias RPC input buffers. File modes are retained as metadata, but the virtual
filesystem does not model users or enforce Unix identity-based permissions.
The filesystem caps data at 256 MiB, directory entries at 65,536, and open
handles at 4,096 so sparse files and metadata cannot grow host memory without
bound.

See `testdata/memfs-rw` for a standalone Go/WASI module that exercises the
writable and read-only modes.
