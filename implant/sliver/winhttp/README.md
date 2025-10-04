# implant/sliver/winhttp

## Overview

Windows HTTP helpers and fallbacks for implant transports. Supplies WinHTTP-based dialers and configuration routines. Runtime components handle winhttp test for implant-side winhttp features.

## Go Files

- `winhttp.go` – Wraps WinHTTP APIs to provide HTTP transport support for implants.
- `winhttp_test.go` *(tests)* – Tests the WinHTTP integration and configuration helpers.
