# implant/sliver/proxy

## Overview

Implant-side proxying helpers for relaying traffic. Configures listeners, upstream connectors, and connection mapping. Runtime components handle provider, provider darwin, provider generic, and provider linux for implant-side proxy features.

## Go Files

- `doc.go` – Package documentation stub describing the proxy interfaces.
- `provider.go` – Declares the provider interface and shared factory logic.
- `provider_darwin.go` – macOS implementation of the proxy provider.
- `provider_darwin_test.go` *(tests)* – Tests macOS-specific provider behavior.
- `provider_generic.go` – Default provider behavior used by non-specialized builds.
- `provider_linux.go` – Linux proxy provider implementation.
- `provider_test.go` *(tests)* – Exercises provider selection and validation.
- `provider_windows.go` – Windows proxy provider implementation.
- `provider_windows_test.go` *(tests)* – Tests Windows-specific provider logic.
- `proxy.go` – Implements core proxy orchestration and connection routing.
- `proxy_test.go` *(tests)* – Validates proxy orchestration behavior.
- `url.go` – Parses proxy URLs and normalizes configuration options.
- `url_test.go` *(tests)* – Tests URL parsing and normalization helpers.
