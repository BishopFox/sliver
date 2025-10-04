# implant/sliver/syscalls

## Overview

Direct syscall wrappers and supporting code for stealth execution. Exposes low-level syscall invocations and helper shims. Runtime components handle syscalls windows, types windows, and zsyscalls windows for implant-side syscalls features.

## Go Files

- `syscalls.go` – Declares syscall helper interfaces used across the implant.
- `syscalls_windows.go` – Implements Windows-specific syscall wrappers.
- `types_windows.go` – Windows type definitions required for syscall invocations.
- `zsyscalls_windows.go` – Generated syscall stubs for Windows builds.
