Client Comm 
======

The client `comm` is similar with the server `comm` package in these aspects:
- An SSH multiplexer (Comm).
- Abstracted listeners, which track listeners started on implant hosts (reverse)
- Connection-wrapping code for handling things like UDP/IP streams, etc.
- And utilities.

However, it is different from it in in these aspects:
- The `Comm` is always in a client position.
- Abstracted dialers/listeners are actually wrapped into Port Forwarders, usable with console commands.
- The client does not need really advanced connection wrapping and monitoring other than for PacketConns.

This package goes hand-in-hand with the `commpb` Protobuf package, where all messages and types used are 
defined, and with the following other `comm` packages: 
- [Server](https://github.com/maxlandon/sliver/tree/Comm/server/comm)
- [Implant](https://github.com/maxlandon/sliver/tree/Comm/sliver/comm)
- [Protobuf Definitions](https://github.com/maxlandon/sliver/tree/Comm/protobuf/commpb)

-----
## Files

A brief explanation of the package files.

### Core
- `comm.go`         - All code for the `Comm` type, adapted for a client console working through gRPC. 
- `comms.go`        - Utility code for piping connections, storing the unique client `Comm` instance, etc. 
- `conn_tcp.go`     - Info attached to direct forwarded TCP connections.
- `conn_udp.go`     - Little connection wrapping code for reverse forwarded UDP connections.

### Port forwarders
- `portfwd.go`          - The `Forwarder` interface type, and all exported functions used by commands. 
- `portfwd_direct_tcp`  - Direct TCP port forwarder
- `portfwd_direct_udp`  - Direct UDP port forwarder
- `portfwd_reverse_tcp` - Reverse TCP port forwarder
- `portfwd_reverse_udp` - Reverse UDP port forwarder

