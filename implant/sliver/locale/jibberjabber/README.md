# implant/sliver/locale/jibberjabber

## Overview

Embedded locale strings provided by jibberjabber. Mirrors upstream localization data bundled into the implant. Runtime components handle jibberjabber unix and jibberjabber windows for implant-side jibberjabber features.

## Go Files

- `jibberjabber.go` – Provides locale detection utilities backed by the jibberjabber data set.
- `jibberjabber_unix.go` – Unix-specific wrappers for querying locale information.
- `jibberjabber_windows.go` – Windows-specific locale detection implementations.
