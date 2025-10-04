# implant/sliver/forwarder

## Overview

Network forwarding helpers that move data through pivot tunnels. Provides connectors, relays, and buffering for implant forwarding. Runtime components handle portforward and socks for implant-side forwarder features.

## Go Files

- `forwarder.go` – Coordinates forwarder registration and shared state across port-forward and SOCKS relays.
- `portforward.go` – Implements implant-side port forwarding logic and connection handlers.
- `socks.go` – Provides SOCKS proxy server functionality within the implant.
