# server/assets/traffic-encoders

## Overview

Managed traffic encoder templates bundled with the server. Provides default encoder definitions and related tests. Key routines cover traffic encoder test within the traffic encoders subsystem.

## Go Files

- `traffic-encoder_test.go` *(tests)* – Validates that the same encoder artifact runs in both the server and implant runtimes, including TCP, UDP, DNS, HTTPS, reactor initialization, cancellation, and ABI validation.

`testdata/network-encoder` is a Go WASI reactor compiled with
`sliver-wasm-go -buildmode=c-shared`. It uses only standard-library networking;
the wrapper converts those calls into the shared `sliver_wasi_net_v1` imports.
