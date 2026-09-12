# client/command/crack

## Overview

Implements the 'crack' command group for the Sliver client console. Handlers map Cobra invocations to crack workflows such as cached benchmark inspection, durable jobs, real-time monitoring, and crack files.

## Go Files

- `commands.go` – Defines the crack command hierarchy, attaching operations for stations and file management.
- `crack-benchmarks.go` – Displays bounded cached crackstation benchmark summaries, freshness metadata, and representative or complete hash-mode rates.
- `crack-files.go` – Manages cracking asset uploads, listings, and removals, handling streaming transfers and chunking.
- `crack-jobs.go` – Lists durable cracking jobs and renders detailed task progress, device telemetry, and recovered credentials.
- `crack-top.go` – Runs the full-screen Bubble Tea monitor, including responsive job/worker panes and a Huh-backed status filter.
- `crack-top-loader.go` – Loads one lean, target-free snapshot containing every active job, bounded recent history, and connected crackstation telemetry.
- `crack-top-telemetry.go` – Calculates weighted shard progress plus per-worker and global live hash rates without treating cached benchmarks as current throughput.
- `crack.go` – Implements station status reporting, health checks, and list rendering for distributed cracking nodes.
- `helpers.go` – Supplies completion helpers and byte formatting utilities for crack subcommands.
