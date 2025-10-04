# implant/sliver/tcpproxy

## Overview

TCP proxy client used by implants for pivoting. Dials upstream pivots, relays streams, and enforces limits.

## Go Files

- `tcpproxy.go` – Implements the implant-side TCP proxy dialer and streaming loops.
