# client/command/environment

## Overview

Implements the 'environment' command group for the Sliver client console. Handlers map Cobra invocations to environment workflows such as GET, SET, and unset.

## Go Files

- `commands.go` – Declares environment command variations and wires subcommands for get/set/unset operations with shared flags.
- `get.go` – Invokes the RPC to retrieve environment variables and prints the results for sessions or beacons.
- `set.go` – Handles setting environment variables remotely, including beacon callback handling and result formatting.
- `unset.go` – Removes environment variables on the remote target and reports task outcomes back to the user.
