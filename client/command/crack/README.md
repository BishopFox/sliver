# client/command/crack

## Overview

Implements the 'crack' command group for the Sliver client console. Handlers map Cobra invocations to crack workflows such as crack files.

## Go Files

- `commands.go` – Defines the crack command hierarchy, attaching operations for stations and file management.
- `crack-files.go` – Manages cracking asset uploads, listings, and removals, handling streaming transfers and chunking.
- `crack.go` – Implements station status reporting, health checks, and list rendering for distributed cracking nodes.
- `helpers.go` – Supplies completion helpers and byte formatting utilities for crack subcommands.
