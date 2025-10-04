# implant/sliver/transports

## Overview

Outbound transport implementations compiled into the implant. Registers available transport clients and shared plumbing. Runtime components handle beacon, connection, session, and transports generic for implant-side transports features.

## Go Files

- `beacon.go`
- `connection.go`
- `session.go`
- `transports.go`
- `transports_generic.go`
- `transports_windows.go`
- `tunnel.go`

## Sub-packages

- `dnsclient/` – DNS-based transport client used for low-and-slow comms. Encodes tasking into DNS queries and parses responses.
- `httpclient/` – HTTP(S) client transport implementation for implants. Configures request schedulers, retry logic, and header shaping.
- `mtls/` – Mutual TLS transport client used by implants. Handles certificate selection, session reuse, and secure dialing.
- `pivotclients/` – Transport clients used when implants dial through pivot chains. Implements nested client behaviors and tunnel negotiation.
- `wireguard/` – WireGuard transport implementation for implants. Manages keypairs, peer configuration, and encrypted sessions.
