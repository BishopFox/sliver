# server

## Overview

Server daemon responsible for managing operators, implants, and infrastructure. Hosts the control plane, APIs, and background services.

## Go Files

- `main.go` – Server entry point that parses flags, loads config, and launches the daemon runtime.

## Sub-packages

- `assets/` – Server-side embedded assets such as default profiles and resources. Packages default configurations, templates, and scripts for distribution.
- `builder/` – Payload builder orchestration running on the server. Coordinates job queues, build options, and artifact storage.
- `c2/` – Command-and-control service wiring exposed by the server. Manages transport listeners, job dispatch, and staging pipelines.
- `certs/` – Certificate generation and management helpers for server transports. Issues and rotates TLS material for listeners.
- `cli/` – CLI entrypoint for running the server daemon. Parses flags, configures logging, and launches the service runtime.
- `codenames/` – Codename generator utilities for implants and operators. Implements deterministic name generation and word lists.
- `configs/` – Configuration loading and validation for the server. Parses config files, environment overrides, and applies defaults.
- `console/` – Administrative console helpers bundled with the server. Provides shared console-mode code reused by client tooling.
- `core/` – Core server runtime coordination, state, and dispatch. Oversees lifecycle management, hub services, and shared state.
- `cryptography/` – Server-side cryptographic helpers and key management. Implements signing, encryption, and key derivation utilities.
- `daemon/` – Control logic for starting and supervising the Sliver daemon. Responsible for graceful shutdown and lifecycle hooks.
- `db/` – Database access layer including migrations and helpers. Establishes database connections, adapters, and query utilities.
- `encoders/` – Managed encoder definitions paralleling the implant encoders. Supplies interpreter integration and server-side validation.
- `generate/` – Server-side code generation and scaffolding utilities. Automates creation of boilerplate assets and configs.
- `gogo/` – Customizations for gogo/protobuf integration with the server. Tweaks protobuf handling and compatibility glue.
- `handlers/` – RPC and event handlers that respond to client and implant requests. Routes gRPC calls to business logic modules.
- `log/` – Logging helpers and structured log setup for the server. Configures log sinks, formats, and telemetry hooks.
- `loot/` – Operator loot storage and retrieval helpers managed by the server. Handles filesystem organization, metadata, and access control.
- `msf/` – Metasploit integration utilities. Implements payload translation and RPC bridging with Metasploit.
- `netstack/` – gVisor-based userland network stack for server transports. Provides packet buffers, endpoint management, and adapters.
- `rpc/` – RPC server implementations and wiring. Boots gRPC servers, registers services, and manages middleware.
- `sgn/` – SGN (Sliver Guard Node) coordination and helpers. Implements SGN enrollment, messaging, and policy logic.
- `transport/` – Server-side transports and listener orchestration. Coordinates C2 listener lifecycles and connection routing.
- `watchtower/` – Watchtower monitoring and background jobs. Runs scheduled tasks, telemetry collection, and alerting hooks.
- `website/` – Static website assets and server-side handling. Serves operator web artifacts and supporting handlers.
