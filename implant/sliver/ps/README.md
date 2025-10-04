# implant/sliver/ps

## Overview

Process enumeration utilities shipped with the implant. Collects process metadata and formats listings for operators. Runtime components handle PS darwin, PS linux, PS types darwin, and PS windows for implant-side ps features.

## Go Files

- `ps.go` – Provides shared process listing logic and structures.
- `ps_darwin.go` – Implements process enumeration on macOS.
- `ps_linux.go` – Implements process enumeration on Linux.
- `ps_types_darwin.go` – Defines Darwin-specific structs for process metadata.
- `ps_windows.go` – Implements process enumeration using Windows APIs.
- `types_darwin.go` – Additional Darwin type definitions supporting process parsing.
