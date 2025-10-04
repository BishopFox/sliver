# client/command/c2profiles

## Overview

Implements the 'c2profiles' command group for the Sliver client console. Handlers map Cobra invocations to c2profiles workflows such as C2 profiles.

## Go Files

- `c2profiles.go` – Implements listing, import/export, and generation routines for HTTP C2 profiles, including protobuf/JSON conversion helpers.
- `commands.go` – Binds the c2profiles command suite with flag handling and completions for profile management operations.
