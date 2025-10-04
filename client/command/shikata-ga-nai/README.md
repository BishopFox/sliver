# client/command/shikata-ga-nai

## Overview

Implements the 'shikata-ga-nai' command group for the Sliver client console. Handlers map Cobra invocations to shikata ga nai workflows such as SGN.

## Go Files

- `commands.go` – Registers shikata-ga-nai encoder commands for payload generation.
- `sgn.go` – Interfaces with the SGN service to build XOR polymorphic payloads and prints results.
