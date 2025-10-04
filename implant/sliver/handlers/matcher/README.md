# implant/sliver/handlers/matcher

## Overview

Matcher logic for routing inbound messages to the right handlers. Builds lookup tables and rule evaluations for message dispatch. Runtime components handle matcher windows for implant-side matcher features.

## Go Files

- `matcher.go` – Implements message matching logic and lookup table generation for handler dispatch.
- `matcher_windows.go` – Provides Windows-specific matcher overrides and optimizations.
