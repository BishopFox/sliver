# implant/sliver/transports/httpclient

## Overview

HTTP(S) client transport implementation for implants. Configures request schedulers, retry logic, and header shaping. Runtime components handle drivers generic, drivers windows, and gohttp for implant-side httpclient features.

## Go Files

- `drivers_generic.go` – Registers HTTP client drivers available on non-Windows builds.
- `drivers_windows.go` – Registers HTTP drivers that rely on Windows-specific APIs.
- `gohttp.go` – Implements the Go stdlib HTTP transport integration.
- `httpclient.go` – Main HTTP client scheduler handling polling and task delivery.
