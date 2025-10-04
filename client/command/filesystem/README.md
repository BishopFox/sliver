# client/command/filesystem

## Overview

Implements the 'filesystem' command group for the Sliver client console. Handlers map Cobra invocations to filesystem workflows such as CAT, CD, chmod, and chown.

## Go Files

- `cat.go` – Downloads remote files and prints contents locally with optional ANSI colorization.
- `cd.go` – Changes the working directory of the active session or beacon.
- `chmod.go` – Adjusts remote file permissions via RPC for Unix-like targets.
- `chown.go` – Alters remote file ownership metadata where supported.
- `chtimes.go` – Updates remote file access and modification timestamps.
- `commands.go` – Assembles the filesystem command group and wires each file operation subcommand.
- `cp.go` – Copies files on the remote target, handling overwrite prompts and path validation.
- `download.go` – Retrieves files from the target and writes them to the local loot directory.
- `grep.go` – Runs server-side grep equivalents with pattern, highlight, and streaming options.
- `head.go` – Streams the first bytes or lines of a remote file.
- `ls.go` – Lists directory contents with filtering, sorting, and long-format support.
- `memfiles-add.go` – Uploads file data into in-memory implants for later retrieval.
- `memfiles-list.go` – Enumerates in-memory file entries stored on the implant.
- `memfiles-rm.go` – Removes staged in-memory files from the implant cache.
- `mkdir.go` – Creates directories on the remote system, with recursive creation if requested.
- `mount.go` – Mounts remote filesystem paths for live browsing through the client.
- `mv.go` – Moves or renames remote files and directories.
- `pwd.go` – Prints the current working directory of the active target.
- `rm.go` – Deletes remote files or directories with optional recursive behavior.
- `upload.go` – Transfers local files to the target and sets desired permissions.
