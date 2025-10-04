# server/generate

## Overview

Server-side code generation and scaffolding utilities. Automates creation of boilerplate assets and configs. Key routines cover binaries, canaries, donut, and external within the generate subsystem.

## Go Files

- `binaries.go` – Generates binary artifacts and handles output formatting.
- `binaries_test.go` *(tests)* – Tests binary generation workflows.
- `canaries.go` – Generates DNS canary assets and configuration.
- `donut.go` – Integrates with Donut to build .NET loaders.
- `external.go` – Coordinates external builder invocations.
- `implants.go` – Generates implant binaries from configs.
- `profiles.go` – Creates implant profiles and configuration templates.
- `profiles_test.go` *(tests)* – Tests profile generation logic.
- `srdi-shellcode.go` – Produces SRDI shellcode payloads.
- `srdi.go` – Shared SRDI generation helpers.
- `wgips.go` – Generates WireGuard IP assignment tables.
