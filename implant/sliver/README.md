# implant/sliver

## Overview

Core Go implementation of the Sliver implant runtime and supporting subsystems. Houses communications, task execution, and platform abstraction layers.

## Go Files

- `main.go` – Implant entry point that wires transports, handlers, and the task runner together at startup.

## Sub-packages

- `constants/` – Compile-time constants and identifiers shared across the implant. Defines feature flags, limits, and well-known keys used during runtime.
- `cryptography/` – Cryptographic primitives and wrappers used by the implant for secure operations. Offers key handling, signing, and envelope encryption helpers.
- `encoders/` – Binary and text encoders embedded with the implant for staging and comms. Hosts encoder registries, factories, and runtime switching logic.
- `evasion/` – Host evasion helpers to reduce detection on compromised systems. Implements anti-scan routines, parent process spoofing, and related checks.
- `extension/` – Extension host runtime allowing implants to load optional capabilities. Manages sandboxing, lifecycle, and communication with extension modules.
- `forwarder/` – Network forwarding helpers that move data through pivot tunnels. Provides connectors, relays, and buffering for implant forwarding.
- `handlers/` – Runtime message handlers that react to server instructions. Dispatches inbound RPC messages to feature-specific executors.
- `hostuuid/` – Host fingerprinting and UUID generation utilities for implants. Collects platform attributes and derives deterministic host identifiers.
- `limits/` – Resource limitation utilities controlling implant behavior. Enforces concurrency caps, throttling, and watchdog timers.
- `locale/` – Localization helpers for implant messages. Supplies translation lookup tables and locale-aware formatting.
- `mount/` – Cross-platform file mount helpers used by implants. Handles mounting, unmounting, and privilege-aware filesystem access.
- `netstack/` – gVisor-based userland network stack adapted for the implant. Integrates packet handling, TCP/IP primitives, and buffer management.
- `netstat/` – Network inspection helpers exposed to operators via implant commands. Retrieves interface statistics, socket tables, and routing data.
- `pivots/` – Pivot channel management for lateral movement through implants. Tracks upstream/downstream links and maintains tunnel state.
- `priv/` – Privilege escalation helpers embedded in the implant. Houses exploit runners, privilege checks, and elevation utilities.
- `procdump/` – Process dump utilities for extracting memory from targets. Implements cross-platform dump routines and artifact packaging.
- `proxy/` – Implant-side proxying helpers for relaying traffic. Configures listeners, upstream connectors, and connection mapping.
- `ps/` – Process enumeration utilities shipped with the implant. Collects process metadata and formats listings for operators.
- `registry/` – Windows registry interaction helpers for implants. Wraps enumeration, read/write, and hive manipulation logic.
- `rportfwd/` – Reverse port forwarding implementation inside the implant. Manages listener registration, channel wiring, and teardown.
- `screen/` – Screen capture helpers for the implant. Captures framebuffer data, encodes it, and returns results to operators.
- `service/` – Windows service management features for the implant. Provides install, start/stop, and query helpers for services.
- `shell/` – Interactive shell functionality exposed by the implant. Coordinates IO loops, PTY integration, and command execution.
- `spoof/` – Identity and network spoofing helpers bundled with the implant. Implements MAC/IP spoofing and related disguise tooling.
- `syscalls/` – Direct syscall wrappers and supporting code for stealth execution. Exposes low-level syscall invocations and helper shims.
- `taskrunner/` – Task scheduler running queued implant jobs. Controls execution order, retries, and task telemetry.
- `tcpproxy/` – TCP proxy client used by implants for pivoting. Dials upstream pivots, relays streams, and enforces limits.
- `transports/` – Outbound transport implementations compiled into the implant. Registers available transport clients and shared plumbing.
- `version/` – Compile-time version metadata baked into implants. Publishes semantic version info, build IDs, and compatibility helpers.
- `winhttp/` – Windows HTTP helpers and fallbacks for implant transports. Supplies WinHTTP-based dialers and configuration routines.
