# implant/sliver/encoders/basex

## Overview

Base-N encoder implementations used by both implant and server. Provides reusable conversion tables and encode/decode routines. Runtime components handle basex test for implant-side basex features.

## Go Files

- `basex.go` – Implements generic base-N encoding/decoding with pluggable alphabets.
- `basex_test.go` *(tests)* – Exercises the base-N conversion logic across supported alphabets.
