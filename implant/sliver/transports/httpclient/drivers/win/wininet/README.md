# implant/sliver/transports/httpclient/drivers/win/wininet

## Overview

WinINet-backed HTTP driver for Windows implants. Wraps WinINet APIs to satisfy the shared transport interface. Runtime components handle client generic, client windows, cookie, and HTTP for implant-side wininet features.

## Go Files

- `client_generic.go`
- `client_windows.go`
- `cookie.go`
- `generated.go`
- `http.go`
- `request.go`
- `response.go`
- `user32_windows.go`
- `utils_windows.go`
- `wininet.go`
- `wininet_windows.go`
