# Repository Guidelines

## Build, Test, and Development Commands
- `make` builds server + client using vendored modules and runs `go-assets.sh`.
- `make client` builds a CGO-free client; `make debug` keeps symbols.
- `make pb` regenerates protobuf stubs from `protobuf/`.
- `./go-tests.sh` runs the curated Go test suite with build tags.
- Docs site: `cd docs/sliver-docs && npm install && npm run dev` for local dev; `npm run build` for a static build.

## Coding Style & Naming Conventions
- Run `gofmt` on all Go code; keep package names lowercase and idiomatic.
- Avoid `CGO` and empty interfaces; prefer explicit types and interfaces.
- Do not import `server` packages from `client`.

## Testing Guidelines
- Prefer `docker build --target test --build-arg GO_TESTS_FLAGS=--skip-generate .` 
- Add or update tests when behavior changes; some server packages run longer tests.

