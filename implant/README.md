# implant

## Overview

Top-level tooling for building and maintaining Sliver implant payloads. Provides build pipelines, dependency vendoring, and shared entrypoints. Runtime components handle generate for implant-side implant features.

## Go Files

- `generate.go` – Implements top-level implant build wiring invoked from the server/client tooling.
- `implant.go` – Entry point for compiling implants and exposing shared build helpers.

## Sub-packages

- `scripts/` – Helper scripts and utilities for implant vendor management and automation. Includes tooling for syncing nested vendored dependencies.
- `sliver/` – Core Go implementation of the Sliver implant runtime and supporting subsystems. Houses communications, task execution, and platform abstraction layers.
