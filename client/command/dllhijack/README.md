# client/command/dllhijack

## Overview

Implements the 'dllhijack' command group for the Sliver client console.

## Go Files

- `commands.go` – Registers the dllhijack command, its flags, and completions for selecting remote DLL targets.
- `dllhijack.go` – Performs the RPC call that plants a Sliver DLL alongside a target executable for hijacking.
