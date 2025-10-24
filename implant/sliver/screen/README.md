# implant/sliver/screen

## Overview

Screen capture helpers for the implant. Captures framebuffer data, encodes it, and returns results to operators. Runtime components handle screenshot generic, screenshot linux, and screenshot windows for implant-side screen features.

## Go Files

- `screen.go` – Coordinates screenshot capture requests and data packaging.
- `screenshot_generic.go` – Provides fallback screenshot logic for unsupported platforms.
- `screenshot_linux.go` – Implements Linux framebuffer capture routines.
- `screenshot_windows.go` – Uses Windows APIs to capture the desktop buffer.
