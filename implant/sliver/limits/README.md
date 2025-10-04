# implant/sliver/limits

## Overview

Resource limitation utilities controlling implant behavior. Enforces concurrency caps, throttling, and watchdog timers. Runtime components handle limits darwin, limits generic, limits linux, and limits windows for implant-side limits features.

## Go Files

- `limits.go` – Defines core limit enforcement structures and logic.
- `limits_darwin.go` – macOS-specific limit adjustments or capabilities.
- `limits_generic.go` – Platform-neutral defaults and stubs for limits.
- `limits_linux.go` – Linux-specific resource limit handling.
- `limits_windows.go` – Windows implementations for managing implant limits.
