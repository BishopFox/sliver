# client/command/shell

## Overview

Implements the 'shell' command group for the Sliver client console. Handlers map Cobra invocations to shell workflows such as filter reader generic and filter reader windows.

## Go Files

- `commands.go` – Exposes the shell command group (`shell`, `shell ls`, `shell attach`) and binds options.
- `completers.go` – Shell-specific command completion helpers.
- `filter-reader_generic.go` – Provides output filtering for shell streams on POSIX targets.
- `filter-reader_windows.go` – Implements CRLF-aware filtering and decoding for Windows shell sessions.
- `ioloop.go` – Handles attached shell stdin forwarding, escape handling, and fast local detach/exit behavior.
- `manager.go` – Tracks managed shell tunnels so detached shells can be listed and re-attached.
- `shell.go` – Launches/attaches shell tunnels and coordinates managed shell lifecycle.
