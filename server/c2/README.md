# server/c2

## Overview

Command-and-control service wiring exposed by the server. Manages transport listeners, job dispatch, and staging pipelines. Key routines cover C2 profile, DNS, HTTP, and jobs within the c2 subsystem.

## Go Files

- `c2_test.go` *(tests)*
- `c2profile.go`
- `dns.go`
- `dns_test.go` *(tests)*
- `http.go`
- `http_test.go` *(tests)*
- `jobs.go`
- `mtls.go`
- `tcp-stager.go`
- `wireguard.go`
