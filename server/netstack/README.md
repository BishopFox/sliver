# server/netstack

## Overview

gVisor-based userland network stack for server transports. Provides packet buffers, endpoint management, and adapters. Key routines cover TUN within the netstack subsystem.

## Go Files

- `tun.go` – Integrates the gVisor TUN stack for server-managed transport endpoints.
