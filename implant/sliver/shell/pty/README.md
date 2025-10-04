# implant/sliver/shell/pty

## Overview

Pseudo-terminal management for implant shells. Handles PTY allocation, resizing, and IO forwarding. Runtime components handle ioctl, ioctl BSD, ioctl darwin, and PTY darwin for implant-side pty features.

## Go Files

- `doc.go`
- `ioctl.go`
- `ioctl_bsd.go`
- `ioctl_darwin.go`
- `pty_darwin.go`
- `pty_dragonfly.go`
- `pty_freebsd.go`
- `pty_linux.go`
- `pty_openbsd.go`
- `pty_unsupported.go`
- `run.go`
- `types.go`
- `types_dragonfly.go`
- `types_freebsd.go`
- `types_openbsd.go`
- `util.go`
- `ztypes_386.go`
- `ztypes_amd64.go`
- `ztypes_arm.go`
- `ztypes_arm64.go`
- `ztypes_dragonfly_amd64.go`
- `ztypes_freebsd_386.go`
- `ztypes_freebsd_amd64.go`
- `ztypes_freebsd_arm.go`
- `ztypes_mipsx.go`
- `ztypes_openbsd_386.go`
- `ztypes_openbsd_amd64.go`
- `ztypes_ppc64.go`
- `ztypes_ppc64le.go`
- `ztypes_s390x.go`
