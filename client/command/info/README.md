# client/command/info

## Overview

Implements the 'info' command group for the Sliver client console. Handlers map Cobra invocations to info workflows such as ping.

## Go Files

- `commands.go` – Registers information-oriented commands and binds them into the console.
- `info.go` – Queries detailed session or beacon metadata and prints rich status tables.
- `ping.go` – Sends ping requests to implants to test connectivity and round-trip latency.
