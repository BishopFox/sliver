# implant/sliver/netstat

## Overview

Network inspection helpers exposed to operators via implant commands. Retrieves interface statistics, socket tables, and routing data. Runtime components handle netstat darwin, netstat linux, netstat types darwin, and netstat windows for implant-side netstat features.

## Go Files

- `netstat.go` – Provides shared netstat command wrappers and data structures.
- `netstat_darwin.go` – Retrieves socket/interface data on macOS systems.
- `netstat_linux.go` – Implements Linux netstat collection logic.
- `netstat_types_darwin.go` – Defines Darwin-specific counter structures used in parsing.
- `netstat_windows.go` – Collects netstat data via Windows APIs.
- `types_darwin.go` – Additional Darwin type definitions supporting netstat parsing.
