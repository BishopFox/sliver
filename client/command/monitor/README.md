# client/command/monitor

## Overview

Implements the 'monitor' command group for the Sliver client console. Handlers map Cobra invocations to monitor workflows such as start and stop.

## Go Files

- `commands.go` – Registers the monitor command suite for live session monitoring.
- `config.go` – Parses monitor configuration flags and resolves target selections.
- `start.go` – Starts monitoring jobs and streams event output to the console.
- `stop.go` – Stops active monitor jobs and reports termination status.
