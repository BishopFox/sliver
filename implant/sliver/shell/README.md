# implant/sliver/shell

## Overview

Interactive shell functionality exposed by the implant. Coordinates IO loops, PTY integration, and command execution. Runtime components handle shell PTY, shell generic, and shell windows for implant-side shell features.

## Go Files

- `shell-pty.go`
- `shell.go`
- `shell_generic.go`
- `shell_windows.go`

## Sub-packages

- `pty/` – Pseudo-terminal management for implant shells. Handles PTY allocation, resizing, and IO forwarding.
- `ssh/` – SSH-based shell transport helpers. Establishes SSH-backed command execution channels for implants.
