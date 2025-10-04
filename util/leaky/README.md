# util/leaky

## Overview

Leak detection helpers for debugging resource usage. Implements buffering utilities to capture leaked output. Utilities focus on leakbuf within the leaky package.

## Go Files

- `leakbuf.go` – Implements a buffer wrapper for detecting unexpected writes during tests.
