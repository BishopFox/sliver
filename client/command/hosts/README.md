# client/command/hosts

## Overview

Implements the 'hosts' command group for the Sliver client console. Handlers map Cobra invocations to hosts workflows such as hosts IOC RM, hosts IOC, and hosts RM.

## Go Files

- `commands.go` – Sets up the hosts command suite and its IOC management subcommands.
- `hosts-ioc-rm.go` – Removes stored indicators of compromise for a host via RPC.
- `hosts-ioc.go` – Lists and prints host IOC data retrieved from the server.
- `hosts-rm.go` – Deletes hosts from inventory and confirms removal operations.
- `hosts.go` – Displays host summaries, filters by query, and renders tabular output.
