# implant/sliver/transports/wireguard

## Overview

WireGuard transport implementation for implants. Manages keypairs, peer configuration, and encrypted sessions. Runtime components handle wireguard generic for implant-side wireguard features.

## Go Files

- `wireguard.go` – Implements the WireGuard transport, including key management and session loops.
- `wireguard_generic.go` – Platform-neutral pieces shared across WireGuard builds.
