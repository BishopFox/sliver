# implant/sliver/encoders

## Overview

Binary and text encoders embedded with the implant for staging and comms. Hosts encoder registries, factories, and runtime switching logic. Runtime components handle base32, base58, base58 alphabet, and base58 genalphabet for implant-side encoders features.

## Go Files

- `base32.go` – Implements Base32 encoding/decoding routines used by the implant.
- `base58.go` – Provides Base58 encode/decode helpers leveraging custom alphabets.
- `base58_alphabet.go` – Defines the canonical Base58 alphabet tables.
- `base58_genalphabet.go` – Generates Base58 alphabet permutations at build time for obfuscation.
- `base64.go` – Wraps Base64 encoding utilities with implant-specific helpers.
- `encoders.go` – Registers available encoders and exposes lookup functions for runtime selection.
- `english.go` – Implements an English word encoder for human-readable payloads.
- `gzip.go` – Handles gzip compression and decompression helpers.
- `hex.go` – Provides hexadecimal encoding utilities.
- `images.go` – Embeds steganographic encoders that hide data inside images.
- `nop.go` – Supplies a passthrough encoder that leaves data untouched.

## Sub-packages

- `basex/` – Base-N encoder implementations used by both implant and server. Provides reusable conversion tables and encode/decode routines.
- `traffic/` – Programmable traffic encoders compiled into the implant for obfuscation. Contains compiler, interpreter, and test harness logic for traffic scripts.
