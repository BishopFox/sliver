# implant/sliver/version

## Overview

Compile-time version metadata baked into implants. Publishes semantic version info, build IDs, and compatibility helpers. Runtime components handle version darwin, version generic, version linux, and version windows for implant-side version features.

## Go Files

- `version.go` – Stores shared implant version metadata getters.
- `version_darwin.go` – macOS build metadata overrides.
- `version_generic.go` – Default version metadata used across platforms.
- `version_linux.go` – Linux-specific build metadata adjustments.
- `version_windows.go` – Windows-specific build metadata adjustments.
