# implant/sliver/hostuuid

## Overview

Host fingerprinting and UUID generation utilities for implants. Collects platform attributes and derives deterministic host identifiers. Runtime components handle UUID, UUID darwin, UUID generic, and UUID linux for implant-side hostuuid features.

## Go Files

- `uuid.go` – Provides shared host UUID generation logic and interface.
- `uuid_darwin.go` – Implements macOS-specific host identifiers.
- `uuid_generic.go` – Fallback UUID generation used when no platform overrides exist.
- `uuid_linux.go` – Linux-specific host UUID derivation routines.
- `uuid_windows.go` – Windows host UUID calculations leveraging system APIs.
