This document describes the technical design of Sliver.

## High Level

There are four major components to the Sliver ecosystem:

- **Server Console -** The server console is the main interface, which is started when you run the `sliver-server` executable. The server console is a superset of the client console. All code is shared between the client/server consoles except server-specific commands related to client (i.e., operator) management. The server console talks over an in-memory gRPC interface to the server.
- **Sliver Server -** The Sliver server is also part of the `sliver-server` executable and manages the internal database, starts/stops network listeners (such as C2 listeners, though there are other types). The main interface used to interact with the server is the gRPC interface, thru which all functionality is implemented. By default the server will only start an in-memory gRPC listener that can only be communicated with from the server console. However, the gRPC interface can also be exposed to the network (i.e., multiplayer mode) over mutual TLS (mTLS).
- **Sliver Client -** The client console is the primary user interface that is used to interact with the Sliver server. Note that most of the code in the server console is actually the client console. The client console can also be compiled into a separate client binary file `sliver-client`, which is generally used to connect to the "mutliplayer" gRPC network listener.
- **Implant -** The implant is the actual malicious code run on the target system you want remote access to.

```
             In         ┌───────────────┐ C2
┌─────────┐  Memory     │               │ Protocol ┌─────────┐
│ Server  ├────────────►│ Sliver Server ├─────────►│ Implant │
│ Console │             │               │          └─────────┘
└─────────┘             └───────────────┘
                               ▲
                               │
                               │gRPC/mTLS
                               │
                          ┌────┴────┐
                          │ Sliver  │
                          │ Client  │
                          └─────────┘
```

By implementing all functionality over this gRPC interface, and only differing the in-memory/mTLS connection types the client code doesn't "know" if it's running in the server console or the client console. Due to this, a single command implementation will work in both the server console and over the network in multiplayer mode.
