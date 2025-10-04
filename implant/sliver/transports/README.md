# implant/sliver/transports

## Overview

Outbound transport implementations compiled into the implant. Registers available transport clients and shared plumbing. Runtime components handle beacon, connection, session, and transports generic for implant-side transports features.

## Go Files

- `beacon.go` – Implements the beacon scheduler and transport selection logic.
- `connection.go` – Wraps low-level connection metadata and lifecycle helpers.
- `session.go` – Manages session state associated with active transports.
- `transports.go` – Registers transport implementations and exposes factory functions.
- `transports_generic.go` – Platform-neutral transport helpers and defaults.
- `transports_windows.go` – Windows-specific transport adjustments and APIs.
- `tunnel.go` – Provides transport tunnel abstractions and helpers.

## Sub-packages

- `dnsclient/` – DNS-based transport client used for low-and-slow comms. Encodes tasking into DNS queries and parses responses.
- `httpclient/` – HTTP(S) client transport implementation for implants. Configures request schedulers, retry logic, and header shaping.
- `mtls/` – Mutual TLS transport client used by implants. Handles certificate selection, session reuse, and secure dialing.
- `pivotclients/` – Transport clients used when implants dial through pivot chains. Implements nested client behaviors and tunnel negotiation.
- `wireguard/` – WireGuard transport implementation for implants. Manages keypairs, peer configuration, and encrypted sessions.
