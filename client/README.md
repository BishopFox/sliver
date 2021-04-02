Client
=======

This directory contains the client implementation, pretty much all of this code is also used by the server console. 

__Important:__ No code in this directory import any code from `server/`.

 * `assets/`            - Configuration object for the client itself (prompts, inputs, etc) and for connecting to a Sliver server.
 * `commands/`          - Command binding (directory) and implementations, per context (subdirectories) (**with README**)
 * `completers/`        - The full completion/hint/syntax processing engine of Sliver, wired to its readline dependency (**with README**)
 * `console/`           - Core of the console (setup, run, prompts, event loop, history sources, etc) (**with README**)
 * `constants/`         - Various shared constant values
 * `context/`           - Multiple packages (commands, console, completers) need access to some shared elements. Avoids circular imports.
 * `help/`              - Console help strings and custom printers, for commands, subcommands and their options.
 * `licenses/`          - License strings for some dependencies
 * `log/`               - The client has access to some logging infrastructure, partly wired to the server. (**with README**)
 * `spin/`              - Console spinner library
 * `transport/`         - Wires the client to the server
 * `version/`           - Version information
 * `main.go`            - Entrypoint
