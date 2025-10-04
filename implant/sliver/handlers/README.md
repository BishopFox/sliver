# implant/sliver/handlers

## Overview

Runtime message handlers that react to server instructions. Dispatches inbound RPC messages to feature-specific executors. Runtime components handle extensions WASM, handlers wireguard, handlers darwin, and handlers generic for implant-side handlers features.

## Go Files

- `extensions-wasm.go`
- `handlers-wireguard.go`
- `handlers.go`
- `handlers_darwin.go`
- `handlers_generic.go`
- `handlers_linux.go`
- `handlers_windows.go`
- `kill-handlers.go`
- `kill-handlers_windows.go`
- `pivot-handlers.go`
- `rpc-handlers-cgo.go`
- `rpc-handlers-generic.go`
- `rpc-handlers.go`
- `rpc-handlers_darwin.go`
- `rpc-handlers_linux.go`
- `rpc-handlers_windows.go`
- `tun-rportfwd.go`
- `tun.go`

## Sub-packages

- `matcher/` – Matcher logic for routing inbound messages to the right handlers. Builds lookup tables and rule evaluations for message dispatch.
- `tunnel_handlers/` – Specialized handlers for managing implant tunnel transports. Coordinates shell, SOCKS, port forwarding, and WASM tunnel flows.
