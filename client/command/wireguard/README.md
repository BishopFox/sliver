# client/command/wireguard

## Overview

Implements the 'wireguard' command group for the Sliver client console. Handlers map Cobra invocations to wireguard workflows such as WG config, WG portfwd ADD, WG portfwd RM, and WG portfwd.

## Go Files

- `commands.go` – Registers WireGuard-centric commands and their subcommands.
- `wg-config.go` – Generates and displays WireGuard configuration files for implants.
- `wg-portfwd-add.go` – Adds WireGuard-based port forwarding rules.
- `wg-portfwd-rm.go` – Removes WireGuard port forwarding entries.
- `wg-portfwd.go` – Lists WireGuard forwarding rules with status information.
- `wg-socks-start.go` – Starts SOCKS proxies over WireGuard tunnels.
- `wg-socks-stop.go` – Stops WireGuard-backed SOCKS proxies.
- `wg-socks.go` – Lists SOCKS proxies running over WireGuard connections.
