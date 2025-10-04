# implant/sliver/mount

## Overview

Cross-platform file mount helpers used by implants. Handles mounting, unmounting, and privilege-aware filesystem access. Runtime components handle mount linux and mount windows for implant-side mount features.

## Go Files

- `mount_linux.go` – Implements mount command wrappers and helpers for Linux targets.
- `mount_windows.go` – Provides Windows-specific mount helpers and drive management.
