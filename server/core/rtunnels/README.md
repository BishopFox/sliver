# server/core/rtunnels

## Overview

Reverse tunnel coordination within the server core. Handles tunnel registration,
server-owned reverse-port-forward authorization, bounded outbound connection
brokering, and multiplexing.

Reverse port forward destinations are immutable operator inputs stored in a
`Registry`. Implant messages carry an opaque `AuthorizationID`; only `Broker`
may turn that ID into an outbound TCP connection. Legacy implant addresses are
accepted solely as canonical registry lookup keys and are never dialed directly.

## Go Files

- `authorization.go` – Owns authorization lifecycle and server-authoritative metadata.
- `broker.go` – Opens bounded, revocation-aware connections from stored dial plans.
- `rtunnels.go` – Manages active reverse tunnel state and lifecycle cleanup.
