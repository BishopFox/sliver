# implant/sliver/handlers/tunnel_handlers

## Overview

Specialized handlers for managing implant tunnel transports. Coordinates shell, SOCKS, port forwarding, and WASM tunnel flows. Runtime components handle close handler, data cache, data handler, and portfwd handler for implant-side tunnel_handlers features.

## Go Files

- `close_handler.go` – Handles tunnel close events and resource cleanup.
- `data_cache.go` – Buffers tunnel payloads to support resend and flow control.
- `data_handler.go` – Processes generic tunnel data frames and dispatches them to consumers.
- `portfwd_handler.go` – Deals with tunnel messages associated with port forwarding channels.
- `shell_handler.go` – Translates tunnel frames into shell IO for interactive sessions.
- `socks_handler.go` – Manages SOCKS tunnel traffic and connection tracking.
- `tunnel_writer.go` – Provides helper routines for writing frames back to the server.
- `utils.go` – Contains shared utilities used by the tunnel handlers.
- `wasm_handler.go` – Routes tunnel traffic destined for WASM-based extensions.
