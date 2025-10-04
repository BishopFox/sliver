# implant/sliver/procdump

## Overview

Process dump utilities for extracting memory from targets. Implements cross-platform dump routines and artifact packaging. Runtime components handle dump, dump darwin, dump linux, and dump windows for implant-side procdump features.

## Go Files

- `dump.go` – Defines shared process dump logic and orchestrates platform hooks.
- `dump_darwin.go` – macOS-specific memory dump implementation.
- `dump_linux.go` – Linux process dumping routines and helpers.
- `dump_windows.go` – Windows process dump support using system APIs.
