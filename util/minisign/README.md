# util/minisign

## Overview

Minisign signing and verification helpers. Wraps minisign key handling, signature generation, and validation. Utilities focus on private, public, and signature within the minisign package.

## Go Files

- `minisign.go` – High-level API for signing and verifying data using Minisign.
- `minisign_test.go` *(tests)* – Tests Minisign helper workflows.
- `private.go` – Private key parsing and serialization helpers.
- `public.go` – Public key handling utilities.
- `public_test.go` *(tests)* – Tests public key helpers.
- `rawsig_test.go` *(tests)* – Tests raw signature parsing/formatting.
- `signature.go` – Signature structure definitions and operations.
- `signature_test.go` *(tests)* – Tests signature encoding/decoding logic.
