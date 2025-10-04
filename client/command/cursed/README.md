# client/command/cursed

## Overview

Implements the 'cursed' command group for the Sliver client console. Handlers map Cobra invocations to cursed workflows such as cursed chrome, cursed console, cursed cookies, and cursed edge.

## Go Files

- `commands.go` – Exposes the cursed command suite and binds individual subcommands for supported targets.
- `cursed-chrome.go` – Launches or hijacks Chrome debugging sessions, locating executables and wiring remote control.
- `cursed-console.go` – Provides the interactive cursed console experience atop Chrome DevTools sessions.
- `cursed-cookies.go` – Fetches and displays browser cookies from an active cursed session.
- `cursed-edge.go` – Mirrors the Chrome workflow for Microsoft Edge, including process discovery logic.
- `cursed-electron.go` – Drives cursed interactions with arbitrary Electron apps, validating binaries and starting injected sessions.
- `cursed-rm.go` – Removes cursed sessions from the client cache and tears down associated resources.
- `cursed-screenshot.go` – Captures screenshots from a cursed browser context via DevTools APIs.
- `cursed.go` – Manages the cursed process registry, selection helpers, and remote debugger configuration defaults.
