# client/command/shell

## Overview

Implements the 'shell' command group for the Sliver client console. Handlers map Cobra invocations to shell workflows such as filter reader generic and filter reader windows.

## Go Files

- `commands.go` – Exposes the interactive shell command and binds its options.
- `filter-reader_generic.go` – Provides output filtering for shell streams on POSIX targets.
- `filter-reader_windows.go` – Implements CRLF-aware filtering and decoding for Windows shell sessions.
- `shell.go` – Launches interactive command shells over RPC and manages IO loops.
