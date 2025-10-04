# client/command/creds

## Overview

Implements the 'creds' command group for the Sliver client console. Handlers map Cobra invocations to creds workflows such as ADD, hash types, RM, and select.

## Go Files

- `add.go` – Adds credentials to the client store via interactive or flag-based input.
- `commands.go` – Registers credential management commands and subcommands.
- `creds.go` – Lists stored credentials and formats them for display.
- `hash-types.go` – Enumerates supported hash types and their identifiers.
- `rm.go` – Removes credentials by ID or filter from the store.
- `select.go` – Selects credentials for use in other workflows, updating defaults.
