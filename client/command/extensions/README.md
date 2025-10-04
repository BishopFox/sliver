# client/command/extensions

## Overview

Implements the 'extensions' command group for the Sliver client console. Handlers map Cobra invocations to extensions workflows such as argparser, install, list, and load.

## Go Files

- `argparser.go` – Builds argument buffers for extensions, including BOF-specific marshaling and validation.
- `commands.go` – Declares the extensions command tree and binds subcommands for list, install, load, and remove actions.
- `extensions.go` – Manages installed/temporary extension manifests, printing metadata and supplying completers.
- `extensions_test.go` *(tests)* – Verifies manifest parsing and conversion routines to guard against format regressions.
- `install.go` – Installs extension bundles from directories or archives, handling overwrite prompts and asset extraction.
- `list.go` – Queries the server for matching extensions and renders results categorized by hash or type.
- `load.go` – Loads extension binaries for the active target, registers generated commands, and dispatches execution RPCs.
- `remove.go` – Removes installed extensions, coordinating manifest lookups and filesystem cleanup.
