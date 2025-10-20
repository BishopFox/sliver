# Repository Guidelines

## Project Structure & Module Organization
Sliver is a Go monorepo. Core client code lives in `client/` (CLI, assets, transports) and server code in `server/` (C2 services, assets, configs). `implant/` holds implant payload sources, while `util/` provides shared helpers that must stay server-side. Protocol buffers live under `protobuf/`, and documentation assets are in `docs/`. Keep generated artifacts (e.g., `sliver-client`, `sliver-server`, `.downloaded_assets`) out of commits.

## Build, Test, and Development Commands
Use `make` targets to stay consistent:
- `make` (or `make default`) compiles client/server binaries with vendored modules and tag defaults.
- `make client` and `make linux-amd64` build platform-specific emitters; prefer `GOOS/GOARCH` env vars over editing sources.
- `./go-assets.sh` refreshes client asset bundles; rerun when static files change.
Regenerate protobuf bindings with `make pb`. For quick checks, `GOFLAGS="-tags client" go build ./client/...` is acceptable, but ensure the matching `server` build also succeeds.

## Coding Style & Naming Conventions
Target Go 1.22+ (see `Makefile` validation). Format code with `gofmt` or `goimports` before committing. Keep package imports segregated (stdlib, third-party, internal). Use lower_snake filenames for assets, PascalCase for exported Go symbols, and prefer explicit names (`sessionID`, `operatorConfig`) over abbreviations. Never import `server` packages inside `client`. Avoid introducing CGO unless justified for cross-platform support.

## Testing Guidelines
Run targeted unit tests with Go’s tooling and project tags, e.g. `go test -tags=client,go_sqlite ./client/command/...`. Full coverage uses `./build.py`; expect long runtimes and ensure ample CPU/RAM. Add tests beside code (`*_test.go`) and mirror package paths. When adding transports or implants, verify integration logs under `~/.sliver/logs/` for regressions and clean up fixtures.

## Commit & Pull Request Guidelines
Sign every commit (`git commit -S`) and write imperative, scoped messages ("Fix DNS canary retry" rather than status updates). Keep vendor updates isolated in their own commit. Pull requests should describe the intent, affected modules, testing performed, and reference GitHub issues. Include before/after screenshots only when UI behavior changes (client console). Ensure CI passes and note any required follow-up.

## Security & Configuration Tips
Default to secure settings: avoid `math/rand` for secrets, prefer vetted crypto (AES-GCM, Curve25519) per `CONTRIBUTING.md`. Lock down file permissions for generated assets and redact secrets from logs. When touching configuration, favor secure defaults even if it means additional setup for operators.
