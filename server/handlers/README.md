# server/handlers

## Overview

RPC and event handlers that respond to client and implant requests. Routes gRPC calls to business logic modules. Key routines cover beacons, data cache, pivot, and sessions within the handlers subsystem.

## Go Files

- `beacons.go` – Handles beacon-related RPC calls and state updates.
- `data_cache.go` – Caches handler data for reuse across requests.
- `handlers.go` – Registers handler functions and shared middleware.
- `pivot.go` – Processes pivot management RPCs.
- `sessions.go` – Handles session CRUD operations and telemetry routing.
- `tunnel_writer.go` – Sends tunnel responses back to clients.
