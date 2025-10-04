# implant/sliver/service

## Overview

Windows service management features for the implant. Provides install, start/stop, and query helpers for services. Runtime components handle service windows for implant-side service features.

## Go Files

- `service.go` – Declares service management interfaces and shared logic.
- `service_windows.go` – Implements Windows service install, control, and status operations.
