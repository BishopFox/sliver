# client/command/jobs

## Overview

Implements the 'jobs' command group for the Sliver client console. Handlers map Cobra invocations to jobs workflows such as DNS, HTTP, HTTPS, and mTLS.

## Go Files

- `commands.go` – Declares the jobs command tree and wires transport-specific subcommands.
- `dns.go` – Lists, starts, and stops DNS listener jobs via RPC.
- `http.go` – Manages HTTP listener jobs, including creation and teardown.
- `https.go` – Handles HTTPS listener lifecycle operations with certificate options.
- `jobs.go` – Provides common utilities for rendering job tables and selecting listener IDs.
- `mtls.go` – Controls mTLS listener jobs and prints connection details.
- `stage.go` – Manages staging servers for delivering payloads to new implants.
- `wg.go` – Configures and monitors WireGuard listener jobs for C2 traffic.
