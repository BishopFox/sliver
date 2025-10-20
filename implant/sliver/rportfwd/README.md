# implant/sliver/rportfwd

## Overview

Reverse port forwarding implementation inside the implant. Manages listener registration, channel wiring, and teardown. Runtime components handle portfwd and tunnel writer for implant-side rportfwd features.

## Go Files

- `portfwd.go` – Handles reverse port forward creation and lifecycle management.
- `tunnel_writer.go` – Sends reverse port forward frames back to the server tunnel.
