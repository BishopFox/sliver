# client/command/registry

## Overview

Implements the 'registry' command group for the Sliver client console. Handlers map Cobra invocations to registry workflows such as REG create, REG delete, REG list, and REG read.

## Go Files

- `commands.go` – Exposes registry management commands for Windows targets.
- `reg-create.go` – Creates new registry keys or values remotely.
- `reg-delete.go` – Deletes registry keys/values with confirmation messaging.
- `reg-list.go` – Enumerates registry subkeys and values for a provided path.
- `reg-read.go` – Reads and prints registry value data.
- `reg-write.go` – Writes registry values with type handling for supported data kinds.
