# Comprehensive end-to-end tests

This suite runs a stock Sliver server and locally generated implants through the multiplayer gRPC API. It exercises every supported command scenario over `mtls`, `wg`, and `http`, once as a session and once as a beacon. DNS is intentionally outside this suite.

## Lifecycle

For each target row, the driver:

1. Creates isolated temporary server, client, home, and target roots. It starts the supplied, unmodified server binary as a loopback-only daemon.
2. Uses the server CLI to create and validate an `e2e-operator` profile for `127.0.0.1`, then connects the custom Go client to the multiplayer listener with mTLS and subscribes to events.
3. Starts localhost mTLS, WireGuard, and HTTP listener jobs.
4. Generates an exact-OS/architecture executable for each transport and mode. Beacons use a ten-second callback interval by default.
5. Runs each implant in its own known filesystem tree and requires the matching session or beacon event before issuing commands.
6. Exercises the catalog in [COVERAGE.md](COVERAGE.md), writes deterministic per-target JSON and Markdown reports, stops only processes and listener jobs created by the test, and removes the isolated working root.

Reports are kept outside the disposable working root. Pass `-results` for a stable location; when it is omitted, the driver creates and logs a preserved temporary results directory.

## Target matrix

`darwin` is Go's name for macOS. The E2E driver must execute on the same OS and architecture as the implant target. A 64-bit server is used on the two 32-bit target runners.

| Target | GitHub runner path | Server architecture |
|---|---|---|
| `darwin/amd64` | `macos-15-intel` | `amd64` |
| `darwin/arm64` | `macos-15` | `arm64` |
| `linux/386` | `linux/386` container under QEMU | `amd64` |
| `linux/amd64` | `ubuntu-24.04` | `amd64` |
| `linux/arm64` | `ubuntu-24.04-arm` | `arm64` |
| `windows/386` | `windows-2022` | `amd64` |
| `windows/amd64` | `windows-2022` | `amd64` |
| `windows/arm64` | `windows-11-arm` | `arm64` |

Every row runs this six-cell transport/mode cross product:

| Transport | Session | Beacon |
|---|---:|---:|
| `mtls` | required | required |
| `wg` | required | required |
| `http` | required | required |

## Safety boundaries

The suite executes real implant commands on the runner, but all mutating fixtures are scoped to resources it creates:

- `HOME`, `USERPROFILE`, `SLIVER_ROOT_DIR`, and `SLIVER_CLIENT_ROOT_DIR` point into the isolated test root. Implants receive an explicit environment allowlist plus isolated home and temporary directories, so host credentials and proxy secrets are not inherited.
- Filesystem writes, `cd`, `rm`, `chmod`, and `chown` are limited to a per-implant tree containing known fixtures and a sentinel that must survive cleanup scenarios.
- Test-created TCP fixtures and non-WireGuard UDP fixtures bind only to loopback. Environment changes use a unique implant-process variable.
- The current WireGuard dependency binds its outer authenticated UDP socket to wildcard addresses even when Sliver is given `127.0.0.1`; the generated implant still connects to `127.0.0.1`, and its C2/key-exchange services remain inside the WireGuard tunnel. An address-aware dependency bind API is required to make this transport strictly loopback-only.
- `Terminate` targets only the helper child started and tracked by the suite. Linux memfd tests close only the descriptor they create.
- Windows registry mutations use a random 128-bit `HKCU\Software\SliverE2E-*` subtree. Cleanup is armed before creation, removes only that known child/root pair, and verifies the root is absent.
- Mount, interface, process, Windows privilege, token-owner, and service checks are read-only inventories.
- Cleanup stops the generated implants, the test server, and listener jobs. It does not delete the explicit results directory.

Do not reuse an externally managed Sliver root or point this harness at a remote daemon.

## Local invocation

Run from the repository root on one of the native matrix targets. The server and driver commands below match the workflow build tags and use vendored modules:

```sh
host_os="$(go env GOOS)"
host_arch="$(go env GOARCH)"
mkdir -p ./e2e-results

go run -buildvcs=false -mod=vendor ./util/cmd/assets
CGO_ENABLED=0 go build -buildvcs=false -mod=vendor -trimpath \
  -tags go_sqlite,server -o ./sliver-server-e2e ./server
CGO_ENABLED=0 go test -c -buildvcs=false -mod=vendor -trimpath \
  -tags client,go_sqlite -o ./sliver-comprehensive-e2e ./test/e2e

./sliver-comprehensive-e2e \
  -test.v \
  -test.timeout=0 \
  -repo . \
  -server ./sliver-server-e2e \
  -server-arch "${host_arch}" \
  -target-os "${host_os}" \
  -target-arch "${host_arch}" \
  -results ./e2e-results \
  -transports mtls,wg,http \
  -implant-modes session,beacon
```

On Windows, give both output files an `.exe` suffix. The `linux/386` row is built and run with [Dockerfile.linux-386](Dockerfile.linux-386) because its 386 driver must execute under a 386 userspace. The selector flags can narrow transports or modes for diagnosis, but a narrowed result set is intentionally incomplete when aggregated against the comprehensive catalog.

Per-target output is named `coverage-<os>-<arch>.json` and `coverage-<os>-<arch>.md`.

## Armory trust chain

The supported Armory scenario downloads immutable release assets directly and verifies every pin before loading code:

| Asset | Pinned version | Purpose |
|---|---|---|
| Official Armory index | `v0.0.45` | Locates the expected package keys and repositories |
| COFFLoader | `v1.0.16` | Loads the selected extension artifact |
| CS-Situational-Awareness `sa-env` | `v0.0.28` | Runs the signed BOF and validates its environment output |
| CS-Situational-Awareness `sa-whoami` | `v0.0.28` | Runs a second signed, read-only BOF and validates stable identity fields |

The driver checks the pinned SHA-256 digest and Minisign signature for the index and package archives, validates repository and public-key identities, requires the signed trusted-comment manifest to match the archived manifest byte-for-byte, and selects only an exact OS/architecture artifact. These signed manifests support `windows/386` and `windows/amd64`; every other target, including `windows/arm64`, is an explicit expected `SKIP` in the aggregate report.

## GitHub Actions and reports

The manual workflow is named **Comprehensive e2e Tests**. Its `workflow_dispatch` entry is followed by an authorization job that checks the triggering actor's repository permission and permits only `admin`. GitHub does not provide an admin-only visibility setting for `workflow_dispatch`, so non-admin collaborators may still see or attempt the dispatch; the authorization job prevents the test jobs from starting. Repository branch/ruleset protection must also prevent non-admin changes to the workflow for this gate to be an enforceable trust boundary. The comprehensive target jobs and report aggregation are separate from the `reflektor` job, which calls the existing `Reflektor Integration Test Matrix` workflow unchanged apart from making it reusable.

The aggregate Markdown and JSON also contain an exhaustive disposition registry for every generated `SliverRPC` method. Finite implant commands are marked `COVERED` or `DEFERRED` with a rationale, while server-only, lifecycle, and tunnel/interactive methods are tracked separately. Descriptor-backed tests fail when a new RPC is added without a disposition or when a method marked covered has no scenario in the executable matrix.

To combine downloaded per-target report directories, run:

```sh
go run -buildvcs=false -mod=vendor ./test/e2e/report \
  -input ./coverage-input \
  -output ./coverage-summary
```

The input scan is recursive. It writes `coverage-summary.json` and `coverage-summary.md`; the command exits nonzero for any recorded failure, recorded skip on a supported cell, or required `NOT RUN` cell. Catalog-generated platform `SKIP` cells do not fail aggregation.
