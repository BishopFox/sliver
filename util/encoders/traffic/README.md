# util/encoders/traffic

## Overview

Traffic encoder interpreter used across server/client. Provides compiler, interpreter, and test helpers for traffic scripts. Utilities focus on compiler, interpreter, testers, and traffic encoder within the traffic package.

## Go Files

- `compiler.go` – Compiles traffic encoder scripts into bytecode.
- `interpreter.go` – Executes traffic encoder bytecode for transformations.
- `testers.go` – Provides helpers for testing encoder scripts.
- `traffic-encoder.go` – High-level API for applying traffic encoders.
