# implant/sliver/evasion

## Overview

Host evasion helpers to reduce detection on compromised systems. Implements anti-scan routines, parent process spoofing, and related checks. Runtime components handle evasion darwin, evasion linux, and evasion windows for implant-side evasion features.

## Go Files

- `evasion.go` – Provides shared evasion utilities and orchestrates platform-specific checks.
- `evasion_darwin.go` – Contains macOS-specific evasion routines and environment validation.
- `evasion_linux.go` – Implements Linux evasion strategies like debugger and sandbox detection.
- `evasion_windows.go` – Houses Windows evasion features, including AMSI bypass helpers.
