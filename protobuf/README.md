# protobuf

## Overview

Generated Protocol Buffer bindings shared between client, server, and implant. Collects common message types for cross-component communication. Key modules cover protobufs.

## Go Files

- `protobufs.go` – Registers all embedded proto descriptors for runtime reflection and versioning.

## Sub-packages

- `clientpb/` – Generated client RPC message bindings. Defines client-facing request and response protobufs.
- `commonpb/` – Generated common/shared message structures. Provides utility messages reused across services.
- `dnspb/` – Generated DNS transport message bindings. Covers protobuf structures for DNS-based communications.
- `rpcpb/` – Generated RPC service definitions and gRPC stubs. Contains service interfaces and server/client scaffolding.
- `sliverpb/` – Generated core Sliver control-plane messages. Represents implant control, job, and telemetry types.
