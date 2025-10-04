# client/command/settings

## Overview

Implements the 'settings' command group for the Sliver client console. Handlers map Cobra invocations to settings workflows such as beacons, opsec, and tables.

## Go Files

- `beacons.go` – Configures beacon polling, jitter, and other runtime options.
- `commands.go` – Registers the settings command namespace and attaches subcommands.
- `opsec.go` – Manages operational security settings like auto tasks and notifications.
- `settings.go` – Lists and updates general client settings persisted to disk.
- `tables.go` – Adjusts table formatting preferences for console output.
