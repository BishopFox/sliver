# client/credentials

## Overview

Credential management utilities including sniffers and import helpers for operators. Provides parsers, storage helpers, and output formatting for recovered secrets. Core logic addresses sniff within the credentials package.

## Go Files

- `credentials.go` – Implements credential storage, lookup, and formatting helpers for the client.
- `sniff.go` – Provides credential sniffing routines that parse captured outputs into structured records.
- `sniff_test.go` *(tests)* – Validates the sniffing/parsing helpers against sample credential data.
