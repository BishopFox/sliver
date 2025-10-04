# implant/sliver/handlers/tunnel_handlers

## Overview

Specialized handlers for managing implant tunnel transports. Coordinates shell, SOCKS, port forwarding, and WASM tunnel flows. Runtime components handle close handler, data cache, data handler, and portfwd handler for implant-side tunnel_handlers features.

## Go Files

- `close_handler.go`
- `data_cache.go`
- `data_handler.go`
- `portfwd_handler.go`
- `shell_handler.go`
- `socks_handler.go`
- `tunnel_writer.go`
- `utils.go`
- `wasm_handler.go`
