# client/command/loot

## Overview

Implements the 'loot' command group for the Sliver client console. Handlers map Cobra invocations to loot workflows such as fetch, local, remote, and rename.

## Go Files

- `commands.go` – Declares the loot command tree and binds fetch/local/removal subcommands.
- `fetch.go` – Downloads remote loot artifacts and writes them to local storage.
- `helpers.go` – Provides helper functions for lookup, filtering, and formatting loot entries.
- `local.go` – Lists, inspects, and opens loot stored on the client filesystem.
- `loot.go` – Shows remote loot inventory and renders summaries.
- `remote.go` – Interacts with server-side loot, supporting filtering and detail views.
- `rename.go` – Renames loot items while synchronizing local and remote metadata.
- `rm.go` – Deletes loot entries from the server or local cache with confirmation prompts.
