# client/command

## Overview

Shared registration glue, scaffolding, and helpers for Sliver client console command groups. Hosts common utilities for binding Cobra commands to RPC-backed actions. Core logic addresses sliver within the command package.

## Go Files

- `command.go` – Provides helpers for binding Cobra commands, flag sets, and visibility filters into the client command tree.
- `server.go` – Defines the server menu composition, wiring each server-side command group onto the root console structure at runtime.
- `main.go` – Builds the implant/Sliver menu, attaching per-target command groups and loading alias/extension manifests before exposure.

## Sub-packages

- `alias/` – Implements the 'alias' command group for the Sliver client console.
- `armory/` – Implements the 'armory' command group for the Sliver client console.
- `backdoor/` – Implements the 'backdoor' command group for the Sliver client console.
- `beacons/` – Implements the 'beacons' command group for the Sliver client console.
- `builders/` – Implements the 'builders' command group for the Sliver client console.
- `c2profiles/` – Implements the 'c2profiles' command group for the Sliver client console.
- `certificates/` – Implements the 'certificates' command group for the Sliver client console.
- `clean/` – Implements the 'clean' command group for the Sliver client console.
- `completers/` – Implements the 'completers' command group for the Sliver client console.
- `crack/` – Implements the 'crack' command group for the Sliver client console.
- `creds/` – Implements the 'creds' command group for the Sliver client console.
- `cursed/` – Implements the 'cursed' command group for the Sliver client console.
- `dllhijack/` – Implements the 'dllhijack' command group for the Sliver client console.
- `environment/` – Implements the 'environment' command group for the Sliver client console.
- `exec/` – Implements the 'exec' command group for the Sliver client console.
- `exit/` – Implements the 'exit' command group for the Sliver client console.
- `extensions/` – Implements the 'extensions' command group for the Sliver client console.
- `filesystem/` – Implements the 'filesystem' command group for the Sliver client console.
- `flags/` – Implements the 'flags' command group for the Sliver client console.
- `generate/` – Implements the 'generate' command group for the Sliver client console.
- `help/` – Implements the 'help' command group for the Sliver client console.
- `hosts/` – Implements the 'hosts' command group for the Sliver client console.
- `info/` – Implements the 'info' command group for the Sliver client console.
- `jobs/` – Implements the 'jobs' command group for the Sliver client console.
- `kill/` – Implements the 'kill' command group for the Sliver client console.
- `licenses/` – Implements the 'licenses' command group for the Sliver client console.
- `loot/` – Implements the 'loot' command group for the Sliver client console.
- `monitor/` – Implements the 'monitor' command group for the Sliver client console.
- `mcp/` – Implements the 'mcp' command group for the Sliver client console.
- `network/` – Implements the 'network' command group for the Sliver client console.
- `operators/` – Implements the 'operators' command group for the Sliver client console.
- `pivots/` – Implements the 'pivots' command group for the Sliver client console.
- `portfwd/` – Implements the 'portfwd' command group for the Sliver client console.
- `privilege/` – Implements the 'privilege' command group for the Sliver client console.
- `processes/` – Implements the 'processes' command group for the Sliver client console.
- `reaction/` – Implements the 'reaction' command group for the Sliver client console.
- `reconfig/` – Implements the 'reconfig' command group for the Sliver client console.
- `registry/` – Implements the 'registry' command group for the Sliver client console.
- `rportfwd/` – Implements the 'rportfwd' command group for the Sliver client console.
- `screenshot/` – Implements the 'screenshot' command group for the Sliver client console.
- `sessions/` – Implements the 'sessions' command group for the Sliver client console.
- `settings/` – Implements the 'settings' command group for the Sliver client console.
- `shell/` – Implements the 'shell' command group for the Sliver client console.
- `shikata-ga-nai/` – Implements the 'shikata-ga-nai' command group for the Sliver client console.
- `socks/` – Implements the 'socks' command group for the Sliver client console.
- `taskmany/` – Implements the 'taskmany' command group for the Sliver client console.
- `tasks/` – Implements the 'tasks' command group for the Sliver client console.
- `update/` – Implements the 'update' command group for the Sliver client console.
- `use/` – Implements the 'use' command group for the Sliver client console.
- `wasm/` – Implements the 'wasm' command group for the Sliver client console.
- `websites/` – Implements the 'websites' command group for the Sliver client console.
- `wireguard/` – Implements the 'wireguard' command group for the Sliver client console.
