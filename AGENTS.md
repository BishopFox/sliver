# Repository Guidelines

## Project Structure & Module Organization
- `client/` holds the CLI client and interactive commands.
- `server/` contains the server runtime, C2 logic, configs, and web assets.
- `implant/` is the implant runtime and transport implementations.
- `protobuf/` stores protobuf definitions and generated Go types.
- `util/` is shared helpers (crypto, encoders, minisign, etc.).
- `docs/sliver-docs/` is the Next.js documentation site and prebuild scripts.
- `vendor/` is the vendored Go dependency tree used by builds.
- Build outputs land in the repo root as `sliver-client*` and `sliver-server*`.

## Build, Test, and Development Commands
- `make` builds server + client using vendored modules and runs `go-assets.sh`.
- `make client` builds a CGO-free client; `make debug` keeps symbols.
- `make pb` regenerates protobuf stubs from `protobuf/`.
- `./go-tests.sh` runs the curated Go test suite with build tags.
- Docs site: `cd docs/sliver-docs && npm install && npm run dev` for local dev; `npm run build` for a static build.
- Go toolchain: the Makefile enforces Go >= 1.24.

## Coding Style & Naming Conventions
- Run `gofmt` on all Go code; keep package names lowercase and idiomatic.
- Avoid `CGO` and empty interfaces; prefer explicit types and interfaces.
- Do not import `server` packages from `client`.
- If `math/rand` is required, import it as `insecureRand` and avoid security use.

## Testing Guidelines
- Go tests follow standard `_test.go` naming within each package.
- Prefer `docker build --target test --build-arg GO_TESTS_FLAGS=--skip-generate .` 
- Add or update tests when behavior changes; some server packages run longer tests.

## Commit & Pull Request Guidelines
- Commits must be signed for PRs to `master`; work in a feature branch.
- Commit subjects are short and imperative (e.g., "Fix ...", "Add ...", "Update ...", "Bump ...").
- Changes to `vendor/` should be in a distinct commit.
- PRs should link the tracking issue, note tests run, and confirm `gofmt` plus unit tests as applicable.

## Security Notes
- Default to secure configuration; fail closed rather than open.
- Avoid insecure algorithms (MD5/SHA1/AES-ECB); prefer AES-GCM and modern curves.
- Treat user input as untrusted: canonicalize paths and use restrictive file permissions.
