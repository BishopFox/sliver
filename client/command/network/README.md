# client/command/network

## Overview

Implements the 'network' command group for the Sliver client console. Handlers map Cobra invocations to network workflows such as ifconfig and netstat.

## Go Files

- `commands.go` – Registers network inspection commands for implants.
- `ifconfig.go` – Retrieves and prints interface configuration details from the target.
- `netstat.go` – Displays remote socket listings similar to netstat output.
