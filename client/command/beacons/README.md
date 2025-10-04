# client/command/beacons

## Overview

Implements the 'beacons' command group for the Sliver client console. Handlers map Cobra invocations to beacons workflows such as prune, RM, and watch.

## Go Files

- `beacons.go` – Retrieves beacon inventories and renders them in tabular form with optional filtering.
- `commands.go` – Declares the beacons command set, binds flags/completions, and wires subcommands like prune, rm, and watch.
- `helpers.go` – Provides helper RPC wrappers for selecting, fetching, and listing beacons.
- `prune.go` – Implements the prune routine that removes stale beacon entries via the backend API.
- `rm.go` – Removes a specific beacon by ID, coordinating confirmation and RPC teardown.
- `watch.go` – Streams beacon events in a watch loop, handling cancellation and console interaction.
