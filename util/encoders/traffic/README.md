# util/encoders/traffic

## Overview

Traffic encoder runtime used by the server. It mirrors the implant runtime and
selects wazero's native compiler or portable interpreter automatically.

## Go Files

- `create.go` - Selects wazero's native compiler when supported and its portable interpreter otherwise.
- `runtime.go` - Instantiates WASI, the legacy encoder host, and the shared `sliver_wasi_net_v1` networking host.
- `testers.go` - Provides helpers for testing encoder scripts.
- `traffic-encoder.go` - High-level API for applying traffic encoders.

## Go WASI traffic encoders

The server and implant runtimes use the same networking ABI from
`implant/sliver/wasmnet`. A Go traffic encoder can therefore use standard
library TCP, UDP, DNS, and HTTP calls when built with Sliver's wrapper:

```text
$SLIVER_ROOT_DIR/go/bin/sliver-wasm-go build -buildmode=c-shared -o encoder.wasm .
```

The module must export `malloc(i32) -> i32`, `free(i32, i32)`,
`encode(i32, i32) -> i64`, and `decode(i32, i32) -> i64`. Go's
`//go:wasmexport` directive can provide these exports. The `i64` transform
result packs the output pointer into the high 32 bits and its size into the
low 32 bits. The host copies and frees returned buffers before releasing the
runtime lock. Legacy `malloc` exports using an `i64` parameter or result remain
accepted when their values fit in 32-bit Wasm linear memory.
