Server
======

This directory contains the Sliver server implementation, and is structured as follows:

 * `assets/` - Static assets embedded in the server binary, and methods for manipulating these assets.
 * `c2/` - The server-side command and control implementations
 * `certs/` - X509 certificate generation and management code
 * `cli` - The command line interface implementation
 * `configs/` - Configuration file parsers
 * `console/` - Server specific console code, the majority of the Sliver console code is in `/client/console`
 * `core/` - Data structures and methods that manage connection state from implants, clients, etc.
 * `cryptography/` - Cryptography code and wrappers around a few of Go's standard `crypto` APIs
 * `daemon/` - Method for starting the server as a daemon
 * `db/` - Database client, helper functions, and ORM models
 * `generate/` - This package generates the implant executables and shared libraries
 * `gogo/` - Go wrappers around the Go compiler tool chain
 * `handlers/` - Methods invoke-able by implants without user interaction
 * `log/` - Wrappers around Logrus
 * `loot/` - Server's local 'loot' implementation 
 * `msf/` - Metasploit helper functions
 * `netstack/` - WireGuard server network stack
 * `rpc/` - Remote procedure call implementations, generally called by the `/client/` code
 * `transport/` - Code that wires the server to the `/client`
 * `watchtower/` - Code that monitors threat intel platforms for implants
 * `website/` - Code that manages static content to host on HTTP(S) C2 domains
 * `main.go` - Entrypoint
