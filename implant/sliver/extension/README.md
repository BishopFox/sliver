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
