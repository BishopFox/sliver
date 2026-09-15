# Comprehensive end-to-end tests

This suite runs a stock Sliver server and locally generated implants through the multiplayer gRPC API. It exercises every supported command scenario over `mtls`, `wg`, and `http`, once as a session and once as a beacon. DNS is intentionally outside this suite.

## Lifecycle

For each target row, the driver:

1. Creates isolated temporary server, client, home, and target roots. It starts the supplied, unmodified server binary as a loopback-only daemon.
2. Uses the server CLI to create and validate an `e2e-operator` profile for `127.0.0.1`, then connects the custom Go client to the multiplayer listener with mTLS and subscribes to events.
3. Starts localhost mTLS, WireGuard, and HTTP listener jobs.
4. Generates an exact-OS/architecture executable for each transport and mode. Beacons use a ten-second callback interval by default.
5. Runs each implant in its own known filesystem tree and requires the matching session or beacon event before issuing commands.
6. Under the default `comprehensive` scope, exercises the catalog in [COVERAGE.md](COVERAGE.md) and, for every session, also runs a real reverse-port-forward lifecycle against a loopback echo service: authoritative listener inventory, bidirectional relay, repeated and concurrent connections, and revocation after stop. The comprehensive driver writes deterministic per-target JSON and Markdown reports. Under the focused `rportfwd` scope, each selected session runs only that lifecycle, and the workflow preserves its driver log in the target artifact. The manual-only `portfwd-socks5` scope starts the production client tunnel loop and runs forward-port-forward and SOCKS5 traffic through the connected session. All scopes stop only processes and listener jobs created by the test and remove the isolated working root.

Reports are kept outside the disposable working root. Pass `-results` for a stable location; when it is omitted, the driver creates and logs a preserved temporary results directory.

The focused `portfwd-socks5` scope writes `portfwd-socks5-e2e.json` and `portfwd-socks5-e2e.md`. These reports contain every scenario outcome and duration, the exact fuzz seed/case/replay configuration, real HTTP byte counts and SHA-256 digests, and RDP negotiation, NLA, and desktop-activation evidence. HTTP metadata is limited to whether a target was configured and its path-free scheme and authority; the complete URL is never persisted. The reports also contain no RDP domain, username, or password values.

## Target matrix

`darwin` is Go's name for macOS. The E2E driver must execute on the same OS and architecture as the implant target.

| Target | GitHub runner path | Server architecture |
|---|---|---|
| `darwin/amd64` | `macos-15-intel` | `amd64` |
| `darwin/arm64` | `macos-15` | `arm64` |
| `linux/amd64` | `ubuntu-24.04` | `amd64` |
| `linux/arm64` | `ubuntu-24.04-arm` | `arm64` |
| `windows/amd64` | `windows-2022` | `amd64` |
| `windows/arm64` | `windows-11-arm` | `arm64` |

Every row runs this six-cell transport/mode cross product:

| Transport | Session | Beacon |
|---|---:|---:|
| `mtls` | required | required |
| `wg` | required | required |
| `http` | required | required |

Reverse port forwarding is session-only, so its production-path lifecycle runs once for each of the three session transport cells on every target row. The focused `rportfwd` suite scope runs exactly those cells and skips the finite command catalog, beacon generation, report aggregation, Reflektor, and shellcode matrices. It is intended for branch-readiness checks of this feature. The default `comprehensive` scope, including reusable release invocations, retains every existing suite and adds the same reverse-port-forward lifecycle.

Forward port forwarding and SOCKS5 are also session-only. The focused `portfwd-socks5` scope exercises the production `client/core` proxy managers and tunnel stream with deterministic binary boundary payloads, a seeded randomized corpus, sequential and concurrent connections, an exact 8 MiB simultaneous bidirectional stream with a 384 KiB/s one-way throughput floor, idle/resume, stop and isolation lifecycles, a real implant-side dial refusal to a locally valid closed-loopback destination followed by same-session recovery, malformed SOCKS input followed by recovery, authenticated SOCKS5, deterministic HTTP through both Go and the host's real `curl` executable, and teardown of established relays when their session disappears. The sustained stream is deliberately large enough to expose pacing and backpressure regressions rather than accepting a tiny correctness-only echo. `curl` is a required focused-suite prerequisite rather than a silently passing skip. Fixtures bind to loopback, so this first phase is suitable for local and Proxmox guest runs without exposing proxy listeners. Windows RDP-over-tunnel validation remains a Proxmox lab scenario; it is not represented as passing by the loopback traffic suite.

This focused scope does not claim transparent support for every TCP shutdown pattern. Generic tunnels currently represent a terminal close, not independent directional FINs, so an application that requires `CloseWrite` followed by an EOF-delimited response is outside the verified contract. Generic tunnels also share one operator gRPC stream: bounded per-tunnel receive admission prevents one unread local consumer from filling unbounded memory, but a transport-level `Send` that never returns can still stall sibling tunnels on that stream. Those per-tunnel budgets do not impose a per-session or global generic-tunnel count limit; the client-bind lease bounds the lifetime of abandoned creations, not their burst cardinality. The suite covers finite send failures, queue saturation, and sibling isolation after bounded admission failure; directional half-close semantics, isolation from a permanently stuck transport send, and count-based generic-tunnel admission require separate protocol or resource-policy designs.

Current client, server, and implant builds negotiate SOCKS flow control only when both endpoints advertise `SOCKS_FLOW_CONTROL_V1` through the exact session generation. Each direction then permits at most 64 unacknowledged data frames and emits cumulative acknowledgements every 16 frames, after the destination adapter has consumed each complete frame. The 128-frame receive queues retain transport-reordering headroom, acknowledgement controls do not allocate or enter payload queues, and a close wakes an endpoint waiting for credit. A mixed-version pair remains on the legacy wire contract because an absent capability is never inferred.

SOCKS flow control fixes ordinary queue-pressure truncation, but it does not make the shared transport fully independent. All local SOCKS connections still serialize writes through one operator gRPC sender, and implant envelopes still share their C2 connection, so a transport call that never returns can cause cross-tunnel head-of-line blocking. The protocol also has one terminal close rather than independent directional FINs; a sender waiting at a full credit window cannot discover local `CloseWrite` until credit returns. Legacy peers remain memory-bounded but do not receive the negotiated per-tunnel credit guarantee. A fair shared-stream scheduler or separate streams and explicit half-close frames are larger follow-up protocol designs rather than claims of this suite.

The live malformed-input corpus defaults to 32 deterministic cases. Use `-socks-fuzz-seed` and `-socks-fuzz-cases` for a broader manual campaign; the bounded range is 1 through 10,000, and every mutation or in-corpus recovery failure includes its seed and zero-based case. The malformed-scenario deadline accounts for the parser, per-case tunnel drain, and periodic recovery deadlines. `-socks-fuzz-replay-case` accepts indices 0 through 9,999 and runs one exact case plus a valid tunneled round trip and session ping, making a range failure directly replayable without rerunning the preceding corpus or increasing `-socks-fuzz-cases`; the seed and zero-based index uniquely identify the input. Before a mutation reaches the live proxy, the driver parses every complete CONNECT and refuses any destination other than `localhost` or a loopback IP literal.

Two coverage-guided Go fuzz targets exercise the dependency parser and Sliver's bounded server-side admission actor without opening a network connection. Normal `./go-tests.sh --unit-only` runs their seed corpora. For a longer manual campaign, use an isolated cache and one bounded target at a time:

```sh
GOCACHE="$(mktemp -d)" go test -mod=vendor \
  -run='^$' -fuzz='^FuzzSocks5ParseRequest$' -fuzztime=10m \
  ./implant/sliver/handlers/tunnel_handlers

GOCACHE="$(mktemp -d)" go test -mod=vendor \
  -tags=server,osusergo,netgo,go_sqlite \
  -run='^$' -fuzz='^FuzzSocksTunnelAdmissionState$' -fuzztime=10m \
  ./server/core
```

For a range run, `-tunnel-target-http-url` adds a real `curl` request through both forward port forwarding and SOCKS5. The driver retrieves one direct baseline for the entire run, then requires every route and transport to match those shared bytes exactly; report validation also rejects byte counts or SHA-256 digests that disagree across evidence rows. Both response bodies and diagnostic output are memory-bounded. The complete target URL is supplied to each `curl` child through its standard-input config rather than argv, and reports retain only the path-free origin. Userinfo, query strings, and fragments remain rejected.

The literal URL flag remains convenient for a non-secret lab target, but its value is visible in the E2E driver's own argv. Use `-tunnel-target-http-url-fd` when the path is sensitive. The descriptor is read once with a 16 KiB bound and a ten-second deadline, must differ from `-tunnel-rdp-credentials-fd`, and is never reported. For example, pipe a URL-producing command to the driver's standard input and pass `-tunnel-target-http-url-fd 0`. If HTTP and RDP runtime inputs are used together, assign two distinct inherited descriptors (for example 3 and 4). The target is opt-in and is never baked into the portable suite.

`-tunnel-target-rdp-address` adds a credential-free RDP negotiation through both tunnel implementations. It validates the TPKT and X.224 connection-confirm response and rejects an RDP negotiation failure. This proves real bidirectional RDP protocol traffic reaches the Windows service; the Proxmox acceptance run additionally performs a FreeRDP authentication and desktop-session check rather than treating negotiation alone as complete RDP coverage.

On a Linux range driver with FreeRDP 3, Xvfb, and GNU `stdbuf` installed, `-tunnel-rdp-credentials-fd` strengthens each RDP scenario with one full `/sec:nla` desktop connection followed by the credential-free negotiation probe. It does not run a redundant `+auth-only` connection, which can leave Windows 11 session setup in flight and contaminate the desktop check. FreeRDP multitransport is disabled so the graphical phase remains on the TCP-only port-forward or SOCKS5 path instead of negotiating UDP outside it. The desktop check line-buffers FreeRDP's captured console output, incrementally scans for the ordered NLA transition and local and remote framebuffer markers followed by `CONNECTION_STATE_ACTIVE`, retains only a bounded 256 KiB diagnostic tail, starts the acceptance clock at activation, and requires the process tree to remain alive for another 15 seconds; a merely live, slowly handshaking, or stalled process does not pass. That one activated connection records both NLA and desktop evidence. Authenticated RDP therefore requires `-command-timeout` of at least 107 seconds for activation, acceptance, bounded process cleanup, the final negotiation probe, and startup slack; the default remains two minutes. The descriptor must contain one JSON object with `domain`, `username`, `password`, `server_name`, and `certificate_fingerprint` fields. `server_name` is the independently verified Windows DNS identity used for TLS and Kerberos even when portfwd connects FreeRDP to loopback. The fingerprint must be the independently verified RDP server certificate in `sha256:<64 hexadecimal digits>` form; colon-separated digest bytes are accepted and canonicalized. FreeRDP receives a deny-on-mismatch policy plus that exact pin instead of disabling certificate validation. The driver consumes the object once, passes FreeRDP one argument per line through an anonymous child-process descriptor, and never places credentials in argv, reports, or logs. Its process tree also receives only a small runtime allowlist rather than GitHub, SSH-agent, or 1Password environment secrets. Pipe the object to descriptor 0 for a remote manual run; never commit a credential file.

Focused tunnel runs default to `-tunnel-acceptance-profile base`. A base report is explicitly marked as non-acceptance even when all selected diagnostics pass, so narrowing transports or omitting external targets cannot be mistaken for range acceptance. Use `-tunnel-acceptance-profile proxmox` only for the range gate. That profile fails preflight unless the exact `mtls,wg,http` transport matrix, an external HTTP target, an RDP target, and the authenticated certificate-pinned RDP credential descriptor are all configured; report completeness independently enforces the same contract and every resulting evidence row. Reports also record SHA-256 digests of the exact server and E2E driver executables plus the server commit and dirty state returned by RPC. Those fields make the executed artifacts identifiable but do not yet attest that an uncommitted manual build came from the current source tree.

The eight native OPFOR CNA scenarios use the same six-cell cross product on every supported Windows target. They are not smoke tests tied to one listener or implant mode: `mtls`, `wg`, and `http` must each pass as both a session and a beacon. Windows amd64 runs all eight scenarios, while Windows arm64 records catalog-generated `SKIP` cells because the OPFOR BOF provider has no arm64 support. [COVERAGE.md](COVERAGE.md) records the platform boundary.

## Shellcode generation and execution matrix

The reusable workflow **Shellcode E2E Test Matrix** is a separate, native-only matrix for generated shellcode. It has a `workflow_call` trigger only; the administrator-gated **Comprehensive e2e Tests** workflow calls it after its authorization job succeeds.

For every required combination, `TestShellcodeE2E` performs this lifecycle:

1. Creates isolated server and client roots, starts the supplied, unmodified Sliver server binary as a localhost-only daemon, connects the custom Go test client to the multiplayer gRPC service, and subscribes to server events.
2. Starts the selected `mtls`, `wg`, or `http` listener on localhost.
3. Calls the `Generate` gRPC command for the exact target, session or beacon mode, and compression setting, and requires a nonempty `.bin` payload. It then keeps the unencoded `none` case or applies each architecture-supported encoder through the `ShellcodeEncoder` gRPC command; beacons use a ten-second callback interval.
4. Creates and compiles a small native C runner, has it load the `.bin` into executable memory, and executes the shellcode locally.
5. Requires the matching session or beacon event from the server before recording the combination result and cleaning up only the processes, jobs, and isolated roots created by the test.

The four native target rows are fixed. The server, test client, C runner, and shellcode match the target architecture.

| Target | GitHub runner | Server architecture | Shellcode backend | Encoder settings | Required combinations |
|---|---|---|---|---|---:|
| `darwin/arm64` | `macos-26` | `arm64` | `beignet` | `none`, `xor`, `xor_dynamic` | 36 |
| `linux/amd64` | `ubuntu-24.04` | `amd64` | `malasada` | `none`, `shikata_ga_nai`, `xor`, `xor_dynamic` | 48 |
| `linux/arm64` | `ubuntu-24.04-arm` | `arm64` | `malasada` | `none`, `xor`, `xor_dynamic` | 36 |
| `windows/amd64` | `windows-2022` | `amd64` | `wasm-donut` | `none`, `shikata_ga_nai`, `xor`, `xor_dynamic` | 48 |

Every encoder setting in a target row crosses these axes:

| Axis | Values |
|---|---|
| Transport | `mtls`, `wg`, `http` |
| Implant mode | `session`, `beacon` |
| Compression | `none` (disabled), `aplib` (enabled) |

Shellcode encoder support is architecture-based rather than OS-based: `amd64` supports `none`, `shikata_ga_nai`, `xor`, and `xor_dynamic`, while `arm64` supports `none`, `xor`, and `xor_dynamic`. Thus the suite has `(3 + 4 + 3 + 4) × 3 transports × 2 modes × 2 compression settings = 168` required combinations.

The workflow passes `-shellcode-sgn-samples 4`. Each logical `shikata_ga_nai` cell therefore encodes and natively executes four independently randomized SGN outputs from the same generated base payload. Every sample is required: the first failed sample fails the cell, while successful later samples can never turn a failure green. When execution fails, the suite executes the exact same in-memory bytes once more in an isolated diagnostic directory; that replay can explain whether the same bytes fail consistently, but it never changes the failed status and neither payload is uploaded in the coverage artifact. Coverage remains 168 logical combinations rather than counting the nested stability samples as separate cells, while a fully passing workflow performs 240 native executions in total. Target and aggregate reports record `completed_samples` and `required_samples`, so a passing SGN cell is auditable as `4/4` from its artifact rather than only from the job log. Local runs also default to four SGN samples; `-shellcode-sgn-samples` may raise, but not lower, that stability floor.

The aggregate table uses four statuses:

- `PASS`: the supported combination generated, its C runner executed, and the expected server event arrived.
- `FAIL`: a supported combination was attempted but generation, runner compilation or execution, or event verification failed.
- `NOT RUN`: a supported combination has no recorded result, including when an earlier target failure prevented it from running. This fails aggregation just like `FAIL`.
- `N/A`: the encoder is not supported by that target architecture. This is allowed, does not fail aggregation, and is not part of the 168 required combinations.

Each target writes `shellcode-coverage-<os>-<arch>.json` and `shellcode-coverage-<os>-<arch>.md` under `shellcode-results`, then uploads that directory as `shellcode-e2e-target-<os>-<arch>` for 14 days. Aggregation writes `shellcode-summary/shellcode-coverage.json` and `shellcode-summary/shellcode-coverage.md`, uploads the directory as `shellcode-e2e-coverage-summary` for 30 days, appends the Markdown table to the workflow summary, and exposes the same Markdown through the reusable-workflow output `shellcode_coverage_markdown`.

## Safety boundaries

The suite executes real implant commands on the runner, but all mutating fixtures are scoped to resources it creates:

- `HOME`, `USERPROFILE`, `SLIVER_ROOT_DIR`, and `SLIVER_CLIENT_ROOT_DIR` point into the isolated test root. Implants receive an explicit environment allowlist plus isolated home and temporary directories, so host credentials and proxy secrets are not inherited.
- Filesystem writes, `cd`, `rm`, `chmod`, and `chown` are limited to a per-implant tree containing known fixtures and a sentinel that must survive cleanup scenarios.
- Test-created TCP fixtures and non-WireGuard UDP fixtures bind only to loopback. Environment changes use a unique implant-process variable.
- The current WireGuard dependency binds its outer authenticated UDP socket to wildcard addresses even when Sliver is given `127.0.0.1`; the generated implant still connects to `127.0.0.1`, and its C2/key-exchange services remain inside the WireGuard tunnel. An address-aware dependency bind API is required to make this transport strictly loopback-only.
- `Terminate` targets only the helper child started and tracked by the suite. Linux memfd tests close only the descriptor they create.
- Windows registry mutations use a random 128-bit `HKCU\Software\SliverE2E-*` subtree. Cleanup is armed before creation, removes only that known child/root pair, and verifies the root is absent.
- Mount, interface, process, Windows privilege, token-owner, and service checks are read-only inventories.
- Native OPFOR Cat reads only a known file created beneath the per-implant root; it never receives a host path or UNC path. FindDotnet performs read-only process inventory.
- FirefoxDump is potentially credential-bearing, so the harness refuses to execute it if any pre-existing `C:\Users\*\AppData\Roaming\Mozilla\Firefox\Profiles` directory or the current user's `Mozilla\Firefox\Profiles` configuration directory exists. Only after that gate passes, the harness creates `<UserConfigDir>\Mozilla\Firefox\Profiles\sliver-opfor-e2e.empty`; the BOF must report that the synthetic profile has none of the requested Firefox data and that it completed exactly one profile. Cleanup removes only the created empty directory non-recursively. It never examines a pre-existing host profile or exercises password, cookie, or NSS decryption against host data.
- FindSysmon invokes only its read-only `reg` action, and only after `reg.exe` confirms `HKLM\SOFTWARE\Microsoft\Windows\CurrentVersion\WINEVT\Channels\Microsoft-Windows-Sysmon/Operational` returns not-found. A present channel or indeterminate query refuses the scenario, and the privileged `driver` action is never invoked.
- Synthetic OPFOR fixtures use repository-owned source and bounded arguments. Malformed-object and deadline cases must return bounded errors and then pass `Ping` on the same target, proving that a loader failure or timeout did not strand the session or beacon.
- Cleanup stops the generated implants, the test server, and listener jobs. It does not delete the explicit results directory.

Do not reuse an externally managed Sliver root or point this harness at a remote daemon.

## Local invocation

Run from the repository root on one of the native matrix targets. The server and driver commands below match the workflow build tags and use vendored modules:

```sh
host_os="$(go env GOOS)"
host_arch="$(go env GOARCH)"
mkdir -p ./e2e-results

go run -buildvcs=false -mod=vendor ./util/cmd/assets
CGO_ENABLED=0 go build -mod=vendor -trimpath \
  -tags go_sqlite,server -o ./sliver-server-e2e ./server
CGO_ENABLED=0 go test -c -mod=vendor -trimpath \
  -tags client,go_sqlite -o ./sliver-comprehensive-e2e ./test/e2e

./sliver-comprehensive-e2e \
  -test.v \
  -test.run=^TestComprehensiveE2E$ \
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

For a local feature-only run, add either `-suite-scope rportfwd` or `-suite-scope portfwd-socks5`. Both scopes automatically select sessions even if the default `session,beacon` mode list is retained. The latter is intentionally local/Proxmox-only for now: the existing GitHub Actions dispatch input has not been changed to accept it. Its default `base` profile is diagnostic only; add `-tunnel-acceptance-profile proxmox` together with all required range target flags for an acceptance run. In GitHub Actions, `rportfwd` remains the only focused choice on the manually dispatched **Comprehensive e2e Tests** workflow.

Run the shellcode group with the same server and test binary by changing the selector to `-test.run=^TestShellcodeE2E$` and writing results to a separate directory such as `./shellcode-results`. The default four SGN samples match the workflow; `-shellcode-sgn-samples` can request a higher stress depth. Shellcode aggregation expects the complete four-target workflow matrix; a single local target is useful as a runtime smoke test but intentionally reports the other required targets as `NOT RUN`.

On Windows, give both output files an `.exe` suffix. The selector flags can narrow transports or modes for diagnosis, but a narrowed result set is intentionally incomplete when aggregated against the comprehensive catalog.

Comprehensive per-target output is named `coverage-<os>-<arch>.json` and `coverage-<os>-<arch>.md`; the focused tunnel scope uses the report names above.

Focused reports validate an exact per-transport scenario and external-evidence matrix. Missing, duplicate, unexpected, or zero-valued evidence rows make the artifact fail, and report generation happens after suite teardown. JSON and Markdown are staged in the results directory; Markdown is published first and the machine-readable JSON PASS marker last, after any stale JSON has been removed, so a pair-write failure cannot leave a plausible `passed: true` result.

## Armory trust chain

The supported Armory scenario downloads immutable release assets directly and verifies every pin before loading code:

| Asset | Pinned version | Purpose |
|---|---|---|
| Official Armory index | `v0.0.45` | Locates the expected package keys and repositories |
| COFFLoader | `v1.0.16` | Loads the selected extension artifact |
| CS-Situational-Awareness `sa-env` | `v0.0.28` | Runs the signed BOF and validates its environment output |
| CS-Situational-Awareness `sa-whoami` | `v0.0.28` | Runs a second signed, read-only BOF and validates stable identity fields |

The driver checks the pinned SHA-256 digest and Minisign signature for the index and package archives, validates repository and public-key identities, requires the signed trusted-comment manifest to match the archived manifest byte-for-byte, and selects only an exact OS/architecture artifact. The current E2E catalog requires the signed `windows/amd64` artifacts; every other target, including `windows/arm64`, is an explicit expected `SKIP` in the aggregate report.

## Native OPFOR asset provenance

The native OPFOR scenarios never download a mutable branch or an unverified release alias. The CNA regression fixtures retain their upstream revisions and licenses in [the corpus source manifest](../../client/command/opfor/testdata/corpus/SOURCES.md); live object files use these additional immutable pins:

| Asset | Immutable source | Verification and target use |
|---|---|---|
| FirefoxDump | [`sliverarmory/firefoxdump` `v0.0.2` / `19bec6fb2def510b299430955fb2be79c1f51820`](https://github.com/sliverarmory/firefoxdump/tree/19bec6fb2def510b299430955fb2be79c1f51820) | Signed [`firefoxdump.tar.gz`](https://github.com/sliverarmory/firefoxdump/releases/download/v0.0.2/firefoxdump.tar.gz), SHA-256 `76260d331ebb57454d06caf9badb512bc21827146e85f425d6337096facfde7f`; detached [`firefoxdump.minisig`](https://github.com/sliverarmory/firefoxdump/releases/download/v0.0.2/firefoxdump.minisig), SHA-256 `22d2d5adfd40612fad82be927101ec2f0e38795b44e34d0a7b539d72f2ed74dd`; Minisign key `RWQT71u8OryWX4sIY2mwJq/41tKfI6SdZivwRlWLP9pBDn6ijresXI/H`. The selected x64 object SHA-256 is `86df6e65c759ce092d18274dfe17cde17c16d8194d609455c918d2dc0a9e78e5`. |
| Beune Cat x64 | [`sliverarmory/bof_collection` `v0.0.1` / `2cb3fb1b39a96484c4c40b8710c1ca9f83e846ee`](https://github.com/sliverarmory/bof_collection/tree/2cb3fb1b39a96484c4c40b8710c1ca9f83e846ee) | Signed [`beune-bof-collection.tar.gz`](https://github.com/sliverarmory/bof_collection/releases/download/v0.0.1/beune-bof-collection.tar.gz), SHA-256 `d69747b567c69c7ed03ef4ae5b6c1f76e76ae88f40f68987f56c0276db1389d7`; detached [`beune-bof-collection.minisig`](https://github.com/sliverarmory/bof_collection/releases/download/v0.0.1/beune-bof-collection.minisig), SHA-256 `56ef773025fb7cd4b397cedd2d68b73f99bd1c0bd5cbbcc8c73d376ebb765cfd`; Minisign key `RWTFh8KpxR0fDBsOK+FYYSo/SxW9hFEQwqqCa0hv1YLyjVXrbXR/hMdf`. The selected x64 object SHA-256 is `c96c07c85fc3809240f87ab37b1aba40c80dcf1c81caa669bde8f8bddd0815e2`. |
| OperatorsKit FindDotnet | [`sliverarmory/OperatorsKit` `66368f4738528d26cc1ccc6d9a3c93d44d63edc1`](https://github.com/sliverarmory/OperatorsKit/tree/66368f4738528d26cc1ccc6d9a3c93d44d63edc1/KIT/FindDotnet) | The commit-addressed [`finddotnet.o`](https://raw.githubusercontent.com/sliverarmory/OperatorsKit/66368f4738528d26cc1ccc6d9a3c93d44d63edc1/KIT/FindDotnet/finddotnet.o) must match SHA-256 `1dcda8bf8db5851e9fc64b40690d3c03f1f86e59ae4b340520603774acff5508`; amd64 only. |
| OperatorsKit FindSysmon | [`sliverarmory/OperatorsKit` `66368f4738528d26cc1ccc6d9a3c93d44d63edc1`](https://github.com/sliverarmory/OperatorsKit/tree/66368f4738528d26cc1ccc6d9a3c93d44d63edc1/KIT/FindSysmon) | The commit-addressed [`findsysmon.o`](https://raw.githubusercontent.com/sliverarmory/OperatorsKit/66368f4738528d26cc1ccc6d9a3c93d44d63edc1/KIT/FindSysmon/findsysmon.o) must match SHA-256 `c2924e690b4ba407fddcfefdba2632e649008084afb7a9dcf9e11f5853cf3050`; amd64 only. |
| Typed/error/timeout fixtures | Current Sliver source revision | The C fixture is compiled for the exact Windows amd64 target; the malformed fixture is a deterministic three-byte non-COFF input. These local fixtures exercise OPFOR callback semantics and recovery rather than third-party behavior. |

FirefoxDump and Beune Cat are authenticated signed release packages: the driver verifies both the pinned archive/signature digests and the detached Minisign signature before extraction, then checks the selected object digest. OperatorsKit publishes commit-addressed raw objects rather than signed packages, so those two rows require both the full immutable commit URL and exact object SHA-256. Any pin, signature, object architecture, safety preflight, or exact-source compilation mismatch fails before BOF execution.

## GitHub Actions and reports

The manual workflow is named **Comprehensive e2e Tests**. Its `workflow_dispatch` entry currently accepts `comprehensive` (default) or `rportfwd` scope; `portfwd-socks5` is deliberately not wired into a workflow until it is stable on the Proxmox range. The dispatch is followed by an authorization job that checks the triggering actor's repository permission and permits only `admin`. GitHub does not provide an admin-only visibility setting for `workflow_dispatch`, so non-admin collaborators may still see or attempt the dispatch; the authorization job prevents the test jobs from starting. Repository branch/ruleset protection must also prevent non-admin changes to the workflow for this gate to be an enforceable trust boundary. The comprehensive target jobs and report aggregation are separate from the `reflektor` and `shellcode` jobs, which call the reusable `Reflektor Integration Test Matrix` and `Shellcode E2E Test Matrix` workflows after authorization. Reusable callers, including the release workflow, default to `comprehensive`; the focused `rportfwd` dispatch skips the unrelated aggregate and reusable matrices.

The aggregate Markdown and JSON also contain an exhaustive disposition registry for every generated `SliverRPC` method. Finite implant commands are marked `COVERED` or `DEFERRED` with a rationale, while server-only, lifecycle, and tunnel/interactive methods are tracked separately. Descriptor-backed tests fail when a new RPC is added without a disposition or when a method marked covered has no scenario in the executable matrix.

The aggregate job writes the standalone command matrix to `command-coverage.md`, includes it in the `comprehensive-e2e-coverage-summary` artifact, and appends it to the workflow summary after the detailed aggregate and RPC disposition report. The same complete command table is available to downstream jobs through the aggregate job output `command_coverage_markdown`, sourced from the step output of the same name via `$GITHUB_OUTPUT`. Each cell combines every catalog scenario and both session and beacon modes for one gRPC command: `✅` means every required result passed, `❌` means at least one required result failed, was skipped at runtime, or was not run, and `N/A` means the command is unsupported on that OS/architecture.

To combine downloaded per-target report directories, run:

```sh
go run -buildvcs=false -mod=vendor ./test/e2e/report \
  -input ./coverage-input \
  -output ./coverage-summary
```

The input scan is recursive. It writes `coverage-summary.json`, `coverage-summary.md`, and `command-coverage.md`; the command exits nonzero for any recorded failure, recorded skip on a supported cell, or required `NOT RUN` cell. Catalog-generated platform `SKIP` cells do not fail aggregation.
