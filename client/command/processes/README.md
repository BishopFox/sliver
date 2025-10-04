# client/command/processes

## Overview

Implements the 'processes' command group for the Sliver client console. Handlers map Cobra invocations to processes workflows such as procdump, PS, pstree, and terminate.

## Go Files

- `commands.go` – Creates the processes command suite and binds subcommands for inspection and control.
- `procdump.go` – Dumps process memory to loot for forensic analysis via RPC tasks.
- `ps.go` – Lists active processes on the target with filtering options.
- `pstree.go` – Renders process hierarchies as a tree view.
- `services.go` – Enumerates Windows services and their states.
- `terminate.go` – Terminates processes by PID with confirmation messaging.
