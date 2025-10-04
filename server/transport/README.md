# server/transport

## Overview

Server-side transports and listener orchestration. Coordinates C2 listener lifecycles and connection routing. Key routines cover local, middleware, mTLS, and tailscale within the transport subsystem.

## Go Files

- `local.go` – Implements local transport listeners and helpers.
- `middleware.go` – Shared middleware for transport handler pipelines.
- `mtls.go` – Manages mTLS server listener setup.
- `tailscale.go` – Integrates Tailscale transport support for Sliver.
