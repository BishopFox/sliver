# client/command/reconfig

## Overview

Implements the 'reconfig' command group for the Sliver client console. Handlers map Cobra invocations to reconfig workflows such as rename.

## Go Files

- `commands.go` – Registers the reconfig command and attaches subcommands such as rename.
- `reconfig.go` – Shows current implant configuration and prompts for updates.
- `rename.go` – Renames implants or beacons on the server for easier identification.
