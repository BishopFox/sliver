# implant/sliver/shell/pty

## Overview

Pseudo-terminal management for implant shells. Handles PTY allocation, resizing, and IO forwarding. Runtime components handle ioctl, ioctl BSD, ioctl darwin, and PTY darwin for implant-side pty features.

## Go Files

- `doc.go` – Package documentation for the PTY helpers.
- `ioctl.go` – Defines portable ioctl wrappers used across platforms.
- `ioctl_bsd.go` – BSD-specific ioctl constants and helpers.
- `ioctl_darwin.go` – macOS ioctl definitions for PTY management.
- `pty_darwin.go` – macOS PTY open/resizing implementation.
- `pty_dragonfly.go` – DragonFly BSD PTY handling.
- `pty_freebsd.go` – FreeBSD PTY creation helpers.
- `pty_linux.go` – Linux PTY allocation and control logic.
- `pty_openbsd.go` – OpenBSD PTY support routines.
- `pty_unsupported.go` – Stub implementations for unsupported platforms.
- `run.go` – Launches commands within PTYs and manages IO loops.
- `types.go` – Shared struct definitions for PTY/ioctl operations.
- `types_dragonfly.go` – DragonFly-specific type definitions.
- `types_freebsd.go` – FreeBSD-specific type definitions.
- `types_openbsd.go` – OpenBSD-specific type definitions.
- `util.go` – Miscellaneous PTY helper functions.
- `ztypes_386.go` – Generated syscall struct definitions for 386 builds.
- `ztypes_amd64.go` – Generated syscall struct definitions for amd64 builds.
- `ztypes_arm.go` – Generated syscall struct definitions for ARM builds.
- `ztypes_arm64.go` – Generated syscall struct definitions for ARM64 builds.
- `ztypes_dragonfly_amd64.go` – Generated DragonFly amd64 syscall types.
- `ztypes_freebsd_386.go` – Generated FreeBSD 386 syscall types.
- `ztypes_freebsd_amd64.go` – Generated FreeBSD amd64 syscall types.
- `ztypes_freebsd_arm.go` – Generated FreeBSD ARM syscall types.
- `ztypes_mipsx.go` – Generated MIPS family syscall types.
- `ztypes_openbsd_386.go` – Generated OpenBSD 386 syscall types.
- `ztypes_openbsd_amd64.go` – Generated OpenBSD amd64 syscall types.
- `ztypes_ppc64.go` – Generated PowerPC64 syscall types.
- `ztypes_ppc64le.go` – Generated PowerPC64 LE syscall types.
- `ztypes_s390x.go` – Generated s390x syscall types.
