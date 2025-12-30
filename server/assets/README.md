# server/assets

## Overview

Server-side embedded assets such as default profiles and resources. Packages default configurations, templates, and scripts for distribution. Key routines cover assets helpers, assets darwin amd64, assets darwin arm64, and assets linux amd64 within the assets subsystem.

## Go Files

- `assets-helpers.go` – Provides helper functions for locating and loading embedded server assets.
- `assets.go` – Registers asset bundles and exposes filesystem abstractions.
- `assets_darwin_amd64.go` – Embeds macOS amd64 asset blobs.
- `assets_darwin_arm64.go` – Embeds macOS arm64 asset blobs.
- `assets_linux_amd64.go` – Embeds Linux amd64 asset blobs.
- `assets_linux_arm64.go` – Embeds Linux arm64 asset blobs.
- `assets_windows_amd64.go` – Embeds Windows amd64 asset blobs.

## Sub-packages

- `traffic-encoders/` – Managed traffic encoder templates bundled with the server. Provides default encoder definitions and related tests.
