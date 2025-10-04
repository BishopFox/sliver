# implant/sliver/transports/dnsclient

## Overview

DNS-based transport client used for low-and-slow comms. Encodes tasking into DNS queries and parses responses. Runtime components handle conf generic, conf windows, resolver generic, and resolver system for implant-side dnsclient features.

## Go Files

- `conf_generic.go` – Defines DNS transport configuration defaults for non-Windows builds.
- `conf_windows.go` – Windows-specific DNS transport configuration overrides.
- `dnsclient.go` – Core DNS transport implementation encoding/decoding payloads.
- `dnsclient_test.go` *(tests)* – Tests DNS client encoding and schedules.
- `resolver-generic.go` – Stub resolver implementation for custom DNS servers.
- `resolver-system.go` – Resolver backed by the system DNS API.
- `resolver.go` – Shared resolver interfaces and helper functions.
