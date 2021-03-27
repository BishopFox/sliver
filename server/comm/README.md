Server Comm 
======

The `comm` package contains the code for the server-side Comm System. This comprises, briefly:
- An SSH multiplexer (wrapped inside a `Comm` type), used by implants and client consoles.
- Tunnels, which are used to produce net.Conns out of Sliver custom RPC transport stacks (DNS/HTTPS)
- Network routes, made of session nodes, gateways and of a `Comm` multiplexer.
- Abstracted listeners (TCP/UDP), which track listeners started on implant hosts (reverse)
- Abstracted dialers (TCP/UDP), for the same purpose but by dialing (bind)
- Direct/reverse TCP/UDP port forwarders mapped to their console clients.
- And utilities...

This package goes hand-in-hand with the `commpb` Protobuf package, where all messages and types used are defined.
Links to the other `comm` packages: 
- [Client](https://github.com/maxlandon/sliver/tree/Comm/client/comm)
- [Implant](https://github.com/maxlandon/sliver/tree/Comm/sliver/comm)
- [Protobuf Definitions](https://github.com/maxlandon/sliver/tree/Comm/protobuf/commpb)


-----
# Package Structure & Files
-----

The following structured list should hopefully summarize the structure of the Server's `comm` package, and
indicate briefly the role of each of its components. (Files do not exactly match alphabetical orderings...)
It is advised to read this list before digging further in the package code, and to come back any time needed.

### Core
- `comm.go`       - The `Comm` object is used both by implant sessions and by console clients. They use different methods.
- `client.go`     - All methods of the `Comm` used by a client console, for handling proxied/forwarded connections.
- `implant.go`    - Most methods of the `Comm` used by an implant session to handle routed connections.
- `comms.go`      - Thread-safe map of the Comms, and utility functions for piping streams (TCP) or packet (UDP) connections.
- `tunnel.go`     - Pseudo net.Conn used by the implant's RPC layer (Sliver RPC) to set up and run its Comm system (on top of it).
- `interfaces.go` - Utility functions check for checking implant host interfaces

### Protocol-specific
- `conn_tcp.go`     - Stream connections coming back from the implant are generally wrapped into a net.Conn (here a pseudo TCPConn)
- `conn_udp.go`     - UDP connections from/to the implant are generally wrapped with the UDP connection is this file.
- `dial_tcp.go`     - API-exposed methods to Dial TCP anywhere (server/implant) and the implant `Comm` method associated.
- `dial_udp.go`     - API-exposed methods to Dial UDP anywhere (server/implant) and the implant `Comm` method associated.
- `listen_tcp`      - TCP-like listener type, and API-exposed functions.
- `listen_udp`      - UDP connection as listener (streams are handled slightly differently if dial or listen), and API-exposed functions.

### The Comm system as library
- `dialers.go`      - Generic dialers that may be used depending on circumstances, in conjunction with routes or the other available Dial funcs.
- `listen.go`       - When used as a library, the Comm system yields TCP-like (stream) listeners, or Packet/UDP connections treated as such.

### Client Consoles port forwarders
- `portfwd.go`          - All clients' port forwarders have a server mapping/handler. The `forwarder` type is a little interface for this.
- `portfwd_direct_tcp`  - Direct TCP port forwarder
- `portfwd_direct_udp`  - Direct UDP port forwarder
- `portfwd_reverse_tcp` - Reverse TCP port forwarder
- `portfwd_reverse_udp` - Reverse UDP port forwarder

### Network routes
- `route.go`        - The `Route` type maps a network to an implant Comm system.
- `routes.go`       - Route add/remove functions, and the global `routes` map.
- `revolver.go`     - Given current network routes, resolve addresses, IPs, or URLS and return the appropriate route `Comm`.


-----
# Components Descriptions
-----

The sections below include a more precise description of the `comm` package components.

## Multiplexing

The routing of connections is possible thanks an SSH session put on top of a `net.Conn` (more fundamentally, an `io.ReadWriteCloser`).
It brings additional encryption, multiplexing capacities and out-of-band request/responses (both at the session level and individual stream level).

There are two ways of getting this `net.Conn`/`io.ReadWriteCloser` object:
- For any implant whose active C2 connection is directly tied to the server (not pivoted) and able to yield a `net.Conn` (TCP, UDP, Named_Pipes, mTLS, QUIC, etc), we use this connection.
- For any implant which C2 stack is custom DNS / HTTPS, we set up a speciall `Tunnel` object, directly tied to some Session RPC handlers.

However, as of now, the Comm system is set up on top of our custom RPC tunnels, for being used transparently through any of the customs C2 stacks (proceduraly-generated HTTPS & DNS).
However, it shoul soon be possible to make use of it directly through net.Conns yielded by other C2 stacks (Mutual being only one example, like are WebSockets, Socks5, QUIC, etc).

## Routes

Similar to Metasploit, the Comm system allows to route (forward, by nature) connections to networks that are only
accessible to implants' hosts. Routes are independent of the Transport/Application protocol stack used by implants.

This will allow us to route TCP/UDP/IP traffic through our implants, so any application-layer traffic as well, theoretically.
Some possible use cases: scans, exploits, automatic host detection of any dialer/listener (bind/reverse handler in Metasploit words).

## Dialers / Listeners

The Comm system thus gives access to some Dial methods (or/and predefined Dialers, with specific timeouts & keepalives depending on use)
These dialers obtain a connection (always transport-level, like TCP/UDP/IP) that has been routed all the way to its destination address, as resolved
by the Comm system (considering available routes and/or the C2 server's network interfaces)

Listeners work on the same principle, with the same transport protocols, but reverse. A detail is that these listeners all have precise IDs,
and that all connections they accept and reverse route back to the server are tagged with this ID, for being recognized and processed by the server.

Use cases of listeners, for instance, are the portfwd commands, handlers (dialers/listeners) modules, etc.

## Concurrency, Connection Closing & Error handling

A few important notes on this:
- The Comm system internally makes use of Protobuf messages (like a RPC) through the SSH Request feature.
- Are concerned by these internal requests: TCP/UDP/IP "handlers" open/close, and their connections. 
- When an implant disconnect, all of its port forwarders, listeners, connections of any type are closed.

## Similarities with Metasploit (Pivoting)

As a consequence of the `comm` package structure, it is very similar to Metasploit likely-named `Comm Subsystem`:
- Their is a one-to-one relationship between a Session `Comm` and a route. A simple mapping.
- The actual real job of the route is to slightly modify string values in various objects (handlers, connections, sessions, etc.)
- Because the implant passes raw streams, we can wrap these streams with any encryption we want, without the implant needing the coresponding code.
