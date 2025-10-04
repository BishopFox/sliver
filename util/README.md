# util

## Overview

Shared utilities imported by server and implant code. Contains path helpers, resource identifiers, and crypto wrappers. Utilities focus on cryptography, files, generics, and implant within the util package.

## Go Files

- `cryptography.go`
- `files.go`
- `generics.go`
- `implant.go`
- `implant_test.go` *(tests)*
- `paths_generic.go`
- `paths_test.go` *(tests)*
- `paths_windows.go`
- `resource_ids.go`

## Sub-packages

- `encoders/` – Common encoder implementations shared across binaries. Offers shared registry wiring and helper encoders.
- `leaky/` – Leak detection helpers for debugging resource usage. Implements buffering utilities to capture leaked output.
- `minisign/` – Minisign signing and verification helpers. Wraps minisign key handling, signature generation, and validation.
