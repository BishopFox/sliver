# implant/sliver/transports/dnsclient

## Overview

DNS-based transport client used for low-and-slow comms. Encodes tasking into DNS queries and parses responses. Runtime components handle conf generic, conf windows, resolver generic, and resolver system for implant-side dnsclient features.

## Go Files

- `conf_generic.go`
- `conf_windows.go`
- `dnsclient.go`
- `dnsclient_test.go` *(tests)*
- `resolver-generic.go`
- `resolver-system.go`
- `resolver.go`
