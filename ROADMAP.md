# Implant build pipeline roadmap

Status: implementation in progress. This document tracks implementation of server-side core and transform specs, build pipelines, and YAML profiles. Check off a milestone only after its exit criteria are demonstrated; update the status and evidence as work lands.

## Scope and decisions

- `builtin/sliver` is the default core. Existing `generate`, `generate beacon`, and `profiles generate` commands remain usable.
- Core and transform specs are versioned YAML files kept on the server. Built-in, administrator-provided, and remote-builder-advertised specs appear in one catalog. The catalog distinguishes **supported** definitions from **currently available** executors.
- A user-provided core initially requires a preinstalled server or builder integration. YAML cannot provide arbitrary commands or upload source code for execution.
- Every initial core speaks the existing Sliver wire protocol and command schema. New runtime protocol adapters and other compiler drivers are future work.
- Specs can describe every Go build mode (`archive`, `c-archive`, `c-shared`, `default`, `exe`, `pie`, `plugin`, `shared`), while each core advertises only outputs it actually implements. The Sliver adapter retains Go with Zig cross-compilation and optional Garble obfuscation. Go build modes and operator-facing output formats are separate fields.
- Profiles store the complete pipeline as server-side YAML. A build freezes its resolved plan and spec digests, while credentials and other per-build secrets are generated separately.
- Remote builder selection is automatic unless a builder is pinned. Among compatible online remote builders with fresh capacity reports, select the highest available memory rounded to the nearest decimal MB (1 MB = 1,000,000 bytes), then the highest effective logical CPU core count, then the lexicographically smallest stable builder name. A pin bypasses ranking but still requires a compatible, online builder.

## Current behavior to preserve

The current generator supports Sliver executables, Windows service executables, shared libraries, and a `c-archive` packaged with headers as a ZIP. Shellcode already has implicit core-build and conversion stages: Windows PIE to Donut, macOS arm64 shared library to Beignet, and Linux amd64/arm64 shared library to Malasada. Shellcode encoders currently run in a separate RPC, sometimes after the server has saved the unencoded artifact.

The Sliver core currently targets Darwin amd64/arm64, Linux 386/amd64/arm64, and Windows 386/amd64/arm64. Its transport inventory includes mTLS, HTTP, HTTPS, DNS, WireGuard, named pipe, and TCP pivot. Named pipe is Windows-only; named pipe and TCP pivot belong to session builds rather than beacon builds. Treat target, mode, protocol, core output, and transform support as valid combinations, not a Cartesian product.

Existing profiles contain one `ImplantConfig`. Remote builders advertise a `sliver` template and compiler targets, remain registered only while connected, and return one final file. The new implementation must migrate those contracts without changing what an existing operator command means.

### Starting points in the repository

| Concern | Current code |
| --- | --- |
| Generate RPC and external build dispatch | [`server/rpc/rpc-generate.go`](server/rpc/rpc-generate.go) |
| Sliver compilation, source rendering, and shellcode conversion | [`server/generate/binaries.go`](server/generate/binaries.go), [`server/generate/donut.go`](server/generate/donut.go) |
| Embedded Sliver source and dependencies | [`implant/implant.go`](implant/implant.go), [`implant/sliver/`](implant/sliver/) |
| Console generation and profiles | [`client/command/generate/`](client/command/generate/) |
| Shellcode encoder implementations | [`server/encoders/shellcode/`](server/encoders/shellcode/) |
| Remote builder registration and execution | [`server/core/builders.go`](server/core/builders.go), [`server/builder/builder.go`](server/builder/builder.go) |
| Protobuf and persistent models | [`protobuf/clientpb/client.proto`](protobuf/clientpb/client.proto), [`protobuf/rpcpb/services.proto`](protobuf/rpcpb/services.proto), [`server/db/models/implant.go`](server/db/models/implant.go), [`server/db/sql.go`](server/db/sql.go) |

## Target contracts

### Specs, artifacts, and providers

- A **CoreSpec** identifies an immutable name/version/digest, runtime protocol, installed executor ID, typed options, valid mode/protocol/target/output combinations, and toolchain needs. A core build returns an artifact with kind, container format where applicable, target OS/architecture, Go build mode where applicable, entry symbols, size, and checksum.
- A **TransformSpec** identifies an immutable name/version/digest, installed executor ID, typed options, and precise input/output artifact rules. Donut, Malasada, Beignet, and each shellcode encoder are separate transforms. An encoder accepts and returns shellcode for its declared architecture.
- A **provider** is the server or a connected remote builder that can execute an exact spec digest for a particular capability tuple. A spec remains listed when its last remote provider disconnects, with availability shown separately.
- A **BuildProfile** contains target, mode, C2 settings, core reference and requested core output, ordered transforms and options, final output, and an automatic or pinned provider policy. The server stores canonical YAML and an immutable resolved snapshot per build. Profile YAML contains no generated private keys.
- A **BuildPlan** names every step, selected artifact type, spec digest, provider policy, and final output. Planning validates all input/output edges and either returns a complete plan or a specific incompatibility or unavailable-provider error.

### Remote capacity and selection

- Builders report available memory in bytes and effective logical CPU cores at registration and on periodic status updates. The server computes and displays integer decimal MB using nearest-MB rounding, and records when it received the report.
- Selection first filters by connection, fresh status, exact spec digest, target, mode/protocol where relevant, input/output type, toolchain, and authorization. It then applies memory, CPU, and stable-name ordering. A pinned builder is checked against the same compatibility and availability rules.
- Freeze the chosen builder and capacity report in the build record. Define the status freshness window and account for in-flight assignments before enabling concurrent automatic scheduling, so repeated requests do not all select a stale high-memory winner.
- Builder advertisements are capability claims, not permission to run arbitrary YAML. Dispatch and upload validate the assigned builder, operator identity, step, spec digest, target, artifact type, size, and checksum.

## Milestones

| Milestone | Deliverable | Depends on | Owner | Status |
| --- | --- | --- | --- | --- |
| M0 | Freeze contracts and baseline behavior | — | Codex | In progress |
| M1 | Server YAML catalog and console inventory | M0 | Codex | In progress |
| M2 | Planner, typed artifacts, and build records | M1 | Codex | In progress |
| M3 | Isolated Sliver core adapter | M2 | Codex | In progress |
| M3.5 | Private source staging and Mage core build handoff | M3 | Codex | In progress |
| M4 | Server-owned transform pipeline | M3.5 | Codex | In progress |
| M5 | Remote step execution and capacity scheduler | M2, M4 | Codex | In progress |
| M6 | Server-side YAML profiles and legacy migration | M2, M4 | Codex | In progress |
| M7 | Non-Sliver core proof and release validation | M5, M6 | Unassigned | Proposed |

### M0 — Contract and baseline

- [x] Record the current generate/profile flags, defaults, aliases, RPC behavior, target and format matrix, and all profile consumers. Include service semantics and the current `archive` = Go `c-archive` ZIP distinction.
- [x] Define versioned YAML schemas for CoreSpec, TransformSpec, and BuildProfile, including canonical names, immutable versions, digests, typed options, capability rules, and strict unknown-field handling.
- [x] Define artifact kinds and container types so PE executable, ELF shared library, Mach-O shared library, Go archive, C archive bundle, and architecture-specific shellcode cannot be confused.
- [x] Define the `sliver-v1` runtime contract and how an alternate core declares the subset of existing commands it implements.
- [x] Set the remote capacity freshness window, CPU/memory measurement rules, in-flight scheduling policy, and deterministic final tie rule.

**Exit:** Reviewed schemas and examples can express every existing Sliver path and every Go build mode without advertising unsupported Sliver outputs. The baseline matrix is an executable acceptance list for later milestones.

**M0 evidence (2026-11-19, local; approved for M1/M2 work):** [Legacy behavior and profile consumers](docs/build-pipeline/legacy-baseline.md), [executable acceptance matrix](docs/build-pipeline/legacy-acceptance.json), [versioned schemas, examples, and digest rules](docs/build-pipeline/README.md), and [artifact, runtime, and capacity contracts](docs/build-pipeline/contracts.md). `python3 tools/check-build-pipeline-baseline.py` passed 32 target/output routes (64 session/beacon cases), 10 edge cases, and 7 protocol cases. `python3 docs/build-pipeline/validate_examples.py` passed 14 immutable specs and 8 profiles; negative checks rejected unknown fields, invalid named-pipe target, duplicate YAML keys/options, invalid digest/version, and unsafe integers. These are source and contract checks, not compiled-implant or live E2E evidence. The user approved proceeding with M1 and M2 together. M0 remains In progress until an implementing PR and its test evidence are recorded.

### M1 — Catalog and introspection

- [x] Ship built-in Sliver and transform YAML with the server. Store administrator and remote spec YAML snapshots server-side, indexed by digest, version, source, provider, and last-seen state.
- [x] Validate specs on import and registration. Reject malformed YAML, unknown executors, duplicate name/version with different content, and invalid capability rules. Permit only installed executor IDs.
- [x] Add read-only catalog RPCs and `cores list`, `cores info <name>`, `transforms list`, and `transforms info <name>`. Show supported tuples, compiler requirements, source, digest, and available providers. Add target and availability filters and server-backed completions.
- [x] Preserve remote-advertised definitions after disconnect, marked unavailable. Keep live provider status separate from immutable spec content.

**Exit:** The console can inspect every built-in and registered spec; an offline remote spec remains visible and cannot be selected for a build. Restarting the server preserves administrator and remote snapshots.

**M1 evidence (2026-11-19, local):** `server/buildcatalog` embeds one Sliver core and six transforms; package tests pass for malformed/unknown spec rejection, immutable digest and conflict checks, atomic remote registration, administrator reload, and offline remote snapshots after restart. Catalog RPC tests pass for exact YAML, target/availability filtering, and online/offline remote inventory. The `cores` and `transforms` console commands and server-backed completions compile and their filter tests pass. See the [implementation guide](docs/build-pipeline/m1-m2-implementation.md). No connected live builder or console session was exercised.

### M2 — Planner and build records

- [x] Add server-side plan and execution APIs with a neutral build request, typed artifacts, step states, progress, and failure details. Keep existing Generate RPCs as compatibility entry points during migration.
- [x] Resolve requested final output to a compatible core output and ordered transform path. Honor explicit steps, reject ambiguous or incompatible paths, and expose a non-executing `generate --plan` preview.
- [x] Map current `generate` flags to `builtin/sliver` by default. Reject unknown formats with a capability error rather than silently selecting an executable.
- [x] Store the original neutral request, resolved step graph, spec digests, selected providers, toolchain identity, artifact hashes, and final output metadata with each M2 local build. Preserve final bytes so `regenerate` retrieves the recorded output, regardless of later profile or catalog edits. Remote step records extend this contract in M5.

**Exit:** Planning produces the same route on repeated identical inputs, identifies missing capabilities before compile, and never mutates a stored plan when a spec or profile changes.

**M2 evidence (2026-11-19, local):** Planner tests pass for direct and Windows/macOS/Linux shellcode routes, explicit encoder suffixes, ambiguity, capability and toolchain rejection, deterministic plan hashes, and frozen stored plans. RPC tests pass for non-executing repeat previews, typed records and checksum-checked artifacts, failed-record states, and revised final bytes. Console tests cover format errors, WireGuard preview without IP allocation, beacon mode, and debug/obfuscation parity. `./go-tests.sh --offline-build-pipeline` passed the five changed package groups using `sliver_lint` assets; tagged `go build` and `python3 tools/check-build-pipeline-baseline.py` passed. `make pb` regenerated the stubs and changed Go files were formatted. The normal `./go-tests.sh --unit-only` gate stopped before tests because generated asset ZIPs were absent; `make .downloaded_assets` packed Go assets but failed to download/verify Zig 0.15.2 from configured mirrors. The full unit, compilation, and live implant E2E gates remain unverified.

The M2 compatibility compiler reports route steps after its monolithic call, so only its final artifact has an observed byte hash. Optional shellcode encoding remains outside the recorded server pipeline until M4. A rare post-build record-write failure can leave the record artifact at an older revision; `regenerate` falls back to the database artifact when its checksum differs. The M1/M2 milestone statuses remain In progress until an implementing PR and passing normal test evidence are linked.

### M3 — Sliver core isolation

- [x] Move Sliver source embedding, template rendering, dependency/vendor setup, certificate and key preparation, and output-specific compilation behind a Sliver core adapter. The generic planner and executor do not import Sliver source packages or interpret Sliver-only `ImplantConfig` flags.
- [x] Keep Go execution behind a compiler adapter using the existing Go, Zig, and Garble toolchain behavior. Leave an explicit driver boundary for a future compiler type.
- [x] Move server/client imports of implant-only transport constants or helpers into neutral shared packages where needed. Keep the implant's dependency on the shared `sliverpb` wire contract explicit.
- [x] Preserve direct executable, service, shared-library, and `c-archive` bundle behavior, including current entry exports and filenames. Declare only proven target/output tuples in the built-in spec.

**Exit:** Direct Sliver builds pass the baseline matrix and keep the same runtime behavior. A new core adapter can be registered without changing Sliver generation functions or generic planner code.

**M3 evidence (2026-11-19, local):** The adapter registry, neutral executor, Go driver, legacy wrappers, and synthetic alternate-core RPC test pass `./go-tests.sh --offline-build-pipeline`. The preferred `docker build --target test --build-arg GO_TESTS_FLAGS=--skip-generate .` passed on Linux/arm64 after compiling server and client and running the normal package and C2 test suite. Targeted `./go-tests.sh --generate-only` runs with unpacked Go, Zig, and Garble assets compiled 46 direct cases available on that image: 16 executables, 6 Windows services, 12 shared libraries, and 12 C archive bundles, spanning session and beacon. The tests check output names, PE/ELF/Mach-O structure, service and `StartW` entrypoints, archive contents, and ZIP checksums. A real `ExecuteBuild` test compiled a native executable and verified its stored SHA-256, `GetBuildArtifact` bytes, and legacy `regenerate` bytes. The existing `TestComprehensiveE2E` driver passed on Linux/arm64 with native mTLS session and beacon implants, including live connection and command execution (1024.94 seconds). The initial archive run exposed false Darwin cross-compiler availability; `GetCompilerTargets` now checks for an installed compiler before advertising cross-OS CGo outputs. The Linux image has no osxcross, so Darwin shared libraries and archives are unavailable there; native Darwin validation still requires a macOS runner. `python3 tools/check-build-pipeline-baseline.py` passed the 32-route static inventory. M3 remains In progress until the roadmap's implementing PR and passing CI evidence are linked.

### M3.5 — Private source and Mage build handoff

- [x] Stage each core's source byte-for-byte in a unique private app-directory workspace. Sliver explicitly renders only its own staged copy; another installed core can skip rendering and keep literal template delimiters.
- [x] Prepare server-owned implant cryptographic material before compile and pass a private material-file reference to the core build. Keep secret values out of command arguments, environment values, and generic build records.
- [x] Let an installed core bind a digest-pinned Magefile and target. Compile the trusted Magefile programmatically for the server host, then run its target per build with cancellation and a private manifest; preserve Mage CLI compatibility.
- [x] Accept one declared primary output from `bin/result.json`, validate its frozen artifact type and target, compute size and SHA-256, and hand typed bytes to the rest of the pipeline.
- [x] Bound and validate source ZIP/TAR extraction and produced ZIP/AR inspection; reject traversal, links, duplicate names, malformed members, and excessive expansion.

**Exit:** Direct Sliver output builds use Mage with unchanged compiler flags and runtime behavior; an alternate core with its own Magefile builds from a literal copied source tree. Produced archives cannot escape their workspace or bypass artifact validation. Run the direct build matrix and a live native session/beacon exercise before treating runtime parity as proven.

**M3.5 evidence (2026-11-19, local):** `./go-tests.sh --offline-build-pipeline` passed the workspace, Mage runner, artifact acceptance, catalog, compiler, adapter, archive safety, RPC, and console tests. The schema validator accepted 14 immutable specs and 8 profiles. A Linux/arm64 Docker build using the current source and the previously bundled toolchain assets unpacked Go and Zig with the hardened extractors, then passed all 46 advertised direct build cases (16 executables, 6 services, 12 shared libraries, 12 C archive bundles) across session and beacon modes. The native `ExecuteBuild` test passed artifact retrieval, SHA-256, legacy regeneration, and a new assertion that the private workspace was removed. The comprehensive E2E driver passed native mTLS session and beacon connection and command checks with a 1-second beacon interval (`119.37s`). Live E2E for other transports and native macOS/Windows remains unverified for M3.5.

See [the source and Mage handoff](docs/build-pipeline/mage-core-builds.md). M3.5 remains In progress until an implementing PR and passing CI evidence are linked. M4 owns execution and recording of every transform as a separate step.

### M4 — Transform pipeline and final artifacts

- [x] Wrap Donut, Beignet, and Malasada as typed transforms with their existing platform, architecture, entry-symbol, and option constraints.
- [x] Wrap Shikata Ga Nai, XOR, and dynamic XOR as shellcode-to-shellcode transforms with the current architecture matrix. Retain existing encoder RPCs as compatibility wrappers.
- [x] Move generation-time post-processing, including applicable PE metadata spoofing, into ordered server-owned steps. Represent `profiles stage` compression, encryption, and size framing as typed steps while keeping listener setup separate from artifact creation.
- [x] Save and checksum the final post-transform bytes on the server before returning them. Keep intermediate artifacts scoped to the build and apply an explicit retention policy.

**Exit:** Existing Windows, macOS, and Linux shellcode requests resolve to their current core artifact and converter. The downloaded file, stored final file, and recorded SHA-256 are identical after optional encoding or other requested transforms.

**M4 evidence (2026-11-19, local):** The resolved executor records each observed intermediate artifact, runs pinned local converters and encoders, and stores only final bytes. Legacy `Generate`, `GenerateStage`, and generation-time PE metadata use the same ordered executor; neutral `ExecuteBuild` carries bounded per-build material that is excluded from the stored plan. `./go-tests.sh --offline-build-pipeline`, Docker `./go-tests.sh --skip-generate`, the 32-route baseline inventory, schema example validation, and `git diff --check` passed. Separate Linux/arm64 Docker generation gates compiled Linux/amd64 and Windows/amd64 shellcode with XOR, a Windows stage with size framing, zlib, and AES, and a Bufnet PE metadata build. The tests verified returned, stored, regenerated, and recorded final bytes and SHA-256. Planner and adapter tests cover the macOS arm64 Beignet route and the platform/architecture matrix; native macOS shellcode execution remains unverified. M4 remains In progress until an implementing PR and passing CI evidence are linked.

### M5 — Remote execution and resource-aware selection

- [x] Extend builder registration/status messages to advertise exact spec digests, capability tuples, toolchain availability, effective CPU cores, and available memory. Refresh dynamic capacity without replacing immutable spec definitions.
- [x] Add authenticated, assigned step requests and typed artifact exchange. The server validates compatibility at dispatch and validates uploaded results before accepting the next step. Keep the old event/file RPCs for legacy configurations until their neutral options are represented.
- [x] Select a compatible remote builder by rounded available decimal MB, then effective CPU cores, then stable name. Support an optional builder pin in commands and profiles. Exclude disconnected or stale-status builders from automatic selection.
- [x] Prevent concurrent assignments from treating the same reported free memory as uncommitted capacity. Record the report and selection decision for each step.
- [x] Expose the reported CPU cores, rounded available MB, report age, spec support, and availability in builder inventory.

**Exit:** Automated and live two-builder exercises prove memory ordering, CPU and name ties, optional pinning, stale/offline exclusion, and concurrent assignment behavior. A wrong builder, spec, target, artifact type, or checksum cannot complete an assigned step.

**M5 evidence (2026-11-19, local):** The [M5 implementation guide](docs/build-pipeline/m5-remote-execution.md) describes registration, status, selection, reservations, assigned steps, and compatibility routing. `./go-tests.sh --offline-build-pipeline` passed 19 package groups on macOS and Linux against the final source. Scheduler tests cover decimal MB rounding, memory/CPU/name ties, pins, stale/offline exclusion, target-specific toolchains, concurrent and exclusive unknown-estimate reservations. Builder tests cover Linux cgroup CPU and memory ancestors, process affinity, and failed capacity readings. Builder and RPC tests reject wrong builder identity, spec, target, artifact type, checksum, and toolchain identity; dedicated notification tests cover broker saturation, full queues, disconnect cancellation, and worker reconnection after quarantine. `TestTwoBuilderAssignedStepStreams` tests two authenticated registration streams and the recorded candidate order. `TestTwoBuilderLiveWorkersExecuteAssignedTransforms` starts two actual builder workers in-process; they fetch assigned steps, execute XOR, submit typed outputs, and route the second assignment to the other builder while the first has an exclusive reservation. A private-network Docker exercise rebuilt the final source, ran the server and builder in separate containers, pinned and compiled a `builtin/sliver` Linux/amd64 core, and retrieved a 15,503,587-byte ELF with SHA-256 `71cb76bfb62255378521a71d6f704f716c70b64b9c53e18f3f3a8fdfc10ce0d5`; the client verified its ELF magic, size, and checksum against the successful build record. The Docker test target built server/client and passed the focused suite; the final source passed the same suite in a Linux container with the repository mounted. Tagged server and builder builds, `make pb`, and `git diff --check` passed. Remote validation in CI remains unverified, so M5 stays In progress.

### M6 — YAML profiles and existing consumers

- [x] Store canonical BuildProfile YAML on the server; add profile import, export, validation, and readable detail commands. Keep `profiles new` and `profiles new beacon` as flag/form front ends that save the equivalent YAML.
- [x] Backfill existing `ImplantConfig` profiles non-destructively and idempotently. Keep older client requests working while the new profile APIs are adopted.
- [x] Make `profiles generate` execute its full ordered pipeline. Revalidate any permitted command-line override and store the resulting per-build plan.
- [x] Update profile consumers, including stage listeners, `psexec`, backdoor, and DLL hijack, to check the profile's final artifact type rather than legacy format flags.
- [x] Decide profile deletion semantics before migration: recommended behavior is to preserve completed builds and snapshots, with an explicit artifact purge action.

**Exit:** Existing profiles still generate their previous output; YAML import/export round-trips; a profile edit or deletion cannot silently change a completed build's frozen plan. Profile consumers accept or reject pipelines by final output type.

**M6 evidence (2026-11-19, local and Linux/arm64):** The [M6 profile guide](docs/build-pipeline/m6-yaml-profiles.md) documents import, validation, export, generation, per-build inputs, migration diagnostics, stage behavior, consumer types, and deletion. `./go-tests.sh --offline-build-pipeline` passed 23 focused package groups on the final source, including canonical YAML round-trip, 255-byte name boundaries, idempotent backfill, revisions and tombstones, secret-free legacy snapshots, complete migrated transform-sequence checks, profile deletion retaining completed builds, the client save-path regression, and all four consumer compatibility checks. The preferred Docker test target with `GO_TESTS_FLAGS=--skip-generate` compiled the server and client and passed the full Linux/arm64 unit and C2 transport suite; a mounted final-source Linux run passed the 23 focused package groups after the client save-path fix. Final Docker `./go-tests.sh --generate-only --run 'TestM6(AuthoredProfileBuildFreezesSnapshot|MigratedProfileBuildPreservesLegacyConfig)'` built an authored YAML native core and a migrated beacon profile through the new profile RPC (23.475s and 8.776s). Those tests verified final artifact bytes and SHA-256, preserved legacy beacon/C2 configuration, and checked that editing or deleting a profile does not change its completed record. `make pb`, `gofmt`, and `git diff --check` passed. No live implant C2 connection was exercised specifically for M6; the earlier M3 and M3.5 live Sliver exercises remain the runtime baseline. M6 remains In progress until an implementing PR and passing CI evidence are linked.

### M7 — Alternate core and release acceptance

- [ ] Add a preinstalled non-Sliver core integration or representative test fixture that speaks `sliver-v1`, declares its real capability subset, and builds locally and through a remote provider. Do not add source-bundle upload or a second runtime protocol in this milestone.
- [ ] Document spec authoring, installed executor registration, profile YAML, builder metrics, automatic selection and pinning, migration, and diagnostics.
- [ ] Complete the compatibility and E2E matrix below and record exact commands, platforms, results, and remaining coverage gaps. Remove legacy paths only after their replacement and compatibility tests pass.

**Exit:** An alternate core works without changes to generic planner or Sliver adapter code, and the final artifact is usable through the existing Sliver server protocol. All release gates below have recorded evidence.

## Release gates and evidence

| Area | Required evidence |
| --- | --- |
| Schema and planning | Invalid mode/protocol/target/output combinations, unknown executors, conflicting versions, cycles, and transform type mismatches are rejected before build. |
| Legacy commands | `generate`, `generate beacon`, and `profiles generate` preserve defaults and supported direct formats; deprecated flags retain their documented translation. |
| Artifact parity | Executable, service, shared-library, C archive, and supported shellcode paths compile; final downloaded bytes equal stored bytes and SHA-256 after every transform. |
| Runtime | Representative session and beacon outputs connect over supported transports; a non-Sliver core proves the shared wire contract. Record build-only versus live E2E coverage separately for each target. |
| Remote builders | Exact capability matching, memory/CPU/name ranking, decimal-MB rounding, fresh-status rules, pinning, assignment authorization, upload validation, and concurrent scheduling are exercised. |
| Profiles | Legacy database migration is idempotent; YAML round-trips; all profile consumers use final artifact compatibility; completed build snapshots remain stable. |
| Go tests and CI | Regenerate changed protobuf stubs with `make pb` and format Go with `gofmt`. Run Go unit tests only through `./go-tests.sh`, including its full generation path for build changes. Use the preferred Docker test target where available. Extend `go-tests.sh --generate-only` if heavy tests move outside `server/generate`, and extend `.github/workflows/unit-tests.yml` changed-path filters so new build packages trigger generation tests on Linux, macOS, and Windows. |

Do not count a successful compile as a live implant E2E result. The existing shellcode E2E workflow covers a smaller target set than the generator's advertised support; track additional target coverage explicitly rather than inferring it from the workflow name.

## Updating this roadmap

When work is scheduled, assign an owner and change `Proposed` to `In progress`. Mark a milestone `Complete` only after checking its finished tasks and linking the implementing PR and passing test or E2E evidence here. Keep unresolved behavior changes visible until decided. Add new compiler drivers, source-bundle upload, and alternative runtime protocols only as separately scoped follow-on milestones.
