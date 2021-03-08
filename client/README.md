Client
=======

This directory contains the client implementation, pretty much all of this code is also used by the server console. 

__Important:__ No code in this directory can import any code from `server/`.

 * `assets/` - Static asset files and management code (e.g. client configs)
 * `command/` - Command implementations
 * `constants/` - Various shared constant values
 * `core/` - Client state management
 * `help/` - Console help
 * `spin/` - Console spinner library
 * `transport/` - Wires the client to the server
 * `version/` - Version information
 * `main.go` - Entrypoint
