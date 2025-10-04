# client/command/portfwd

## Overview

Implements the 'portfwd' command group for the Sliver client console. Handlers map Cobra invocations to portfwd workflows such as portfwd ADD and portfwd RM.

## Go Files

- `commands.go` – Sets up the port forwarding command suite.
- `portfwd-add.go` – Adds new local-to-remote port forwarders on the active session.
- `portfwd-rm.go` – Removes configured port forwarders by ID.
- `portfwd.go` – Lists existing port forwards and renders their configuration.
