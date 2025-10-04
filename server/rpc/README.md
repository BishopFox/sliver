# server/rpc

## Overview

RPC server implementations and wiring. Boots gRPC servers, registers services, and manages middleware. Key routines cover RPC backdoor, RPC beacons, RPC C2 profile, and RPC certificates within the rpc subsystem.

## Go Files

- `errors.go` – Defines shared RPC error helpers and codes.
- `rpc-backdoor.go` – Handles RPC endpoints for backdoor operations.
- `rpc-beacons.go` – Implements beacon management RPCs.
- `rpc-c2profile.go` – Serves C2 profile CRUD and generation requests.
- `rpc-certificates.go` – Exposes certificate issuance and listing RPCs.
- `rpc-client-logs.go` – Streams client log data via RPC.
- `rpc-crackstations.go` – Manages crackstation nodes through RPC commands.
- `rpc-creds.go` – Provides credential store RPC endpoints.
- `rpc-env.go` – Handles environment variable get/set RPC calls.
- `rpc-events.go` – Streams server events to clients.
- `rpc-execute.go` – Executes commands/shellcode on implants via RPC.
- `rpc-extensions.go` – Manages extension installation and invocation RPCs.
- `rpc-filesystem.go` – Exposes filesystem operations over RPC.
- `rpc-generate.go` – Drives payload/profile generation commands.
- `rpc-hijack.go` – Handles DLL hijack RPC requests.
- `rpc-hosts.go` – Offers host inventory RPCs.
- `rpc-jobs.go` – Manages background job operations via RPC.
- `rpc-kill.go` – Provides RPC to kill implants or sessions.
- `rpc-loot.go` – Manages loot retrieval and updates.
- `rpc-monitor.go` – Controls monitor job lifecycle through RPC.
- `rpc-msf.go` – Bridges Metasploit integration RPC calls.
- `rpc-net.go` – Delivers network info RPC responses.
- `rpc-operators.go` – Operator management RPC endpoints.
- `rpc-ping.go` – Implements ping RPC for reachability checks.
- `rpc-pivots.go` – Handles pivot setup/teardown RPCs.
- `rpc-portfwd.go` – Manages port-forward RPC operations.
- `rpc-priv.go` – Runs privilege escalation RPC tasks.
- `rpc-process.go` – Provides process listing/termination RPCs.
- `rpc-reconfig.go` – Handles implant reconfiguration RPC actions.
- `rpc-registry.go` – Exposes Windows registry RPC endpoints.
- `rpc-rportfwd.go` – Manages reverse port forwarding RPC commands.
- `rpc-screenshot.go` – Requests screenshots via RPC.
- `rpc-service.go` – Controls service-related RPC calls.
- `rpc-sessions.go` – Manages session lifecycle through RPC.
- `rpc-shell.go` – Provides interactive shell RPC streaming.
- `rpc-shellcode.go` – Executes raw shellcode over RPC.
- `rpc-socks.go` – Controls SOCKS proxy operations via RPC.
- `rpc-stager.go` – Handles stager delivery RPC endpoints.
- `rpc-tasks.go` – Lists and cancels tasks through RPC.
- `rpc-tunnel.go` – Manages tunnel creation and traffic forwarding RPCs.
- `rpc-wasm.go` – Executes WASM modules via RPC interface.
- `rpc-website.go` – Serves website management RPC functions.
- `rpc-wg.go` – Manages WireGuard configuration via RPC.
- `rpc.go` – Boots the RPC server and registers all service handlers.
