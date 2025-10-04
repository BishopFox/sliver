# client/command/pivots

## Overview

Implements the 'pivots' command group for the Sliver client console. Handlers map Cobra invocations to pivots workflows such as details, graph, start, and stop.

## Go Files

- `commands.go` – Declares the pivots command family and ties start/stop operations together.
- `details.go` – Prints detailed information about a specific pivot chain.
- `graph.go` – Renders pivot relationships as ASCII graphs for situational awareness.
- `helpers.go` – Supplies shared RPC helpers and completions used across pivot commands.
- `pivots.go` – Lists pivot sessions and summarizes their status.
- `start.go` – Initiates new pivot listeners or routes for the active target.
- `stop.go` – Stops running pivots and cleans up resources on the implant.
