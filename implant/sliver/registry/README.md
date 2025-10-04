# implant/sliver/registry

## Overview

Windows registry interaction helpers for implants. Wraps enumeration, read/write, and hive manipulation logic. Runtime components handle registry windows for implant-side registry features.

## Go Files

- `registry.go` – Provides shared registry interfaces and request structs.
- `registry_windows.go` – Implements Windows registry operations via native APIs.
