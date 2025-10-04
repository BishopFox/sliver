# implant/sliver/taskrunner

## Overview

Task scheduler running queued implant jobs. Controls execution order, retries, and task telemetry. Runtime components handle dotnet windows, task, task darwin, and task linux for implant-side taskrunner features.

## Go Files

- `dotnet_windows.go` – Handles .NET task execution specifics on Windows.
- `task.go` – Defines the core task runner, queue, and lifecycle logic.
- `task_darwin.go` – macOS adaptations for task execution.
- `task_linux.go` – Linux adaptations for task execution.
- `task_windows.go` – Windows-specific task execution helpers.
