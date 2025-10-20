# implant/sliver/transports/httpclient/drivers/win/wininet

## Overview

WinINet-backed HTTP driver for Windows implants. Wraps WinINet APIs to satisfy the shared transport interface. Runtime components handle client generic, client windows, cookie, and HTTP for implant-side wininet features.

## Go Files

- `client_generic.go` – Provides platform-independent scaffolding shared by WinINet clients.
- `client_windows.go` – Windows-specific WinINet client wiring and initialization.
- `cookie.go` – Implements the custom cookie jar for WinINet sessions.
- `generated.go` – Auto-generated WinINet constants and structures.
- `http.go` – High-level HTTP client implementation using WinINet handles.
- `request.go` – Builds WinINet request handles from transport metadata.
- `response.go` – Parses WinINet responses and converts them to transport objects.
- `user32_windows.go` – lightweight wrappers for required user32 calls.
- `utils_windows.go` – Miscellaneous helper functions for WinINet interactions.
- `wininet.go` – Links WinINet DLL procedures needed by the driver.
- `wininet_windows.go` – Windows-specific syscall bindings for WinINet APIs.
