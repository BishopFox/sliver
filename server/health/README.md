# server/health

## Overview

Daemon-local HTTP health endpoint for liveness probes, bound to loopback by default. Serves `GET /healthz` returning `{"status":"ok"}`; no version or build information is disclosed. The daemon owns the listener lifecycle — this is not a core.Job.

## Go Files

- `health.go` – HTTP server, `/healthz` handler, and listen address validation.
