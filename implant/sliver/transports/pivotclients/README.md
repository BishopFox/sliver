# implant/sliver/transports/pivotclients

## Overview

Transport clients used when implants dial through pivot chains. Implements nested client behaviors and tunnel negotiation. Runtime components handle namedpipe, namedpipe generic, namedpipe windows, and pivotclient for implant-side pivotclients features.

## Go Files

- `namedpipe.go` – Common named pipe transport helpers.
- `namedpipe_generic.go` – Platform-neutral named pipe client stubs.
- `namedpipe_windows.go` – Windows named pipe transport implementation.
- `pivotclient.go` – Core pivot client abstraction coordinating nested transports.
- `tcp.go` – TCP-based pivot client implementation.
