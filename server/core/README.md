# server/core

## Overview

Core server runtime coordination, state, and dispatch. Oversees lifecycle management, hub services, and shared state. Key routines cover clients, connnection, crackstations, and events within the core subsystem.

## Go Files

- `builders.go`
- `clients.go`
- `connnection.go`
- `core.go`
- `crackstations.go`
- `events.go`
- `hosts.go`
- `jobs.go`
- `pivots.go`
- `sessions.go`
- `socks.go`
- `tunnels.go`

## Sub-packages

- `rtunnels/` – Reverse tunnel coordination within the server core. Handles tunnel registration, negotiation, and multiplexing.
