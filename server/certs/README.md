# server/certs

## Overview

Certificate generation and management helpers for server transports. Issues and rotates TLS material for listeners. Key routines cover acme, CA, HTTPS, and mTLS within the certs subsystem.

## Go Files

- `acme.go` – Integrates with ACME providers to issue certificates automatically.
- `ca.go` – Manages the internal certificate authority lifecycle.
- `certs.go` – High-level certificate manager used by transports.
- `certs_test.go` *(tests)* – Tests certificate issuance and rotation workflows.
- `https.go` – Provides HTTPS certificate helpers and caching.
- `mtls.go` – Generates and manages mTLS client certificates.
- `operators.go` – Issues operator-specific certificates and keys.
- `subject.go` – Builds X.509 subject information based on configuration.
- `tlskeys.go` – Handles TLS keypair generation and persistence.
- `wireguard.go` – Manages certificates/keys for WireGuard listeners when needed.
