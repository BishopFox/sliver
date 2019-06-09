Server
======

This directory contains the Sliver server implementation, and is strucutred as follows:

 * `assets/` - Static assets embedded in the server binary, and methods for manipulating these assets.
 * `c2/` - The server-side command and control implementations
 * `certs/` - X509 certificate generation and management code
 * `console/` - Server specific console code, the majority of the Sliver console code is in `/client/console`
 * `core/` - Data structures and methods that manage connection state from implants, clients, etc.
 * `cryptography/` - Cryptography code and wrappers around a few of Go's standard `crypto` APIs
 * `encoders/` - Data encoders and decoders
 * `generate/` - This package generates the implant executables and shared libraries
 * `gobfuscate/` - Compile-time obfuscation library
 * `gogo/` - Go wrappers around the Go compiler toolchain
 * `handlers/` - Methods invokable by Slivers without user interaction
 * `log/` - Wrappers around Logrus
 * `msf/` - Metasploit helper functions
 * `rpc/` - Remote procedure call implementations, generally called by the `/client/` code
 * `transport/` - Code that wires the server to the `/client`
 * `website/` - Code that manages static content to host on HTTP(S) C2 domains
 * `main.go` - Entrypoint
