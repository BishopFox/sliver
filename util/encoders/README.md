# util/encoders

## Overview

Common encoder implementations shared across binaries. Offers shared registry wiring and helper encoders. Utilities focus on base32, base58, base58 alphabet, and base58 genalphabet within the encoders package.

## Go Files

- `base32.go` – Implements reusable Base32 encoder/decoder functions.
- `base32_test.go` *(tests)* – Tests Base32 encoding routines.
- `base58.go` – Provides Base58 encoding/decoding logic.
- `base58_alphabet.go` – Defines Base58 alphabets used by encoders.
- `base58_genalphabet.go` – Generates Base58 alphabets dynamically.
- `base58_test.go` *(tests)* – Tests Base58 helpers.
- `base64.go` – Supplies Base64 encoding utilities.
- `base64_test.go` *(tests)* – Tests Base64 helpers.
- `encoders.go` – Registers encoders and exposes factory methods.
- `encoders_test.go` *(tests)* – Tests encoder registry behavior.
- `english.go` – Implements an English word encoder for readability.
- `english_test.go` *(tests)* – Tests English encoder output.
- `gzip.go` – Provides gzip compression helpers.
- `gzip_test.go` *(tests)* – Tests gzip helper functions.
- `hex.go` – Hex encoder/decoder utilities.
- `hex_test.go` *(tests)* – Tests hex helpers.
- `images.go` – Encodes data into images for covert transport.
- `images_test.go` *(tests)* – Tests image encoder correctness.
- `nop.go` – Passthrough encoder that returns data unchanged.

## Sub-packages

- `basex/` – Generic base-N encoder primitives. Supplies reusable alphabets and encode/decode logic.
- `traffic/` – Traffic encoder interpreter used across server/client. Provides compiler, interpreter, and test helpers for traffic scripts.
