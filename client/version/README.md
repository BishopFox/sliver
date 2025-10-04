# client/version

## Overview

Build-time version and semantic metadata for the client binary. Exposes git information, release channels, and helper formatters. Core logic addresses sliver version and updates within the version package.

## Go Files

- `sliver-version.go` – Stores build metadata (version strings, commit hashes) and helper formatters.
- `updates.go` – Tracks update state, caches release info, and signals when new versions are available.
