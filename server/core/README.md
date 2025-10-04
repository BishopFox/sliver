# server/core

## Overview

Core server runtime coordination, state, and dispatch. Oversees lifecycle management, hub services, and shared state. Key routines cover clients, connnection, crackstations, and events within the core subsystem.

## Go Files

- `builders.go` – Tracks builder jobs and integration with the core scheduler.
- `clients.go` – Manages connected operators and their state.
- `connnection.go` – Handles active connection bookkeeping (typo in filename retained).
- `core.go` – Initializes core services and orchestrates shared state.
- `crackstations.go` – Maintains crackstation worker metadata and status.
- `events.go` – Broadcasts and records core events for subscribers.
- `hosts.go` – Tracks discovered hosts and associated metadata.
- `jobs.go` – Manages background jobs and status reporting.
- `pivots.go` – Coordinates pivot graphs and routing information.
- `sessions.go` – Manages implant sessions and lifecycle hooks.
- `socks.go` – Tracks SOCKS proxy instances managed by the server.
- `tunnels.go` – Maintains tunnel state and routing entries.

## Sub-packages

- `rtunnels/` – Reverse tunnel coordination within the server core. Handles tunnel registration, negotiation, and multiplexing.
