# client/command/crack

## Overview

Implements the 'crack' command group for the Sliver client console. Handlers map Cobra invocations to crack workflows such as cached benchmark inspection, durable jobs, real-time monitoring, and crack files.

## Go Files

- `commands.go` – Defines the crack command hierarchy, attaching operations for stations and file management.
- `crack-benchmarks.go` – Displays bounded cached crackstation benchmark summaries, freshness metadata, and representative or complete hash-mode rates.
- `crack-files.go` – Manages cracking asset uploads, listings, and removals, handling streaming transfers and chunking.
- `crack-jobs.go` – Lists durable cracking jobs, renders detailed task progress, manages pause/resume/cancel lifecycle actions, and removes terminal jobs after confirmation.
- `top.go` – Preserves the crack command's compatibility entry point and delegates to the dedicated monitor package.
- `top/` – Owns the full-screen Bubble Tea monitor, its lean real-time snapshot loader, live telemetry, history, charts, actions, and tests.
- `internal/hashcatdisplay/` – Shares Hashcat mode names and rate formatting between the command handlers and monitor without coupling the packages.
- `crack.go` – Implements station status reporting, health checks, and list rendering for distributed cracking nodes.
- `helpers.go` – Supplies completion helpers and byte formatting utilities for crack subcommands.
