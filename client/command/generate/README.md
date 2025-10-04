# client/command/generate

## Overview

Implements the 'generate' command group for the Sliver client console. Handlers map Cobra invocations to generate workflows such as canaries, generate beacon, generate info, and implants RM.

## Go Files

- `canaries.go` – Lists DNS canaries, filters burned entries, and prints status tables.
- `commands.go` – Defines the generate command hierarchy spanning implant, profile, and canary tooling.
- `generate-beacon.go` – Builds staged beacons from profile/flag inputs and triggers compilation via RPC.
- `generate-info.go` – Displays build configuration details for a generated implant artifact.
- `generate.go` – Orchestrates implant compilation pipelines, parsing C2 flags and selecting targets/builders.
- `helpers.go` – Supplies completions, helper lookups, and binary retrieval utilities used across generate commands.
- `implants-rm.go` – Deletes stored implant builds from the server catalog.
- `implants-stage.go` – Uploads generated implants into staging slots for later deployment.
- `implants.go` – Lists implant builds, filters them, and resolves selected build metadata.
- `profiles-generate.go` – Generates new implant profiles from templates and selected options.
- `profiles-new.go` – Creates empty implant profiles populated with default settings.
- `profiles-rm.go` – Removes saved implant profiles and confirms deletion state.
- `profiles-stage.go` – Stages implant profiles to facilitate rapid build operations.
- `profiles.go` – Lists profiles, prints detailed info, and resolves profile selections for other commands.
- `regenerate.go` – Rebuilds implants from existing configuration IDs without re-specifying flags.
- `traffic-encoders.go` – Manages generation of HTTP/DNS traffic encoder assets for implants.
