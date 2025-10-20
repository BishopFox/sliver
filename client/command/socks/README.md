# client/command/socks

## Overview

Implements the 'socks' command group for the Sliver client console. Handlers map Cobra invocations to socks workflows such as rootcommands, socks start, and socks stop.

## Go Files

- `commands.go` – Registers SOCKS proxy commands within the client console.
- `rootCommands.go` – Provides shared root-level wiring for SOCKS commands used in multiple menus.
- `socks-start.go` – Starts local SOCKS proxy listeners tied to implants.
- `socks-stop.go` – Stops running SOCKS proxies and frees ports.
- `socks.go` – Lists SOCKS proxies and prints their connection details.
