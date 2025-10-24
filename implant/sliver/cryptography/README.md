# implant/sliver/cryptography

## Overview

Cryptographic primitives and wrappers used by the implant for secure operations. Offers key handling, signing, and envelope encryption helpers. Runtime components handle crypto, implant, minisign, and tlskeys for implant-side cryptography features.

## Go Files

- `crypto.go` – Wraps core encryption/signing helpers and exposes high-level APIs to other implant packages.
- `implant.go` – Manages implant keypairs, including generation, storage, and rotation helpers.
- `minisign.go` – Handles Minisign signature verification for bundled artifacts and updates.
- `tlskeys.go` – Generates and caches TLS key material used by outbound transports.
