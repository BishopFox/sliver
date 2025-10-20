# implant/sliver/encoders/traffic

## Overview

Programmable traffic encoders compiled into the implant for obfuscation. Contains compiler, interpreter, and test harness logic for traffic scripts. Runtime components handle compiler, interpreter, and traffic encoder for implant-side traffic features.

## Go Files

- `compiler.go` – Compiles traffic encoder scripts into executable bytecode for implants.
- `interpreter.go` – Executes compiled traffic encoder bytecode at runtime.
- `traffic-encoder.go` – Provides high-level APIs to apply compiled traffic encoders to payloads.
