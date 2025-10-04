# client

## Overview

Main command-line client for interacting with and administering Sliver servers. Provides the Cobra entrypoint and aggregates console, command, and transport subsystems.

## Go Files

- `main.go` – Entry point that initializes the CLI, processes flags, and launches the interactive console.

## Sub-packages

- `assets/` – Manages embedded client asset manifests such as operator profiles, settings, and armory catalogs. Handles serialization, bundling, and lookup of static client metadata.
- `cli/` – Defines the Cobra root command and CLI initialization for the Sliver client binary. Sets up persistent flags, environment bootstrapping, and command wiring.
- `command/` – Shared registration glue, scaffolding, and helpers for Sliver client console command groups. Hosts common utilities for binding Cobra commands to RPC-backed actions.
- `console/` – Renders the interactive console UI, logging, and terminal helpers for the client. Coordinates readline behavior, prompt styling, and message formatting.
- `constants/` – Centralizes constants shared across the client runtime and command packages. Captures strings, flag names, and feature toggles used throughout the CLI.
- `core/` – Maintains client application state, background workers, and RPC coordination. Orchestrates session tracking, event streams, and health checks.
- `credentials/` – Credential management utilities including sniffers and import helpers for operators. Provides parsers, storage helpers, and output formatting for recovered secrets.
- `licenses/` – Holds embedded license metadata bundled with the client distribution. Supplies license retrieval utilities used for attribution commands.
- `overlord/` – Supervises long-running client jobs, scheduling, and task orchestration. Tracks job lifecycle events and exposes status updates to the console.
- `packages/` – Deals with packaged extensions and module discovery for the client. Scans, validates, and caches extension manifests and assets.
- `spin/` – Lightweight terminal spinner utilities used for progress feedback. Wraps spinner styles and timing helpers shared across commands.
- `tcpproxy/` – Implements the local TCP proxy used for port forwarding from the client side. Manages listener lifecycles, connection pumps, and cleanup hooks.
- `transport/` – Client-side transports and RPC wiring to communicate with Sliver servers. Builds authenticated gRPC clients, TLS dialers, and reconnect logic.
- `version/` – Build-time version and semantic metadata for the client binary. Exposes git information, release channels, and helper formatters.
