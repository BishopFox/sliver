# implant/sliver/locale

## Overview

Localization helpers for implant messages. Supplies translation lookup tables and locale-aware formatting. Runtime components handle locale generic for implant-side locale features.

## Go Files

- `locale.go` – Provides locale detection and message formatting utilities.
- `locale_generic.go` – Fallback implementations used when platform-specific locale support is unavailable.

## Sub-packages

- `jibberjabber/` – Embedded locale strings provided by jibberjabber. Mirrors upstream localization data bundled into the implant.
