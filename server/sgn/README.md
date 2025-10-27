# server/sgn

## Overview

SGN (Sliver Guard Node) coordination and helpers. Implements SGN enrollment, messaging, and policy logic.

## Go Files

- `sgn.go` – Implements SGN coordination logic and message handling, including the Shikata Ga Nai encoder helpers.
- `sgn_test.go` – Unit tests covering SGN configuration wiring and helper utilities.

## SGN Encoder Helpers

The server helper wraps the [`github.com/moloch--/sgn`](https://github.com/moloch--/sgn) encoder and exposes a simple `SGNConfig` with the following knobs:

- `Architecture` – `386`/`amd64` (case-insensitive) selection passed to `sgn.NewEncoder`.
- `Iterations` – number of encode passes mapped to `Encoder.EncodingCount`.
- `MaxObfuscation` – byte budget forwarded to `Encoder.ObfuscationLimit`.
- `PlainDecoder` – keep the decoder stub in clear text.
- `Safe` – enable register preservation via `Encoder.SaveRegisters`.
- `BadChars` / `Asci` – optional post-processing filters that brute force new seeds until constraints pass.

These options mirror the upstream CLI flags so server-side tasks can reuse the same behavior.

## Test Fixtures

Shellcode fixtures used by the unit tests live under `testdata/` with a `.bin` extension. They are produced via `msfvenom` using a dedicated Go generator:

```bash
go generate ./server/sgn
```

The generator invokes `msfvenom` three times (reverse TCP/HTTP stagers and an exec payload) and writes raw shellcode into the `testdata` directory. Ensure the Metasploit framework is installed and `msfvenom` is on `$PATH` before running the generation step.

## Testing

The log subsystem expects a writable Sliver root directory. Point it to a temporary location when running tests:

```bash
export SLIVER_ROOT_DIR=$(pwd)/.tmp-sliver
export GOCACHE=$(pwd)/.tmp-gocache
mkdir -p "$SLIVER_ROOT_DIR" "$GOCACHE"
go test ./server/sgn
```

The test suite focuses on option wiring and constraint helpers rather than the full stochastic encoding pipeline.
