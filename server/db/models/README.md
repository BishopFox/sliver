# server/db/models

## Overview

Database models and ORM definitions for server state. Defines data schemas, relationships, and query helpers. Key routines cover beacon, canary, certificates, and crackstations within the models subsystem.

## Go Files

- `beacon.go` – Defines the beacon ORM model and associated relations.
- `canary.go` – Stores DNS canary records and status flags.
- `certificates.go` – Persists certificate metadata for listeners and operators.
- `crackstations.go` – Models cracking nodes and benchmark data.
- `credentials.go` – Holds captured credential entries and tags.
- `host.go` – Represents discovered hosts and host-specific metadata.
- `http-c2.go` – Stores HTTP C2 configuration data.
- `implant.go` – Tracks implant builds, configs, and artifacts.
- `jobs.go` – Records long-running server jobs and status.
- `keyex.go` – Persists key exchange state for implants.
- `keyvalue.go` – Generic key/value storage for misc settings.
- `loot.go` – Describes loot artifacts and file locations.
- `monitor.go` – Stores monitoring job configurations.
- `operator.go` – Represents operators, permissions, and MFA data.
- `resource_id.go` – Allocates human-readable resource identifiers.
- `website.go` – Captures hosted website configurations and assets.
- `wgkeys.go` – Stores WireGuard peer keys and metadata.
