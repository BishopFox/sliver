# Build pipeline acceptance matrix

This matrix separates source validation, artifact compilation, assigned remote
steps, and live implant behavior. Results labeled **recorded M0–M6** were run
before M7 changes; they must be rerun when their affected source changes.
The [roadmap](../../ROADMAP.md) records the underlying milestone evidence.

## Recorded compatibility evidence

| Gate | Exact command or recorded test | Platform | Result | Scope still open |
| --- | --- | --- | --- | --- |
| Legacy route inventory | `python3 tools/check-build-pipeline-baseline.py` | Local macOS, M0/M3 | Passed 32 target/output routes (64 session/beacon cases), 10 edge cases, and 7 protocol cases. | Static inventory; no artifact or connection. |
| Spec and profile examples | `python3 docs/build-pipeline/validate_examples.py` | Local macOS, M3.5/M4 | Passed 14 immutable specs and 8 profiles, including digest, schema, duplicate-key/option, and artifact-edge checks. | Rerun after adding the M7 spec; the host must have PyYAML and `jsonschema`. |
| Focused build-system tests | `./go-tests.sh --offline-build-pipeline` | macOS and Linux/arm64, M6 | Passed 23 package groups on the final M6 source. Includes catalog, planner, Mage/workspace/archive safety, transforms, scheduler, builder/RPC assignment, profiles, and client consumer checks. | No compiler assets are unpacked in this mode. Rerun after M7. |
| Full non-generation suite | `DOCKER_CONFIG=/tmp/sliver-m6-docker-config docker buildx build --target test --build-arg GO_TESTS_FLAGS=--skip-generate -t sliver-m6-full-test:local --load .` | Linux/arm64 Docker, M6 | Passed server/client build, unit packages, and C2 transport tests. The final save-path edit also passed a mounted-source Linux focused run. | This skips implant compilation; rerun on M7 source. The temporary Docker config path was specific to that run. |
| Direct Sliver artifacts | `TestSliverDirectFormatsAdvertisedTargets`, `TestSliverExecutablesAdvertisedTargets`, and `TestSliverArchiveAdvertisedTargets` through `./go-tests.sh --generate-only` | Linux/arm64 Docker, M3.5 | Passed 46 advertised direct cases: 16 executables, 6 services, 12 shared libraries, and 12 C archive bundles across session and beacon. Structure, exports, names, headers, and ZIP checksums were checked. | The exact prior `--run` regex was not retained. Native Darwin shared library/archive requires a macOS runner; live execution was not asserted for all 46. |
| Ordered transforms | `TestM4GenerateShellcodeFinalParity` and `TestM4GenerateStageFinalParity` through `./go-tests.sh --generate-only` | Linux/arm64 Docker, M4 | Compiled Linux/amd64 and Windows/amd64 shellcode with XOR, plus a Windows stage with length framing, zlib, and AES. Returned, stored, regenerated, and recorded final bytes and SHA-256 agreed. | macOS arm64 Beignet is planner/adapter tested, not native runtime tested. The exact prior `--run` command was not retained. |
| YAML profile builds | `./go-tests.sh --generate-only --run 'TestM6(AuthoredProfileBuildFreezesSnapshot|MigratedProfileBuildPreservesLegacyConfig)'` | Linux/arm64 Docker, M6 | Passed native authored core compilation (23.475s) and migrated beacon compilation (8.776s). Verified final bytes/SHA-256, legacy beacon/C2 settings, and a completed record after profile edit/deletion. | No M6-specific live C2 connection. |
| Two-builder scheduling and exchange | `./go-tests.sh --offline-build-pipeline` (`TestTwoBuilderAssignedStepStreams`, `TestTwoBuilderLiveWorkersExecuteAssignedTransforms`) | macOS and Linux/arm64, M5/M6 | Passed memory/CPU/name ordering, pins, stale/offline exclusion, reservations, authenticated assignment, typed XOR output, and rejection of wrong identity/spec/type/checksum. | The stream test simulates uploads; the live worker test runs two workers in-process. |
| Two-process remote Sliver core | Private-network Docker server/builder exercise recorded in [M5 evidence](../../ROADMAP.md) | Linux/amd64 target, Linux Docker, M5 | Pinned remote `builtin/sliver` compile returned a 15,503,587-byte ELF; the client checked magic, size, and SHA-256 `71cb76bfb62255378521a71d6f704f716c70b64b9c53e18f3f3a8fdfc10ce0d5` against the record. | The exact ad hoc Docker invocation was not retained; rerun with a reproducible command for release evidence. |
| Live Sliver mTLS | `TestComprehensiveE2E` driver recorded in [M3.5 evidence](../../ROADMAP.md) | Linux/arm64 Docker, M3.5 | Native session and beacon connected and executed command checks (119.37s). | The exact prior invocation was not retained. Other C2 transports and native Windows/macOS runtime are separate gates. |

The test names in the direct-artifact and transform rows identify the tests
that ran. They are not a reconstructed claim about the exact command line used
in those earlier runs.

## Final M7 source verification

The Linux/arm64 generation run used the completed
`sliver-m7-clean-test:local` image, built from the two-commit
`feat/implant-build-pipeline-clean` branch based on `origin/master`. The
macOS/arm64 run used compiler archives temporarily copied from that image;
they were removed from the managed worktree afterward. All Go tests were
invoked through `./go-tests.sh`.

| Gate | Exact command | Platform | Result and scope |
| --- | --- | --- | --- |
| Legacy route inventory | `python3 tools/check-build-pipeline-baseline.py` | macOS/arm64 | Passed 32 routes (64 session/beacon cases), 10 edge cases, and 7 protocol cases. Static compatibility check. |
| Spec and profile examples | `PYTHONPATH=/tmp/sliver-schema-validate PYTHONDONTWRITEBYTECODE=1 python3 docs/build-pipeline/validate_examples.py` | macOS/arm64 | Passed 15 immutable specs and 9 profiles, including the M7 minimal core and authored profile. |
| Focused build-system suite | `./go-tests.sh --offline-build-pipeline` | macOS/arm64 and Linux/arm64 Docker | Passed 27 package groups on each platform after review, including disconnect cleanup for pending legacy external builds. Covers catalog, planning, source copy, archive safety, remote assignment, saves, profile and client compatibility, and the minimal core fixture. |
| Preferred full test target | `DOCKER_CONFIG=/tmp/sliver-m6-docker-config docker buildx build --target test --build-arg GO_TESTS_FLAGS=--skip-generate -t sliver-m7-clean-test:local --load --progress=plain .` | Linux/arm64 Docker | Passed server/client builds and 106 test groups, including C2 transport tests. Generation is deliberately skipped in this target. |
| Full generation path | `docker run --rm -e GOPROXY=off sliver-m7-clean-test:local ./go-tests.sh --generate-only` | Linux/arm64 Docker | Passed the complete `server/generate` package (428.413s) and 10 tagged RPC artifact tests. Those include native Sliver, shellcode/stage parity, M6 profiles, and all four M7 cases. |
| M7 target matrix | `TestM7MinimalCoreAdvertisedTargets` within full generation | Linux/arm64 Docker; Linux/amd64, Linux/arm64, Darwin/arm64 targets | Passed ELF/Mach-O target checks and equality of returned, stored, retrieved bytes and SHA-256 for each advertised target. These matrix checks are build-only. |
| M7 local live session | `TestM7MinimalCoreLiveMTLSPing` within full generation | Linux/arm64 Docker | The Mage-built alternate core registered over loopback mTLS and answered `MsgPing`; final artifact bytes and record/retrieval hash matched. |
| M7 in-process remote worker | `TestM7RemoteMinimalCoreLiveWorker` within full generation | Linux/arm64 Docker | Authenticated assignment used the exact installed spec and bounded material, then produced a recorded artifact that registered and answered Ping. Worker and server ran in one test process. |
| M7 separate-process remote worker | `TestM7RemoteMinimalCoreSeparateProcessWorker` within full generation | Linux/arm64 Docker | A separate worker process connected over authenticated localhost gRPC, fetched an exact-digest assignment, built and submitted the core, and its recorded artifact registered and answered Ping. |
| Native macOS live sessions | `./go-tests.sh --generate-only --run '^TestM7(MinimalCoreLiveMTLSPing|RemoteMinimalCoreLiveWorker|RemoteMinimalCoreSeparateProcessWorker)$'` | macOS/arm64 | All three passed on the clean branch. The separate worker was PID 6699, used digest `sha256:6e6654928672f3862aa49f14639b9f038dbf7e661223644e3b3f8586de10b78c`, and produced an artifact with SHA-256 `ed75321324c16b7ea5f674335322d9768f20bf9bd78843b13c01261e034e2213` that registered and answered Ping. The hash identifies this run, not a reproducible binary: each build receives new cryptographic material. |

The exact Linux generation container invocation was:

```sh
docker run --rm -e GOPROXY=off sliver-m7-clean-test:local \
  ./go-tests.sh --generate-only
```

## Remaining release coverage

- The M7 alternate core declares only session executables over mTLS. Its live
  Linux/arm64 and Darwin/arm64 sessions passed; Linux/amd64 was built and
  inspected, but not launched. No alternate-core beacon, Windows target, other
  output type, or other C2 transport is claimed.
- Existing Sliver-core direct formats, transforms, profiles, remote builders,
  and session/beacon transport coverage are recorded above. Native Windows
  compilation and live Windows artifact behavior still require CI or a Windows
  runner. The generator's advertised shellcode target set exceeds the
  shellcode E2E workflow's smaller runtime set.
- Linux Docker did not provide osxcross for Darwin CGo outputs. Native macOS
  shared-library/archive and macOS arm64 Beignet generation remain separate
  gates.
- Legacy external-builder disconnect now revokes pending assignments and
  authentication rows. A server crash before stream teardown can still leave
  a pre-saved legacy build row: assignments are memory-only, and other paths
  also create checksum-empty temporary rows. Safe crash recovery needs a
  persisted pending-assignment lease or a change to pre-save semantics.
- No implementing PR, GitHub Actions run on the final M7 commit, or terminal
  cross-platform CI result exists. Under the roadmap's publication rule M7
  remains In progress until those links and remaining matrix decisions are
  recorded.
