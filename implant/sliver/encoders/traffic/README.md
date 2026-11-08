# implant/sliver/encoders/traffic

## Overview

Programmable traffic encoders compiled into the implant for obfuscation.

## Go Files

- `create.go` - Selects wazero's native compiler when supported and its portable interpreter otherwise.
- `runtime.go` - Instantiates WASI, the legacy encoder host, and the shared `sliver_wasi_net_v1` networking host.
- `traffic-encoder.go` - Provides high-level APIs to apply compiled traffic encoders to payloads.

The implant runtime intentionally mirrors `util/encoders/traffic`: the same
Wasm artifact and the same `sliver_wasi_net_v1` imports run on both sides of
an HTTP C2 connection. See the server-side traffic runtime README for the Go
reactor build command and exported function ABI.
