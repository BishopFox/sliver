# client/command/rportfwd

## Overview

Implements the 'rportfwd' command group for the Sliver client console. Handlers map Cobra invocations to rportfwd workflows such as portfwd ADD, portfwd RM, and portfwd.

## Go Files

- `commands.go` – Registers reverse port forwarding commands tied to implant callbacks.
- `portfwd-add.go` – Adds new reverse port forwards from remote hosts back to the operator.
- `portfwd-rm.go` – Removes reverse port forwards by identifier.
- `portfwd.go` – Lists active reverse port forwards and summarizes their endpoints.
