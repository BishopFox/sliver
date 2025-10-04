# util/encoders

## Overview

Common encoder implementations shared across binaries. Offers shared registry wiring and helper encoders. Utilities focus on base32, base58, base58 alphabet, and base58 genalphabet within the encoders package.

## Go Files

- `base32.go`
- `base32_test.go` *(tests)*
- `base58.go`
- `base58_alphabet.go`
- `base58_genalphabet.go`
- `base58_test.go` *(tests)*
- `base64.go`
- `base64_test.go` *(tests)*
- `encoders.go`
- `encoders_test.go` *(tests)*
- `english.go`
- `english_test.go` *(tests)*
- `gzip.go`
- `gzip_test.go` *(tests)*
- `hex.go`
- `hex_test.go` *(tests)*
- `images.go`
- `images_test.go` *(tests)*
- `nop.go`

## Sub-packages

- `basex/` – Generic base-N encoder primitives. Supplies reusable alphabets and encode/decode logic.
- `traffic/` – Traffic encoder interpreter used across server/client. Provides compiler, interpreter, and test helpers for traffic scripts.
