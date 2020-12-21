Sliver Comm 
======


The sliver-side `comm` package is quite similar to its server-side equivalent with respect to:
- An SSH multiplexer (Comm).
- Tunnels, which are used to produce net.Conns out of Sliver custom RPC transport stacks (DNS/HTTPS)
- This README follows the same structure as the server comm README.

However it has some differences due to the fact that an implant might have to act as the C2 server, when being a pivot.


### Routes

The route code is smaller, as we just have to save them to a map (not much checks done, the server did them).

### Multiplexing

The `Comm` type has methods for starting an SSH connection as client (normal) or as server (pivot).

### Dialers / Listeners

The implant may be mandated to start listeners or dialers on behalf of the server, and to pass these connections back to it.
Therefore all dialers and listeners started also have their reference IDs, with which they tag their connections before piping.

### Pivots

Some changes and infos to come here, as pivoting might be influenced by many things in the `transports` and `comm` packages,
as well as the underlying C2 stack used by the pivoted implant.


-----
## Files

- `comm-client.go`  - Code for setting up and handling the Comm system when we are a directly talking to the C2 server.
- `comm-pivot.go`   - Comm system when we are a pivot.
- `comm.go`         - All code for the `Comm` type, an SSH multiplexer (setup, start, serve, etc.)
- `comms.go`        - Gathers the global `Comms`, `Tunnels` maps, and their Add(), Remove(), Get() methods.
- `route.go`        - Defines the Route object, some of its methods for conn handling, and the `Routes` map.
- `tunnel.go`       - A specific tunnel used as a basis for multiplexing DNS/HTTPS based Sliver connections.
- `tcp.go`          - Handles TCP traffic to be routed on the implant' host network. 
- `udp.go`          - Handles UDP traffic to be routed on the implant' host network.
