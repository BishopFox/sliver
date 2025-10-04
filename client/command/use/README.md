# client/command/use

## Overview

Implements the 'use' command group for the Sliver client console. Handlers map Cobra invocations to use workflows such as beacons and sessions.

## Go Files

- `beacons.go` – Switches the active target to a selected beacon.
- `commands.go` – Registers the use command and subcommands for session selection.
- `sessions.go` – Sets the active session context for subsequent commands.
- `use.go` – Implements shared helper logic for switching targets and updating prompts.
