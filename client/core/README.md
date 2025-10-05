# client/core

## Overview

Maintains client application state, background workers, and RPC coordination. Orchestrates session tracking, event streams, and health checks. Core logic addresses BOF, curses, portfwd, and reactions within the core package.

## Go Files

- `bof.go` – Defines Beacon Object File argument buffers and helpers for executing BOF modules.
- `curses.go` – Manages cursed browser sessions, caching processes and coordinating lifecycle events.
- `portfwd.go` – Tracks active port forwards and exposes shared state for command packages.
- `reactions.go` – Stores reaction definitions and evaluates triggers when new events arrive.
- `socks.go` – Maintains SOCKS proxy metadata and orchestrates background goroutines.
- `tunnel.go` – Represents individual tunnels and their configuration details.
- `tunnel_io.go` – Implements IO handling and relay loops for tunnel data streams.
- `tunnels.go` – Coordinates tunnel creation, lookup, and teardown across the client.
