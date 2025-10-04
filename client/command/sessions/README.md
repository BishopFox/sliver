# client/command/sessions

## Overview

Implements the 'sessions' command group for the Sliver client console. Handlers map Cobra invocations to sessions workflows such as background, close, interactive, and prune.

## Go Files

- `background.go` – Sends interactive sessions to the background while keeping them connected.
- `close.go` – Closes sessions cleanly via RPC.
- `commands.go` – Assembles the sessions command hierarchy and subcommands.
- `helpers.go` – Provides lookup utilities and prompt helpers for session selection.
- `interactive.go` – Brings sessions into the foreground interactive console.
- `prune.go` – Removes disconnected or dead sessions from the server inventory.
- `sessions.go` – Lists active sessions and renders tabular summaries.
