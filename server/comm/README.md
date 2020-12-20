Comm 
======

The `comm` package contains the code for the Comm System. This comprises, briefly:
- An SSH multiplexer (Comm).
- Network routes, made of session nodes, gateways and of an entry multiplexer.
- Abstracted listeners, which track listeners started on implant hosts (reverse)
- Abstracted dialers, for the same purpose but by dialing (bind)
- Tunnels, which are used to produce net.Conns out of Sliver custom RPC transport stacks (DNS/HTTPS)
- And utilities


### Routes

Similar to Metasploit, the Comm system allows to route (forward, by nature) connections to networks that are only
accessible to implants' hosts. Routes are independent of the Transport/Application protocol stack used by implants.

This will allow us to route TCP/UDP/IP traffic through our implants, so any application-layer traffic as well, theoretically.
Some possible use cases: scans, exploits, automatic host detection of any dialer/listener (bind/reverse handler in Metasploit words).

### Multiplexing

The routing of connections is possible thanks an SSH session put on top of a `net.Conn` (more fundamentally, an `io.ReadWriteCloser`).
It brings additional encryption, multiplexing capacities and out-of-band request/responses (both at the session level and individual stream level).

There are two ways of getting this `net.Conn`/`io.ReadWriteCloser` object:
- For any implant whose active C2 connection is directly tied to the server (not pivoted) and able to yield a `net.Conn` (TCP, UDP, Named_Pipes, mTLS, QUIC, etc), we use this connection.
- For any implant which C2 stack is custom DNS / HTTPS, we set up a speciall `Tunnel` object, directly tied to some Session RPC handlers.

### Pivots

When a handler/listener is started on a pivot and upon registration of the pivoted imlant, depending on transport used, we either setup an SSH
session on top of the pivoted net.Conn, or we just reverse forward the RPC messages (DNS/HTTPS). This would give, briefly:


           |--------DNS 1 (route 3)--------------> |               |--------DNS 1 (route 3)--------------> | IMPLANT (route 3)
**C2 Server**|--------TCPConn 1 (route 1)----------> |               
           |--------UDPConn 1 (route 4)----------> | PIVOT_IMPLANT |--------TCPConn 1 (route 1)----------> | IMPLANT (route 1 & 4)
           |--------TCPConn 2 (route 2)----------> |               |--------UDPConn 1 (route 4)----------> |
           |--------TCP/HTTP 1 (route 2)---------> |                                   
                                                                   |--------TCPConn 2 (route 2)----------> | IMPLANT (route 2)
                                                                   |--------TCP/HTTP 1 (route 2)---------> |


The import point of this crappy diagram is to show that we do not create SSH session inside of an SSH session, inside another, etc...
This might happen, however, when the route being considered has a session gateway whose C2 stack mandates us to use a `Tunnel` object.

### Dialers / Listeners

The Comm system thus gives access to some Dial methods (or/and predefined Dialers, with specific timeouts & keepalives depending on use)
These dialers obtain a connection (always transport-level, like TCP/UDP/IP) that has been routed all the way to its destination address, as resolved
by the Comm system (considering available routes and/or the C2 server's network interfaces)

Listeners work on the same principle, with the same transport protocols, but reverse. A detail is that these listeners all have precise IDs,
and that all connections they accept and reverse route back to the server are tagged with this ID, for being recognized and processed by the server.

Use cases of listeners, for instance, are the portfwd commands, handlers (dialers/listeners) modules, etc.


-----
## Files

A brief explanation of the package files.

- `comm.go`         - All code for the `Comm` type, an SSH multiplexer (setup, start, serve, etc.)
- `comms.go`        - Gathers the global `Comms`, `Tunnels`, and `Listeners` maps, and their Add(), Remove(), Get() methods.
- `connection.go`   - All streams that come out of a Comm are only io.ReadWriteClosers. Based on their attached info, forge a net.Conn with specifics.
- `dialers.go`      - Dialers and Dial() methods to be used by handlers, portforwards and/or routes.
- `interfaces.go`   - Checks network interfaces for implants.
- `listener.go`     - Abstracted listeners that get their connections from the routing system.
- `nodes.go`        - Methods for sending route requests to node Sessions.
- `resolver.go`     - Methods for getting a network route to an address (string, net.IP, or url.URL types)  
- `route.go`        - Defines the Route object, with init(), remove(), close(), and other setup methods.
- `routes.go`       - Glocal map of network routes, and code to add a new Route, or delete an active one.
- `tunnel.go`       - A specific tunnel used as a basis for multiplexing DNS/HTTPS based Sliver connections.
