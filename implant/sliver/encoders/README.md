# implant/sliver/encoders

## Overview

Binary and text encoders embedded with the implant for staging and comms. Hosts encoder registries, factories, and runtime switching logic. Runtime components handle base32, base58, base58 alphabet, and base58 genalphabet for implant-side encoders features.

## Go Files

- `base32.go`
- `base58.go`
- `base58_alphabet.go`
- `base58_genalphabet.go`
- `base64.go`
- `encoders.go`
- `english.go`
- `gzip.go`
- `hex.go`
- `images.go`
- `nop.go`

## Sub-packages

- `basex/` – Base-N encoder implementations used by both implant and server. Provides reusable conversion tables and encode/decode routines.
- `traffic/` – Programmable traffic encoders compiled into the implant for obfuscation. Contains compiler, interpreter, and test harness logic for traffic scripts.
