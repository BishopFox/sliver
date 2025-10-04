# client/command/reaction

## Overview

Implements the 'reaction' command group for the Sliver client console. Handlers map Cobra invocations to reaction workflows such as reload, save, SET, and unset.

## Go Files

- `commands.go` – Wires the reaction command set for automations and triggers.
- `helpers.go` – Supplies shared lookup and persistence helpers for reaction rules.
- `reaction.go` – Lists existing reactions and displays their trigger conditions.
- `reload.go` – Reloads reaction definitions from disk into the client runtime.
- `save.go` – Persists in-memory reaction definitions back to disk.
- `set.go` – Creates or modifies reaction rules interactively.
- `unset.go` – Removes configured reaction rules from the store.
