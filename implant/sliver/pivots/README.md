# implant/sliver/pivots

## Overview

Pivot channel management for lateral movement through implants. Tracks upstream/downstream links and maintains tunnel state. Runtime components handle named pipe, named pipe windows, pivots generic, and pivots windows for implant-side pivots features.

## Go Files

- `named-pipe.go` – Provides common named pipe pivot helpers shared across platforms.
- `named-pipe_windows.go` – Implements Windows-specific named pipe pivot setup.
- `pivots.go` – Core pivot manager tracking active connections and state.
- `pivots_generic.go` – Platform-neutral pivot functionality and stubs.
- `pivots_windows.go` – Windows-specific pivot routines and APIs.
- `tcp.go` – Implements TCP-based pivot connectors used by the implant.
