# implant/sliver/handlers

## Overview

Runtime message handlers that react to server instructions. Dispatches inbound RPC messages to feature-specific executors. Runtime components handle extensions WASM, handlers wireguard, handlers darwin, and handlers generic for implant-side handlers features.

## Go Files

- `extensions-wasm.go` – Routes extension-related RPC messages to the WASM runtime.
- `handlers-wireguard.go` – Handles WireGuard control messages and state updates.
- `handlers.go` – Registers core handler functions and shared dispatch utilities.
- `handlers_darwin.go` – macOS-specific handler bindings and feature toggles.
- `handlers_generic.go` – Generic handler implementations used when no platform specialization is required.
- `handlers_linux.go` – Linux-focused handler logic and build tags.
- `handlers_windows.go` – Windows-specific handler implementations.
- `kill-handlers.go` – Processes kill commands for sessions and tasks in a platform-agnostic way.
- `kill-handlers_windows.go` – Windows-specific kill command handling.
- `pivot-handlers.go` – Manages pivot-related messages and tunnel setup instructions.
- `rpc-handlers-cgo.go` – CGO-enabled RPC handler variants for platforms that require it.
- `rpc-handlers-generic.go` – Generic RPC handler implementations shared across builds.
- `rpc-handlers.go` – Core RPC handler registration and dispatch loop.
- `rpc-handlers_darwin.go` – macOS-specific RPC handler tweaks.
- `rpc-handlers_linux.go` – Linux RPC handler customizations.
- `rpc-handlers_windows.go` – Windows RPC handler customizations.
- `tun-rportfwd.go` – Handles tunnel messages specific to reverse port forwarding channels.
- `tun.go` – Processes generic tunnel control messages and orchestrates link state.

## Sub-packages

- `matcher/` – Matcher logic for routing inbound messages to the right handlers. Builds lookup tables and rule evaluations for message dispatch.
- `tunnel_handlers/` – Specialized handlers for managing implant tunnel transports. Coordinates shell, SOCKS, port forwarding, and WASM tunnel flows.
