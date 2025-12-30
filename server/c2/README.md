# server/c2

## Overview

Command-and-control service wiring exposed by the server. Manages transport listeners, job dispatch, and staging pipelines. Key routines cover C2 profile, DNS, HTTP, and jobs within the c2 subsystem.

## Go Files

- `c2_test.go` *(tests)* – Tests core C2 orchestration behaviors.
- `c2profile.go` – Manages C2 profile metadata and binding to listeners.
- `dns.go` – Implements DNS listener management and query handling.
- `dns_test.go` *(tests)* – Validates DNS C2 behavior.
- `http.go` – Manages HTTP/HTTPS listener lifecycle and request routing.
- `http_test.go` *(tests)* – Tests HTTP C2 operations.
- `jobs.go` – Tracks running C2 jobs and listener status reporting.
- `mtls.go` – Configures mTLS listeners and certificate usage.
- `tcp-stager.go` – Provides TCP stager listener logic for payload delivery.
- `wireguard.go` – Handles WireGuard listener setup and management.
