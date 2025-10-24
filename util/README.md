# util

## Overview

Shared utilities imported by server and implant code. Contains path helpers, resource identifiers, and crypto wrappers. Utilities focus on cryptography, files, generics, and implant within the util package.

## Go Files

- `cryptography.go` – Provides shared cryptographic helpers and constants.
- `files.go` – File handling utilities for copying, hashing, and temp files.
- `generics.go` – Miscellaneous generic helper functions used across packages.
- `implant.go` – Utilities for working with implant metadata and artifacts.
- `implant_test.go` *(tests)* – Tests implant utility helpers.
- `paths_generic.go` – Platform-neutral path helpers and directories.
- `paths_test.go` *(tests)* – Tests path helper behavior.
- `paths_windows.go` – Windows-specific path utilities.
- `resource_ids.go` – Generates deterministic resource identifiers.

## Sub-packages

- `encoders/` – Common encoder implementations shared across binaries. Offers shared registry wiring and helper encoders.
- `leaky/` – Leak detection helpers for debugging resource usage. Implements buffering utilities to capture leaked output.
- `minisign/` – Minisign signing and verification helpers. Wraps minisign key handling, signature generation, and validation.
