# implant/sliver/taskrunner

## Overview

Task scheduler running queued implant jobs. Controls execution order, retries, and task telemetry. Runtime components handle dotnet windows, task, task darwin, and task linux for implant-side taskrunner features.

## Go Files

- `dotnet_windows.go`
- `task.go`
- `task_darwin.go`
- `task_linux.go`
- `task_windows.go`
