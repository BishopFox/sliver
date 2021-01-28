Sliver Comm 
======


The sliver `comm` is similar with the server `comm` package in these aspects:
- An SSH multiplexer (Comm).
- Abstracted listeners, which track listeners started on implant hosts (reverse)
- Connection-wrapping code for handling things like UDP/IP streams, etc.
- And utilities.

However, it is different from it in in these aspects:
- The `Comm` is always in a client position.
- The client does not need really advanced connection wrapping an monitoring other than for PacketConns.
- The package includes Named Pipes dialers/listeners, as they are somewhat equivalent to Unix pipes.
- The `Comm` has no notion of route nor its aware of the currently active routes on the server.

This package goes hand-in-hand with the `commpb` Protobuf package, where all messages and types used are defined.
Links to the other `comm` packages: 
- [Client](https://github.com/maxlandon/sliver/tree/Comm/client/comm)
- [Server](https://github.com/maxlandon/sliver/tree/Comm/server/comm)
- [Protobuf Definitions](https://github.com/maxlandon/sliver/tree/Comm/protobuf/commpb)

-----
## Files

### Core
- `comm.go`         - All code for the `Comm` type, an SSH multiplexer (setup, start, serve, etc.)
- `comms.go`        - Gathers the global `Comms`, `Tunnels` maps, and their Add(), Remove(), Get() methods.
- `tunnel.go`       - A specific tunnel used as a basis for multiplexing DNS/HTTPS based Sliver connections.

### Protocol-specific
- `dial-tcp.go`           - Handles TCP traffic coming from the server, to be dialed forward. 
- `dial-udp.go`           - Handles UDP traffic coming from the server. 
- `listen-tcp.go`         - Custom TCP listener that can be closed by the server.
- `listen-udp.go`         - Custom UDP listener (PacketConn), can be closed by the server.
- `named-pipe.go`         - Included in all builds.
- `named-pipe_windows.go` - Code for dialing and listening on Windows named pipes.
