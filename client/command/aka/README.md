# client/command/aka

## Overview

Implements the 'aka' command group for the Sliver client console. "Aka"s are
command aliases (like 'alias' in 'bash') that allow operators to specify
shorthand commands and default arguments for commonly used commands.

### Examples:

- `aka create bg background`: Allows an operator to run `bg` from the implant 
menu to quickly move back to the server menu
- `aka create execw execute -o "C:\Windows\System32\cmd.exe" /c`: Shorthand for
  executing binaries on Windows targets like: `execw arp -a`

## Go Files

- `commands.go` – Wires Cobra commands for listing, creating, and deleting command aliases in the client console.
- `create.go` – Handles creating command aliases to use by the client.
- `delete.go` – Removes command aliases currently tracked by the client. 
- `hook.go` – The main hooking logic to intercept commands ran by the client console. The hook will intercept processing before hitting the Cobra tree to transform the command alias to a full command with default arguments from the command alias and user-provided arguments to the alias.
- `list.go` - Lists the current command aliases tracked by the client.
- `storage.go` - Supports the saving of aliases to disk to keep between runs of the client. 
