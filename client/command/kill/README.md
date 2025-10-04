# client/command/kill

## Overview

Implements the 'kill' command group for the Sliver client console.

## Go Files

- `commands.go` – Registers the kill command and binds timeout/target flags.
- `kill.go` – Terminates sessions or beacons on the server, handling confirmations and output messaging.
