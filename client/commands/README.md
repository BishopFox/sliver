Commands
=======

**Package**

On each readline loop of the client, all commands (that are fundamentally structs) are bound to the console.
They are then processed and executed by a command parser. The root `commands` directory contains a generic `BindCommands()`
function, which is used to register sets of commands to the console, depending on various elements of the current context.
For instance, sliver commands are only attached when interacting with an implant.

The `commands.go` file also contains some options applying to all commmands, such as timeouts, and a few utility functions.


**Sub-packages**

The subpackages contain the actual command implementations, classified by context, and/or by platform if commands are Sliver ones.
In each of these packages, a similar `BindCommands()` function is declared, and also performs some conditional code for some commands
being registered (for instance, mTLS listeners may be available in either server/sliver or both contexts, if we were to do advanced pivoting)

 * `server/`        - All commands that have no effects on a Sliver session, or pertaining to C2 transports, in here.
 * `sliver/`        - All session commands, including a directory for Windows-based host commands.
 * `transports/`    - All C2 channels/transports listeners/dialers are separated, because they might be available in both contexts.
